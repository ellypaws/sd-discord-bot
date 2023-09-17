package handlers

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"net/http"
)

// ErrorHandler responds to the interaction with an error message and a deletion button.
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
