package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
	"strings"
)

var componentHandlers = map[string]func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	deleteButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			errorEphemeral(s, i.Interaction, err)
		}
	},

	dimensionSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	checkpointSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine sd model name setting menu")
			return
		}
		newModel := i.MessageComponentData().Values[0]
		bot.processImagineSDModelNameSetting(s, i, newModel)
	},

	batchCountSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	batchSizeSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	rerollButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.processImagineReroll(s, i)
	},

	upscaleButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.Interaction.Data.(discordgo.MessageComponentInteractionData).CustomID
		interactionIndex := strings.TrimPrefix(customID, upscaleButton+"_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			log.Printf("Error parsing interaction index: %v", err)

			return
		}

		bot.processImagineUpscale(s, i, interactionIndexInt)
	},

	variantButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.Interaction.Data.(discordgo.MessageComponentInteractionData).CustomID
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
	components := []discordgo.MessageComponent{components[deleteButton]}

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
	components := []discordgo.MessageComponent{components[deleteButton]}

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
