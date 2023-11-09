package discord_bot

import (
	"errors"
	"fmt"
	"log"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	developmentMode    bool
	botSession         *discordgo.Session
	guildID            string
	imagineQueue       imagine_queue.Queue
	registeredCommands map[string]*discordgo.ApplicationCommand
	imagineCommand     string
	removeCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
	config             *Config
}

type Config struct {
	DevelopmentMode    bool
	BotToken           string
	GuildID            string
	ImagineQueue       imagine_queue.Queue
	ImagineCommand     string
	RemoveCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
}

var config *Config

func (b *botImpl) imagineCommandString() string {
	if b.developmentMode {
		return "dev_" + b.imagineCommand
	}

	return b.imagineCommand
}

func (b *botImpl) imagineSettingsCommandString() string {
	if b.developmentMode {
		return "dev_" + b.imagineCommand + "_settings"
	}

	return b.imagineCommand + "_settings"
}

func New(cfg *Config) (Bot, error) {
	config = cfg
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	if cfg.GuildID == "" {
		return nil, errors.New("missing guild ID")
	}

	if cfg.ImagineQueue == nil {
		return nil, errors.New("missing imagine queue")
	}

	if cfg.ImagineCommand == "" {
		return nil, errors.New("missing imagine command")
	}

	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = botSession.Open()
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		developmentMode:    cfg.DevelopmentMode,
		botSession:         botSession,
		imagineQueue:       cfg.ImagineQueue,
		registeredCommands: make(map[string]*discordgo.ApplicationCommand, 0),
		imagineCommand:     cfg.ImagineCommand,
		removeCommands:     cfg.RemoveCommands,
		StableDiffusionApi: cfg.StableDiffusionApi,
		config:             cfg,
	}

	err = bot.registerCommands()
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionMessageComponent:
			switch customID := i.MessageComponentData().CustomID; {
			case strings.HasPrefix(customID, upscaleButton):
				componentHandlers[upscaleButton](bot, s, i, customID)
			case strings.HasPrefix(customID, variantButton):
				componentHandlers[variantButton](bot, s, i, customID)
			}
		case discordgo.InteractionApplicationCommandAutocomplete:
			switch i.ApplicationCommandData().Name {
			case bot.imagineCommandString():
				bot.processImagineAutocomplete(s, i)
			}
		}
	})

	bot.registerHandlers(botSession)

	return bot, nil
}

func (bot *botImpl) registerHandlers(session *discordgo.Session) {
	session.AddHandler(func(session *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		// commands
		case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(bot, session, i)
			}
		//buttons
		case discordgo.InteractionMessageComponent:
			log.Printf("Component with customID `%v` was pressed, attempting to respond\n", i.MessageComponentData().CustomID)
			if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
				//bot.p.Send(logger.Message(fmt.Sprintf(
				//	"Handler found, executing on message `%v`\nRan by: <@%v>\nUsername: %v",
				//	i.Message.ID,
				//	i.Member.User.ID,
				//	i.Member.User.Username,
				//)))
				//bot.p.Send(logger.Message(fmt.Sprintf("https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)))
				h(bot, session, i)
			} else {
				log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
			}
		}
	})
	//currentProgress = len(commandHandlers) + len(componentHandlers) + len(components)
	//bot.p.Send(load.Goal{
	//	Current: currentProgress,
	//	Total:   totalProgress,
	//	Show:    true,
	//})
	//session.AddHandler(func(session *discordgo.Session, r *discordgo.Ready) {
	//	bot.p.Send(logger.Message(fmt.Sprintf("Logged in as: %v#%v", session.State.User.Username, session.State.User.Discriminator)))
	//})
}

func (bot *botImpl) registerCommands() error {
	bot.registeredCommands = make(map[string]*discordgo.ApplicationCommand, len(commands))
	commands[imagineCommand].Name = imagineCommand
	commands[imagineSettingsCommand].Name = imagineSettingsCommand
	for key, command := range commands {
		if command.Name == "" {
			// clean the key because it might be a description of some sort
			// only get the first word, and clean to only alphanumeric characters or -
			sanitized := strings.ReplaceAll(key, " ", "-")
			sanitized = strings.ToLower(sanitized)

			// remove all non valid characters
			for _, c := range sanitized {
				if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
					sanitized = strings.ReplaceAll(sanitized, string(c), "")
				}
			}
			command.Name = sanitized
		}
		cmd, err := bot.botSession.ApplicationCommandCreate(bot.botSession.State.User.ID, bot.guildID, command)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot create '%v' command: %v", command.Name, err))
		}
		bot.registeredCommands[key] = cmd
		//bot.p.Send(logger.Message(fmt.Sprintf("Registered command: %v", cmd.Name)))
		//currentProgress++
		//bot.p.Send(load.Goal{
		//	Current: currentProgress,
		//	Total:   totalProgress,
		//	Show:    true,
		//})
	}

	return nil
}

func (b *botImpl) Start() {
	b.imagineQueue.StartPolling(b.botSession)

	err := b.teardown()
	if err != nil {
		log.Printf("Error tearing down bot: %v", err)
	}
}

func (b *botImpl) teardown() error {
	// Delete all commands added by the bot
	if b.removeCommands {
		log.Printf("Removing all commands added by bot...")

		for _, v := range b.registeredCommands {
			log.Printf("Removing command '%v'...", v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.guildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	return b.botSession.Close()
}

func shortenString(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}
