package novelai

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
)

type ItemType = string

const (
	ItemTypeImage        ItemType = "Text to Image"
	ItemTypeVibeTransfer ItemType = "Vibe Transfer"
	ItemTypeImg2Img      ItemType = "Image to Image"
)

type NAIQueueItem struct {
	Type ItemType

	Request *entities.NovelAIRequest

	Created            time.Time
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
	Interrupt          chan *discordgo.Interaction

	pos  int
	user *discordgo.User
}

func (q *NAIQueueItem) Interaction() *discordgo.Interaction {
	return q.DiscordInteraction
}

func (q *NAIQueue) NewItem(interaction *discordgo.Interaction, options ...func(*NAIQueueItem)) *NAIQueueItem {
	item := q.DefaultQueueItem()
	item.DiscordInteraction = interaction
	item.user = utils.GetUser(interaction)

	for _, option := range options {
		option(item)
	}

	return item
}

func (q *NAIQueue) DefaultQueueItem() *NAIQueueItem {
	return &NAIQueueItem{
		Type:      ItemTypeImage,
		Request:   entities.DefaultNovelAIRequest(),
		Created:   time.Now(),
		Interrupt: nil,
	}
}

func WithPrompt(prompt string) func(*NAIQueueItem) {
	return func(item *NAIQueueItem) {
		item.Request.Input = prompt
	}
}
