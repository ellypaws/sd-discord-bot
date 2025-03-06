package llm

import (
	"errors"
	"fmt"
	"log"

	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/utils"

	"github.com/bwmarrin/discordgo"
)

const LLMCommand = "llm"

const (
	promptOption       = "prompt"
	systemPromptOption = "system_prompt"
	maxTokensOption    = "max_tokens"
	llmModelOption     = "model" // TODO: Retrieve /v1/models from endpoint
)

func (q *LLMQueue) handlers() queue.CommandHandlers {
	return queue.CommandHandlers{
		discordgo.InteractionApplicationCommand: {
			LLMCommand: q.processLLMCommand,
		},
	}
}

func (q *LLMQueue) processLLMCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	optionMap := utils.GetOpts(i.ApplicationCommandData())

	prompt, ok := optionMap[promptOption]
	if !ok {
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a prompt.")
	}

	item := q.NewItem(i.Interaction, WithPrompt(prompt.StringValue()))

	if len(item.Request.Messages) < 2 {
		return handlers.ErrorEdit(s, i.Interaction, errors.New("unexpected error: LLM request messages is less than 2"))
	}

	if s, ok := optionMap[systemPromptOption]; ok {
		item.Request.Messages[0].Content = s.StringValue()
	}

	if m, ok := optionMap[maxTokensOption]; ok {
		item.Request.MaxTokens = m.IntValue()
	}

	position, err := q.Add(item)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error adding imagine to queue.", err)
	}

	queueString := fmt.Sprintf(
		"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to generate \n```\n%s\n```",
		position,
		utils.GetUser(i.Interaction).ID,
		prompt.StringValue(),
	)

	message, err := handlers.EditInteractionResponse(s, i.Interaction, queueString, components[cancel])
	if err != nil {
		return err
	}
	if item.DiscordInteraction != nil && item.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting message ID for interaction %v", item.DiscordInteraction.ID)
		item.DiscordInteraction.Message = message
	}

	return nil
}
