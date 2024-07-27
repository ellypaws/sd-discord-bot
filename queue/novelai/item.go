package novelai

import (
	"github.com/bwmarrin/discordgo"
	"stable_diffusion_bot/entities"
	"time"
)

type ItemType = string

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
	item := q.DefaultQueueItem()
	item.DiscordInteraction = interaction

	if item.DiscordInteraction.Member != nil {
		item.user = item.DiscordInteraction.Member.User
	}
	if item.DiscordInteraction.User != nil {
		item.user = item.DiscordInteraction.User
	}

	for _, option := range options {
		option(item)
	}

	return item
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
	return func(item *NAIQueueItem) {
		item.Request.Input = prompt
	}
}
