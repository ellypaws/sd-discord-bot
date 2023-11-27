package imagine_queue

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"time"
)

// TODO: Implement separate processing for Img2Img, possibly use github.com/SpenserCai/sd-webui-go/intersvc
// Deprecated: still using processCurrentImagine
func (q *queueImplementation) processImg2ImgImagine() {
	q.processCurrentImagine()
}

func (q *queueImplementation) imageToImage(newGeneration *entities.ImageGenerationRequest, imagine *QueueItem, generationDone chan bool) (error, bool) {
	newGeneration.NIter = 1
	newGeneration.BatchSize = 1
	img2img := entities.ImageToImageRequest{
		AlwaysonScripts:                   newGeneration.AlwaysonScripts,
		BatchSize:                         &newGeneration.BatchSize,
		CFGScale:                          &newGeneration.CFGScale,
		DenoisingStrength:                 &newGeneration.DenoisingStrength,
		Height:                            &newGeneration.Height,
		ImageCFGScale:                     &newGeneration.CFGScale,
		IncludeInitImages:                 nil,
		InitImages:                        nil,
		NIter:                             &newGeneration.NIter,
		NegativePrompt:                    &newGeneration.NegativePrompt,
		OverrideSettings:                  newGeneration.OverrideSettings,
		OverrideSettingsRestoreAfterwards: newGeneration.OverrideSettingsRestoreAfterwards,
		Prompt:                            newGeneration.Prompt,
		RefinerCheckpoint:                 newGeneration.RefinerCheckpoint,
		RefinerSwitchAt:                   newGeneration.RefinerSwitchAt,
		RestoreFaces:                      &newGeneration.RestoreFaces,
		SChurn:                            newGeneration.SChurn,
		SMinUncond:                        newGeneration.SMinUncond,
		SNoise:                            newGeneration.SNoise,
		STmax:                             newGeneration.STmax,
		STmin:                             newGeneration.STmin,
		SamplerIndex:                      newGeneration.SamplerIndex,
		SamplerName:                       &newGeneration.SamplerName,
		SaveImages:                        newGeneration.SaveImages,
		ScriptArgs:                        newGeneration.ScriptArgs,
		ScriptName:                        newGeneration.ScriptName,
		Seed:                              &newGeneration.Seed,
		SeedResizeFromH:                   newGeneration.SeedResizeFromH,
		SeedResizeFromW:                   newGeneration.SeedResizeFromW,
		SendImages:                        newGeneration.SendImages,
		Steps:                             &newGeneration.Steps,
		Styles:                            newGeneration.Styles,
		Subseed:                           &newGeneration.Subseed,
		SubseedStrength:                   &newGeneration.SubseedStrength,
		Tiling:                            newGeneration.Tiling,
		Width:                             &newGeneration.Width,
	}

	if len(q.currentImagine.Attachments) == 0 {
		err := errors.New("No attached images found, skipping img2img generation")
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
		return err, true
	}

	calculateGCD := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}

	for snowflake := range q.currentImagine.Attachments {
		width, height, err := utils.GetImageSizeFromBase64(q.currentImagine.Images[snowflake])
		if err != nil {
			handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
			return err, true
		}

		//calculate aspect ratio. e.g. 512x768 = 2:3 to the nearest whole number
		gcd := calculateGCD(width, height)
		aspectRatio := fmt.Sprintf("%d:%d", width/gcd, height/gcd)

		*img2img.Width, *img2img.Height = aspectRatioCalculation(aspectRatio, initializedWidth, initializedHeight)

		img2img.InitImages = append(img2img.InitImages, q.currentImagine.Images[snowflake])
	}

	marshal, err := img2img.Marshal()
	if err != nil {
		return err, true
	}

	// save to file
	err = os.WriteFile("img2img.json", marshal, 0644)

	resp, err := q.stableDiffusionAPI.ImageToImageRequest(&img2img)

	if err != nil {
		log.Printf("Error processing image: %v\n", err)

		errorContent := fmt.Sprint("I'm sorry, but I had a problem imagining your image. ", err)

		handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, errorContent)

		return err, true
	}

	generationDone <- true

	//type ImageToImageResponse struct {
	//	Images     []string          `json:"images,omitempty"`
	//	Info       string            `json:"info"`
	//	Parameters map[string]string `json:"parameters"`
	//}

	if len(resp.Images) == 0 {
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, errors.New("no images returned"))
		return errors.New("no images returned"), true
	}

	imageBufs := make([]*bytes.Buffer, len(resp.Images))

	for idx, image := range resp.Images {
		image, err := base64.StdEncoding.DecodeString(image)
		if err != nil {
			handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
			return err, true
		}

		imageBuf := bytes.NewBuffer(image)

		imageBufs[idx] = imageBuf
	}

	q.compositeRenderer, err = composite_renderer.New(false)
	if err != nil {
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
		return err, true
	}

	compositeImage, err := q.compositeRenderer.TileImages(imageBufs)
	if err != nil {
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
		return err, true
	}

	finishedContent := imagineMessageContent(newGeneration, imagine.DiscordInteraction.Member.User, 1)

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				// append timestamp for grid image result
				Name:   "imagine_" + time.Now().Format("20060102150405") + ".png",
				Reader: compositeImage,
			},
		},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						// Label is what the user will see on the button.
						Label: "1",
						// Style provides coloring of the button. There are not so many styles tho.
						Style: discordgo.SecondaryButton,
						// Disabled allows bot to disable some buttons for users.
						Disabled: true,
						// CustomID is a thing telling Discord which data to send when this button will be pressed.
						CustomID: "imagine_variation_1",
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
						Disabled: true,
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
					handlers.Components[handlers.DeleteGeneration].(discordgo.ActionsRow).Components[0],
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)

		return err, true
	}
	return nil, false
}
