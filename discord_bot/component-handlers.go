package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/handlers"
	"strconv"
	"strings"
)

var componentHandlers = map[string]func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	handlers.DeleteButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			errorEphemeral(s, i.Interaction, err)
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

// errorFollowup [ErrorFollowup] sends an error message as a followup message with a deletion button.
func errorFollowup(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	if errorContent == nil || len(errorContent) == 0 {
		return
	}
	var errorString string

	switch content := errorContent[0].(type) {
	case string:
		errorString = content
	case error:
		errorString = fmt.Sprint(content) // Convert the error to a string
	default:
		errorString = "An unknown error has occurred"
		errorString += "\nReceived:" + fmt.Sprint(content)
	}
	components := []discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]}

	logError(errorString, i)

	_, _ = bot.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Content:    *sanitizeToken(&errorString),
		Components: components,
	})
}

// errorEdit [ErrorResponse] responds to the interaction with an error message and a deletion button.
func errorEdit(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	if errorContent == nil || len(errorContent) == 0 {
		return
	}
	var errorString string

	switch content := errorContent[0].(type) {
	case string:
		errorString = content
	case error:
		errorString = fmt.Sprint(content) // Convert the error to a string
	default:
		errorString = "An unknown error has occurred"
		errorString += "\nReceived:" + fmt.Sprint(content)
	}
	components := []discordgo.MessageComponent{handlers.Components[handlers.DeleteButton]}

	logError(errorString, i)

	_, _ = bot.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:    sanitizeToken(&errorString),
		Components: &components,
	})
}

// errorEphemeral [ErrorEphemeral] responds to the interaction with an ephemeral error message when the deletion button doesn't work.
func errorEphemeral(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	if errorContent == nil || len(errorContent) == 0 {
		return
	}
	blankEmbed, toPrint := errorEmbed(errorContent, i)

	_ = bot.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			// Note: this isn't documented, but you can use that if you want to.
			// This flag just allows you to create messages visible only for the caller of the command
			// (user who triggered the command)
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: toPrint,
			Embeds:  blankEmbed,
		},
	})
}

func errorEphemeralFollowup(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	if errorContent == nil || len(errorContent) == 0 {
		return
	}
	blankEmbed, toPrint := errorEmbed(errorContent, i)

	_, _ = bot.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Content: *sanitizeToken(&toPrint),
		Embeds:  blankEmbed,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

func errorEmbed(errorContent []any, i *discordgo.Interaction) ([]*discordgo.MessageEmbed, string) {
	var errorString string

	switch content := errorContent[0].(type) {
	case string:
		errorString = content
	case error:
		errorString = fmt.Sprint(content) // Convert the error to a string
	default:
		errorString = "An unknown error has occurred"
		errorString += "\nReceived:" + fmt.Sprint(content)
	}

	logError(errorString, i)

	// decode ED4245 to int
	color, _ := strconv.ParseInt("ED4245", 16, 64)

	embed := []*discordgo.MessageEmbed{
		{
			Type: discordgo.EmbedTypeRich,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Error",
					Value:  *sanitizeToken(&errorString),
					Inline: false,
				},
			},
			Color: int(color),
		},
	}

	var toPrint string

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		toPrint = fmt.Sprintf(
			"Could not run the [command](https://github.com/ellypaws/go-clippy) `%v`",
			i.ApplicationCommandData().Name,
		)
	case discordgo.InteractionMessageComponent:
		toPrint = fmt.Sprintf(
			"Could not run the [button](https://github.com/ellypaws/go-clippy) `%v` on message https://discord.com/channels/%v/%v/%v",
			i.MessageComponentData().CustomID,
			i.GuildID,
			i.ChannelID,
			i.Message.ID,
		)
	}
	return embed, toPrint
}

func sanitizeToken(errorString *string) *string {
	if config == nil {
		logError("WARNING: Config is nil", nil)
		return errorString
	}
	if strings.Contains(*errorString, config.BotToken) {
		//log.Println("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\"")
		//log.Println("Error message:", errorString)
		logError("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\"", nil)
		logError(*errorString, nil)
		sanitizedString := strings.ReplaceAll(*errorString, config.BotToken, "[TOKEN]")
		errorString = &sanitizedString
	}
	return errorString
}

func logError(errorString string, i *discordgo.Interaction) {
	//GetBot().p.Send(logger.Message(fmt.Sprintf("WARNING: A command failed to execute: %v", errorString)))
	//if i.Type == discordgo.InteractionMessageComponent {
	//	//log.Printf("Command: %v", i.MessageComponentData().CustomID)
	//	GetBot().p.Send(logger.Message(fmt.Sprintf("Command: %v", i.MessageComponentData().CustomID)))
	//}
	log.Printf("ERROR: %v", errorString)
	if i == nil {
		return
	}
	log.Printf("User: %v", i.Member.User.Username)
	//if i.Type == discordgo.InteractionMessageComponent {
	//	//log.Printf("Link: https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)
	//	GetBot().p.Send(logger.Message(fmt.Sprintf("Link: https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)))
	//}
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

		checkpointDropdown.Components[0] = discordgo.SelectMenu{
			Options: modelOptions,
		}
		handlers.Components[handlers.CheckpointSelect] = checkpointDropdown
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
