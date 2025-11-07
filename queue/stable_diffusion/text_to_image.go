package stable_diffusion

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
)

func (q *SDQueue) processImagineGrid(queue *SDQueueItem) error {
	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	config, originalConfig, err := q.switchToModels(queue)
	if err != nil {
		return fmt.Errorf("error switching to models: %w", err)
	}

	q.logger.Info("Processing imagine", "id", queue.DiscordInteraction.ID, "prompt", textToImage.Prompt)

	embed, webhook, err := showInitialMessage(queue, q)
	if err != nil {
		return err
	}

	request, err = q.recordToRepository(request)
	if err != nil {
		return fmt.Errorf("error recording to repository: %w", err)
	}

	generationDone := make(chan bool, 1)
	defer close(generationDone)

	go q.updateProgressBar(queue, generationDone, webhook)

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
		images, err := q.imageToImage()
		generationDone <- true
		if err != nil {
			return err
		}

		err = q.showFinalMessage(queue, &entities.TextToImageResponse{Images: images}, embed, webhook)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown queue type: %v", queue.Type)
	}

	err = q.revertModels(config, originalConfig)
	if err != nil {
		return handlers.ErrorFollowupEphemeral(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
	}

	return nil
}

func showInitialMessage(queue *SDQueueItem, q *SDQueue) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := queue.ImageGenerationRequest
	newContent := imagineMessageSimple(request, utils.GetUser(queue.DiscordInteraction), 0, nil, nil)

	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, queue, queue.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message, err := handlers.EditInteractionResponse(q.botSession, queue.DiscordInteraction, webhook)
	if err != nil {
		return nil, nil, err
	}

	err = q.storeMessageInteraction(queue, message)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving message interaction: %v", err)
	}

	return embed, webhook, nil
}

func (q *SDQueue) storeMessageInteraction(queue *SDQueueItem, message *discordgo.Message) (err error) {
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
	request.MemberID = utils.GetUser(queue.DiscordInteraction).ID
	request.SortOrder = 0
	request.Processed = true
	return nil
}

func (q *SDQueue) showFinalMessage(queue *SDQueueItem, response *entities.TextToImageResponse, embed *discordgo.MessageEmbed, webhook *discordgo.WebhookEdit) error {
	request := queue.ImageGenerationRequest
	totalImages := q.totalImageCount(request)

	imageBuffers, thumbnailBuffers := q.retrieveImagesFromResponse(response, queue)

	mention := fmt.Sprintf("<@%v>", utils.GetUser(queue.DiscordInteraction).ID)
	// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
	embed = generationEmbedDetails(embed, queue, queue.Interrupt != nil)

	webhook = &discordgo.WebhookEdit{
		Content:    &mention,
		Components: rerollVariationComponents(min(len(imageBuffers), totalImages), queue.Type == ItemTypeImg2Img || (queue.Raw != nil && queue.Raw.Debug)),
	}

	if err := utils.EmbedImages(webhook, embed, imageBuffers[:min(len(imageBuffers), totalImages)], thumbnailBuffers, q.compositor); err != nil {
		return fmt.Errorf("error creating image embed: %w", err)
	}

	_, err := handlers.EditInteractionResponse(q.botSession, queue.DiscordInteraction, webhook)
	return err
}

func (q *SDQueue) recordSeeds(response *entities.TextToImageResponse, request *entities.ImageGenerationRequest, config *entities.Config) {
	q.logger.Info("Seed information", "seeds", response.Seeds, "subseeds", response.Subseeds)
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
			q.logger.Error("Failed to create image generation record", "error", createErr)
		}
	}
}

func (q *SDQueue) totalImageCount(request *entities.ImageGenerationRequest) int {
	if request.BatchSize == 0 {
		q.logger.Warn("BatchSize is zero", "requestID", request.ID)
		request.BatchSize = max(request.BatchSize, 1)
	}
	if request.NIter == 0 {
		q.logger.Warn("NIter is zero", "requestID", request.ID)
		request.NIter = max(request.NIter, 1)
	}

	totalImages := request.NIter * request.BatchSize
	return totalImages
}

func (q *SDQueue) retrieveImagesFromResponse(response *entities.TextToImageResponse, item *SDQueueItem) (images, thumbnails []io.Reader) {
	images = make([]io.Reader, len(response.Images))

	for idx, image := range response.Images {
		decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
		if decodeErr != nil {
			q.logger.Error("Failed to decode image", "error", decodeErr)
		}

		images[idx] = bytes.NewBuffer(decodedImage)
	}

	if image := item.ControlnetItem.Image; image != nil {
		thumbnails = append(thumbnails, image)
	}

	if image := item.Img2ImgItem.Image; image != nil {
		thumbnails = append(thumbnails, image)
	}

	generation := item.ImageGenerationRequest
	totalImages := q.totalImageCount(generation)
	if len(images) > totalImages {
		q.logger.Info("Received extra images", "count", len(images), "controlnet", item.ControlnetItem.Enabled)
		thumbnails = append(thumbnails, images[totalImages:]...)
	}

	return images, thumbnails
}

func (q *SDQueue) textInference(queue *SDQueueItem) (response *entities.TextToImageResponse, err error) {
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

func (q *SDQueue) recordToRepository(request *entities.ImageGenerationRequest) (*entities.ImageGenerationRequest, error) {
	var ok bool
	if request.Prompt, ok = strings.CutSuffix(request.Prompt, "{DEBUG}"); ok {
		byteArr, _ := request.TextToImageRequest.Marshal()
		q.logger.Debug("{DEBUG} TextToImageRequest", "request", string(byteArr))
	}

	// return newGeneration from image_generations.Create as we need newGeneration.CreatedAt later on
	request, err := q.imageGenerationRepo.Create(context.Background(), request)
	if err != nil {
		q.logger.Error("Failed to create image generation record", "error", err)
		return nil, err
	}
	return request, nil
}

func (q *SDQueue) updateProgressBar(item *SDQueueItem, generationDone chan bool, webhook *discordgo.WebhookEdit) {
	request := item.ImageGenerationRequest
	timeout := time.NewTimer(5 * time.Minute)
	for {
		select {
		case <-generationDone:
			return
		case _, ok := <-item.Interrupt:
			if !ok {
				return
			}
			err := q.stableDiffusionAPI.Interrupt()
			if err != nil {
				_ = handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
				return
			}
			message, err := handlers.EditInteractionResponse(q.botSession, item.DiscordInteraction, "Generation Interrupted", webhook, handlers.Components[handlers.DeleteGeneration])
			if err != nil {
				return
			}
			if item.DiscordInteraction.Message == nil && message != nil {
				q.logger.Debug("Setting discord interaction message", "message", message)
				item.DiscordInteraction.Message = message
			}
			return
		case <-time.After(1 * time.Second):
			progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
			if progressErr != nil {
				q.logger.Error("Failed to get generation progress", "error", progressErr)
				_ = handlers.ErrorEdit(q.botSession, item.DiscordInteraction, fmt.Sprintf("Error getting current progress: %v", progressErr))
				return
			}

			if progress.Progress == 0 {
				continue
			}

			var ram, cuda *entities.ReadableMemory
			mem, err := q.stableDiffusionAPI.GetMemory()
			if err != nil {
				q.logger.Error("Failed to get memory information", "error", err)
			} else {
				ram = mem.RAM.Readable()
				cuda = mem.Cuda.Readable()
			}

			mem, err = stable_diffusion_api.GetMemory()
			if err != nil {
				q.logger.Error("Failed to get memory information", "error", err)
			} else {
				ram = mem.RAM.Readable()
			}

			progressContent := imagineMessageSimple(request, utils.GetUser(item.DiscordInteraction), progress.Progress, ram, cuda)

			// TODO: Use handlers.Responses[handlers.EditInteractionResponse] instead and adjust to return errors
			_, progressErr = q.botSession.InteractionResponseEdit(item.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &progressContent,
			})
			if progressErr != nil {
				q.logger.Error("Failed to edit interaction", "error", progressErr)
				return
			}
		case <-timeout.C:
			q.logger.Warn("Generation timeout reached")
			_ = handlers.ErrorEdit(q.botSession, item.DiscordInteraction, "Timeout reached")
			return
		}
	}
}

func (q *SDQueue) switchToModels(queue *SDQueueItem) (config, originalConfig *entities.Config, err error) {
	config, err = q.stableDiffusionAPI.GetConfig()
	originalConfig = config
	if err != nil {
		return nil, nil, fmt.Errorf("error getting config: %w", err)
	}

	config, err = q.updateModels(queue, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error updating models: %w", err)
	}

	return config, originalConfig, nil
}

func (q *SDQueue) revertModels(config *entities.Config, originalConfig *entities.Config) error {
	if !ptrStringCompare(config.SDModelCheckpoint, originalConfig.SDModelCheckpoint) ||
		!ptrStringCompare(config.SDVae, originalConfig.SDVae) ||
		!ptrStringCompare(config.SDHypernetwork, originalConfig.SDHypernetwork) {
		q.logger.Info("Switching back to original models",
			"checkpoint", safeDereference(originalConfig.SDModelCheckpoint),
			"vae", safeDereference(originalConfig.SDVae),
			"hypernetwork", safeDereference(originalConfig.SDHypernetwork))
		return q.stableDiffusionAPI.UpdateConfiguration(entities.Config{
			SDModelCheckpoint: originalConfig.SDModelCheckpoint,
			SDVae:             originalConfig.SDVae,
			SDHypernetwork:    originalConfig.SDHypernetwork,
		})
	}
	return nil
}

func (q *SDQueue) updateModels(c *SDQueueItem, config *entities.Config) (*entities.Config, error) {
	request := c.ImageGenerationRequest
	if !ptrStringCompare(request.Checkpoint, config.SDModelCheckpoint) ||
		!ptrStringCompare(request.VAE, config.SDVae) ||
		!ptrStringCompare(request.Hypernetwork, config.SDHypernetwork) {
		_, err := handlers.EditInteractionResponse(q.botSession, c.DiscordInteraction,
			fmt.Sprintf("Changing models to: \n**Checkpoint**: `%v` -> `%v`\n**VAE**: `%v` -> `%v`\n**Hypernetwork**: `%v` -> `%v`",
				safeDereference(config.SDModelCheckpoint), safeDereference(request.Checkpoint),
				safeDereference(config.SDVae), safeDereference(request.VAE),
				safeDereference(config.SDHypernetwork), safeDereference(request.Hypernetwork),
			),
			handlers.Components[handlers.CancelDisabled])
		if err != nil {
			return nil, err
		}

		// Insert code to update the configuration here
		err = q.stableDiffusionAPI.UpdateConfiguration(
			q.lookupModel(request, config,
				[]stable_diffusion_api.Cacheable{
					stable_diffusion_api.CheckpointCache,
					stable_diffusion_api.VAECache,
					stable_diffusion_api.HypernetworkCache,
				}))
		if err != nil {
			return nil, fmt.Errorf("error updating configuration: %w", err)
		}
		config, err = q.stableDiffusionAPI.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("error getting config: %w", err)
		}
		request.Checkpoint = config.SDModelCheckpoint
		request.VAE = config.SDVae
		request.Hypernetwork = config.SDHypernetwork
	}
	return config, nil
}
