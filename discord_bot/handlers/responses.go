package handlers

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var ResponseError = errors.New("error responding to interaction")

func Wrap(err error) error {
	if err != nil {
		return fmt.Errorf("%w: %w", ResponseError, err)
	}
	return nil
}

func ThinkResponse(bot *discordgo.Session, i *discordgo.InteractionCreate) error {
	return Wrap(bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}))
}

func EphemeralThink(bot *discordgo.Session, i *discordgo.InteractionCreate) error {
	return Wrap(bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
}

func PendingResponse(bot *discordgo.Session, i *discordgo.InteractionCreate) error {
	return Wrap(bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Bot is responding...",
		},
	}))
}

func MessageResponse(bot *discordgo.Session, i *discordgo.Interaction, message ...any) error {
	err := bot.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message[0].(string),
		},
	})
	if err != nil {
		return Wrap(err)
	}
	return nil
}

func FollowupResponse(bot *discordgo.Session, i *discordgo.Interaction, message ...any) (*discordgo.Message, error) {
	webhookParams := contentToWebhookParams(message...)
	msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
	if err != nil {
		return nil, Wrap(err)
	}
	return msg, nil
}

func FollowupEdit(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) (*discordgo.Message, error) {
	webhookEdit := webhookFromContents(content...)
	contentEdit(webhookEdit, message)
	contentEdit(webhookEdit, content...)
	msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, webhookEdit)
	if err != nil {
		return nil, Wrap(err)
	}
	return msg, nil
}

func EphemeralFollowup(bot *discordgo.Session, i *discordgo.Interaction, message ...any) (*discordgo.Message, error) {
	webhookParams := contentToWebhookParams(message...)
	msg, err := bot.FollowupMessageCreate(i, true, &webhookParams)
	if err != nil {
		return nil, Wrap(err)
	}
	return msg, nil
}

func EditMessage(bot *discordgo.Session, i *discordgo.Interaction, message *discordgo.Message, content ...any) (*discordgo.Message, error) {
	webhookEdit := webhookFromContents(content...)
	contentEdit(webhookEdit, message)
	contentEdit(webhookEdit, content...)
	msg, err := bot.FollowupMessageEdit(i, message.Reference().MessageID, webhookEdit)
	if err != nil {
		return nil, Wrap(err)
	}

	return msg, nil
}

func UpdateFromComponent(bot *discordgo.Session, i *discordgo.Interaction, content ...any) error {
	interactionResponse := discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{},
	}
	responseEdit(interactionResponse.Data, content...)

	err := bot.InteractionRespond(i, &interactionResponse)
	if err != nil {
		return Wrap(err)
	}

	return nil
}

func EditInteractionResponse(bot *discordgo.Session, i *discordgo.Interaction, content ...any) (*discordgo.Message, error) {
	webhookEdit := webhookFromContents(content...)
	contentEdit(webhookEdit, content...)

	msg, err := bot.InteractionResponseEdit(i, webhookEdit)
	if err != nil {
		return nil, Wrap(err)
	}

	return msg, nil
}

func EphemeralResponding(bot *discordgo.Session, i *discordgo.InteractionCreate) error {
	return Wrap(bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Bot is responding...",
		},
	}))
}

func EphemeralContent(bot *discordgo.Session, i *discordgo.Interaction, message ...any) error {
	return Wrap(bot.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: message[0].(string),
		},
	}))
}

func HelloResponse(bot *discordgo.Session, i *discordgo.InteractionCreate) error {
	return Wrap(bot.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Hey there! Congratulations, you just executed your first slash command",
		},
	}))
}

func webhookFromContents(content ...any) *discordgo.WebhookEdit {
	webhookEdit := &discordgo.WebhookEdit{}
	for _, m := range content {
		switch c := m.(type) {
		case discordgo.WebhookEdit:
			webhookEdit = &c
		case *discordgo.WebhookEdit:
			webhookEdit = c
		}
	}
	return webhookEdit
}

func contentToWebhookParams(content ...any) discordgo.WebhookParams {
	webhookParams := discordgo.WebhookParams{}
	for _, m := range content {
		switch c := m.(type) {
		case discordgo.WebhookParams:
			webhookParams = c
		case string:
			webhookParams.Content = c
		case discordgo.MessageComponent:
			webhookParams.Components = append(webhookParams.Components, c)
		case discordgo.MessageFlags:
			webhookParams.Flags = c
		}
	}
	return webhookParams
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
			// log.Println("String content: ", c)
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

func responseEdit(resp *discordgo.InteractionResponseData, messages ...any) {
	if resp == nil {
		resp = &discordgo.InteractionResponseData{}
	}
	if len(messages) == 0 {
		return
	}
	var newEmbeds []*discordgo.MessageEmbed
	var newComponents []discordgo.MessageComponent
	for _, m := range messages {
		switch c := m.(type) {
		case *discordgo.Message:
			resp.Content = c.Content
			resp.Embeds = c.Embeds
			resp.Components = c.Components
		case string:
			resp.Content = c
		case discordgo.MessageEmbed:
			newEmbeds = append(newEmbeds, &c)
		case discordgo.MessageComponent:
			newComponents = append(newComponents, c)
		case []discordgo.MessageComponent:
			newComponents = append(newComponents, c...)
		}
	}
	if len(newComponents) > 0 {
		resp.Components = newComponents
	}
	if len(newEmbeds) > 0 {
		resp.Embeds = newEmbeds
	}
}

func DeleteAboveFollowup(bot *discordgo.Session, i *discordgo.Interaction) error {
	_, err := EphemeralFollowup(bot, i, "Delete generation", Components[DeleteButton])
	return Wrap(err)
}
