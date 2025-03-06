package llm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ellypaws/inkbunny-sd/llm"

	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/utils"
)

const DefaultLLMSystem = `You are an exceptionally detailed AI.
You can use markdown to format your text.
`

const LLama3 = `lmstudio-community/Meta-Llama-3-8B-Instruct-GGUF/Meta-Llama-3-8B-Instruct-Q8_0.gguf`

func (q *LLMQueue) processLLM() error {
	defer q.done()
	item := q.current

	request := item.Request
	if request == nil {
		return handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Errorf("LLM request of type %v is nil", item.Type))
	}

	embed, webhook, err := showProcessingLLM(item, q)
	if err != nil {
		return handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Errorf("error showing processing LLM: %w", err))
	}

	response, err := q.host.Infer(request)
	if err != nil {
		return handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Errorf("error processing LLM request: %w", err))
	}
	if len(response.Choices) == 0 {
		return handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Errorf("LLM response was invalid"))
	}

	webhook = llmResponseEmbed(item, &response, embed)

	if len(response.Choices[0].Message.Content) > 900 {
		attachLLMResponse(&response, webhook)
	}

	_, err = handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, webhook)
	return err
}

func showProcessingLLM(item *LLMItem, q *LLMQueue) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := item.Request

	content := fmt.Sprintf(
		"Processing LLM request for <@%s>",
		utils.GetUser(item.DiscordInteraction).ID,
	)
	embed := llmEmbed(new(discordgo.MessageEmbed), request, item, item.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	}

	_, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, webhook)
	if err != nil {
		return nil, nil, err
	}

	return embed, webhook, nil
}

func llmResponseEmbed(item *LLMItem, response *llm.Response, embed *discordgo.MessageEmbed) *discordgo.WebhookEdit {
	timeSince := time.Since(item.Created).Round(time.Second).String()
	if item.Created.IsZero() {
		timeSince = "unknown"
	}
	mention := fmt.Sprintf("<@%s> generated in %s", utils.GetUser(item.DiscordInteraction).ID, timeSince)
	message := response.Choices[0].Message.Content
	if len(message) > 900 {
		message = fmt.Sprintf("%s ...\n<truncated, see file>", message[:900])
	}

	embed.Fields[0].Value = fmt.Sprintf("`%s`", response.Model)
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Output",
		Value:  fmt.Sprintf("%s", message),
		Inline: false,
	})

	return &discordgo.WebhookEdit{
		Content:    &mention,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.DeleteGeneration]},
	}
}

func attachLLMResponse(response *llm.Response, webhook *discordgo.WebhookEdit) {
	webhook.Files = []*discordgo.File{
		{
			Name:        fmt.Sprintf("output-%s.txt", time.Now().Format("2006-01-02-15-04-05")),
			ContentType: "text/plain",
			Reader:      strings.NewReader(response.Choices[0].Message.Content),
		},
	}
}

func llmEmbed(embed *discordgo.MessageEmbed, request *llm.Request, item *LLMItem, interrupted bool) *discordgo.MessageEmbed {
	if item == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T", item)
		return embed
	}
	if request == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T or %T", request, item)
		return embed
	}
	if embed == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T, creating...", embed)
		embed = &discordgo.MessageEmbed{}
	}
	user := utils.GetUser(item.DiscordInteraction)
	embed.Title = item.Type
	if interrupted {
		embed.Title += " (Interrupted)"
	}
	embed.Type = discordgo.EmbedTypeArticle
	embed.URL = "https://github.com/ellypaws/sd-discord-bot/"
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name:         user.Username,
		IconURL:      user.AvatarURL(""),
		ProxyIconURL: "https://i.keiau.space/data/00144.png",
	}

	if item.Created.IsZero() {
		item.Created = time.Now()
	}

	embed.Description = fmt.Sprintf("<@%s> asked me to process `%d` tokens",
		user.ID, request.MaxTokens)

	embed.Timestamp = time.Now().Format(time.RFC3339)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text:    "https://github.com/ellypaws/sd-discord-bot/",
		IconURL: "https://i.keiau.space/data/00144.png",
	}
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Model",
			Value:  fmt.Sprintf("`%s`", request.Model),
			Inline: false,
		},
		{
			Name:   "Tokens",
			Value:  fmt.Sprintf("`%d`", request.MaxTokens),
			Inline: true,
		},
		{
			Name:   "Temperature",
			Value:  strings.TrimRight(fmt.Sprintf("`%.2f`", request.Temperature), "0"),
			Inline: true,
		},
		{
			Name:  "Prompt",
			Value: fmt.Sprintf("```\n%s\n```", request.Messages[1].Content),
		},
	}
	return embed
}
