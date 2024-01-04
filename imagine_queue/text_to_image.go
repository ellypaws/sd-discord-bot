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
	"strings"
	"time"
)

func (q *queueImplementation) processImagineGrid(queue *entities.QueueItem) error {
	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	} else {
		config, err = q.updateModels(request, queue, config)
		if err != nil {
			return fmt.Errorf("error updating models: %w", err)
		}
	}

	log.Printf("Processing imagine #%s: %v\n", queue.DiscordInteraction.ID, textToImage.Prompt)

	embed, webhook, err := showInitialMessage(queue, q)
	if err != nil {
		return err
	}

	request, err = q.recordToRepository(request, err)
	if err != nil {
		return fmt.Errorf("error recording to repository: %w", err)
	}

	generationDone := make(chan bool)

	go q.updateProgressBar(queue, generationDone, config, originalConfig, webhook)

	switch queue.Type {
	case ItemTypeImagine, ItemTypeReroll, ItemTypeVariation, ItemTypeRaw:
		response, err := q.textInference(queue)
		generationDone <- true
		if err != nil {
			return fmt.Errorf("error inferencing generation: %w", err)
		}

		if response == nil {
			return fmt.Errorf("response of type %v is nil: %v", queue.Type, err)
		}

		q.recordSeeds(response, request, config)

		err = q.showFinalMessage(queue, response, embed, webhook)
		if err != nil {
			return err
		}
	case ItemTypeImg2Img:
		err := q.imageToImage(generationDone, embed, webhook)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown queue type: %v", queue.Type)
	}

	return nil
}

func showInitialMessage(queue *entities.QueueItem, q *queueImplementation) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := queue.ImageGenerationRequest
	newContent := imagineMessageSimple(request, queue.DiscordInteraction.Member.User, 0, nil, nil)

	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, queue, queue.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, webhook)

	err := q.storeMessageInteraction(queue, message)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving message interaction: %v", err)
	}

	return embed, webhook, nil
}

func (q *queueImplementation) storeMessageInteraction(queue *entities.QueueItem, message *discordgo.Message) (err error) {
	request := queue.ImageGenerationRequest

	if queue.DiscordInteraction == nil {
		return fmt.Errorf("queue.DiscordInteraction is nil")
	}

	if message == nil {
		message, err = q.botSession.InteractionResponse(queue.DiscordInteraction)
		if err != nil {
			return err
		}
	}

	// store message ID in c.DiscordInteraction.Message
	queue.DiscordInteraction.Message = message

	request.InteractionID = queue.DiscordInteraction.ID
	request.MessageID = queue.DiscordInteraction.Message.ID
	request.MemberID = queue.DiscordInteraction.Member.User.ID
	request.SortOrder = 0
	request.Processed = true
	return nil
}

func (q *queueImplementation) showFinalMessage(queue *entities.QueueItem, response *entities.TextToImageResponse, embed *discordgo.MessageEmbed, webhook *discordgo.WebhookEdit) error {
	request := queue.ImageGenerationRequest
	totalImages := totalImageCount(request)

	imageBuffers, thumbnailBuffers := retrieveImagesFromResponse(response, queue)

	mention := fmt.Sprintf("<@%v>", queue.DiscordInteraction.Member.User.ID)
	// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
	embed = generationEmbedDetails(embed, queue, queue.Interrupt != nil)

	webhook = &discordgo.WebhookEdit{
		Content:    &mention,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: rerollVariationComponents(min(len(imageBuffers), totalImages), queue.Type == ItemTypeImg2Img || (queue.Raw != nil && queue.Raw.Debug)),
	}

	if err := imageEmbedFromBuffers(webhook, embed, imageBuffers[:min(len(imageBuffers), totalImages)], thumbnailBuffers); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, webhook)
	return nil
}

func (q *queueImplementation) recordSeeds(response *entities.TextToImageResponse, request *entities.ImageGenerationRequest, config *entities.Config) {
	log.Printf("Seeds: %v Subseeds:%v", response.Seeds, response.Subseeds)
	for idx := range *response.Seeds {
		subGeneration := request
		subGeneration.SortOrder = idx + 1
		subGeneration.Seed = (*response.Seeds)[idx]
		subGeneration.Subseed = (*response.Subseeds)[idx]
		subGeneration.Checkpoint = response.Info.SDModelName
		subGeneration.VAE = response.Info.SDVaeName
		subGeneration.Hypernetwork = config.SDHypernetwork

		_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
		if createErr != nil {
			log.Printf("Error creating image generation record: %v\n", createErr)
		}
	}
}

func totalImageCount(request *entities.ImageGenerationRequest) int {
	const maxImages = 4
	if request.BatchSize == 0 {
		log.Printf("Warning: newGeneration.Batchsize == 0")
		request.BatchSize = between(request.BatchSize, 1, maxImages)
	}
	if request.NIter == 0 {
		log.Printf("Warning: newGeneration.NIter == 0")
		request.NIter = between(request.NIter, 1, maxImages/request.BatchSize)
	}

	totalImages := request.NIter * request.BatchSize
	return totalImages
}

func retrieveImagesFromResponse(response *entities.TextToImageResponse, queue *entities.QueueItem) (images, thumbnails []*bytes.Buffer) {
	images = make([]*bytes.Buffer, len(response.Images))

	for idx, image := range response.Images {
		decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
		if decodeErr != nil {
			log.Printf("Error decoding image: %v\n", decodeErr)
		}

		images[idx] = bytes.NewBuffer(decodedImage)
	}

	if queue.ControlnetItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(queue.ControlnetItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, bytes.NewBuffer(decodedBytes))
	}

	if queue.Img2ImgItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(queue.Img2ImgItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, bytes.NewBuffer(decodedBytes))
	}

	generation := queue.ImageGenerationRequest
	totalImages := totalImageCount(generation)
	if len(images) > totalImages {
		log.Printf("received extra images: len(imageBufs): %v, controlnet: %v", len(images), queue.ControlnetItem.Enabled)
		thumbnails = append(thumbnails, images[totalImages:]...)
	}

	return images, thumbnails
}

func (q *queueImplementation) textInference(queue *entities.QueueItem) (response *entities.TextToImageResponse, err error) {
	generation := queue.ImageGenerationRequest
	switch queue.Type {
	case ItemTypeRaw:
		if q.currentImagine.Raw.Unsafe {
			response, err = q.stableDiffusionAPI.TextToImageRaw(q.currentImagine.Raw.Blob)
		} else {
			marshal, marshalErr := q.currentImagine.Raw.Marshal()
			if marshalErr != nil {
				return nil, fmt.Errorf("error marshalling raw: %w", marshalErr)
			}
			response, err = q.stableDiffusionAPI.TextToImageRaw(marshal)
		}
	default:
		response, err = q.stableDiffusionAPI.TextToImageRequest(generation.TextToImageRequest)
	}
	return response, err
}

func (q *queueImplementation) recordToRepository(request *entities.ImageGenerationRequest, err error) (*entities.ImageGenerationRequest, error) {
	var ok bool
	if request.Prompt, ok = strings.CutSuffix(request.Prompt, "{DEBUG}"); ok {
		byteArr, _ := request.TextToImageRequest.Marshal()
		log.Printf("{DEBUG} TextToImageRequest: %v", string(byteArr))
	}

	// return newGeneration from image_generations.Create as we need newGeneration.CreatedAt later on
	request, err = q.imageGenerationRepo.Create(context.Background(), request)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
		return nil, err
	}
	return request, nil
}

func (q *queueImplementation) updateProgressBar(queue *entities.QueueItem, generationDone chan bool, config, originalConfig *entities.Config, webhook *discordgo.WebhookEdit) {
	request := queue.ImageGenerationRequest
	for {
		select {
		case queue.DiscordInteraction = <-queue.Interrupt:
			err := q.stableDiffusionAPI.Interrupt()
			if err != nil {
				errorResponse(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
				return
			}
			message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, "Generation Interrupted", webhook, handlers.Components[handlers.DeleteGeneration])
			if queue.DiscordInteraction.Message == nil && message != nil {
				log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
				queue.DiscordInteraction.Message = message
			}
		case <-generationDone:
			err := q.revertModels(config, originalConfig)
			if err != nil {
				errorResponse(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
			}
			return
		case <-time.After(1 * time.Second):
			progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
			if progressErr != nil {
				log.Printf("Error getting current progress: %v", progressErr)
				errorResponse(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error getting current progress: %v", progressErr))
				return
			}

			if progress.Progress == 0 {
				continue
			}

			var ram, cuda *entities.ReadableMemory
			mem, err := q.stableDiffusionAPI.GetMemory()
			if err != nil {
				log.Printf("Error getting memory: %v", err)
			} else {
				ram = mem.RAM.Readable()
				cuda = mem.Cuda.Readable()
			}

			mem, err = stable_diffusion_api.GetMemory()
			if err != nil {
				log.Printf("Error getting memory: %v", err)
			} else {
				ram = mem.RAM.Readable()
			}

			progressContent := imagineMessageSimple(request, queue.DiscordInteraction.Member.User, progress.Progress, ram, cuda)

			// TODO: Use handlers.Responses[handlers.EditInteractionResponse] instead and adjust to return errors
			_, progressErr = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &progressContent,
			})
			if progressErr != nil {
				log.Printf("Error editing interaction: %v", progressErr)
				return
			}
		}
	}
}

func (q *queueImplementation) revertModels(config *entities.Config, originalConfig *entities.Config) error {
	if !ptrStringCompare(config.SDModelCheckpoint, originalConfig.SDModelCheckpoint) ||
		!ptrStringCompare(config.SDVae, originalConfig.SDVae) ||
		!ptrStringCompare(config.SDHypernetwork, originalConfig.SDHypernetwork) {
		log.Printf("Switching back to original models: %v, %v, %v",
			safeDereference(originalConfig.SDModelCheckpoint),
			safeDereference(originalConfig.SDVae),
			safeDereference(originalConfig.SDHypernetwork),
		)
		return q.stableDiffusionAPI.UpdateConfiguration(entities.Config{
			SDModelCheckpoint: originalConfig.SDModelCheckpoint,
			SDVae:             originalConfig.SDVae,
			SDHypernetwork:    originalConfig.SDHypernetwork,
		})
	}
	return nil
}

func (q *queueImplementation) updateModels(request *entities.ImageGenerationRequest, c *entities.QueueItem, config *entities.Config) (*entities.Config, error) {
	if !ptrStringCompare(request.Checkpoint, config.SDModelCheckpoint) ||
		!ptrStringCompare(request.VAE, config.SDVae) ||
		!ptrStringCompare(request.Hypernetwork, config.SDHypernetwork) {
		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction,
			fmt.Sprintf("Changing models to: \n**Checkpoint**: `%v` -> `%v`\n**VAE**: `%v` -> `%v`\n**Hypernetwork**: `%v` -> `%v`",
				safeDereference(config.SDModelCheckpoint), safeDereference(request.Checkpoint),
				safeDereference(config.SDVae), safeDereference(request.VAE),
				safeDereference(config.SDHypernetwork), safeDereference(request.Hypernetwork),
			),
			handlers.Components[handlers.CancelDisabled])

		// Insert code to update the configuration here
		err := q.stableDiffusionAPI.UpdateConfiguration(
			q.lookupModel(request, config,
				[]stable_diffusion_api.Cacheable{
					stable_diffusion_api.CheckpointCache,
					stable_diffusion_api.VAECache,
					stable_diffusion_api.HypernetworkCache,
				}))
		if err != nil {
			log.Printf("Error updating configuration: %v", err)
			return nil, err
		}
		config, err = q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
			return nil, err
		}
		request.Checkpoint = config.SDModelCheckpoint
		request.VAE = config.SDVae
		request.Hypernetwork = config.SDHypernetwork
	}
	return config, nil
}
