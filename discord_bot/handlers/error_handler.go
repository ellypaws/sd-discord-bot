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

const (
	ErrorResponse          errorEnum = iota // errorResponseType Respond with an error message and a deletion button.
	ErrorFollowup                           // errorResponseType Respond with an error message as a followup message with a deletion button.
	ErrorEphemeral                          // errorResponseType Respond with an ephemeral error message and a deletion button.
	ErrorFollowupEphemeral                  // errorResponseType Respond with an ephemeral error message as a followup message with a deletion button.
)

type errorResponseType MsgResponseType
type errorEnum int

var Errors = map[errorEnum]errorResponseType{
	ErrorResponse:          errorResponseType(ErrorEdit),
	ErrorFollowup:          errorResponseType(errorFollowup),
	ErrorEphemeral:         errorResponseType(ErrorEphemeralResponse),
	ErrorFollowupEphemeral: errorResponseType(errorEphemeralFollowup),
}

// ErrorHandler responds to the interaction with an error message and a deletion button.
// Deprecated: Use ErrorEdit instead.
var ErrorHandler = Errors[ErrorResponse]

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
	embed, toPrint := errorEmbed(i, errorContent...)

	logError(toPrint, i)

	_, _ = bot.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Content:    *sanitizeToken(&toPrint),
		Components: []discordgo.MessageComponent{Components[DeleteButton]},
		Embeds:     embed,
	})
}

// ErrorEdit [ErrorResponse] responds to the interaction with an error message and a deletion button.
func ErrorEdit(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	embed, toPrint := errorEmbed(i, errorContent...)

	logError(toPrint, i)

	_, err := bot.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:    sanitizeToken(&toPrint),
		Components: &[]discordgo.MessageComponent{Components[DeleteButton]},
		Embeds:     &embed,
	})
	if err != nil {
		log.Printf("Error editing interaction for error (%v): %v", toPrint, err)
	}
}

// ErrorEphemeralResponse [ErrorEphemeral] responds to the interaction with an ephemeral error message.
func ErrorEphemeralResponse(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	embed, toPrint := errorEmbed(i, errorContent...)

	logError(toPrint, i)

	_ = bot.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			// Note: this isn't documented, but you can use that if you want to.
			// This flag just allows you to create messages visible only for the caller of the command
			// (user who triggered the command)
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: toPrint,
			Embeds:  embed,
		},
	})
}

func errorEphemeralFollowup(bot *discordgo.Session, i *discordgo.Interaction, errorContent ...any) {
	embed, toPrint := errorEmbed(i, errorContent...)

	logError(toPrint, i)

	_, _ = bot.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: *sanitizeToken(&toPrint),
		Embeds:  embed,
	})
}

func formatError(errorContent ...any) string {
	if errorContent == nil || len(errorContent) < 1 {
		errorContent = []any{"An unknown error has occurred"}
	}

	var errors []string
	for _, content := range errorContent {
		switch content := content.(type) {
		case string:
			errors = append(errors, content)
		case []string:
			errors = append(errors, content...)
		case error:
			errors = append(errors, content.Error())
		case []any:
			errors = append(errors, formatError(content...)) // Recursively format the error
		//case any:
		//	errors = append(errors, fmt.Sprintf("%v", content))
		default:
			errors = append(errors, fmt.Sprintf("An unknown error has occured\nReceived: %v", content))
		}
	}

	errorString := strings.Join(errors, "\n")
	if len(errors) > 1 {
		errorString = "Multiple errors have occurred:\n" + errorString
	}

	return errorString
}

func errorEmbed(i *discordgo.Interaction, errorContent ...any) ([]*discordgo.MessageEmbed, string) {
	errorString := formatError(errorContent)

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

	var toPrint strings.Builder
	// Could not run the [command] `command` on message https://discord.com/channels/123456789012345678/1234567890123456789/1234567890123456789
	toPrint.Grow(192)

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		toPrint.WriteString(fmt.Sprintf(
			"Could not run the [command] `%v`",
			i.ApplicationCommandData().Name,
		))
	case discordgo.InteractionMessageComponent:
		toPrint.WriteString(fmt.Sprintf(
			"Could not run the [button] `%v`",
			i.MessageComponentData().CustomID,
		))
		if i.Message != nil {
			toPrint.WriteString(fmt.Sprintf(" on message https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID))
		}
	}
	return embed, toPrint.String()
}

func sanitizeToken(errorString *string) *string {
	if errorString == nil {
		return errorString
	}
	if Token == nil {
		log.Println("WARNING: Token is nil")
		return errorString
	}
	if strings.Contains(*errorString, *Token) {
		//log.Println("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\"")
		//log.Println("Error message:", errorString)
		log.Printf("WARNING: Bot token was found in the error message. Replacing it with \"Bot Token\": %v", *errorString)
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
	//byteArr, _ := json.MarshalIndent(i, "", " ")
	//log.Printf("Interaction: %v", string(byteArr))
	if i == nil || i.Member == nil {
		log.Printf("WARNING: Member is nil!")
		return
	}
	log.Printf("User: %v", i.Member.User.Username)
	//if i.Type == discordgo.InteractionMessageComponent {
	//	//log.Printf("Link: https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)
	//	GetBot().p.Send(logger.Message(fmt.Sprintf("Link: https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)))
	//}
}
