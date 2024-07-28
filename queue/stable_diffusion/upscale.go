package stable_diffusion

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
	"strings"
	"time"
)

func (q *SDQueue) processUpscaleImagine() error {
	defer q.done()
	queue := q.currentImagine
	var err error
	queue.ImageGenerationRequest, err = q.getPreviousGeneration(queue)
	if err != nil {
		return handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Errorf("error getting prompt for upscale: %w", err))

	}

	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	if textToImage == nil {
		return handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Errorf("textToImageRequest of type %v is nil", queue.Type))
	}

	config, originalConfig, err := q.switchToModels(queue)
	if err != nil {
		return handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Errorf("error switching to models: %w", err))
	}

	newContent := upscaleMessageContent(queue.DiscordInteraction.Member.User, 0, 0)
	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, queue, queue.Interrupt != nil)

	_, err = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		return handlers.Wrap(err)
	}

	generationDone := make(chan bool)

	go q.updateUpscaleProgress(queue, generationDone, config, originalConfig)

	resp, err := q.upscale(request)
	generationDone <- true
	if err != nil {
		log.Printf("Error processing image upscale: %v\n", err)
		return handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, "I'm sorry, but I had a problem upscaling your image.", err)
	}

	log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d", queue.DiscordInteraction.ID, queue.DiscordInteraction.Message.ID, queue.InteractionIndex)

	if err := q.finalUpscaleMessage(queue, resp, embed); err != nil {
		return handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Errorf("error finalizing upscale message: %w", err))
	}

	return nil
}

func (q *SDQueue) upscale(request *entities.ImageGenerationRequest) (*stable_diffusion_api.UpscaleResponse, error) {
	textToImage := request.TextToImageRequest
	// Use face segm model if we're upscaling but there's no ADetailer models
	if textToImage.Scripts.ADetailer == nil {
		textToImage.Scripts.NewADetailerWithArgs()
		textToImage.Scripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", request)
	}

	textToImage.BatchSize = 1
	textToImage.NIter = 1

	return q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:         0,
		UpscalingResize:    2,
		Upscaler1:          "R-ESRGAN 2x+",
		TextToImageRequest: textToImage,
	})
}

func (q *SDQueue) finalUpscaleMessage(queue *SDQueueItem, resp *stable_diffusion_api.UpscaleResponse, embed *discordgo.MessageEmbed) error {
	textToImage := queue.ImageGenerationRequest.TextToImageRequest

	decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
	if decodeErr != nil {
		return fmt.Errorf("error decoding image: %w", decodeErr)
	}
	if len(decodedImage) == 0 {
		return fmt.Errorf("decoded image is empty")
	}

	var scriptsString string
	var scripts []string

	if queue.Type != ItemTypeRaw {
		if textToImage.Scripts.ADetailer != nil {
			scripts = append(scripts, "ADetailer")
		}
		if textToImage.Scripts.ControlNet != nil {
			scripts = append(scripts, "ControlNet")
		}
		if textToImage.Scripts.CFGRescale != nil {
			scripts = append(scripts, "CFGRescale")
		}
	} else {
		for script := range queue.Raw.RawScripts {
			scripts = append(scripts, script)
		}
	}

	if len(scripts) > 0 {
		scriptsString = fmt.Sprintf("\n**Scripts**: [`%v`]", strings.Join(scripts, ", "))
	}

	finishedContent := fmt.Sprintf("<@%s> asked me to upscale their image. (seed: %d) Here's the result:%v",
		queue.DiscordInteraction.Member.User.ID,
		textToImage.Seed,
		scriptsString,
	)

	if len(finishedContent) > 2000 {
		finishedContent = finishedContent[:2000]
	}

	webhook := &discordgo.WebhookEdit{
		Content: &finishedContent,
		Components: &[]discordgo.MessageComponent{
			handlers.Components[handlers.DeleteGeneration],
		},
	}

	if err := utils.EmbedImages(webhook, embed, []io.Reader{bytes.NewBuffer(decodedImage)}, nil, q.compositor); err != nil {
		log.Printf("Error creating image embed: %v\n", err)
		return err
	}

	_, err := handlers.EditInteractionResponse(q.botSession, queue.DiscordInteraction, webhook)
	return err
}

func (q *SDQueue) updateUpscaleProgress(queue *SDQueueItem, generationDone chan bool, config, originalConfig *entities.Config) {
	lastProgress := float64(0)
	fetchProgress := float64(0)
	upscaleProgress := float64(0)
	for {
		select {
		case queue.DiscordInteraction = <-queue.Interrupt:
			err := q.stableDiffusionAPI.Interrupt()
			if err != nil {
				_ = handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
				return
			}
			message, err := handlers.EditInteractionResponse(q.botSession, queue.DiscordInteraction, "Generation Interrupted", handlers.Components[handlers.DeleteGeneration])
			if err != nil {
				return
			}
			if queue.DiscordInteraction.Message == nil && message != nil {
				log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
				queue.DiscordInteraction.Message = message
			}
		case <-generationDone:
			err := q.revertModels(config, originalConfig)
			if err != nil {
				_ = handlers.ErrorEdit(q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
				return
			}
			return
		case <-time.After(1 * time.Second):
			progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
			if progressErr != nil {
				log.Printf("Error getting current progress: %v", progressErr)
				return
			}

			if progress.Progress == 0 {
				continue
			}

			if progress.Progress < lastProgress || upscaleProgress > 0 {
				upscaleProgress = progress.Progress
				fetchProgress = 1
			} else {
				fetchProgress = progress.Progress
			}

			lastProgress = progress.Progress

			progressContent := upscaleMessageContent(queue.DiscordInteraction.Member.User, fetchProgress, upscaleProgress)

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
