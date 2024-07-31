package llm

import (
	"github.com/bwmarrin/discordgo"
)

func (q *LLMQueue) commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        LLMCommand,
			Description: "Ask the bot to generate text using an LLM",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				commandOptions[promptOption],
				commandOptions[systemPromptOption],
				commandOptions[maxTokensOption],
			},
		},
	}
}

var commandOptions = map[string]*discordgo.ApplicationCommandOption{
	promptOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        promptOption,
		Description: "The text prompt to imagine",
		Required:    true,
	},
	systemPromptOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        systemPromptOption,
		Description: "The system prompt to generate with",
		Required:    false,
	},
	maxTokensOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        maxTokensOption,
		Description: "The maximum number of tokens to generate. Use -1 for infinite (default: 1024)",
		Required:    false,
	},
}
