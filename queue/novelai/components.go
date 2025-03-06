package novelai

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/utils"
)

type Handler = func(*discordgo.Session, *discordgo.InteractionCreate) error

const (
	prefix = "novelai_"
	cancel = prefix + "cancel"
)

var components = map[string]discordgo.MessageComponent{
	cancel: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: cancel,
			},
		},
	},
}

func (q *NAIQueue) components() map[string]Handler {
	return map[string]Handler{
		cancel: q.removeImagineFromQueue,
	}
}

func (q *NAIQueue) removeImagineFromQueue(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if utils.GetUser(i.Interaction).ID != i.Message.InteractionMetadata.User.ID {
		return handlers.ErrorEphemeral(s, i.Interaction, "You can only cancel your own generations")
	}

	log.Printf("Removing imagine from queue: %#v", i.Message.InteractionMetadata)

	err := q.Remove(i.Message.InteractionMetadata)
	if err != nil {
		log.Printf("Error removing imagine from queue: %v", err)
		return handlers.ErrorEdit(s, i.Interaction, "Error removing imagine from queue")
	}
	log.Printf("Removed imagine from queue: %#v", i.Message.InteractionMetadata)

	return handlers.UpdateFromComponent(s, i.Interaction, "Generation cancelled", handlers.Components[handlers.DeleteButton])
}
