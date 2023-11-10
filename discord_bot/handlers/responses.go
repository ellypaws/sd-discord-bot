package handlers

import (
	"github.com/bwmarrin/discordgo"
)

const (
	ThinkResponse   = iota // NewResponseType Respond with a "Bot is thinking..." message
	EphemeralThink         // NewResponseType Respond with an ephemeral message saying "Bot is thinking..."
	pendingResponse        // NewResponseType Respond with a "Bot is responding..." message
	messageResponse        // msgResponseType Respond with a message

	followupResponse  // msgReturnType Send a followup message
	followupEdit      // editResponseType Edit a followup message by providing a [*discordgo.Message]
	ephemeralFollowup // msgReturnType Respond with an ephemeral followup message

	editMessage             // editResponseType Edit a [*discordgo.Message]
	editInteractionResponse // msgReturnType Edit the interaction response message

	ephemeralResponding // NewResponseType Respond with an ephemeral message saying "Bot is responding..."
	ephemeralContent    // msgResponseType Respond with an ephemeral message with the provided content

	HelloResponse // newResponseType Respond with a message saying "Hey there! Congratulations, you just executed your first slash command"
)

type NewResponseType func(bot *discordgo.Session, i *discordgo.InteractionCreate)
type newReturnType func(bot *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.Message
type msgResponseType func(bot *discordgo.Session, i *discordgo.Interaction, content ...any)
type msgReturnType func(bot *discordgo.Session, i *discordgo.Interaction, content ...any) *discordgo.Message
type editResponseType func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message

var Responses = map[int]any{
	ThinkResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			ErrorEphemeralResponse(bot, i.Interaction, err)
		}
	}),
	EphemeralThink: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			}})
		if err != nil {
			ErrorEphemeralResponse(bot, i.Interaction, err)
		}
	}),
	pendingResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				//Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Bot is responding...",
			},
		})
		if err != nil {
			ErrorEdit(bot, i.Interaction, err)
		}
	}),
	messageResponse: msgResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
		err := bot.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message[0].(string),
			},
		})
		if err != nil {
			errorFollowup(bot, i, err)
		}
	}),
	followupResponse: msgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) *discordgo.Message {
		webhookParams := discordgo.WebhookParams{}
		for _, m := range message {
			switch content := m.(type) {
			case string:
				webhookParams.Content = content
			case discordgo.MessageComponent:
				webhookParams.Components = append(webhookParams.Components, content)
			case discordgo.MessageFlags:
				webhookParams.Flags = content
			}
		}
		msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
		if err != nil {
			errorFollowup(bot, i, err)
		}
		return msg
	}),

	followupEdit: editResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message {
		webhookEdit := discordgo.WebhookEdit{}
		contentEdit(&webhookEdit, message)
		contentEdit(&webhookEdit, content...)

		msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, &webhookEdit)
		if err != nil {
			errorFollowup(bot, i, err)
		}
		return msg
	}),

	ephemeralFollowup: msgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) *discordgo.Message {
		webhookParams := discordgo.WebhookParams{
			Flags: discordgo.MessageFlagsEphemeral,
		}
		for _, m := range message {
			switch content := m.(type) {
			case string:
				webhookParams.Content = content
			case discordgo.MessageComponent:
				webhookParams.Components = append(webhookParams.Components, content)
			}
		}
		msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
		if err != nil {
			errorFollowup(bot, i, err)
		}
		return msg
	}),

	editMessage: editResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) *discordgo.Message {
		webhookEdit := discordgo.WebhookEdit{}
		contentEdit(&webhookEdit, message)
		contentEdit(&webhookEdit, content...)

		msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, &webhookEdit)
		if err != nil {
			errorFollowup(bot, i, err)
		}
		return msg
	}),

	editInteractionResponse: msgReturnType(func(bot *discordgo.Session, i *discordgo.Interaction, content ...any) *discordgo.Message {
		webhookEdit := discordgo.WebhookEdit{}
		contentEdit(&webhookEdit, content...)

		msg, err := bot.InteractionResponseEdit(i, &webhookEdit)
		if err != nil {
			ErrorEphemeralResponse(bot, i, err)
		}
		return msg
	}),

	ephemeralResponding: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Bot is responding...",
			},
		})
		if err != nil {
			ErrorEdit(bot, i.Interaction, err)
		}
	}),
	ephemeralContent: msgResponseType(func(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
		err := bot.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Note: this isn't documented, but you can use that if you want to.
				// This flag just allows you to create messages visible only for the caller of the command
				// (user who triggered the command)
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: message[0].(string),
			},
		})
		if err != nil {
			errorFollowup(bot, i, err)
		}
	}),
	HelloResponse: NewResponseType(func(bot *discordgo.Session, i *discordgo.InteractionCreate) {
		err := bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Hey there! Congratulations, you just executed your first slash command",
			},
		})
		if err != nil {
			ErrorEdit(bot, i.Interaction, err)
		}
	}),
}

func contentEdit(webhookEdit *discordgo.WebhookEdit, messages ...any) {
	if len(messages) == 0 {
		return
	}
	var newEmbeds []*discordgo.MessageEmbed
	var newComponents []discordgo.MessageComponent
	for _, m := range messages {
		switch c := m.(type) {
		case *discordgo.Message:
			webhookEdit.Content = &c.Content
			webhookEdit.Embeds = &c.Embeds
			webhookEdit.Components = &c.Components
		case string:
			//log.Println("String content: ", c)
			webhookEdit.Content = &c
		case discordgo.MessageEmbed:
			newEmbeds = append(newEmbeds, &c)
		case discordgo.MessageComponent:
			newComponents = append(newComponents, c)
		case []discordgo.MessageComponent:
			newComponents = append(newComponents, c...)
		}
	}
	if len(newComponents) > 0 {
		webhookEdit.Components = &newComponents
	}
	if len(newEmbeds) > 0 {
		webhookEdit.Embeds = &newEmbeds
	}
}

func EphemeralFollowup(bot *discordgo.Session, i *discordgo.Interaction, message ...any) {
	Responses[ephemeralFollowup].(msgReturnType)(bot, i, message...)
}

func DeleteAboveFollowup(bot *discordgo.Session, i *discordgo.Interaction) {
	EphemeralFollowup(bot, i, "Delete generation", Components[DeleteButton])
}
