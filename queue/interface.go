package queue

import (
	"github.com/bwmarrin/discordgo"
)

type Queue[item Item] interface {
	Start(botSession *discordgo.Session)
	NewItem(interaction *discordgo.Interaction, options ...func(item)) item
	Add(item item) (int, error)
	Remove(message *discordgo.MessageInteraction) error
	Interrupt(i *discordgo.Interaction) error

	Stop()
}

type StartStop interface {
	Start(botSession *discordgo.Session)
	Stop()
}

type Item interface {
	Interaction() *discordgo.Interaction
}
