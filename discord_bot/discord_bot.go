package discord_bot

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"sort"
	"strings"
	"sync"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/queue/llm"
	"stable_diffusion_bot/queue/novelai"
	"stable_diffusion_bot/queue/stable_diffusion"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	developmentMode bool
	botSession      *discordgo.Session

	registeredCommands map[Command]*discordgo.ApplicationCommand
	config             *Config
}

type Config struct {
	DevelopmentMode    bool
	BotToken           string
	GuildID            string
	ImagineQueue       queue.Queue[*stable_diffusion.SDQueueItem]
	NovelAIQueue       queue.Queue[*novelai.NAIQueueItem]
	LLMQueue           queue.Queue[*llm.LLMItem]
	ImagineCommand     *Command
	RemoveCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
}

func (b *botImpl) imagineCommandString() Command {
	if b.developmentMode && !strings.HasPrefix(imagineCommand, "dev_") {
		imagineCommand = fmt.Sprintf("dev_%v", strings.TrimPrefix(*b.config.ImagineCommand, "dev_"))
		return imagineCommand
	}

	return *b.config.ImagineCommand
}

func (b *botImpl) imagineSettingsCommandString() Command {
	if b.developmentMode && !strings.HasPrefix(imagineSettingsCommand, "dev_") {
		imagineSettingsCommand = fmt.Sprintf("dev_%v_settings", strings.TrimPrefix(*b.config.ImagineCommand, "dev_"))
		return imagineSettingsCommand
	}

	return fmt.Sprintf("%v_settings", *b.config.ImagineCommand)
}

func New(cfg *Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	handlers.Token = &cfg.BotToken

	if cfg.GuildID == "" {
		//return nil, errors.New("missing guild ID")
		log.Printf("Guild ID not provided, commands will be registered globally")
	}

	if cfg.ImagineQueue == nil {
		return nil, errors.New("missing imagine queue")
	}

	if cfg.ImagineCommand == nil || *cfg.ImagineCommand == "" {
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
		registeredCommands: make(map[Command]*discordgo.ApplicationCommand),
		config:             cfg,
	}

	//Read the imagineCommand from the config and remake the maps
	bot.customImagineCommand()

	if bot.config.NovelAIQueue == nil {
		delete(commands, novelAICommand)
	}

	err = bot.registerCommands()
	if err != nil {
		return nil, err
	}

	bot.registerHandlers(botSession)

	return bot, nil
}

func (b *botImpl) registerHandlers(session *discordgo.Session) {
	session.AddHandler(func(session *discordgo.Session, i *discordgo.InteractionCreate) {
		var handler func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error
		var ok bool
		switch i.Type {
		// commands
		case discordgo.InteractionApplicationCommand:
			handler, ok = commandHandlers[i.ApplicationCommandData().Name]
			//If we're using *Command, we have to range through the map to dereference the Command string
			//for key, command := range commandHandlers {
			//	if string(*key) == i.ApplicationCommandData().Name {
			//		h = command
			//		ok = true
			//	}
			//}
		// buttons
		case discordgo.InteractionMessageComponent:
			log.Printf("Component with customID `%v` was pressed, attempting to respond\n", i.MessageComponentData().CustomID)
			handler, ok = componentHandlers[handlers.Component(i.MessageComponentData().CustomID)]
			//bot.p.Send(logger.Message(fmt.Sprintf(
			//	"Handler found, executing on message `%v`\nRan by: <@%v>\nUsername: %v",
			//	i.Message.ID,
			//	i.Member.User.ID,
			//	i.Member.User.Username,
			//)))
			//bot.p.Send(logger.Message(fmt.Sprintf("https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)))

			if !ok {
				switch customID := i.MessageComponentData().CustomID; {
				case strings.HasPrefix(customID, string(handlers.UpscaleButton)):
					handler, ok = componentHandlers[handlers.UpscaleButton]
				case strings.HasPrefix(customID, string(handlers.VariantButton)):
					handler, ok = componentHandlers[handlers.VariantButton]
				default:
					log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
				}
			}
		// autocomplete
		case discordgo.InteractionApplicationCommandAutocomplete:
			handler, ok = autocompleteHandlers[i.ApplicationCommandData().Name]
		// modals
		case discordgo.InteractionModalSubmit:
			handler, ok = modalHandlers[i.ModalSubmitData().CustomID]
		default:
			log.Printf("Unknown interaction type '%v'", i.Type)
		}

		if !ok || handler == nil {
			var interactionType string = "unknown"
			var interactionName string = "unknown"
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				interactionType = "command"
				interactionName = i.ApplicationCommandData().Name
			case discordgo.InteractionMessageComponent:
				interactionType = "component"
				interactionName = i.MessageComponentData().CustomID
			case discordgo.InteractionApplicationCommandAutocomplete:
				interactionType = "autocomplete"

				data := i.ApplicationCommandData()
				for _, opt := range data.Options {
					if !opt.Focused {
						continue
					}
					interactionName = fmt.Sprintf("command: /%v option: %v (%v)", data.Name, opt.Name)
					break
				}
			case discordgo.InteractionModalSubmit:
				interactionType = "modal"
				interactionName = i.ModalSubmitData().CustomID
			}
			log.Printf("WARNING: Cannot find handler for interaction [%v] '%v'", interactionType, interactionName)
			return
		}

		err := handler(b, session, i)

		var username string = "unknown"
		if i.Member != nil {
			username = i.Member.User.Username
		}
		if i.User != nil {
			username = i.User.Username
		}

		if err != nil {
			if errors.Is(err, handlers.ResponseError) {
				log.Printf("Error responding to interaction for %s: %v", username, err)
				return
			}
			err := handlers.ErrorEdit(session, i.Interaction, err)
			if err != nil {
				log.Printf("Error showing error message to user %s: %v", username, err)
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

func (b *botImpl) registerCommands() error {
	b.registeredCommands = make(map[Command]*discordgo.ApplicationCommand, len(commands))
	for key, command := range commands {
		if command.Name == llmCommand && b.config.LLMQueue == nil {
			continue
		}
		if command.Name == novelAICommand && b.config.NovelAIQueue == nil {
			continue
		}
		if command.Name == "" {
			// clean the key because it might be a description of some sort
			// only get the first word, and clean to only alphanumeric characters or -
			sanitized := strings.ReplaceAll(key, " ", "-")
			sanitized = strings.ToLower(sanitized)

			// remove all non-valid characters
			for _, c := range sanitized {
				if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
					sanitized = strings.ReplaceAll(sanitized, string(c), "")
				}
			}
			command.Name = sanitized
		}
		//b.controlnetTypes()
		cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.config.GuildID, command)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot create '%v' command: %v", command.Name, err))
		}
		b.registeredCommands[key] = cmd
		log.Printf("Registered %v command as: /%v", key, cmd.Name)
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

// customImagineCommand is used to read the imagineCommand from the config and remake the maps
// the keys are copied from the commands and commandHandlers map, deleted, and then re-added with the new command
func (b *botImpl) customImagineCommand() {
	//imagine
	b.rebuildMap((*botImpl).imagineCommandString, &imagineCommand, commands, commandHandlers)

	//imagine_settings
	b.rebuildMap((*botImpl).imagineSettingsCommandString, &imagineSettingsCommand, commands, commandHandlers)
}

func (b *botImpl) rebuildMap(f func(*botImpl) Command, key *Command, m map[Command]*discordgo.ApplicationCommand, h map[Command]Handler) {
	oldKey := *key

	*key = f(b)
	if *key == oldKey {
		return
	}
	log.Printf("Rebuilding map for '%v' to '%v'", oldKey, *key)

	m[*key] = m[oldKey]
	m[*key].Name = *key
	h[*key] = h[oldKey]
	delete(m, oldKey)
	delete(h, oldKey)
}

func (b *botImpl) Start() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	queues := []queue.StartStop{
		b.config.ImagineQueue,
		b.config.NovelAIQueue,
		b.config.LLMQueue,
	}

	slices.DeleteFunc(queues, IsNil)
	for _, q := range queues {
		go q.Start(b.botSession)
	}

	if len(queues) == 0 {
		log.Println("No queues to start, exiting...")
		stop <- os.Interrupt
	} else {
		log.Println("Press Ctrl+C to exit")
	}

	<-stop
	var wg sync.WaitGroup
	for _, q := range queues {
		wg.Add(1)
		go func(q queue.StartStop) {
			q.Stop()
			wg.Done()
		}(q)
	}
	wg.Wait()

	err := b.teardown()
	if err != nil {
		log.Printf("Error tearing down bot: %v", err)
	}
}

func IsNil(q queue.StartStop) bool {
	return q == nil
}

func (b *botImpl) teardown() error {
	// Delete all commands added by the bot
	if b.config.RemoveCommands {
		log.Printf("Removing all commands added by bot...")

		for key, v := range b.registeredCommands {
			log.Printf("Removing command [key:%v], '%v'...", key, v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.config.GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	return b.botSession.Close()
}

// Deprecated: If we want to dynamically update the controlnet types, we can do it here
func (b *botImpl) controlnetTypes() {
	if false {
		controlnet, err := stable_diffusion_api.ControlnetTypesCache.GetCache(b.config.StableDiffusionApi)
		if err != nil {
			log.Printf("Error getting controlnet types: %v", err)
			panic(err)
		}
		// modify the choices of controlnetType by using the controlnetTypes cache
		var keys map[string]bool = make(map[string]bool)
		for key := range controlnet.(*stable_diffusion_api.ControlnetTypes).ControlTypes {
			if keys[key] {
				continue
			}
			keys[key] = true

			commandOptions[controlnetType].Choices = append(commandOptions[controlnetType].Choices,
				&discordgo.ApplicationCommandOptionChoice{
					Name:  key,
					Value: key,
				})
			if len(commandOptions[controlnetType].Choices) >= 25 {
				break
			}
		}
		sort.Slice(commandOptions[controlnetType].Choices, func(i, j int) bool {
			return cmp.Less(commandOptions[controlnetType].Choices[i].Name, commandOptions[controlnetType].Choices[j].Name)
		})
	}
}

func shortenString(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}
