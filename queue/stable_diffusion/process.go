package stable_diffusion

import (
	"encoding/json"
	"fmt"
	"log"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
)

func (q *SDQueue) processCurrentImagine() error {
	queue := q.currentImagine

	request, err := queue.ImageGenerationRequest, error(nil)
	if request == nil {
		return fmt.Errorf("ImageGenerationRequest of type %v is nil", queue.Type)
	}

	textToImage := request.TextToImageRequest
	if textToImage == nil {
		return fmt.Errorf("TextToImageRequest of type %v is nil", queue.Type)
	}

	// only set width and height if it is not a raw json request
	if queue.Type != ItemTypeRaw || (queue.Type == ItemTypeRaw && queue.Raw != nil && queue.Raw.Unsafe) {
		err = calculateDimensions(q, queue)
		if err != nil {
			return fmt.Errorf("error calculating dimensions: %w", err)
		}
	}

	fillBlankModels(q, request)

	initializeScripts(queue)

	err = q.processImagineGrid(queue)
	if err != nil {
		return fmt.Errorf("error processing imagine grid: %w", err)
	}

	return nil
}

func calculateDimensions(q *SDQueue, queue *SDQueueItem) (err error) {
	textToImage := queue.TextToImageRequest
	if textToImage.Width == 0 {
		textToImage.Width, err = q.defaultWidth()
		if err != nil {
			return fmt.Errorf("error getting default width: %w", err)
		}
	}

	if textToImage.Height == 0 {
		textToImage.Height, err = q.defaultHeight()
		if err != nil {
			return fmt.Errorf("error getting default height: %w", err)
		}
	}

	if queue.AspectRatio != "" && queue.AspectRatio != "1:1" {
		textToImage.Width, textToImage.Height = aspectRatioCalculation(queue.AspectRatio, textToImage.Width, textToImage.Height)
	}

	if textToImage.EnableHr && textToImage.HrScale > 1.0 {
		textToImage.HrResizeX = int(float64(textToImage.Width) * textToImage.HrScale)
		textToImage.HrResizeY = int(float64(textToImage.Height) * textToImage.HrScale)
	} else {
		textToImage.EnableHr = false
		textToImage.HrResizeX = textToImage.Width
		textToImage.HrResizeY = textToImage.Height
	}
	return
}

// fillBlankModels fills in the blank models with the current models from the config
func fillBlankModels(q *SDQueue, request *entities.ImageGenerationRequest) {
	config, err := q.stableDiffusionAPI.GetConfig()
	if err != nil {
		log.Printf("Error getting config: %v", err)
	} else {
		if !ptrStringNotBlank(request.Checkpoint) {
			request.Checkpoint = config.SDModelCheckpoint
		}
		if !ptrStringNotBlank(request.VAE) {
			request.VAE = config.SDVae
		}
		if !ptrStringNotBlank(request.Hypernetwork) {
			request.Hypernetwork = config.SDHypernetwork
		}
	}
}

// initializeScripts sets up ADetailer and Controlnet scripts
func initializeScripts(queue *SDQueueItem) {
	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	if queue.ADetailerString != "" {
		log.Printf("q.currentImagine.ADetailerString: %v", queue.ADetailerString)

		request.NewADetailer()

		textToImage.Scripts.ADetailer.AppendSegModelByString(queue.ADetailerString, request)
	}

	if queue.ControlnetItem.Enabled {
		initializeControlnet(queue)
	}

	if request.Scripts.ADetailer != nil {
		jsonMarshalScripts, err := json.MarshalIndent(&request.Scripts.ADetailer, "", "  ")
		if err != nil {
			log.Printf("Error marshalling scripts: %v", err)
		} else {
			log.Println("Final scripts (Adetailer): ", string(jsonMarshalScripts))
		}
	}
}

func initializeControlnet(queue *SDQueueItem) {
	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest

	var controlnetImage string
	switch {
	case queue.ControlnetItem.Image != nil:
		controlnetImage = queue.ControlnetItem.Image.String()
	case queue.Img2ImgItem.Image != nil:
		// not needed for Img2Img as it automatically uses it if InputImage is null, only used for width/height
		controlnetImage = queue.Img2ImgItem.Image.String()
	default:
		queue.ControlnetItem.Enabled = false
	}
	width, height, err := utils.GetBase64ImageSize(controlnetImage)
	var controlnetResolution int
	if err != nil {
		log.Printf("Error getting image size: %v", err)
	} else {
		controlnetResolution = between(max(width, height), min(request.Width, request.Height), 1024)
	}

	textToImage.Scripts.ControlNet = &entities.ControlNet{
		Args: []*entities.ControlNetParameters{
			{
				InputImage:   &controlnetImage,
				Module:       queue.ControlnetItem.Preprocessor,
				Model:        queue.ControlnetItem.Model,
				Weight:       1.0,
				ResizeMode:   queue.ControlnetItem.ResizeMode,
				ProcessorRes: controlnetResolution,
				ControlMode:  queue.ControlnetItem.ControlMode,
				PixelPerfect: false,
			},
		},
	}
	if queue.Type == ItemTypeImg2Img && queue.ControlnetItem.Image == nil {
		// controlnet will automatically use img2img if it is null
		request.Scripts.ControlNet.Args[0].InputImage = nil
	}

	if !queue.Enabled {
		request.Scripts.ControlNet = nil
	}

	log.Printf("q.currentImagine.ControlnetItem.Enabled: %v", queue.ControlnetItem.Enabled)
}
