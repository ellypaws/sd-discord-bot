package discord_bot

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/queue"
)

// componentHandlers is a map of common component handlers.
// TODO: Verify we're using the correct response function such as ErrorEdit or ErrorEphemeral.
// The former is used when we want to edit the original message, the latter acts as the initial response to an interaction.
var componentHandlers = queue.Components{
	handlers.DeleteButton: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, err)
		}
		return nil
	},

	handlers.DeleteGeneration: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
		if err := handlers.EphemeralThink(s, i); err != nil {
			return err
		}

		var originalInteractionUser string

		switch {
		case i.Message.Interaction != nil && i.Message.Interaction.User != nil:
			originalInteractionUser = i.Message.Interaction.User.ID
		case i.Message.Interaction != nil && i.Message.Interaction.Member != nil:
			originalInteractionUser = i.Message.Interaction.Member.User.ID
		case len(i.Message.Mentions) > 0:
			log.Printf("WARN: Using mentions to determine original interaction user")
			originalInteractionUser = i.Message.Mentions[0].ID
		default:
			err := handlers.ErrorEdit(s, i.Interaction, "Unable to determine original interaction user")
			if err != nil {
				return err
			}
			log.Printf("Unable to determine original interaction user: %#v", i)
			byteArr, _ := json.MarshalIndent(i, "", "  ")
			log.Printf("Interaction: %v", string(byteArr))
			return nil
		}

		if i.Member.User.ID != originalInteractionUser {
			return handlers.ErrorEdit(s, i.Interaction, "You can only delete your own generations")
		}
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			return handlers.ErrorEdit(s, i.Interaction, fmt.Errorf("error deleting message: %w", err))
		}

		_, err = handlers.EditInteractionResponse(s, i.Interaction, "Generation deleted")
		return err
	},

	handlers.DeleteAboveButton: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
		msg, err := s.InteractionResponse(i.Interaction)
		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, fmt.Errorf("failed to retrieve interaction response: %v, %v", i, err))
		}

		err = s.ChannelMessageDelete(i.ChannelID, msg.ID)

		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, err)
		}

		return nil
	},
}
