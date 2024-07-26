package discord_bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/queue/stable_diffusion"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// TODO: Verify we're using the correct response function such as ErrorEdit or ErrorEphemeral.
// The former is used when we want to edit the original message, the latter acts as the initial response to an interaction.
var componentHandlers = map[handlers.Component]Handler{
	handlers.DeleteButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, err)
		}
		return nil
	},

	handlers.DeleteGeneration: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	handlers.DeleteAboveButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	handlers.DimensionSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

		return bot.processImagineDimensionSetting(s, i, widthInt, heightInt)
	},

	handlers.CheckpointSelect:   (*botImpl).processImagineModelSetting,
	handlers.VAESelect:          (*botImpl).processImagineModelSetting,
	handlers.HypernetworkSelect: (*botImpl).processImagineModelSetting,

	handlers.BatchCountSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

		return bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
	},

	handlers.BatchSizeSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

		return bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
	},

	handlers.RerollButton: (*botImpl).processImagineReroll,

	handlers.UpscaleButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, string(handlers.UpscaleButton+"_"))

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, "error parsing interaction index", err)
		}

		return bot.processImagineUpscale(s, i, interactionIndexInt)
	},

	handlers.VariantButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) error {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			return handlers.ErrorEphemeral(s, i.Interaction, "error parsing interaction index", err)
		}

		return bot.processImagineVariation(s, i, interactionIndexInt)
	},

	handlers.Cancel:    (*botImpl).removeImagineFromQueue, // Cancel button is used when still in queue
	handlers.Interrupt: (*botImpl).interrupt,              // Interrupt button is used when currently generating, using the api.Interrupt() method
}

func (b *botImpl) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	position, queueError := b.config.ImagineQueue.Add(&stable_diffusion.SDQueueItem{
		ImageGenerationRequest: &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				InteractionID: i.Interaction.ID,
				MessageID:     i.Message.ID,
				MemberID:      i.Member.User.ID,
				CreatedAt:     time.Now(),
			},
			TextToImageRequest: &entities.TextToImageRequest{},
		},
		Type:               stable_diffusion.ItemTypeReroll,
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

func (b *botImpl) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) error {
	position, err := b.config.ImagineQueue.Add(&stable_diffusion.SDQueueItem{
		Type:               stable_diffusion.ItemTypeUpscale,
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

func (b *botImpl) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) error {
	position, queueError := b.config.ImagineQueue.Add(&stable_diffusion.SDQueueItem{
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
		Type:               stable_diffusion.ItemTypeVariation,
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

// patch from upstream
func (b *botImpl) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	config, err := b.config.StableDiffusionApi.GetConfig()
	if err != nil {
		log.Printf("Error retrieving config: %v", err)
	} else {
		populateOption(b, handlers.CheckpointSelect, stable_diffusion_api.CheckpointCache, config)
		populateOption(b, handlers.VAESelect, stable_diffusion_api.VAECache, config)
		populateOption(b, handlers.HypernetworkSelect, stable_diffusion_api.HypernetworkCache, config)
	}

	// set default dimension from config
	dimensions := handlers.Components[handlers.DimensionSelect].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	dimensions.Options[0].Default = settings.Width == 512 && settings.Height == 512
	dimensions.Options[1].Default = settings.Width == 768 && settings.Height == 768
	handlers.Components[handlers.DimensionSelect].(discordgo.ActionsRow).Components[0] = dimensions

	batchSlice := []int{1, 2, 4}
	// set default batch count from config
	batchCount := handlers.Components[handlers.BatchCountSelect].(discordgo.ActionsRow)
	for i, option := range batchCount.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchCount {
			option.Default = true
		} else {
			option.Default = false
		}
		batchCount.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchCountSelect] = batchCount

	// set the default batch size from config
	batchSize := handlers.Components[handlers.BatchSizeSelect].(discordgo.ActionsRow)
	for i, option := range batchSize.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchSize {
			option.Default = true
		} else {
			option.Default = false
		}
		batchSize.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchSizeSelect] = batchSize

	return []discordgo.MessageComponent{
		handlers.Components[handlers.CheckpointSelect],
		handlers.Components[handlers.VAESelect],
		handlers.Components[handlers.HypernetworkSelect],
		handlers.Components[handlers.DimensionSelect],
		//handlers.Components[handlers.BatchCountSelect],
		handlers.Components[handlers.BatchSizeSelect],
	}
}

// populateOption will fill in the options for a given dropdown component that implements stable_diffusion_api.Cacheable
func populateOption(b *botImpl, handler handlers.Component, cache stable_diffusion_api.Cacheable, config *entities.Config) {
	checkpointDropdown := handlers.Components[handler].(discordgo.ActionsRow)
	var modelOptions []discordgo.SelectMenuOption

	models, err := cache.GetCache(b.config.StableDiffusionApi)
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
		return
	} else {
		var modelNames []string
		var currentModel *string

		switch toRange := models.(type) {
		case *stable_diffusion_api.SDModels:
			currentModel = config.SDModelCheckpoint
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.ModelName),
					Value: shortenString(model.Title),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.ModelName)
				}
				if model.Hash != nil {
					modelOptions[i].Description = fmt.Sprintf("[%v]", *model.Hash)
				}
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.VAEModels:
			currentModel = config.SDVae
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.ModelName),
					Value: shortenString(model.ModelName),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.ModelName)
				}
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.HypernetworkModels:
			currentModel = config.SDHypernetwork
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.Name),
					Value: shortenString(model.Name),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.Name)
				}
				modelNames = append(modelNames, model.Name)
			}
		}

		var Default bool
		for i, model := range modelOptions {
			if model.Default {
				modelOptions[i].Emoji = &discordgo.ComponentEmoji{
					Name: "✨",
				}
				Default = true
				break
			}
		}

		if currentModel != nil && *currentModel != "" && *currentModel != "None" && !Default {
			modelOptions = append([]discordgo.SelectMenuOption{{
				Label:   shortenString(*currentModel),
				Value:   shortenString(*currentModel),
				Default: true,
				Emoji: &discordgo.ComponentEmoji{
					Name: "✨",
				},
			}}, modelOptions...)
		}

		if len(modelOptions) == 0 {
			modelOptions = append(modelOptions, discordgo.SelectMenuOption{
				Label:       "No models found",
				Value:       "None",
				Description: "Are you sure you have the right API URL?",
				Default:     false,
			})
		} else {
			modelOptions = append([]discordgo.SelectMenuOption{{
				Label:       "None",
				Value:       "None",
				Description: "Unset the model",
				Emoji: &discordgo.ComponentEmoji{
					Name: "❌",
				},
			}}, modelOptions...)
		}
		component := checkpointDropdown.Components[0].(discordgo.SelectMenu)
		component.Options = modelOptions

		handlers.Components[handler].(discordgo.ActionsRow).Components[0] = component
	}
}

func (b *botImpl) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) error {
	botSettings, err := b.config.ImagineQueue.(*stable_diffusion.SDQueue).UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			return handlers.Wrap(err)
		}

		return nil
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	return handlers.Wrap(err)
}

func (b *botImpl) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) error {
	botSettings, err := b.config.ImagineQueue.(*stable_diffusion.SDQueue).UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			return handlers.Wrap(err)
		}

		return nil
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose default settings for the imagine command:",
			Components: messageComponents,
		},
	})
	return handlers.Wrap(err)
}

func (b *botImpl) processImagineModelSetting(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if len(i.MessageComponentData().Values) == 0 {
		return fmt.Errorf("no values for %v", i.MessageComponentData().CustomID)
	}
	newModelName := i.MessageComponentData().Values[0]

	var config entities.Config
	var modelType string
	switch i.MessageComponentData().CustomID {
	case string(handlers.CheckpointSelect):
		config = entities.Config{SDModelCheckpoint: &newModelName}
		modelType = "checkpoint"
	case string(handlers.VAESelect):
		config = entities.Config{SDVae: &newModelName}
		modelType = "vae"
	case string(handlers.HypernetworkSelect):
		config = entities.Config{SDHypernetwork: &newModelName}
		modelType = "hypernetwork"
	}

	err := handlers.UpdateFromComponent(s, i.Interaction,
		fmt.Sprintf("Updating [**%v**] model to `%v`...", modelType, newModelName),
		i.Interaction.Message.Components,
	)
	if err != nil {
		return err
	}

	err = b.config.StableDiffusionApi.UpdateConfiguration(config)
	if err != nil {
		log.Printf("error updating sd model name settings: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction,
			fmt.Sprintf("Error updating [%v] model name settings...", modelType))
	}

	botSettings, err := b.config.ImagineQueue.(*stable_diffusion.SDQueue).GetBotDefaultSettings()
	if err != nil {
		log.Printf("error retrieving bot settings: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction, "Error retrieving bot settings...")
	}

	newComponents := b.settingsMessageComponents(botSettings)
	_, err = handlers.EditInteractionResponse(s, i.Interaction,
		fmt.Sprintf("Updated [**%v**] model to `%v`", modelType, newModelName),
		newComponents,
	)
	if err != nil {
		return err
	}

	time.AfterFunc(5*time.Second, func() {
		_, _ = handlers.EditInteractionResponse(s, i.Interaction,
			"Choose default settings for the imagine command:",
			newComponents,
		)
	})

	return nil
}

// check if the user using the cancel button is the same user that started the generation, then remove it from the queue
func (b *botImpl) removeImagineFromQueue(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Member.User.ID != i.Message.Interaction.User.ID {
		return handlers.ErrorEphemeral(s, i.Interaction, "You can only cancel your own generations")
	}

	log.Printf("Removing imagine from queue: %#v", i.Message.Interaction)

	err := b.config.ImagineQueue.Remove(i.Message.Interaction)
	if err != nil {
		log.Printf("Error removing imagine from queue: %v", err)
		return handlers.ErrorEdit(s, i.Interaction, "Error removing imagine from queue")
	}
	log.Printf("Removed imagine from queue: %#v", i.Message.Interaction)

	return handlers.UpdateFromComponent(s, i.Interaction, "Generation cancelled", handlers.Components[handlers.DeleteButton])
}

// check if the user using the interrupt button is the same user that started the generation
func (b *botImpl) interrupt(s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	err := b.config.ImagineQueue.Interrupt(i.Interaction)
	if err != nil {
		log.Printf("Error interrupting generation: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction, err)
	}

	return handlers.UpdateFromComponent(s, i.Interaction, "Generation interrupted", handlers.Components[handlers.InterruptDisabled])
}
