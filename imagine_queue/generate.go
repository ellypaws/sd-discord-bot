package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/stable_diffusion_api"
	"time"
)

func (q *queueImplementation) processImagineGrid(newGeneration *entities.ImageGenerationRequest, c *QueueItem) error {
	config, err := q.stableDiffusionAPI.GetConfig()
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return err
	} else {
		if !ptrStringCompare(newGeneration.Checkpoint, config.SDModelCheckpoint) ||
			!ptrStringCompare(newGeneration.VAE, config.SDVae) ||
			!ptrStringCompare(newGeneration.Hypernetwork, config.SDHypernetwork) {
			handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction,
				fmt.Sprintf("Changing models to: \n**Checkpoint**: `%v` -> `%v`\n**VAE**: `%v` -> `%v`\n**Hypernetwork**: `%v` -> `%v`",
					safeDereference(config.SDModelCheckpoint), safeDereference(newGeneration.Checkpoint),
					safeDereference(config.SDVae), safeDereference(newGeneration.VAE),
					safeDereference(config.SDHypernetwork), safeDereference(newGeneration.Hypernetwork),
				),
				handlers.Components[handlers.CancelDisabled])

			// Insert code to update the configuration here
			err := q.stableDiffusionAPI.UpdateConfiguration(q.switchModel(newGeneration, config, []stable_diffusion_api.Cacheable{
				stable_diffusion_api.CheckpointCache,
				stable_diffusion_api.VAECache,
				stable_diffusion_api.HypernetworkCache,
			}))
			if err != nil {
				log.Printf("Error updating configuration: %v", err)
				return err
			}
			config, err = q.stableDiffusionAPI.GetConfig()
			if err != nil {
				log.Printf("Error getting config: %v", err)
				return err
			}
		}
	}

	log.Printf("Processing imagine #%s: %v\n", c.DiscordInteraction.ID, newGeneration.Prompt)

	newContent := imagineMessageContent(newGeneration, c.DiscordInteraction.Member.User, 0)

	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, newGeneration, c, c.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message, err := q.botSession.InteractionResponseEdit(c.DiscordInteraction, webhook)
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
		return err
	}

	// store message ID in c.DiscordInteraction.Message
	if c.DiscordInteraction != nil && c.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting c.DiscordInteraction.Message to message: %v", message)
		c.DiscordInteraction.Message = message
	}

	defaultBatchCount, err := q.defaultBatchCount()
	if err != nil {
		log.Printf("Error getting default batch count: %v", err)
		return err
	}

	defaultBatchSize, err := q.defaultBatchSize()
	if err != nil {
		log.Printf("Error getting default batch size: %v", err)
		return err
	}
	newGeneration.InteractionID = c.DiscordInteraction.ID
	newGeneration.MessageID = message.ID
	newGeneration.MemberID = c.DiscordInteraction.Member.User.ID
	newGeneration.SortOrder = 0
	newGeneration.BatchCount = defaultBatchCount
	newGeneration.BatchSize = defaultBatchSize
	newGeneration.Processed = true

	// return newGeneration from image_generations.Create as we need newGeneration.CreatedAt later on
	newGeneration, err = q.imageGenerationRepo.Create(context.Background(), newGeneration)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
		return err
	}

	generationDone := make(chan bool)

	go func() {
		for {
			select {
			case c.DiscordInteraction = <-c.Interrupt:
				err := q.stableDiffusionAPI.Interrupt()
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
					return
				}
				message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction, "Generation Interrupted", webhook, handlers.Components[handlers.DeleteGeneration])
				if c.DiscordInteraction.Message == nil && message != nil {
					log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
					c.DiscordInteraction.Message = message
				}
			case <-generationDone:
				return
			case <-time.After(1 * time.Second):
				progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
				if progressErr != nil {
					log.Printf("Error getting current progress: %v", progressErr)
					handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, fmt.Sprintf("Error getting current progress: %v", progressErr))

					return
				}

				if progress.Progress == 0 {
					continue
				}

				progressContent := imagineMessageContent(newGeneration, c.DiscordInteraction.Member.User, progress.Progress)

				_, progressErr = q.botSession.InteractionResponseEdit(c.DiscordInteraction, &discordgo.WebhookEdit{
					Content: &progressContent,
				})
				if progressErr != nil {
					log.Printf("Error editing interaction: %v", err)
				}
			}
		}
	}()

	switch c.Type {
	case ItemTypeImagine, ItemTypeReroll, ItemTypeVariation:
		resp, err := q.stableDiffusionAPI.TextToImageRequest(newGeneration.TextToImageRequest)

		generationDone <- true

		if err != nil {
			log.Printf("Error processing image: %v\n", err)

			errorContent := fmt.Sprint("I'm sorry, but I had a problem imagining your image. ", err)

			//_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			//	Content: &errorContent,
			//})

			handlers.ErrorHandler(q.botSession, c.DiscordInteraction, errorContent)
			//handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, errorContent)

			return err
		}

		// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
		embed = generationEmbedDetails(embed, newGeneration, c, c.Interrupt != nil)

		log.Printf("embed: %v", embed)

		log.Printf("Seeds: %v Subseeds:%v", resp.Seeds, resp.Subseeds)

		imageBufs := make([]*bytes.Buffer, len(resp.Images))

		for idx, image := range resp.Images {
			decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
			if decodeErr != nil {
				log.Printf("Error decoding image: %v\n", decodeErr)
			}

			imageBufs[idx] = bytes.NewBuffer(decodedImage)
		}

		for idx := range resp.Seeds {
			subGeneration := newGeneration
			subGeneration.SortOrder = idx + 1
			subGeneration.Seed = resp.Seeds[idx]
			subGeneration.Subseed = int64(resp.Subseeds[idx])
			subGeneration.Checkpoint = config.SDModelCheckpoint
			subGeneration.VAE = config.SDVae
			subGeneration.Hypernetwork = config.SDHypernetwork

			_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
			if createErr != nil {
				log.Printf("Error creating image generation record: %v\n", createErr)
			}
		}

		var thumbnailBuffers []*bytes.Buffer

		if c.ControlnetItem.MessageAttachment != nil {
			decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(c.ControlnetItem.MessageAttachment.Image))
			if err != nil {
				log.Printf("Error decoding image: %v\n", err)
			}
			thumbnailBuffers = append(thumbnailBuffers, bytes.NewBuffer(decodedBytes))
		}

		if len(imageBufs) > 4 {
			log.Printf("received extra images: len(imageBufs): %v, controlnet: %v", len(imageBufs), c.ControlnetItem.Enabled)
			thumbnailBuffers = append(thumbnailBuffers, imageBufs[4:]...)
		}

		empty := ""

		webhook = &discordgo.WebhookEdit{
			Content:    &empty,
			Components: rerollVariationComponents(min(len(imageBufs), 4), c.Type == ItemTypeImg2Img),
		}

		imageEmbedFromBuffers(webhook, embed, imageBufs[:min(len(imageBufs), 4)], thumbnailBuffers)

		_, err = q.botSession.InteractionResponseEdit(c.DiscordInteraction, webhook)
		if err != nil {
			log.Printf("Error editing interaction: %v\n", err)
			return err
		}
	case ItemTypeImg2Img:
		err, done := q.imageToImage(newGeneration, c, generationDone)
		if done {
			return err
		}
	}
	//handlers.EphemeralFollowup(q.botSession, imagine.DiscordInteraction, "Delete generation", handlers.Components[handlers.DeleteAboveButton])

	return nil
}
