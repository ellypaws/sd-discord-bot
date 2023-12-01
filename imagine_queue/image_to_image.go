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
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
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

	c := q.currentImagine

	if len(c.Attachments) == 0 {
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

	width, height, err := utils.GetImageSizeFromBase64(safeDereference(c.Img2ImgItem.Image))
	if err != nil {
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, err)
		return err, true
	}

	//calculate aspect ratio. e.g. 512x768 = 2:3 to the nearest whole number
	gcd := calculateGCD(width, height)
	aspectRatio := fmt.Sprintf("%d:%d", width/gcd, height/gcd)

	*img2img.Width, *img2img.Height = aspectRatioCalculation(aspectRatio, initializedWidth, initializedHeight)

	img2img.InitImages = append(img2img.InitImages, safeDereference(c.Img2ImgItem.Image))

	marshal, err := img2img.Marshal()
	if err != nil {
		return err, true
	}

	// save to file
	err = os.WriteFile("img2img.json", marshal, 0644)

	resp, err := q.stableDiffusionAPI.ImageToImageRequest(&img2img)

	// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt
	var embed *discordgo.MessageEmbed
	embed = generationEmbedDetails(embed, newGeneration, c)

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

	primaryImage := imageBufs[0]
	var thumbnailBuffers []*bytes.Buffer

	log.Printf("c.ControlnetItem.Enabled: %v, c.ControlnetItem.MessageAttachment != nil: %v", c.ControlnetItem.Enabled, c.ControlnetItem.MessageAttachment != nil)

	if c.ControlnetItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(c.ControlnetItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnailReader := bytes.NewBuffer(decodedBytes)
		thumbnailBuffers = append(thumbnailBuffers, thumbnailReader)
	}

	if c.Img2ImgItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(c.Img2ImgItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnailReader := bytes.NewBuffer(decodedBytes)
		thumbnailBuffers = append(thumbnailBuffers, thumbnailReader)
	}

	if len(imageBufs) > 1 {
		thumbnailBuffers = append(thumbnailBuffers, imageBufs[1:]...)
	}

	empty := ""

	webhook := &discordgo.WebhookEdit{
		Content:    &empty,
		Components: rerollVariationComponents(min(len(imageBufs), 1), c.Type == ItemTypeImg2Img),
	}

	thumbnailTile, err := q.compositeRenderer.TileImages(thumbnailBuffers)
	if err != nil {
		log.Printf("Error tiling thumbnails: %v\n", err)
		//byteArray, _ := json.Marshal(thumbnailBuffers)
		//log.Printf("thumbnailBuffers: %v", string(byteArray))
	}

	var primaryImageReader *bytes.Reader
	if primaryImage != nil {
		primaryImageReader = bytes.NewReader(primaryImage.Bytes())
	}

	var thumbnailTileReader *bytes.Reader
	if thumbnailTile != nil {
		thumbnailTileReader = bytes.NewReader(thumbnailTile.Bytes())
	}

	if len(thumbnailBuffers) > 0 {
		imageEmbedFromReader(webhook, embed, primaryImageReader, thumbnailTileReader)
	} else {
		// because we don't have the original webhook that contains the image file
		err := imageAttachmentAsThumbnail(webhook, embed, primaryImageReader, c.Img2ImgItem.MessageAttachment, true)
		if err != nil {
			log.Printf("Error attaching image as thumbnail: %v", err)
			return err, true
		}
	}

	_, err = q.botSession.InteractionResponseEdit(c.DiscordInteraction, webhook)
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
		return err, true
	}
	return nil, false
}
