package stable_diffusion

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	p "stable_diffusion_bot/gui/progress"
	"stable_diffusion_bot/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
)

func (q *SDQueue) next() error {
	if len(q.queue) == 0 {
		return nil
	}
	if q.currentImagine != nil {
		log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
		return errors.New("currentImagine is not nil")
	}
	q.currentImagine = <-q.queue
	defer q.done()

	if q.currentImagine.DiscordInteraction == nil {
		// If the interaction is nil, we can't respond. Make sure to set the implementation before adding to the queue.
		// Example: queue.DiscordInteraction = i.Interaction
		log.Panicf("DiscordInteraction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", q.currentImagine)
	}

	q.mu.Lock()
	if q.cancelledItems[q.currentImagine.DiscordInteraction.ID] {
		delete(q.cancelledItems, q.currentImagine.DiscordInteraction.ID)
		q.mu.Unlock()
		return nil
	}
	q.mu.Unlock()

	var err error
	switch q.currentImagine.Type {
	case ItemTypeImagine, ItemTypeRaw:
		err = q.processCurrentImagine()
	case ItemTypeReroll, ItemTypeVariation:
		err = q.processVariation()
	case ItemTypeImg2Img:
		err = q.processImg2ImgImagine()
	case ItemTypeUpscale:
		err = q.processUpscaleImagine()
	default:
		return handlers.ErrorEdit(q.botSession, q.currentImagine.DiscordInteraction, fmt.Errorf("unknown item type: %v", q.currentImagine.Type))
	}

	if err != nil {
		return handlers.ErrorEdit(q.botSession, q.currentImagine.DiscordInteraction, fmt.Errorf("error processing current item: %w", err))
	}

	return nil
}

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

func (q *SDQueue) done() {
	q.mu.Lock()
	q.currentImagine = nil
	q.mu.Unlock()
}

func between[T cmp.Ordered](value, minimum, maximum T) T {
	return min(max(minimum, value), maximum)
}

func betweenPtr[T cmp.Ordered](value, minimum, maximum T) *T {
	out := min(max(minimum, value), maximum)
	return &out
}

func (q *SDQueue) getPreviousGeneration(queue *SDQueueItem) (*entities.ImageGenerationRequest, error) {
	if queue.DiscordInteraction == nil {
		return nil, errors.New("interaction is nil")
	}

	if queue.DiscordInteraction.Message == nil {
		return nil, errors.New("interaction message is nil")
	}

	interactionID := queue.DiscordInteraction.ID
	sortOrder := queue.InteractionIndex
	messageID := queue.DiscordInteraction.Message.ID

	log.Printf("Reimagining interaction: %v, Message: %v", interactionID, messageID)

	var err error
	queue.ImageGenerationRequest, err = q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, sortOrder)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return nil, err
	}

	log.Printf("Found generation: %v", queue.ImageGenerationRequest)

	return queue.ImageGenerationRequest, nil
}

// Deprecated: use imagineMessageSimple instead
func imagineMessageContent(request *entities.ImageGenerationRequest, user *discordgo.User, progress float64) string {
	var out = strings.Builder{}

	seedString := fmt.Sprintf("%d", request.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine with step: `%d` cfg: `%s` seed: `%s` sampler: `%s`",
		user.ID,
		request.Steps,
		strconv.FormatFloat(request.CFGScale, 'f', 1, 64),
		seedString,
		request.SamplerName,
	))

	out.WriteString(fmt.Sprintf(" `%d x %d`", request.Width, request.Height))

	if request.EnableHr {
		// " -> (x %x) = %d x %d"
		out.WriteString(fmt.Sprintf(" -> (x `%s` by hires.fix) = `%d x %d`",
			strconv.FormatFloat(request.HrScale, 'f', 1, 64),
			request.HrResizeX,
			request.HrResizeY),
		)
	}

	if ptrStringNotBlank(request.Checkpoint) {
		out.WriteString(fmt.Sprintf("\n**Checkpoint**: `%v`", *request.Checkpoint))
	}

	if ptrStringNotBlank(request.VAE) {
		out.WriteString(fmt.Sprintf("\n**VAE**: `%v`", *request.VAE))
	}

	if ptrStringNotBlank(request.Hypernetwork) {
		if ptrStringNotBlank(request.VAE) {
			out.WriteString(" ")
		} else {
			out.WriteString("\n")
		}
		out.WriteString(fmt.Sprintf("**Hypernetwork**: `%v`", *request.Hypernetwork))
	}

	if progress >= 0 && progress < 1 {
		out.WriteString(fmt.Sprintf("\n**Progress**:\n```ansi\n%v\n```", p.Get().ViewAs(progress)))
	}

	out.WriteString(fmt.Sprintf("\n```\n%s\n```", request.Prompt))

	if request.Scripts.ADetailer != nil && len(request.Scripts.ADetailer.Args) > 0 {
		var models []string
		for _, v := range request.Scripts.ADetailer.Args {
			models = append(models, v.AdModel)
		}
		out.WriteString(fmt.Sprintf("\n**ADetailer**: [%v]", strings.Join(models, ", ")))
	}

	if request.Scripts.ControlNet != nil && len(request.Scripts.ControlNet.Args) > 0 {
		var preprocessor []string
		var model []string
		for _, v := range request.Scripts.ControlNet.Args {
			preprocessor = append(preprocessor, v.Module)
			model = append(model, v.Model)
		}
		out.WriteString(fmt.Sprintf("\n**ControlNet**: [%v]\n**Preprocessor**: [%v]", strings.Join(preprocessor, ", "), strings.Join(model, ", ")))
	}

	if out.Len() > 2000 {
		return out.String()[:2000]
	}
	return out.String()
}

func imagineMessageSimple(request *entities.ImageGenerationRequest, user *discordgo.User, progress float64, ram, vram *entities.ReadableMemory) string {
	var out = strings.Builder{}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine", user.ID))
	out.WriteString(fmt.Sprintf(" `%d x %d`", request.Width, request.Height))

	if ram != nil {
		out.WriteString(fmt.Sprintf(" **RAM**: `%s`/`%s`", ram.Used, ram.Total))
	}
	if vram != nil {
		out.WriteString(fmt.Sprintf(" **VRAM**:`%s`/`%s`", vram.Used, vram.Total))
	}

	if request.EnableHr {
		// " -> (x %x) = %d x %d"
		if request.HrResizeX == 0 {
			request.HrResizeX = scaleDimension(request.Width, request.HrScale)
		}
		if request.HrResizeY == 0 {
			request.HrResizeY = scaleDimension(request.Height, request.HrScale)
		}
		out.WriteString(fmt.Sprintf(" -> (x `%s` by hires.fix) = `%d x %d`",
			strconv.FormatFloat(request.HrScale, 'f', 1, 64),
			request.HrResizeX,
			request.HrResizeY),
		)
	}

	if progress >= 0 && progress < 1 {
		out.WriteString(fmt.Sprintf("\n**Progress**:\n```ansi\n%v\n```", p.Get().ViewAs(progress)))
	}

	if out.Len() > 2000 {
		return out.String()[:2000]
	}
	return out.String()
}

func scaleDimension(dimension int, scale float64) int {
	return int(float64(dimension) * scale)
}

// lookupModel searches through []stable_diffusion_api.Cacheable models to find the model to load
func (q *SDQueue) lookupModel(request *entities.ImageGenerationRequest, config *entities.Config, c []stable_diffusion_api.Cacheable) (POST entities.Config) {
	for _, c := range c {
		var toLoad *string
		var loadedModel *string
		switch c.(type) {
		case *stable_diffusion_api.SDModels:
			toLoad = request.Checkpoint
			loadedModel = config.SDModelCheckpoint
			//log.Printf("Checkpoint: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.VAEModels:
			toLoad = request.VAE
			loadedModel = config.SDVae
			//log.Printf("VAE: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.HypernetworkModels:
			toLoad = request.Hypernetwork
			loadedModel = config.SDHypernetwork
			//log.Printf("Hypernetwork: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		}

		if ptrStringCompare(toLoad, loadedModel) {
			log.Printf("Model %T \"%v\" already loaded as \"%v\"", toLoad, safeDereference(toLoad), safeDereference(loadedModel))
		}

		if toLoad != nil {
			switch safeDereference(toLoad) {
			case "":
				// set to nil if empty string
				toLoad = nil
			case "None":
				// keep "None" to unload the model
			default:
				// lookup from the list of models
				cache, err := c.GetCache(q.stableDiffusionAPI)
				if err != nil {
					log.Println("Failed to get cached models:", err)
					continue
				}

				results := fuzzy.FindFrom(*toLoad, cache)

				if len(results) > 0 {
					firstResult := cache.String(results[0].Index)
					toLoad = &firstResult
				} else {
					log.Printf("Couldn't find model %v", safeDereference(toLoad))
					//log.Printf("Available models: %v", cache)
				}
			}
		}

		switch c.(type) {
		case *stable_diffusion_api.SDModels:
			POST.SDModelCheckpoint = toLoad
		case *stable_diffusion_api.VAEModels:
			POST.SDVae = toLoad
		case *stable_diffusion_api.HypernetworkModels:
			POST.SDHypernetwork = toLoad
		}
	}

	if POST.SDModelCheckpoint != nil || POST.SDVae != nil || POST.SDHypernetwork != nil {
		marshal, _ := POST.Marshal()
		log.Printf("Switching models to %#v", string(marshal))
	}
	return
}

func upscaleMessageContent(user *discordgo.User, fetchProgress, upscaleProgress float64) string {
	if fetchProgress >= 0 && fetchProgress <= 1 && upscaleProgress < 1 {
		if upscaleProgress == 0 {
			return fmt.Sprintf("Currently upscaling the image for you... Fetch progress: %.0f%%", fetchProgress*100)
		} else {
			return fmt.Sprintf("Currently upscaling the image for you... Fetch progress: %.0f%% Upscale progress: %.0f%%",
				fetchProgress*100, upscaleProgress*100)
		}
	} else {
		return fmt.Sprintf("<@%s> asked me to upscale their image. Here's the result:",
			user.ID)
	}
}

func ptrStringCompare(s1 *string, s2 *string) bool {
	if s1 == nil || s2 == nil {
		return s1 == s2
	}
	return *s1 == *s2
}

func ptrStringNotBlank(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}

func safeDereference(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
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
