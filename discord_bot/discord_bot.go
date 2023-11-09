package discord_bot

import (
	"errors"
	"log"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	developmentMode    bool
	botSession         *discordgo.Session
	guildID            string
	imagineQueue       imagine_queue.Queue
	registeredCommands []*discordgo.ApplicationCommand
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
		registeredCommands: make([]*discordgo.ApplicationCommand, 0),
		imagineCommand:     cfg.ImagineCommand,
		removeCommands:     cfg.RemoveCommands,
		StableDiffusionApi: cfg.StableDiffusionApi,
		config:             cfg,
	}

	err = bot.addImagineCommand()
	if err != nil {
		return nil, err
	}

	err = bot.addImagineSettingsCommand()
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case bot.imagineCommandString():
				bot.processImagineCommand(s, i)
			case bot.imagineSettingsCommandString():
				bot.processImagineSettingsCommand(s, i)
			default:
				log.Printf("Unknown command '%v'", i.ApplicationCommandData().Name)
			}
		case discordgo.InteractionMessageComponent:
			switch customID := i.MessageComponentData().CustomID; {
			case customID == "imagine_reroll":
				bot.processImagineReroll(s, i)
			case strings.HasPrefix(customID, "imagine_upscale_"):
				interactionIndex := strings.TrimPrefix(customID, "imagine_upscale_")

				interactionIndexInt, err := strconv.Atoi(interactionIndex)
				if err != nil {
					log.Printf("Error parsing interaction index: %v", err)

					return
				}

				bot.processImagineUpscale(s, i, interactionIndexInt)
			case strings.HasPrefix(customID, "imagine_variation_"):
				interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

				interactionIndexInt, err := strconv.Atoi(interactionIndex)
				if err != nil {
					log.Printf("Error parsing interaction index: %v", err)

					return
				}

				bot.processImagineVariation(s, i, interactionIndexInt)
			case customID == dimensionSelect:
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine dimension setting menu")

					return
				}

				sizes := strings.Split(i.MessageComponentData().Values[0], "_")

				width := sizes[0]
				height := sizes[1]

				widthInt, err := strconv.Atoi(width)
				if err != nil {
					log.Printf("Error parsing width: %v", err)

					return
				}

				heightInt, err := strconv.Atoi(height)
				if err != nil {
					log.Printf("Error parsing height: %v", err)

					return
				}

				bot.processImagineDimensionSetting(s, i, widthInt, heightInt)
			case customID == checkpointSelect:
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine sd model name setting menu")
					return
				}
				newModel := i.MessageComponentData().Values[0]
				bot.processImagineSDModelNameSetting(s, i, newModel)

			// patch from upstream
			case customID == batchCountSelect:
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine batch count setting menu")

					return
				}

				batchCount := i.MessageComponentData().Values[0]

				batchCountInt, intErr := strconv.Atoi(batchCount)
				if intErr != nil {
					log.Printf("Error parsing batch count: %v", err)

					return
				}

				var batchSizeInt int

				// calculate the corresponding batch size
				switch batchCountInt {
				case 1:
					batchSizeInt = 4
				case 2:
					batchSizeInt = 2
				case 4:
					batchSizeInt = 1
				default:
					log.Printf("Unknown batch count: %v", batchCountInt)

					return
				}

				bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
			case customID == batchSizeSelect:
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine batch count setting menu")

					return
				}

				batchSize := i.MessageComponentData().Values[0]

				batchSizeInt, err := strconv.Atoi(batchSize)
				if err != nil {
					log.Printf("Error parsing batch count: %v", err)

					return
				}

				var batchCountInt int

				// calculate the corresponding batch count
				switch batchSizeInt {
				case 1:
					batchCountInt = 4
				case 2:
					batchCountInt = 2
				case 4:
					batchCountInt = 1
				default:
					log.Printf("Unknown batch size: %v", batchSizeInt)

					return
				}

				bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)

			default:
				log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
			}
		case discordgo.InteractionApplicationCommandAutocomplete:
			switch i.ApplicationCommandData().Name {
			case bot.imagineCommandString():
				bot.processImagineAutocomplete(s, i)
			}
		}
	})
	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionMessageComponent { // Validate the interaction type
			if i.MessageComponentData().CustomID == "delete_error_message" {
				err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
				if err != nil {
					return
				}
			}
		}
	})

	return bot, nil
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
