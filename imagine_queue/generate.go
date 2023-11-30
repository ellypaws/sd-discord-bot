package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/stable_diffusion_api"
	"strings"
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
				))

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

	var files []*discordgo.File
	var embeds []*discordgo.MessageEmbed
	for snowflake, attachment := range c.Attachments {
		imageReader, err := utils.GetImageReaderByBase64(safeDereference(c.Attachments[snowflake].Image))
		if err != nil {
			log.Printf("Error getting image reader: %v", err)
			continue
		}
		var title string
		if c.ControlnetItem.MessageAttachment != nil && snowflake == c.ControlnetItem.MessageAttachment.ID {
			title = fmt.Sprintf("Controlnet")
		}
		if c.Img2ImgItem.MessageAttachment != nil && snowflake == c.Img2ImgItem.MessageAttachment.ID {
			title = fmt.Sprintf("Img2Img")
		}
		if strings.Contains(attachment.ContentType, "image") {
			files = append(files, &discordgo.File{
				ContentType: attachment.ContentType,
				Name:        attachment.Filename,
				Reader:      imageReader,
			})
			embeds = append(embeds, &discordgo.MessageEmbed{
				Type:  discordgo.EmbedTypeImage,
				Title: title,
				Image: &discordgo.MessageEmbedImage{
					URL: fmt.Sprintf("attachment://%v", attachment.Filename),
				},
			})
		} else {
			log.Printf("Attachment is not an image: %#v", attachment)
		}
	}

	message, err := q.botSession.InteractionResponseEdit(c.DiscordInteraction, &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Files:      files,
		Embeds:     &embeds,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
		return err
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

	_, err = q.imageGenerationRepo.Create(context.Background(), newGeneration)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
		return err
	}

	generationDone := make(chan bool)

	go func() {
		for {
			select {
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

		generationDone <- true

		log.Printf("Seeds: %v Subseeds:%v", resp.Seeds, resp.Subseeds)

		imageBufs := make([]*bytes.Buffer, len(resp.Images))

		for idx, image := range resp.Images {
			decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
			if decodeErr != nil {
				log.Printf("Error decoding image: %v\n", decodeErr)
			}

			imageBuf := bytes.NewBuffer(decodedImage)

			imageBufs[idx] = imageBuf
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

		compositeImage, err := q.compositeRenderer.TileImages(imageBufs[:min(len(imageBufs), 4)])
		if err != nil {
			log.Printf("Error tiling images: %v\n", err)
			handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
			return err
		}
		files = append(files, &discordgo.File{
			ContentType: "image/png",
			// append timestamp for grid image result
			Name:   "imagine_" + time.Now().Format("20060102150405") + ".png",
			Reader: compositeImage,
		})
		embeds[0].Image.URL = fmt.Sprintf("attachment://%v", files[0].Name)

		if c.Enabled && c.Type != ItemTypeImg2Img {
			extraImage, err := q.compositeRenderer.TileImages(imageBufs[4:])
			if err != nil {
				log.Printf("Error tiling images: %v\n", err)
				handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
				return err
			}
			files = append(files, &discordgo.File{
				ContentType: "image/png",
				Name:        "controlnet.png",
				Reader:      extraImage,
			})
			embeds[0].Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: "attachment://controlnet.png",
			}
		}

		finishedContent := imagineMessageContent(newGeneration, c.DiscordInteraction.Member.User, 1)

		_, err = q.botSession.InteractionResponseEdit(c.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &finishedContent,
			Files:   files,
			Embeds:  &embeds,
			Components: &[]discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "1",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_variation_1",
							Emoji: discordgo.ComponentEmoji{
								Name: "♻️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "2",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_variation_2",
							Emoji: discordgo.ComponentEmoji{
								Name: "♻️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "3",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_variation_3",
							Emoji: discordgo.ComponentEmoji{
								Name: "♻️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "4",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_variation_4",
							Emoji: discordgo.ComponentEmoji{
								Name: "♻️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "Re-roll",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.PrimaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_reroll",
							Emoji: discordgo.ComponentEmoji{
								Name: "🎲",
							},
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "1",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_upscale_1",
							Emoji: discordgo.ComponentEmoji{
								Name: "⬆️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "2",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_upscale_2",
							Emoji: discordgo.ComponentEmoji{
								Name: "⬆️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "3",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_upscale_3",
							Emoji: discordgo.ComponentEmoji{
								Name: "⬆️",
							},
						},
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "4",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.SecondaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_upscale_4",
							Emoji: discordgo.ComponentEmoji{
								Name: "⬆️",
							},
						},
						handlers.Components[handlers.DeleteGeneration].(discordgo.ActionsRow).Components[0],
					},
				},
			},
		})
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
