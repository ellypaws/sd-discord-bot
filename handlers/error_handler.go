package handlers

import "github.com/bwmarrin/discordgo"

// errorHandler responds to the interaction with an error message and a deletion button.
func ErrorHandler(s *discordgo.Session, i *discordgo.Interaction, errorContent interface{}) {
	var errorString string

	switch v := errorContent.(type) {
	case string:
		errorString = v
	case error:
		errorString = v.Error() // Convert the error to a string
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
