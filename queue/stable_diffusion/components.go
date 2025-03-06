package stable_diffusion

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"slices"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/utils"
	"strconv"
	"strings"
	"time"
)

type customID = string

const (
	CheckpointSelect   customID = "imagine_sd_model_name_menu"
	VAESelect          customID = "imagine_vae_model_name_menu"
	HypernetworkSelect customID = "imagine_hypernetwork_model_name_menu"
	DimensionSelect    customID = "imagine_dimension_setting_menu"
	BatchCountSelect   customID = "imagine_batch_count_setting_menu"
	BatchSizeSelect    customID = "imagine_batch_size_setting_menu"

	JSONInput customID = "raw"
)

const (
	RerollButton  customID = "imagine_reroll"
	UpscaleButton customID = "imagine_upscale"
	VariantButton customID = "imagine_variation"
)

var components = map[customID]discordgo.MessageComponent{
	CheckpointSelect:   modelSelectMenu(CheckpointSelect),
	VAESelect:          modelSelectMenu(VAESelect),
	HypernetworkSelect: modelSelectMenu(HypernetworkSelect),

	DimensionSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  DimensionSelect,
				MinValues: nil,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Size: 512x512",
						Value:   "512_512",
						Default: true,
					},
					{
						Label:   "Size: 768x768",
						Value:   "768_768",
						Default: false,
					},
					{
						Label:   "Size: 1024x1024",
						Value:   "1024_1024",
						Default: false,
					},
					{
						Label:   "Size: 832x1216",
						Value:   "832_1216",
						Default: false,
					},
				},
			},
		},
	},
	BatchCountSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  BatchCountSelect,
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch count: 1",
						Value:   "1",
						Default: false,
					},
					{
						Label:   "Batch count: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch count: 4",
						Value:   "4",
						Default: true,
					},
				},
			},
		},
	},
	BatchSizeSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  BatchSizeSelect,
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch size: 1",
						Value:   "1",
						Default: true,
					},
					{
						Label:   "Batch size: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch size: 4",
						Value:   "4",
						Default: false,
					},
				},
			},
		},
	},

	JSONInput: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    JSONInput,
				Label:       "JSON blob",
				Style:       discordgo.TextInputParagraph,
				Placeholder: "{\"height\":768,\"width\":512,\"prompt\":\"masterpiece\"}",
				Value:       "",
				Required:    true,
				MinLength:   1,
				MaxLength:   4000,
			},
		},
	},
}

var minValues = 1

func modelSelectMenu(ID customID) discordgo.ActionsRow {
	display := strings.TrimPrefix(ID, "imagine_")
	display = strings.TrimSuffix(ID, "_model_name_menu")
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    ID,
				Placeholder: fmt.Sprintf("Change %s Model", display),
				MinValues:   &minValues,
				MaxValues:   1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:       display,
						Value:       "Placeholder",
						Description: "Placeholder",
						Default:     false,
					},
				},
			},
		},
	}
}

func (q *SDQueue) components() map[string]queue.Handler {
	h := map[string]queue.Handler{
		DimensionSelect: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
			if len(i.MessageComponentData().Values) == 0 {
				return errors.New("no values for imagine dimension setting menu")
			}

			sizes := strings.Split(i.MessageComponentData().Values[0], "_")

			width := sizes[0]
			height := sizes[1]

			widthInt, err := strconv.Atoi(width)
			if err != nil {
				return fmt.Errorf("error parsing width: %w", err)
			}

			heightInt, err := strconv.Atoi(height)
			if err != nil {
				return fmt.Errorf("error parsing height: %w", err)
			}

			return q.processImagineDimensionSetting(s, i, widthInt, heightInt)
		},

		CheckpointSelect:   q.processImagineModelSetting,
		VAESelect:          q.processImagineModelSetting,
		HypernetworkSelect: q.processImagineModelSetting,

		BatchCountSelect: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
			if len(i.MessageComponentData().Values) == 0 {
				return errors.New("no values for imagine batch count setting menu")
			}

			batchCount := i.MessageComponentData().Values[0]

			batchCountInt, err := strconv.Atoi(batchCount)
			if err != nil {
				return handlers.ErrorEphemeral(s, i.Interaction, "error parsing batch count", err)
			}

			var batchSizeInt int

			// calculate the corresponding batch size
			switch batchCountInt {
			case 1:
				batchSizeInt = 4
			case 2:
				batchSizeInt = 2
			case 4:
				batchSizeInt = 1
			default:
				return fmt.Errorf("unknown batch count: %v", batchCountInt)
			}

			return q.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
		},

		BatchSizeSelect: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
			if len(i.MessageComponentData().Values) == 0 {
				return errors.New("no values for imagine batch count setting menu")
			}

			batchSize := i.MessageComponentData().Values[0]

			batchSizeInt, err := strconv.Atoi(batchSize)
			if err != nil {
				return handlers.ErrorEphemeral(s, i.Interaction, "error parsing batch size", err)
			}

			var batchCountInt int

			// calculate the corresponding batch count
			switch batchSizeInt {
			case 1:
				batchCountInt = 4
			case 2:
				batchCountInt = 2
			case 4:
				batchCountInt = 1
			default:
				return handlers.ErrorEphemeral(s, i.Interaction, fmt.Errorf("unknown batch size: %v", batchSizeInt))
			}

			return q.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
		},

		RerollButton:  q.processImagineReroll,
		UpscaleButton: q.upscaleComponentHandler,
		VariantButton: q.variantComponentHandler,

		handlers.Cancel:    q.removeImagineFromQueue, // Cancel button is used when still in queue
		handlers.Interrupt: q.interrupt,              // Interrupt button is used when currently generating, using the api.Interrupt() method
	}

	for i := range 4 {
		h[UpscaleButton+"_"+strconv.Itoa(i+1)] = q.upscaleComponentHandler
		h[VariantButton+"_"+strconv.Itoa(i+1)] = q.variantComponentHandler
	}

	return h
}

func (q *SDQueue) upscaleComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	interactionIndex := strings.TrimPrefix(customID, UpscaleButton+"_")

	interactionIndexInt, err := strconv.Atoi(interactionIndex)
	if err != nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "error parsing interaction index", err)
	}

	return q.processImagineUpscale(s, i, interactionIndexInt)
}

func (q *SDQueue) variantComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	interactionIndex := strings.TrimPrefix(customID, VariantButton+"_")

	interactionIndexInt, err := strconv.Atoi(interactionIndex)
	if err != nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "error parsing interaction index", err)
	}

	return q.processImagineVariation(s, i, interactionIndexInt)
}

func (q *SDQueue) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	position, queueError := q.Add(&SDQueueItem{
		ImageGenerationRequest: &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				InteractionID: i.Interaction.ID,
				MessageID:     i.Message.ID,
				MemberID:      i.Member.User.ID,
				CreatedAt:     time.Now(),
			},
			TextToImageRequest: new(entities.TextToImageRequest),
		},
		Type:               ItemTypeReroll,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "Error adding imagine to queue", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm reimagining that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		return handlers.Wrap(err)
	}

	return nil
}

func (q *SDQueue) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) error {
	position, err := q.Add(&SDQueueItem{
		Type:               ItemTypeUpscale,
		InteractionIndex:   upscaleIndex,
		DiscordInteraction: i.Interaction,
	})
	if err != nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "Error adding imagine to queue", err)
	}

	return handlers.Wrap(s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm upscaling that for you... You are currently #%d in line.", position),
		},
	}))
}

func (q *SDQueue) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) error {
	position, queueError := q.Add(&SDQueueItem{
		ImageGenerationRequest: &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				InteractionID: i.Interaction.ID,
				MessageID:     i.Message.ID,
				MemberID:      i.Member.User.ID,
				SortOrder:     variationIndex,
				CreatedAt:     time.Now(),
			},
			TextToImageRequest: &entities.TextToImageRequest{},
		},
		Type:               ItemTypeVariation,
		InteractionIndex:   variationIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "Error adding imagine to queue")
	}

	return handlers.Wrap(s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm imagining more variations for you... You are currently #%d in line.", position),
		},
	}))
}

// check if the user using the cancel button is the same user that started the generation, then remove it from the queue
func (q *SDQueue) removeImagineFromQueue(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if utils.GetUser(i).ID != i.Message.InteractionMetadata.User.ID {
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

// check if the user using the interrupt button is the same user that started the generation
func (q *SDQueue) interrupt(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Member == nil {
		return handlers.ErrorEphemeral(s, i.Interaction, "Member not found")
	}

	var mentionedIDs []string

	for _, mention := range i.Message.Mentions {
		mentionedIDs = append(mentionedIDs, mention.ID)
	}

	if len(mentionedIDs) == 0 {
		return handlers.ErrorEphemeral(s, i.Interaction, "Could not determine who started the generation as there are no detected mentions")
	}

	if !slices.Contains(mentionedIDs, i.Member.User.ID) {
		return handlers.ErrorEphemeral(s, i.Interaction,
			// strings.Join with <@ID> and newlines.
			fmt.Sprintf("You can only interrupt your own generations.\nValid users: <@%v>", strings.Join(mentionedIDs, ">\n<@")))
	}

	err := q.Interrupt(i.Interaction)
	if err != nil {
		log.Printf("Error interrupting generation: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction, err)
	}

	return handlers.UpdateFromComponent(s, i.Interaction, "Generation interrupted", handlers.Components[handlers.InterruptDisabled])
}
