package stable_diffusion

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ellypaws/inkbunny-sd/llm"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"strings"
	"time"
)

const DefaultLLMSystem = `You are an exceptionally detailed AI.
You can use markdown to format your text.
`

const LLama3 = `lmstudio-community/Meta-Llama-3-8B-Instruct-GGUF/Meta-Llama-3-8B-Instruct-Q8_0.gguf`

func (q *SDQueue) processLLM() {
	defer q.done()
	queue := q.currentImagine

	request, err := queue.LLMRequest, error(nil)
	if request == nil {
		errorResponse(q.botSession, queue.DiscordInteraction, fmt.Errorf("LLM request of type %v is nil", queue.Type))
		return
	}

	embed, webhook, err := showProcessingLLM(queue, q)
	if err != nil {
		errorResponse(q.botSession, queue.DiscordInteraction, fmt.Errorf("error showing processing LLM: %w", err))
		return
	}

	response, err := q.llmConfig.Infer(request)
	if err != nil {
		errorResponse(q.botSession, queue.DiscordInteraction, fmt.Errorf("error processing LLM request: %w", err))
		return
	}
	if len(response.Choices) == 0 {
		errorResponse(q.botSession, queue.DiscordInteraction, fmt.Errorf("LLM response was invalid"))
		return
	}

	webhook = llmResponseEmbed(queue, &response, embed)

	if len(response.Choices[0].Message.Content) > 900 {
		attachLLMResponse(&response, webhook)
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, webhook)
}

func showProcessingLLM(queue *SDQueueItem, q *SDQueue) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := queue.LLMRequest

	content := fmt.Sprintf(
		"Processing LLM request for <@%s>",
		queue.DiscordInteraction.Member.User.ID,
	)
	embed := llmEmbed(new(discordgo.MessageEmbed), request, queue, queue.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(
		q.botSession,
		queue.DiscordInteraction,
		webhook,
	)

	return embed, webhook, nil
}

func llmResponseEmbed(queue *SDQueueItem, response *llm.Response, embed *discordgo.MessageEmbed) *discordgo.WebhookEdit {
	timeSince := time.Since(queue.LLMCreated).Round(time.Second).String()
	if queue.LLMCreated.IsZero() {
		timeSince = "unknown"
	}
	mention := fmt.Sprintf("<@%s> generated in %s", queue.DiscordInteraction.Member.User.ID, timeSince)
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

func llmEmbed(embed *discordgo.MessageEmbed, request *llm.Request, queue *SDQueueItem, interrupted bool) *discordgo.MessageEmbed {
	if queue == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T", queue)
		return embed
	}
	if request == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T or %T", request, queue)
		return embed
	}
	if embed == nil {
		log.Printf("WARNING: generationEmbedDetails called with nil %T, creating...", embed)
		embed = &discordgo.MessageEmbed{}
	}
	embed.Title = "LLM Instruct"
	if interrupted {
		embed.Title += " (Interrupted)"
	}
	embed.Type = discordgo.EmbedTypeArticle
	embed.URL = "https://github.com/ellypaws/sd-discord-bot/"
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name:         queue.DiscordInteraction.Member.User.Username,
		IconURL:      queue.DiscordInteraction.Member.User.AvatarURL(""),
		ProxyIconURL: "https://i.keiau.space/data/00144.png",
	}

	if queue.LLMCreated.IsZero() {
		queue.LLMCreated = time.Now()
	}

	embed.Description = fmt.Sprintf("<@%s> asked me to process `%d` tokens",
		queue.DiscordInteraction.Member.User.ID, request.MaxTokens)

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
