package handlers

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var Token *string

// ErrorHandler responds to the interaction with an error message and a deletion button.
// Deprecated: Use errorEdit instead.
func ErrorHandler(s *discordgo.Session, i *discordgo.Interaction, errorContent any) {
	var errorString string

	switch content := errorContent.(type) {
	case string:
		errorString = content
	case error:
		errorString = fmt.Sprint(content) // Convert the error to a string
	default:
		errorString = "An unknown error has occurred"
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Delete this message",
					Style:    discordgo.DangerButton,
					CustomID: "delete_error_message",
				},
			},
		},
	}

	_, _ = s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:    &errorString,
		Components: &components,
	})
}

func CheckAPIAlive(apiHost string) bool {
	resp, err := http.Get(apiHost)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

const DeadAPI = "API is not running"

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
	components := []discordgo.MessageComponent{Components[DeleteButton]}

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
	components := []discordgo.MessageComponent{Components[DeleteButton]}

	logError(errorString, i)

	_, _ = bot.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:    sanitizeToken(&errorString),
		Components: &components,
	})
}

// ErrorEphemeralResponse [ErrorEphemeral] responds to the interaction with an ephemeral error message when the deletion button doesn't work.
func ErrorEphemeralResponse(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
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
	if Token == nil {
		logError("WARNING: Token is nil", nil)
		return errorString
	}
	if strings.Contains(*errorString, *Token) {
		//log.Println("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\"")
		//log.Println("Error message:", errorString)
		logError("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\"", nil)
		logError(*errorString, nil)
		sanitizedString := strings.ReplaceAll(*errorString, *Token, "[TOKEN]")
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
