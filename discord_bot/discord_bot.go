package discord_bot

import (
	"errors"
	"fmt"
	"github.com/sahilm/fuzzy"
	"log"
	"regexp"
	"stable_diffusion_bot/entities"
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

const (
	checkpointSelect = "imagine_sd_model_name_menu"
	dimensionSelect  = "imagine_dimension_setting_menu"
	batchCountSelect = "imagine_batch_count_setting_menu"
	batchSizeSelect  = "imagine_batch_size_setting_menu"
)

func New(cfg Config) (Bot, error) {
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

				interactionIndexInt, intErr := strconv.Atoi(interactionIndex)
				if intErr != nil {
					log.Printf("Error parsing interaction index: %v", err)

					return
				}

				bot.processImagineUpscale(s, i, interactionIndexInt)
			case strings.HasPrefix(customID, "imagine_variation_"):
				interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

				interactionIndexInt, intErr := strconv.Atoi(interactionIndex)
				if intErr != nil {
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

				widthInt, intErr := strconv.Atoi(width)
				if intErr != nil {
					log.Printf("Error parsing width: %v", err)

					return
				}

				heightInt, intErr := strconv.Atoi(height)
				if intErr != nil {
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

				batchSizeInt, intErr := strconv.Atoi(batchSize)
				if intErr != nil {
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

const extraLoras = 3

const (
	promptOption       = "prompt"
	negativeOption     = "negative_prompt"
	samplerOption      = "sampler_name"
	aspectRatio        = "aspect_ratio"
	loraOption         = "lora"
	hiresFixOption     = "use_hires_fix"
	hiresFixSize       = "hires_fix_size"
	restoreFacesOption = "restore_faces"
	adModelOption      = "ad_model"
)

func (b *botImpl) addImagineCommand() error {
	log.Printf("Adding command '%s'...", b.imagineCommandString())

	options := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        promptOption,
			Description: "The text prompt to imagine",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        negativeOption,
			Description: "Negative prompt",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        aspectRatio,
			Description: "The aspect ratio to use. Default is 1:1 (note: you can specify your own aspect ratio)",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "1:1",
					Value: "1:1",
				},
				{
					Name:  "2:3",
					Value: "2:3",
				},
				{
					Name:  "3:2",
					Value: "3:2",
				},
				{
					Name:  "3:4",
					Value: "3:4",
				},
				{
					Name:  "4:3",
					Value: "4:3",
				},
				{
					Name:  "16:9",
					Value: "16:9",
				},
				{
					Name:  "9:16",
					Value: "9:16",
				},
			},
		},
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         loraOption,
			Description:  "The lora(s) to apply",
			Required:     false,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        samplerOption,
			Description: "sampler",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Euler a",
					Value: "Euler a",
				},
				{
					Name:  "DDIM",
					Value: "DDIM",
				},
				{
					Name:  "UniPC",
					Value: "UniPC",
				},
				{
					Name:  "Euler",
					Value: "Euler",
				},
				{
					Name:  "DPM2 a Karras",
					Value: "DPM2 a Karras",
				},
				{
					Name:  "DPM++ 2S a Karras",
					Value: "DPM++ 2S a Karras",
				},
				{
					Name:  "DPM++ 2M Karras",
					Value: "DPM++ 2M Karras",
				},
				{
					Name:  "DPM++ 3M SDE Karras",
					Value: "DPM++ 3M SDE Karras",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        hiresFixOption,
			Description: "use hires.fix or not. default=No for better performance",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Yes",
					Value: "true",
				},
				{
					Name:  "No",
					Value: "false",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        hiresFixSize,
			Description: "upscale multiplier for hires.fix. default=2",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "1.5",
					Value: "1.5",
				},
				{
					Name:  "2",
					Value: "2",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        restoreFacesOption,
			Description: "Use Codeformer to restore faces",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Yes",
					Value: "true",
				},
				{
					Name:  "No",
					Value: "false",
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        adModelOption,
			Description: "The model to use for adetailer",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Face",
					Value: "face_yolov8n.pt",
				},
				{
					Name:  "Body",
					Value: "person_yolov8n-seg.pt",
				},
				{
					Name:  "Both",
					Value: "person_yolov8n-seg.pt,face_yolov8n.pt",
				},
			},
		},
	}

	for i := 0; i < extraLoras; i++ {
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         loraOption + fmt.Sprintf("%d", i+2),
			Description:  "The lora(s) to apply",
			Required:     false,
			Autocomplete: true,
		})
	}

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineCommandString(),
		Description: "Ask the bot to imagine something",
		Options:     options,
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineCommandString(), err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) addImagineSettingsCommand() error {
	log.Printf("Adding command '%s'...", b.imagineSettingsCommandString())

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineSettingsCommandString(),
		Description: "Change the default settings for the imagine command",
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineSettingsCommandString(), err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeReroll,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm reimagining that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeUpscale,
		InteractionIndex:   upscaleIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm upscaling that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeVariation,
		InteractionIndex:   variationIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm imagining more variations for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var position int
	var queueError error
	var prompt string
	negative := ""
	sampler := "Euler a"
	hiresFix := false
	restoreFaces := false
	var stringValue string

	if option, ok := optionMap[promptOption]; ok {
		prompt = option.StringValue()

		if nopt, ok := optionMap[negativeOption]; ok {
			negative = nopt.StringValue()
		}

		if smpl, ok := optionMap[samplerOption]; ok {
			sampler = smpl.StringValue()
		}

		if hires, ok := optionMap[hiresFixOption]; ok {
			hiresFix, _ = strconv.ParseBool(hires.StringValue())
		}

		if hires, ok := optionMap[restoreFacesOption]; ok {
			restoreFaces, _ = strconv.ParseBool(hires.StringValue())
		}

		if aDetailOpt, ok := optionMap[adModelOption]; ok {
			stringValue = aDetailOpt.StringValue()
			// adModel = strings.Split(stringValue, ",")
			// use AppendSegModelByString instead
		}

		strength := regexp.MustCompile(`:([\d\.]+)$`)

		for i := 0; i < extraLoras+1; i++ {
			loraKey := loraOption
			if i != 1 {
				loraKey += fmt.Sprintf("%d", i+1)
			}

			if lora, ok := optionMap[loraKey]; ok {
				loraValue := lora.StringValue()
				if loraValue != "" {
					// add :1 if no strength is specified
					if !strength.MatchString(loraValue) {
						loraValue += ":1"
					}
					prompt += ", <lora:" + loraValue + ">"
				}
			}
		}

		imagine := &imagine_queue.QueueItem{
			Prompt:             prompt,
			NegativePrompt:     negative,
			SamplerName1:       sampler,
			Type:               imagine_queue.ItemTypeImagine,
			UseHiresFix:        hiresFix,
			RestoreFaces:       restoreFaces,
			DiscordInteraction: i.Interaction,
			ADetailerString:    stringValue,
		}

		if restoreFacesOption, ok := optionMap["restore_faces"]; ok {
			restoreFacesValue, err := strconv.ParseBool(restoreFacesOption.StringValue())
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			}
			imagine.RestoreFaces = restoreFacesValue
		}

		position, queueError = b.imagineQueue.AddImagine(imagine)
		if queueError != nil {
			log.Printf("Error adding imagine to queue: %v\n", queueError)
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(
				"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \"%s\", with sampler: %s",
				position,
				i.Member.User.ID,
				prompt,
				sampler),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	log.Printf("running autocomplete handler")
	var input string
	for index, opt := range data.Options {
		if opt.Focused {
			log.Printf("Focused option (%v): %v", index, opt.Name)
			input = opt.StringValue()

			var choices []*discordgo.ApplicationCommandOptionChoice

			if input != "" {
				log.Printf("Autocompleting '%v'", input)
				cache, err := b.StableDiffusionApi.SDLorasCache()
				if err != nil {
					log.Printf("Error retrieving loras cache: %v", err)
				}
				results := fuzzy.FindFrom(input, cache)

				for index, result := range results {
					if index > 25 {
						break
					}
					regExp := regexp.MustCompile(`(?:models\\\\)?Lora\\\\(.*)`)

					alias := regExp.FindStringSubmatch(cache[result.Index].Path)

					var nameToUse string
					switch {
					case alias != nil && alias[1] != "":
						// replace double slash with single slash
						regExp := regexp.MustCompile(`\\{2,}`)
						nameToUse = regExp.ReplaceAllString(alias[1], `\`)
					default:
						nameToUse = cache[result.Index].Name
					}

					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  nameToUse,
						Value: cache[result.Index].Name,
					})
				}
			} else {
				choices = []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Type a lora name. Add a colon after to specify the strenth. (e.g. \"clay:0.5\")",
						Value: "placeholder",
					},
				}
			}

			if input != "" {
				choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
					Name:  input,
					Value: input,
				})
			}

			// make sure we're under 100 char limit and under 25 choices
			for i, choice := range choices {
				if len(choice.Name) > 100 {
					// TODO: check if discord counts bytes or chars
					choices[i].Name = choice.Name[:100]
				}
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices[:min(25, len(choices))], // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
				},
			})
			break
		}

	}
}

func shortenString(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}

// patch from upstream
func (b *botImpl) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	minValues := 1

	models, err := b.StableDiffusionApi.SDModels()
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
	}
	var modelOptions []discordgo.SelectMenuOption

	for i, model := range models {
		if i > 20 {
			break
		}
		modelOptions = append(modelOptions, discordgo.SelectMenuOption{
			Label:   shortenString(model.ModelName),
			Value:   shortenString(model.Title),
			Default: settings.SDModelName == model.Title,
		})
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    checkpointSelect,
					Placeholder: "Change SD Model",
					MinValues:   &minValues,
					MaxValues:   1,
					Options:     modelOptions,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  dimensionSelect,
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Size: 512x512",
							Value:   "512_512",
							Default: settings.Width == 512 && settings.Height == 512,
						},
						{
							Label:   "Size: 768x768",
							Value:   "768_768",
							Default: settings.Width == 768 && settings.Height == 768,
						},
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  batchCountSelect,
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Batch count: 1",
							Value:   "1",
							Default: settings.BatchCount == 1,
						},
						{
							Label:   "Batch count: 2",
							Value:   "2",
							Default: settings.BatchCount == 2,
						},
						{
							Label:   "Batch count: 4",
							Value:   "4",
							Default: settings.BatchCount == 4,
						},
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  batchSizeSelect,
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Batch size: 1",
							Value:   "1",
							Default: settings.BatchSize == 1,
						},
						{
							Label:   "Batch size: 2",
							Value:   "2",
							Default: settings.BatchSize == 2,
						},
						{
							Label:   "Batch size: 4",
							Value:   "4",
							Default: settings.BatchSize == 4,
						},
					},
				},
			},
		},
	}
}

func (b *botImpl) processImagineSettingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error getting default settings for settings command: %v", err)

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:      "Settings",
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) {
	botSettings, err := b.imagineQueue.UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}
}

func (b *botImpl) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) {
	botSettings, err := b.imagineQueue.UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineSDModelNameSetting(s *discordgo.Session, i *discordgo.InteractionCreate, newModelName string) {
	botSettings, err := b.imagineQueue.UpdateModelName(newModelName)
	if err != nil {
		log.Printf("error updating sd model name settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating sd model name settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
