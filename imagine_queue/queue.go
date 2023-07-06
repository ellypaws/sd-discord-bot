package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"regexp"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/repositories"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	botID = "bot"

	initializedWidth  = 512
	initializedHeight = 512
	initializedBatchCount = 4
	initializedBatchSize  = 1
)

type queueImpl struct {
	botSession          *discordgo.Session
	stableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	queue               chan *QueueItem
	currentImagine      *QueueItem
	mu                  sync.Mutex
	imageGenerationRepo image_generations.Repository
	compositeRenderer   composite_renderer.Renderer
	defaultSettingsRepo default_settings.Repository
	botDefaultSettings  *entities.DefaultSettings
}

type Config struct {
	StableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	ImageGenerationRepo image_generations.Repository
	DefaultSettingsRepo default_settings.Repository
}

func New(cfg Config) (Queue, error) {
	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
	}

	if cfg.ImageGenerationRepo == nil {
		return nil, errors.New("missing image generation repository")
	}

	if cfg.DefaultSettingsRepo == nil {
		return nil, errors.New("missing default settings repository")
	}

	compositeRenderer, err := composite_renderer.New(composite_renderer.Config{})
	if err != nil {
		return nil, err
	}

	return &queueImpl{
		stableDiffusionAPI:  cfg.StableDiffusionAPI,
		imageGenerationRepo: cfg.ImageGenerationRepo,
		queue:               make(chan *QueueItem, 100),
		compositeRenderer:   compositeRenderer,
		defaultSettingsRepo: cfg.DefaultSettingsRepo,
	}, nil
}

type ItemType int

const (
	ItemTypeImagine ItemType = iota
	ItemTypeReroll
	ItemTypeUpscale
	ItemTypeVariation
)

type QueueItem struct {
	Prompt             string
	NegativePrompt     string
	SamplerName1       string
	Type               ItemType
	UseHiresFix        bool
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
}

func (q *queueImpl) AddImagine(item *QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *queueImpl) StartPolling(botSession *discordgo.Session) {
	q.botSession = botSession

	botDefaultSettings, err := q.initializeOrGetBotDefaults()
	if err != nil {
		log.Printf("Error getting/initializing bot default settings: %v", err)

		return
	}

	q.botDefaultSettings = botDefaultSettings

	log.Println("Press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	stopPolling := false

	for {
		select {
		case <-stop:
			stopPolling = true
		case <-time.After(1 * time.Second):
			if q.currentImagine == nil {
				q.pullNextInQueue()
			}
		}

		if stopPolling {
			break
		}
	}

	log.Printf("Polling stopped...\n")
}

func (q *queueImpl) pullNextInQueue() {
	if len(q.queue) > 0 {
		element := <-q.queue

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentImagine = element

		q.processCurrentImagine()
	}
}

func (q *queueImpl) fillInBotDefaults(settings *entities.DefaultSettings) (*entities.DefaultSettings, bool) {
	updated := false

	if settings == nil {
		settings = &entities.DefaultSettings{
			MemberID: botID,
		}
	}

	if settings.Width == 0 {
		settings.Width = initializedWidth
		updated = true
	}

	if settings.Height == 0 {
		settings.Height = initializedHeight
		updated = true
	}

	if settings.BatchCount == 0 {
		settings.BatchCount = initializedBatchCount
		updated = true
	}

	if settings.BatchSize == 0 {
		settings.BatchSize = initializedBatchSize
		updated = true
	}

	return settings, updated
}


func (q *queueImpl) initializeOrGetBotDefaults() (*entities.DefaultSettings, error) {
	botDefaultSettings, err := q.GetBotDefaultSettings()
	if err != nil && !errors.Is(err, &repositories.NotFoundError{}) {
		return nil, err
	}

	botDefaultSettings, updated := q.fillInBotDefaults(botDefaultSettings)
	if updated {
		botDefaultSettings, err = q.defaultSettingsRepo.Upsert(context.Background(), botDefaultSettings)
		if err != nil {
			return nil, err
		}

		log.Printf("Initialized bot default settings: %+v\n", botDefaultSettings)
	} else {
		log.Printf("Retrieved bot default settings: %+v\n", botDefaultSettings)
	}

	return botDefaultSettings, nil
}

func (q *queueImpl) GetBotDefaultSettings() (*entities.DefaultSettings, error) {
	if q.botDefaultSettings != nil {
		return q.botDefaultSettings, nil
	}

	defaultSettings, err := q.defaultSettingsRepo.GetByMemberID(context.Background(), botID)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = defaultSettings

	return defaultSettings, nil
}

func (q *queueImpl) defaultWidth() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Width, nil
}

func (q *queueImpl) defaultHeight() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Height, nil
}

func (q *queueImpl) defaultBatchCount() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchCount, nil
}

func (q *queueImpl) defaultBatchSize() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchSize, nil
}


func (q *queueImpl) UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	defaultSettings.Width = width
	defaultSettings.Height = height

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated default dimensions to: %dx%d\n", width, height)

	return newDefaultSettings, nil
}

func (q *queueImpl) UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	defaultSettings.BatchCount = batchCount
	defaultSettings.BatchSize = batchSize

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated default batch count/size to: %d/%d\n", batchCount, batchSize)

	return newDefaultSettings, nil
}

type dimensionsResult struct {
	SanitizedPrompt string
	Width           int
	Height          int
}

type stepsResult struct {
	SanitizedPrompt string
	Steps           int
}

type cfgScaleResult struct {
	SanitizedPrompt string
	CFGScale        float64
}

type seedResult struct {
	SanitizedPrompt string
	Seed            int64
}

type zoomScaleResult struct {
	SanitizedPrompt string
	ZoomScale       float64
}



const (
	emdash = '\u2014'
	hyphen = '\u002D'
)

func fixEmDash(prompt string) string {
	return strings.ReplaceAll(prompt, string(emdash), string(hyphen)+string(hyphen))
}

var arRegex = regexp.MustCompile(`\s?--ar ([\d]*):([\d]*)\s?`)

func extractDimensionsFromPrompt(prompt string, width, height int) (*dimensionsResult, error) {
	// Sanitize em dashes. Some phones will autocorrect to em dashes
	prompt = fixEmDash(prompt)

	arMatches := arRegex.FindStringSubmatch(prompt)

	if len(arMatches) == 3 {
		log.Printf("Aspect ratio overwrite: %#v", arMatches)

		prompt = arRegex.ReplaceAllString(prompt, "")

		firstDimension, err := strconv.Atoi(arMatches[1])
		if err != nil {
			return nil, err
		}

		secondDimension, err := strconv.Atoi(arMatches[2])
		if err != nil {
			return nil, err
		}

		if firstDimension > secondDimension {
			scaledWidth := float64(height) * (float64(firstDimension) / float64(secondDimension))

			// Round up to the nearest 8
			width = (int(scaledWidth) + 7) & (-8)
		} else if secondDimension > firstDimension {
			scaledHeight := float64(width) * (float64(secondDimension) / float64(firstDimension))

			// Round up to the nearest 8
			height = (int(scaledHeight) + 7) & (-8)
		}

		log.Printf("New dimensions: width: %v, height: %v", width, height)
	}

	return &dimensionsResult{
		SanitizedPrompt: prompt,
		Width:           width,
		Height:          height,
	}, nil
}


func quotePromptAsMonospace(promptIn string) (quotedprompt string) {
	// backtick(code) is shown as monospace in Discord client
	return "`" + promptIn + "`"
}

// recieve sampling process steps value
var stepRegex = regexp.MustCompile(`\s?--step ([\d]*)\s?`)

func extractStepsFromPrompt(prompt string, defaultsteps int) (*stepsResult, error) {

	stepMatches := stepRegex.FindStringSubmatch(prompt)
	stepsValue  := defaultsteps

	if len(stepMatches) == 2 {
		log.Printf("steps overwrite: %#v", stepMatches)

		prompt = stepRegex.ReplaceAllString(prompt, "")

		s, err := strconv.Atoi(stepMatches[1])
		if err != nil {
			return nil, err
		}
		stepsValue = s

		if s < 1 {
			stepsValue = defaultsteps
		}
	}

	return &stepsResult{
		SanitizedPrompt: prompt,
		Steps:           stepsValue,
	}, nil
}

var cfgscaleRegex = regexp.MustCompile(`\s?--cfgscale (\d\d?\.?\d?)\s?`)
func extractCFGScaleFromPrompt(prompt string, defaultScale float64) (*cfgScaleResult, error) {

	cfgscaleMatches := cfgscaleRegex.FindStringSubmatch(prompt)
	cfgValue  := defaultScale

	if len(cfgscaleMatches) == 2 {
		log.Printf("CFG Scale overwrite: %#v", cfgscaleMatches)

		prompt = cfgscaleRegex.ReplaceAllString(prompt, "")
		c, err := strconv.ParseFloat(cfgscaleMatches[1], 64)
		if err != nil {
			return nil, err
		}
		cfgValue = c

		if c < 1.0 || c > 30.0 {
			cfgValue = defaultScale
		}
	}

	return &cfgScaleResult{
		SanitizedPrompt: prompt,
		CFGScale:        cfgValue,
	}, nil
}

var seedRegex = regexp.MustCompile(`\s?--seed ([\d]+)\s?`)
func extractSeedFromPrompt(prompt string) (*seedResult, error) {

	seedMatches := seedRegex.FindStringSubmatch(prompt)
	var seedValue int64 = 0
	var Seed_MaxValue int64 = int64(math.MaxInt64) // although SD accepts: 12345678901234567890

	if len(seedMatches) == 2 {
		log.Printf("Seed overwrite: %#v", seedMatches)

		prompt = seedRegex.ReplaceAllString(prompt, "")
		s, err := strconv.ParseInt(seedMatches[1], 10, 64)
		if err != nil {
			return nil, err
		}		
		if int64(s) > Seed_MaxValue {
			seedValue = Seed_MaxValue
		} else {
			seedValue = int64(s)
		}

	} else {
		seedValue = int64(-1)
	}

	return &seedResult{
		SanitizedPrompt: prompt,
		Seed:            seedValue,
	}, nil
}

// hires.fix upscaleby 
var zoomRegex = regexp.MustCompile(`\s?--zoom (\d\d?\.?\d?)\s?`)
func extractZoomScaleFromPrompt(prompt string, defaultZoomScale float64) (*zoomScaleResult, error) {

	zoomMatches := zoomRegex.FindStringSubmatch(prompt)
	zoomValue  := defaultZoomScale

	if len(zoomMatches) == 2 {
		log.Printf("Zoom Scale overwrite: %#v", zoomMatches)

		prompt = zoomRegex.ReplaceAllString(prompt, "")
		z, err := strconv.ParseFloat(zoomMatches[1], 64)
		if err != nil {
			return nil, err
		}
		zoomValue = z

		if z < 1.0 || z > 4.0 {
			zoomValue = defaultZoomScale
		}
	}

	return &zoomScaleResult{
		SanitizedPrompt: prompt,
		ZoomScale:        zoomValue,
	}, nil
}



const defaultNegative = "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
		"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
		"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy"


func (q *queueImpl) processCurrentImagine() {
	go func() {
		defer func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			q.currentImagine = nil
		}()

		if q.currentImagine.Type == ItemTypeUpscale {
			q.processUpscaleImagine(q.currentImagine)

			return
		}

		defaultWidth, err := q.defaultWidth()
		if err != nil {
			log.Printf("Error getting default width: %v", err)

			return
		}

		defaultHeight, err := q.defaultHeight()
		if err != nil {
			log.Printf("Error getting default height: %v", err)

			return
		}

		// add optional parameter: Negative prompt	
		negativePrompt := ""

		if q.currentImagine.NegativePrompt == "" {
			negativePrompt = defaultNegative
		} else {
			negativePrompt = q.currentImagine.NegativePrompt
		}

		// add optional parameter: sampler
		samplerName1 := ""
		if q.currentImagine.SamplerName1 == "" {
			samplerName1 = "Euler a"
		} else {
			samplerName1 = q.currentImagine.SamplerName1
		}

		promptRes, err := extractDimensionsFromPrompt(q.currentImagine.Prompt, defaultWidth, defaultHeight)
		if err != nil {
			log.Printf("Error extracting dimensions from prompt: %v", err)

			return
		}

		scaledWidth := defaultWidth
		scaledHeight := defaultHeight
		hiresWidth := defaultWidth
		hiresHeight := defaultHeight
	
		if promptRes.Width > defaultWidth || promptRes.Height > defaultHeight {
			scaledWidth = promptRes.Width
			scaledHeight = promptRes.Height
			hiresWidth = promptRes.Width
			hiresHeight = promptRes.Height
		}
		
		// add optional parameter: enable hires.fix
		enableHR1 := false
		upscaleRate1  := 1.0
		upscalerName1 := ""

		// extract --zoom parameter

		defaultZoomValue1 := 2.0
		promptResZ, errZ := extractZoomScaleFromPrompt(promptRes.SanitizedPrompt, defaultZoomValue1)
		if errZ != nil {
			log.Printf("Error extracting zoom scale from prompt: %v", errZ)

			return
		}

		enableHR1 = q.currentImagine.UseHiresFix 
		if enableHR1 == true {
			upscaleRate1 = promptResZ.ZoomScale
			upscalerName1 = "Latent"
			hiresWidth = 0
			hiresHeight = 0
			// hrSecondPassSteps = 10
		} else {
			enableHR1 = false
			upscaleRate1 = 1.0
			upscalerName1 = ""
			hiresWidth = scaledWidth
			hiresHeight = scaledHeight
		}

		stepValue := 20 // default steps value
		promptRes2, err := extractStepsFromPrompt(promptResZ.SanitizedPrompt, stepValue)
		if err != nil {
			log.Printf("Error extracting step from prompt: %v", err)
		} else if promptRes2.Steps != stepValue {
			stepValue = promptRes2.Steps
		}

		cfgScaleValue := 9.0 // default CFG scale value
		promptRes3, err := extractCFGScaleFromPrompt(promptRes2.SanitizedPrompt, cfgScaleValue)
		if err != nil {
			log.Printf("Error extracting cfg scale from prompt: %v", err)
		} else if promptRes3.CFGScale != cfgScaleValue {
			cfgScaleValue = promptRes3.CFGScale
		}

		seedValue := int64(-1) // default seed is random
		promptRes4, err := extractSeedFromPrompt(promptRes3.SanitizedPrompt)
		if err != nil {
			log.Printf("Error extracting seed from prompt: %v", err)
		} else if promptRes4.Seed != seedValue {
			seedValue = promptRes4.Seed
		}


		// prompt will displayed as Monospace in Discord
		var quotedPrompt = quotePromptAsMonospace(promptRes4.SanitizedPrompt)
		promptRes.SanitizedPrompt = quotedPrompt

		// new generation with defaults
		newGeneration := &entities.ImageGeneration{
			Prompt: promptRes.SanitizedPrompt,
			NegativePrompt:    negativePrompt,
			Width:             scaledWidth,
			Height:            scaledHeight,
			RestoreFaces:      true,
			EnableHR:          enableHR1,
			HRUpscaleRate:     upscaleRate1,
			HRUpscaler:        upscalerName1,
			HiresWidth:        hiresWidth,
			HiresHeight:       hiresHeight,
			DenoisingStrength: 0.7,
			Seed:              seedValue,
			Subseed:           -1,
			SubseedStrength:   0,
			SamplerName:       samplerName1,
			CfgScale:          cfgScaleValue,
			Steps:             stepValue,
			Processed:         false,
		}

		if q.currentImagine.Type == ItemTypeReroll || q.currentImagine.Type == ItemTypeVariation {
			foundGeneration, err := q.getPreviousGeneration(q.currentImagine, q.currentImagine.InteractionIndex)
			if err != nil {
				log.Printf("Error getting prompt for reroll: %v", err)

				return
			}

			// if we are rerolling, or generating variations, we simply replace some defaults
			newGeneration = foundGeneration

			// for variations, we need random subseeds
			newGeneration.Subseed = -1

			// for variations, the subseed strength determines how much variation we get
			if q.currentImagine.Type == ItemTypeVariation {
				newGeneration.SubseedStrength = 0.15
			}
		}

		err = q.processImagineGrid(newGeneration, q.currentImagine)
		if err != nil {
			log.Printf("Error processing imagine grid: %v", err)

			return
		}
	}()
}

func (q *queueImpl) getPreviousGeneration(imagine *QueueItem, sortOrder int) (*entities.ImageGeneration, error) {
	interactionID := imagine.DiscordInteraction.ID
	messageID := ""

	if imagine.DiscordInteraction.Message != nil {
		messageID = imagine.DiscordInteraction.Message.ID
	}

	log.Printf("Reimagining interaction: %v, Message: %v", interactionID, messageID)

	generation, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, sortOrder)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return nil, err
	}

	log.Printf("Found generation: %v", generation)

	return generation, nil
}

func imagineMessageContent(generation *entities.ImageGeneration, user *discordgo.User, progress float64) string {
	if progress >= 0 && progress < 1 {
		return fmt.Sprintf("<@%s> asked me to imagine \"%s\". Currently dreaming it up for them. Progress: %.0f%%",
			user.ID, generation.Prompt, progress*100)
	} else {
		seedString := fmt.Sprintf("%d", generation.Seed)
		if seedString == "-1" {
			seedString = "at ramdom(-1)"
		}

		sizeString := ""
		if generation.EnableHR == true {
			sizeString = fmt.Sprintf("%d x %d -> (x %s by hires.fix)",
						generation.Width,
						generation.Height,
						strconv.FormatFloat(generation.HRUpscaleRate,'f', 1, 64))
		} else {
			sizeString = fmt.Sprintf("%d x %d",
						generation.Width,
						generation.Height)
		}
		return fmt.Sprintf("<@%s> asked me to imagine \"%s\" at step %d cfgscale %s seed %s with sampler %s. resolution: %s. here is what I imagined for them.",
			user.ID,
			generation.Prompt,
			generation.Steps,
			strconv.FormatFloat(generation.CfgScale,'f', 1, 64),
			seedString,
			generation.SamplerName,
			sizeString,
		)
	}
}

func (q *queueImpl) processImagineGrid(newGeneration *entities.ImageGeneration, imagine *QueueItem) error {
	log.Printf("Processing imagine #%s: %v\n", imagine.DiscordInteraction.ID, newGeneration.Prompt)

	newContent := imagineMessageContent(newGeneration, imagine.DiscordInteraction.Member.User, 0)

	message, err := q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
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

	newGeneration.InteractionID = imagine.DiscordInteraction.ID
	newGeneration.MessageID = message.ID
	newGeneration.MemberID = imagine.DiscordInteraction.Member.User.ID
	newGeneration.SortOrder = 0
	newGeneration.BatchCount = defaultBatchCount
	newGeneration.BatchSize = defaultBatchSize
	newGeneration.Processed = true

	_, err = q.imageGenerationRepo.Create(context.Background(), newGeneration)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
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

					return
				}

				if progress.Progress == 0 {
					continue
				}

				progressContent := imagineMessageContent(newGeneration, imagine.DiscordInteraction.Member.User, progress.Progress)

				_, progressErr = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
					Content: &progressContent,
				})
				if progressErr != nil {
					log.Printf("Error editing interaction: %v", err)
				}
			}
		}
	}()

	resp, err := q.stableDiffusionAPI.TextToImage(&stable_diffusion_api.TextToImageRequest{
		Prompt:            newGeneration.Prompt,
		NegativePrompt:    newGeneration.NegativePrompt,
		Width:             newGeneration.Width,
		Height:            newGeneration.Height,
		RestoreFaces:      newGeneration.RestoreFaces,
		EnableHR:          newGeneration.EnableHR,
		HRUpscaleRate:     newGeneration.HRUpscaleRate,
		HRUpscaler:        newGeneration.HRUpscaler,
		HRResizeX:         newGeneration.HiresWidth,
		HRResizeY:         newGeneration.HiresHeight,
		DenoisingStrength: newGeneration.DenoisingStrength,
		BatchSize:         newGeneration.BatchSize,
		Seed:              newGeneration.Seed,
		Subseed:           newGeneration.Subseed,
		SubseedStrength:   newGeneration.SubseedStrength,
		SamplerName:       newGeneration.SamplerName,
		CfgScale:          newGeneration.CfgScale,
		Steps:             newGeneration.Steps,
		NIter:             newGeneration.BatchCount,
	})
	if err != nil {
		log.Printf("Error processing image: %v\n", err)

		errorContent := "I'm sorry, but I had a problem imagining your image."

		_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &errorContent,
		})

		return err
	}

	generationDone <- true

	finishedContent := imagineMessageContent(newGeneration, imagine.DiscordInteraction.Member.User, 1)

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
		subGeneration := &entities.ImageGeneration{
			InteractionID:     newGeneration.InteractionID,
			MessageID:         newGeneration.MessageID,
			MemberID:          newGeneration.MemberID,
			SortOrder:         idx + 1,
			Prompt:            newGeneration.Prompt,
			NegativePrompt:    newGeneration.NegativePrompt,
			Width:             newGeneration.Width,
			Height:            newGeneration.Height,
			RestoreFaces:      newGeneration.RestoreFaces,
			EnableHR:          newGeneration.EnableHR,
			HRUpscaleRate:     newGeneration.HRUpscaleRate,
			HRUpscaler:        newGeneration.HRUpscaler,
			HiresWidth:        newGeneration.HiresWidth,
			HiresHeight:       newGeneration.HiresHeight,
			DenoisingStrength: newGeneration.DenoisingStrength,
			BatchCount:	   newGeneration.BatchCount,
			BatchSize:         newGeneration.BatchSize,
			Seed:              resp.Seeds[idx],
			Subseed:           resp.Subseeds[idx],
			SubseedStrength:   newGeneration.SubseedStrength,
			SamplerName:       newGeneration.SamplerName,
			CfgScale:          newGeneration.CfgScale,
			Steps:             newGeneration.Steps,
			Processed:         true,
		}

		_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
		if createErr != nil {
			log.Printf("Error creating image generation record: %v\n", createErr)
		}
	}

	compositeImage, err := q.compositeRenderer.TileImages(imageBufs)
	if err != nil {
		log.Printf("Error tiling images: %v\n", err)

		return err
	}

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				// append timestamp for grid image result
				Name:        "imagine_" + time.Now().Format("20060102150405") + ".png",
				Reader:      compositeImage,
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
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)

		return err
	}

	return nil
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

func (q *queueImpl) processUpscaleImagine(imagine *QueueItem) {
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

	resp, err := q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:      0,
		UpscalingResize: 2,
		Upscaler1:       "ESRGAN_4x",
		TextToImageRequest: &stable_diffusion_api.TextToImageRequest{
			Prompt:            generation.Prompt,
			NegativePrompt:    generation.NegativePrompt,
			Width:             generation.Width,
			Height:            generation.Height,
			RestoreFaces:      generation.RestoreFaces,
			EnableHR:          generation.EnableHR,
			HRUpscaleRate:     generation.HRUpscaleRate,
			HRUpscaler:        generation.HRUpscaler,
			HRResizeX:         generation.HiresWidth,
			HRResizeY:         generation.HiresHeight,
			DenoisingStrength: generation.DenoisingStrength,
			BatchSize:         1,
			Seed:              generation.Seed,
			Subseed:           generation.Subseed,
			SubseedStrength:   generation.SubseedStrength,
			SamplerName:       generation.SamplerName,
			CfgScale:          generation.CfgScale,
			Steps:             generation.Steps,
			NIter:             1,
		},
	})
	if err != nil {
		log.Printf("Error processing image upscale: %v\n", err)

		errorContent := "I'm sorry, but I had a problem upscaling your image."

		_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &errorContent,
		})

		return
	}

	generationDone <- true

	decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
	if decodeErr != nil {
		log.Printf("Error decoding image: %v\n", decodeErr)

		return
	}

	imageBuf := bytes.NewBuffer(decodedImage)

	log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, imagine.InteractionIndex)

	finishedContent := fmt.Sprintf("<@%s> asked me to upscale their image. (seed: %d) Here's the result:",
		imagine.DiscordInteraction.Member.User.ID,
		generation.Seed)

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				// add timestamp to output file 
				Name:        "imagine_" + time.Now().Format("20060102150405") + ".png",
				Reader:      imageBuf,
			},
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)

		return
	}
}
