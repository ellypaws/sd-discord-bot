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

	Registrar

	Stop()
}

type Handler = func(*discordgo.Session, *discordgo.InteractionCreate) error

type Command = string
type CommandHandlers = map[discordgo.InteractionType]map[Command]Handler
type Components map[string]Handler

type Registrar interface {
	Commands() []*discordgo.ApplicationCommand
	Handlers() CommandHandlers
	Components() Components
}

type HandlerStartStopper interface {
	Registrar
	StartStop
}

type StartStop interface {
	Start(botSession *discordgo.Session)
	Stop()
}

type Item interface {
	Interaction() *discordgo.Interaction
}
