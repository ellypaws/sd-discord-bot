package novelai

import (
	"github.com/bwmarrin/discordgo"
	"stable_diffusion_bot/entities"
	"time"
)

type ItemType string

const (
	ItemTypeImage        ItemType = "Text to Image"
	ItemTypeVibeTransfer ItemType = "Vibe Transfer"
	ItemTypeImg2Img      ItemType = "Image to Image"
)

type NAIQueueItem struct {
	Type ItemType

	Request     *entities.NovelAIRequest
	Attachments map[string]*entities.MessageAttachment

	Created            time.Time
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
	Interrupt          chan *discordgo.Interaction

	user *discordgo.User
}

func (q *NAIQueueItem) Interaction() *discordgo.Interaction {
	return q.DiscordInteraction
}

func (q *NAIQueue) NewItem(interaction *discordgo.Interaction, options ...func(*NAIQueueItem)) *NAIQueueItem {
	queue := q.DefaultQueueItem()
	queue.DiscordInteraction = interaction

	if queue.DiscordInteraction.Member != nil {
		queue.user = queue.DiscordInteraction.Member.User
	}
	if queue.DiscordInteraction.User != nil {
		queue.user = queue.DiscordInteraction.User
	}

	for _, option := range options {
		option(queue)
	}

	return queue
}

func (q *NAIQueue) DefaultQueueItem() *NAIQueueItem {
	return &NAIQueueItem{
		Type:        ItemTypeImage,
		Request:     entities.DefaultNovelAIRequest(),
		Attachments: nil,
		Created:     time.Now(),
		Interrupt:   nil,
	}
}

func WithPrompt(prompt string) func(*NAIQueueItem) {
	return func(queue *NAIQueueItem) {
		queue.Request.Input = prompt
	}
}
