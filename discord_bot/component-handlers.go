package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"regexp"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"strconv"
	"strings"
)

var componentHandlers = map[string]func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	handlers.DeleteButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, err)
		}
	},

	handlers.DeleteGeneration: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		handlers.Responses[handlers.EphemeralThink].(handlers.NewResponseType)(s, i)
		msg, _ := bot.botSession.ChannelMessage(i.ChannelID, i.Message.ID)

		content := msg.Content
		userRegex := regexp.MustCompile("<@!?(\\d+)>")
		userID := userRegex.FindStringSubmatch(content)[1]

		if userID != i.Member.User.ID {
			handlers.ErrorEdit(s, i.Interaction, "You can only delete your own generations")
			return
		}
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			handlers.ErrorEdit(s, i.Interaction, err)
			return
		}

		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, "Generation deleted")
	},

	handlers.DeleteAboveButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		// delete the original interaction response
		msg, err := s.InteractionResponse(i.Interaction)
		if i == nil || err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, fmt.Errorf("failed to retrieve interaction response: %v, %v", i, err))
			return
		}

		err = s.ChannelMessageDelete(i.ChannelID, msg.ID)

		if err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, err)
		}
	},

	handlers.DimensionSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	},

	handlers.CheckpointSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine sd model name setting menu")
			return
		}
		newModel := i.MessageComponentData().Values[0]
		bot.processImagineSDModelNameSetting(s, i, newModel)
	},

	handlers.BatchCountSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine batch count setting menu")

			return
		}

		batchCount := i.MessageComponentData().Values[0]

		batchCountInt, err := strconv.Atoi(batchCount)
		if err != nil {
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
	},

	handlers.BatchSizeSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	},

	handlers.RerollButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.processImagineReroll(s, i)
	},

	handlers.UpscaleButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, handlers.UpscaleButton+"_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			log.Printf("Error parsing interaction index: %v", err)

			return
		}

		bot.processImagineUpscale(s, i, interactionIndexInt)
	},

	handlers.VariantButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			log.Printf("Error parsing interaction index: %v", err)

			return
		}

		bot.processImagineVariation(s, i, interactionIndexInt)
	},
}

// patch from upstream
func (b *botImpl) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	models, err := b.StableDiffusionApi.SDModelsCache()

	// populate checkpoint dropdown and set default
	checkpointDropdown := handlers.Components[handlers.CheckpointSelect].(discordgo.ActionsRow)
	var modelOptions []discordgo.SelectMenuOption
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
	} else {
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

		component := checkpointDropdown.Components[0].(discordgo.SelectMenu)
		component.Options = modelOptions

		handlers.Components[handlers.CheckpointSelect].(discordgo.ActionsRow).Components[0] = component
	}

	// set default dimension from config
	dimensions := handlers.Components[handlers.DimensionSelect].(discordgo.ActionsRow)
	dimensions.Components[0].(discordgo.SelectMenu).Options[0].Default = settings.Width == 512 && settings.Height == 512
	dimensions.Components[0].(discordgo.SelectMenu).Options[1].Default = settings.Width == 768 && settings.Height == 768
	handlers.Components[handlers.DimensionSelect] = dimensions

	// set default batch count from config
	batchCount := handlers.Components[handlers.BatchCountSelect].(discordgo.ActionsRow)
	for i, option := range batchCount.Components[0].(discordgo.SelectMenu).Options {
		if i == settings.BatchCount {
			option.Default = true
		} else {
			option.Default = false
		}
		batchCount.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchCountSelect] = batchCount

	// set the default batch size from config
	batchSize := handlers.Components[handlers.BatchSizeSelect].(discordgo.ActionsRow)
	for i, option := range batchSize.Components[0].(discordgo.SelectMenu).Options {
		if i == settings.BatchSize {
			option.Default = true
		} else {
			option.Default = false
		}
		batchSize.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchSizeSelect] = batchSize

	return []discordgo.MessageComponent{
		handlers.Components[handlers.CheckpointSelect],
		handlers.Components[handlers.DimensionSelect],
		handlers.Components[handlers.BatchCountSelect],
		handlers.Components[handlers.BatchSizeSelect],
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
