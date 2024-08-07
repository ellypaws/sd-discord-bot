package stable_diffusion

import (
	"errors"
	"fmt"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"

	"github.com/bwmarrin/discordgo"
)

// TODO: Implement separate processing for Img2Img, possibly use github.com/SpenserCai/sd-webui-go/intersvc
// Deprecated: still using processCurrentImagine
func (q *SDQueue) processImg2ImgImagine() error {
	//defer q.done()
	return q.processCurrentImagine()
}

func (q *SDQueue) imageToImage(generationDone chan bool, embed *discordgo.MessageEmbed, webhook *discordgo.WebhookEdit) error {
	queue := q.currentImagine
	img2img := t2iToImg2Img(queue.TextToImageRequest)

	err := calculateImg2ImgDimensions(queue, &img2img)
	if err != nil {
		return err
	}

	resp, err := q.stableDiffusionAPI.ImageToImageRequest(&img2img)
	generationDone <- true
	if err != nil {
		return err
	}

	err = q.showFinalMessage(queue, &entities.TextToImageResponse{Images: resp.Images}, embed, webhook)
	if err != nil {
		return err
	}
	return nil
}

func calculateImg2ImgDimensions(queue *SDQueueItem, img2img *entities.ImageToImageRequest) error {
	if queue.Img2ImgItem.Image == nil {
		return errors.New("no attached images found, skipping img2img generation")
	}

	width, height, err := utils.GetBase64ImageSize(queue.Img2ImgItem.Image.String())
	if err != nil {
		return err
	}

	//calculate aspect ratio. e.g. 512x768 = 2:3 to the nearest whole number
	gcd := calculateGCD(width, height)
	aspectRatio := fmt.Sprintf("%d:%d", width/gcd, height/gcd)

	*img2img.Width, *img2img.Height = aspectRatioCalculation(aspectRatio, initializedWidth, initializedHeight)

	img2img.InitImages = append(img2img.InitImages, queue.Img2ImgItem.Image.String())
	return err
}

func calculateGCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func t2iToImg2Img(textToImage *entities.TextToImageRequest) entities.ImageToImageRequest {
	img2img := entities.ImageToImageRequest{
		Scripts:                           textToImage.Scripts,
		BatchSize:                         textToImage.BatchSize,
		CFGScale:                          &textToImage.CFGScale,
		DenoisingStrength:                 &textToImage.DenoisingStrength,
		Height:                            &textToImage.Height,
		ImageCFGScale:                     &textToImage.CFGScale,
		IncludeInitImages:                 nil,
		InitImages:                        nil,
		NIter:                             textToImage.NIter,
		NegativePrompt:                    &textToImage.NegativePrompt,
		OverrideSettings:                  textToImage.OverrideSettings,
		OverrideSettingsRestoreAfterwards: textToImage.OverrideSettingsRestoreAfterwards,
		Prompt:                            textToImage.Prompt,
		RefinerCheckpoint:                 textToImage.RefinerCheckpoint,
		RefinerSwitchAt:                   textToImage.RefinerSwitchAt,
		RestoreFaces:                      &textToImage.RestoreFaces,
		SChurn:                            textToImage.SChurn,
		SMinUncond:                        textToImage.SMinUncond,
		SNoise:                            textToImage.SNoise,
		STmax:                             textToImage.STmax,
		STmin:                             textToImage.STmin,
		SamplerIndex:                      textToImage.SamplerIndex,
		SamplerName:                       &textToImage.SamplerName,
		SaveImages:                        textToImage.SaveImages,
		ScriptArgs:                        textToImage.ScriptArgs,
		ScriptName:                        textToImage.ScriptName,
		Seed:                              &textToImage.Seed,
		SeedResizeFromH:                   textToImage.SeedResizeFromH,
		SeedResizeFromW:                   textToImage.SeedResizeFromW,
		SendImages:                        textToImage.SendImages,
		Steps:                             &textToImage.Steps,
		Styles:                            textToImage.Styles,
		Subseed:                           &textToImage.Subseed,
		SubseedStrength:                   &textToImage.SubseedStrength,
		Tiling:                            textToImage.Tiling,
		Width:                             &textToImage.Width,
	}
	return img2img
}
