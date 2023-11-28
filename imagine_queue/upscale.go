package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/stable_diffusion_api"
	"time"
)

func (q *queueImplementation) processUpscaleImagine(imagine *QueueItem) {
	go func() {
		defer func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			q.currentImagine = nil
		}()
		interactionID := imagine.DiscordInteraction.ID
		messageID := ""

		if imagine.DiscordInteraction.Message != nil {
			messageID = imagine.DiscordInteraction.Message.ID
		}

		log.Printf("Upscaling image: %v, Message: %v, Upscale Index: %d",
			interactionID, messageID, imagine.InteractionIndex)

		generation, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, imagine.InteractionIndex)
		if err != nil {
			log.Printf("Error getting image generation: %v", err)

			return
		}

		log.Printf("Found generation: %v", generation)

		config, err := q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
			handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, fmt.Sprintf("Error getting config: %v", err))
			return
		}

		log.Printf("Current checkpoint: %v", safeDereference(config.SDModelCheckpoint))
		log.Printf("Generation checkpoint: %v", safeDereference(generation.Checkpoint))

		if generation.Checkpoint != nil && !ptrStringCompare(config.SDModelCheckpoint, generation.Checkpoint) {
			log.Printf("Changing checkpoint to: %v", *generation.Checkpoint)

			updateModelMessage := fmt.Sprintf("Changing checkpoint to %v -> %v", safeDereference(config.SDModelCheckpoint), safeDereference(generation.Checkpoint))

			_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &updateModelMessage,
			})
			if err != nil {
				log.Printf("Error editing interaction: %v", err)
			}

			err = q.stableDiffusionAPI.UpdateConfiguration(q.switchModel(generation, config, []stable_diffusion_api.Cacheable{
				stable_diffusion_api.CheckpointCache,
				stable_diffusion_api.VAECache,
				stable_diffusion_api.HypernetworkCache,
			}))
			if err != nil {
				log.Printf("Error updating models: %v", err)
				handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, fmt.Sprintf("Error updating models: %v", err))

				return
			}
		}

		newContent := upscaleMessageContent(imagine.DiscordInteraction.Member.User, 0, 0)

		_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &newContent,
		})
		if err != nil {
			log.Printf("Error editing interaction: %v", err)
		}

		generationDone := make(chan bool)

		go func() {
			lastProgress := float64(0)
			fetchProgress := float64(0)
			upscaleProgress := float64(0)
			elapsedTime := 0

			for {
				select {
				case <-generationDone:
					return
				case <-time.After(1 * time.Second):
					progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
					if progressErr != nil {
						log.Printf("Error getting current progress: %v", progressErr)
						return
					}
					elapsedTime += 1

					if elapsedTime > 60 {
						msg := "Upscale timed out after 60 seconds"
						log.Printf(msg)

						_, _ = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
							Content: &msg,
						})

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

					progressContent := upscaleMessageContent(imagine.DiscordInteraction.Member.User, fetchProgress, upscaleProgress)

					_, progressErr = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
						Content: &progressContent,
					})
					if progressErr != nil {
						log.Printf("Error editing interaction: %v", err)
					}
				}
			}
		}()

		//// Check if ADetailer is in the scripts and add it to the object generation with method by using AppendToArgs
		//_, exist := generation.AlwaysOnScripts["ADetailer"]
		//if !exist {
		//	model := entities.ADetailerParameters{AdModel: "face_yolov8n.pt"}
		//	generation.AlwaysOnScripts["ADetailer"] = &entities.ADetailer{}
		//	generation.AlwaysOnScripts["ADetailer"].AppendSegModel(model)
		//}

		// Use face segm model if we're upscaling but there's no ADetailer models
		if generation.AlwaysonScripts == nil {
			generation.NewScripts()
		}
		if generation.AlwaysonScripts.ADetailer == nil {
			generation.AlwaysonScripts.NewADetailerWithArgs()
			generation.AlwaysonScripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", generation)
		}

		t2iRequest := generation.TextToImageRequest
		t2iRequest.BatchSize = 1
		t2iRequest.NIter = 1

		resp, err := q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
			ResizeMode:         0,
			UpscalingResize:    2,
			Upscaler1:          "R-ESRGAN 2x+",
			TextToImageRequest: t2iRequest,
		})
		if err != nil {
			log.Printf("Error processing image upscale: %v\n", err)

			errorContent := fmt.Sprint("I'm sorry, but I had a problem upscaling your image. ", err)

			//_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			//	Content: &errorContent,
			//})

			handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, errorContent)
			//handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, errorContent)

			generationDone <- true
			return
		}

		generationDone <- true

		decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
		if decodeErr != nil {
			log.Printf("Error decoding image: %v\n", decodeErr)

			return
		}

		imageBuf := bytes.NewBuffer(decodedImage)

		// save imageBuf to disk
		//err = ioutil.WriteFile("upscaled.png", imageBuf.Bytes(), 0644)

		log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d",
			interactionID, messageID, imagine.InteractionIndex)

		var scriptsString string

		if generation.AlwaysonScripts != nil {
			scripts, err := json.Marshal(generation.AlwaysonScripts)
			if err != nil {
				log.Printf("Error marshalling scripts: %v", err)
			} else {
				scriptsString = string(scripts)
			}
		}

		finishedContent := fmt.Sprintf("<@%s> asked me to upscale their image. (seed: %d) Here's the result:\n\n Scripts: ```json\n%v\n```",
			imagine.DiscordInteraction.Member.User.ID,
			generation.Seed,
			scriptsString,
		)

		_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &finishedContent,
			Files: []*discordgo.File{
				{
					ContentType: "image/png",
					// add timestamp to output file
					Name:   "imagine_" + time.Now().Format("20060102150405") + ".png",
					Reader: imageBuf,
				},
			},
		})
		if err != nil {
			log.Printf("Error editing interaction: %v\n", err)

			return
		}
	}()
}
