package llm

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ellypaws/inkbunny-sd/llm"
)

type ItemType = string

const (
	ItemTypeInstruct ItemType = "Instruct"
)

type LLMItem struct {
	Type ItemType

	Request *llm.Request

	Created            time.Time
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
	Interrupt          chan *discordgo.Interaction
}

func (q *LLMItem) Interaction() *discordgo.Interaction {
	return q.DiscordInteraction
}

func (q *LLMQueue) NewItem(interaction *discordgo.Interaction, options ...func(*LLMItem)) *LLMItem {
	item := q.DefaultQueueItem()
	item.DiscordInteraction = interaction

	for _, option := range options {
		option(item)
	}

	return item
}

func (q *LLMQueue) DefaultQueueItem() *LLMItem {
	messages := make([]llm.Message, 1, 2)
	messages[0] = llm.Message{
		Role:    llm.SystemRole,
		Content: DefaultLLMSystem,
	}

	return &LLMItem{
		Type: ItemTypeInstruct,
		Request: &llm.Request{
			Messages:      messages,
			Model:         LLama3,
			Temperature:   0.7,
			MaxTokens:     1024,
			Stream:        false,
			StreamChannel: nil,
		},
		Created:   time.Now(),
		Interrupt: nil,
	}
}

func WithPrompt(prompt string) func(*LLMItem) {
	return func(item *LLMItem) {
		item.Request.Messages = append(item.Request.Messages, llm.UserMessage(prompt))
	}
}
