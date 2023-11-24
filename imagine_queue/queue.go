package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sahilm/fuzzy"
	"log"
	"math"
	"os"
	"os/signal"
	"regexp"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	p "stable_diffusion_bot/gui/progress"
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

	initializedWidth      = 512
	initializedHeight     = 512
	initializedBatchCount = 4
	initializedBatchSize  = 1
)

type queueImplementation struct {
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

	return &queueImplementation{
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
	Steps              int
	SamplerName1       string
	Type               ItemType
	UseHiresFix        bool
	HiresUpscaleRate   float64
	CfgScale           float64
	AspectRatio        string
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction
	RestoreFaces       bool
	ADetailerString    string // use AppendSegModelByString
	Checkpoint         *string
	VAE                *string
	Hypernetwork       *string
}

func DefaultQueueItem() *QueueItem {
	return &QueueItem{
		NegativePrompt:   defaultNegative,
		Steps:            20,
		SamplerName1:     "Euler a",
		Type:             ItemTypeImagine,
		UseHiresFix:      false,
		HiresUpscaleRate: 1.0,
		CfgScale:         7.0,
	}
}

func NewQueueItem(options ...func(*QueueItem)) *QueueItem {
	q := DefaultQueueItem()

	for _, option := range options {
		option(q)
	}

	return q
}

func WithPrompt(prompt string) func(*QueueItem) {
	return func(q *QueueItem) {
		q.Prompt = prompt
	}
}

func (q *queueImplementation) AddImagine(item *QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *queueImplementation) StartPolling(botSession *discordgo.Session) {
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

func (q *queueImplementation) pullNextInQueue() {
	if len(q.queue) > 0 {
		element := <-q.queue

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentImagine = element

		q.processCurrentImagine()
	}
}

func (q *queueImplementation) fillInBotDefaults(settings *entities.DefaultSettings) (*entities.DefaultSettings, bool) {
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

func (q *queueImplementation) initializeOrGetBotDefaults() (*entities.DefaultSettings, error) {
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

func (q *queueImplementation) GetBotDefaultSettings() (*entities.DefaultSettings, error) {
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

func (q *queueImplementation) defaultWidth() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Width, nil
}

func (q *queueImplementation) defaultHeight() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Height, nil
}

func (q *queueImplementation) defaultBatchCount() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchCount, nil
}

func (q *queueImplementation) defaultBatchSize() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchSize, nil
}

func (q *queueImplementation) UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error) {
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

func (q *queueImplementation) UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error) {
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

// Deprecated: No longer store the SDModelName to DefaultSettings struct, use stable_diffusion_api.GetConfig instead
func (q *queueImplementation) UpdateModelName(modelName string) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	//defaultSettings.SDModelName = modelName

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated model to: %s\n", modelName)
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

var arRegex = regexp.MustCompile(`\s?--ar (\d*):(\d*)\s?`)

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

// input is 2:3 for example, without the `--ar` part
func aspectRatioCalculation(aspectRatio string, w, h int) (width, height int) {
	// split
	aspectRatioSplit := strings.Split(aspectRatio, ":")
	if len(aspectRatioSplit) != 2 {
		return w, h
	}

	// convert to int
	widthRatio, err := strconv.Atoi(aspectRatioSplit[0])
	if err != nil {
		return w, h
	}
	heightRatio, err := strconv.Atoi(aspectRatioSplit[1])
	if err != nil {
		return w, h
	}

	// calculate
	if widthRatio > heightRatio {
		scaledWidth := float64(h) * (float64(widthRatio) / float64(heightRatio))

		// Round up to the nearest 8
		width = (int(scaledWidth) + 7) & (-8)
		height = h
	} else if heightRatio > widthRatio {
		scaledHeight := float64(w) * (float64(heightRatio) / float64(widthRatio))

		// Round up to the nearest 8
		height = (int(scaledHeight) + 7) & (-8)
		width = w
	} else {
		width = w
		height = h
	}

	return width, height
}

// Deprecated: This was inadvertently adding backticks to the prompt inside the database as well
func quotePromptAsMonospace(promptIn string) (quotedprompt string) {
	// backtick(code) is shown as monospace in Discord client
	return "`" + promptIn + "`"
}

// recieve sampling process steps value
var stepRegex = regexp.MustCompile(`\s?--step (\d*)\s?`)

func extractStepsFromPrompt(prompt string, defaultsteps int) (*stepsResult, error) {

	stepMatches := stepRegex.FindStringSubmatch(prompt)
	stepsValue := defaultsteps

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
	cfgValue := defaultScale

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

var seedRegex = regexp.MustCompile(`\s?--seed (\d+)\s?`)

func extractSeedFromPrompt(prompt string) (*seedResult, error) {

	seedMatches := seedRegex.FindStringSubmatch(prompt)
	var seedValue int64 = 0
	var SeedMaxvalue = int64(math.MaxInt64) // although SD accepts: 12345678901234567890

	if len(seedMatches) == 2 {
		log.Printf("Seed overwrite: %#v", seedMatches)

		prompt = seedRegex.ReplaceAllString(prompt, "")
		s, err := strconv.ParseInt(seedMatches[1], 10, 64)
		if err != nil {
			return nil, err
		}
		seedValue = min(SeedMaxvalue, s)

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
	zoomValue := defaultZoomScale

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
		ZoomScale:       zoomValue,
	}, nil
}

const defaultNegative = "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
	"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
	"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy"

func (q *queueImplementation) processCurrentImagine() {
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

		newGeneration, err := &entities.ImageGeneration{
			Prompt:            q.currentImagine.Prompt,
			NegativePrompt:    q.currentImagine.NegativePrompt,
			Width:             initializedWidth,
			Height:            initializedHeight,
			RestoreFaces:      q.currentImagine.RestoreFaces,
			EnableHR:          q.currentImagine.UseHiresFix,
			HRUpscaleRate:     q.currentImagine.HiresUpscaleRate,
			HRUpscaler:        "R-ESRGAN 2x+",
			HiresWidth:        initializedWidth,
			HiresHeight:       initializedHeight,
			DenoisingStrength: 0.7,
			Seed:              int64(-1),
			Subseed:           -1,
			SubseedStrength:   0,
			SamplerName:       q.currentImagine.SamplerName1,
			CfgScale:          q.currentImagine.CfgScale,
			Steps:             q.currentImagine.Steps,
			Processed:         false,
			Checkpoint:        q.currentImagine.Checkpoint,
			VAE:               q.currentImagine.VAE,
			Hypernetwork:      q.currentImagine.Hypernetwork,
		}, error(nil)

		newGeneration.Width, err = q.defaultWidth()
		if err != nil {
			log.Printf("Error getting default width: %v", err)
		}

		newGeneration.Height, err = q.defaultHeight()
		if err != nil {
			log.Printf("Error getting default height: %v", err)
		}

		// add optional parameter: Negative prompt
		if q.currentImagine.NegativePrompt == "" {
			newGeneration.NegativePrompt = defaultNegative
		}

		// add optional parameter: sampler
		if q.currentImagine.SamplerName1 == "" {
			newGeneration.SamplerName = "Euler a"
		}

		if q.currentImagine.AspectRatio != "" && q.currentImagine.AspectRatio != "1:1" {
			newGeneration.Width, newGeneration.Height = aspectRatioCalculation(q.currentImagine.AspectRatio, newGeneration.Width, newGeneration.Height)
		} else {
			dimensions, err := extractDimensionsFromPrompt(newGeneration.Prompt, newGeneration.Width, newGeneration.Height)
			if err != nil {
				log.Printf("Error extracting dimensions from prompt: %v", err)
			} else {
				newGeneration.Prompt = dimensions.SanitizedPrompt
				newGeneration.Width = max(dimensions.Width, newGeneration.Width)
				newGeneration.Height = max(dimensions.Height, newGeneration.Height)
			}
		}

		// extract --zoom parameter
		zoom, errZ := extractZoomScaleFromPrompt(newGeneration.Prompt, newGeneration.HRUpscaleRate)
		if errZ != nil {
			log.Printf("Error extracting zoom scale from prompt: %v", errZ)
		} else if newGeneration.EnableHR {
			newGeneration.HRUpscaleRate = max(newGeneration.HRUpscaleRate, zoom.ZoomScale)
			newGeneration.HiresWidth = int(float64(newGeneration.Width) * newGeneration.HRUpscaleRate)
			newGeneration.HiresHeight = int(float64(newGeneration.Height) * newGeneration.HRUpscaleRate)
			// hrSecondPassSteps = 10
		} else {
			newGeneration.HRUpscaleRate = 1.0
			newGeneration.HRUpscaler = ""
			newGeneration.HiresWidth = newGeneration.Width
			newGeneration.HiresHeight = newGeneration.Height
		}

		steps, err := extractStepsFromPrompt(newGeneration.Prompt, newGeneration.Steps)
		if err != nil {
			log.Printf("Error extracting step from prompt: %v", err)
		} else if steps.Steps != newGeneration.Steps {
			newGeneration.Prompt = steps.SanitizedPrompt
			newGeneration.Steps = steps.Steps
		}

		cfgScale, err := extractCFGScaleFromPrompt(newGeneration.Prompt, newGeneration.CfgScale)
		if err != nil {
			log.Printf("Error extracting cfg scale from prompt: %v", err)
		} else if cfgScale.CFGScale != newGeneration.CfgScale {
			newGeneration.Prompt = cfgScale.SanitizedPrompt
			newGeneration.CfgScale = cfgScale.CFGScale
		}

		seed, err := extractSeedFromPrompt(cfgScale.SanitizedPrompt)
		if err != nil {
			log.Printf("Error extracting seed from prompt: %v", err)
		} else if seed.Seed != newGeneration.Seed {
			newGeneration.Seed = seed.Seed
		}

		// prompt will display as Monospace in Discord
		//var quotedPrompt = quotePromptAsMonospace(promptRes4.SanitizedPrompt)
		//promptRes.SanitizedPrompt = quotedPrompt

		config, err := q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
		} else {
			if !ptrStringNotBlank(newGeneration.Checkpoint) {
				newGeneration.Checkpoint = config.SDModelCheckpoint
			}
			if !ptrStringNotBlank(newGeneration.VAE) {
				newGeneration.VAE = config.SDVae
			}
			if !ptrStringNotBlank(newGeneration.Hypernetwork) {
				newGeneration.Hypernetwork = config.SDHypernetwork
			}
		}

		// segModelOptions will never be nil and at least an empty string in the slice [""]
		// because of strings.Split() in discord_bot.go

		//additionalScript := make(map[string]*entities.ADetailer)
		//alternatively additionalScript := map[string]*stable_diffusion_api.ADetailer{}

		if q.currentImagine.ADetailerString != "" {
			log.Printf("q.currentImagine.ADetailerString: %v", q.currentImagine.ADetailerString)

			newGeneration.NewADetailer()

			newGeneration.AlwaysOnScripts.ADetailer.AppendSegModelByString(q.currentImagine.ADetailerString, newGeneration)
		}

		if newGeneration.AlwaysOnScripts != nil {
			jsonMarshalScripts, err := json.MarshalIndent(&newGeneration.AlwaysOnScripts, "", "  ")
			if err != nil {
				log.Printf("Error marshalling scripts: %v", err)
			} else {
				log.Println("Final scripts: ", string(jsonMarshalScripts))
			}
		}

		// Should not create a new map here, because it will be overwritten by the map in newGeneration
		// if newGeneration.AlwaysOnScripts == nil {
		// 	newGeneration.AlwaysOnScripts = make(map[string]*entities.ADetailer)
		// }

		//if additionalScript["ADetailer"] != nil {
		//	newGeneration.AlwaysOnScripts["ADetailer"] = additionalScript["ADetailer"]
		//}

		switch q.currentImagine.Type {
		case ItemTypeReroll, ItemTypeVariation:
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

func (q *queueImplementation) getPreviousGeneration(imagine *QueueItem, sortOrder int) (*entities.ImageGeneration, error) {
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
	var out = strings.Builder{}

	var scriptsString string

	if generation.AlwaysOnScripts != nil && generation.AlwaysOnScripts.ADetailer != nil {
		scripts, err := json.Marshal(generation.AlwaysOnScripts)
		if err != nil {
			log.Printf("Error marshalling scripts: %v", err)
			return fmt.Sprintf("Error marshalling scripts: %v", err)
		} else {
			scriptsString = string(scripts)
		}
	}

	seedString := fmt.Sprintf("%d", generation.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine with step: `%d` cfg: `%s` seed: `%s` sampler: `%s`",
		user.ID,
		generation.Steps,
		strconv.FormatFloat(generation.CfgScale, 'f', 1, 64),
		seedString,
		generation.SamplerName,
	))

	out.WriteString(fmt.Sprintf(" `%d x %d`", generation.Width, generation.Height))

	if generation.EnableHR == true {
		// " -> (x %x) = %d x %d"
		out.WriteString(fmt.Sprintf(" -> (x `%s` by hires.fix) = `%d x %d`",
			strconv.FormatFloat(generation.HRUpscaleRate, 'f', 1, 64),
			generation.HiresWidth,
			generation.HiresHeight),
		)
	}

	if ptrStringNotBlank(generation.Checkpoint) {
		out.WriteString(fmt.Sprintf("\n**Checkpoint**: `%v`", *generation.Checkpoint))
	}

	if ptrStringNotBlank(generation.VAE) {
		out.WriteString(fmt.Sprintf("\n**VAE**: `%v`", *generation.VAE))
	}

	if ptrStringNotBlank(generation.Hypernetwork) {
		if ptrStringNotBlank(generation.VAE) {
			out.WriteString(" ")
		} else {
			out.WriteString("\n")
		}
		out.WriteString(fmt.Sprintf("**Hypernetwork**: `%v`", *generation.Hypernetwork))
	}

	if progress >= 0 && progress < 1 {
		out.WriteString(fmt.Sprintf("\n**Progress**:\n```ansi\n%v\n```", p.Get().ViewAs(progress)))
	}

	out.WriteString(fmt.Sprintf("\n```\n%s\n```", generation.Prompt))

	if scriptsString != "" {
		out.WriteString(fmt.Sprintf("**Scripts**: ```json\n%v\n```", scriptsString))
	}

	return out.String()
}

func (q *queueImplementation) processImagineGrid(newGeneration *entities.ImageGeneration, imagine *QueueItem) error {
	config, err := q.stableDiffusionAPI.GetConfig()
	if err != nil {
		log.Printf("Error getting config: %v", err)
	} else {
		if !ptrStringCompare(newGeneration.Checkpoint, config.SDModelCheckpoint) ||
			!ptrStringCompare(newGeneration.VAE, config.SDVae) ||
			!ptrStringCompare(newGeneration.Hypernetwork, config.SDHypernetwork) {
			handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, imagine.DiscordInteraction,
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
			}
		}
	}

	log.Printf("Processing imagine #%s: %v\n", imagine.DiscordInteraction.ID, newGeneration.Prompt)

	newContent := imagineMessageContent(newGeneration, imagine.DiscordInteraction.Member.User, 0)

	message, err := q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
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
		AlwaysOnScripts:   newGeneration.AlwaysOnScripts,
	})
	if err != nil {
		log.Printf("Error processing image: %v\n", err)

		errorContent := fmt.Sprint("I'm sorry, but I had a problem imagining your image. ", err)

		//_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		//	Content: &errorContent,
		//})

		handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, errorContent)
		//handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, errorContent)

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
			BatchCount:        newGeneration.BatchCount,
			BatchSize:         newGeneration.BatchSize,
			Seed:              resp.Seeds[idx],
			Subseed:           resp.Subseeds[idx],
			SubseedStrength:   newGeneration.SubseedStrength,
			SamplerName:       newGeneration.SamplerName,
			CfgScale:          newGeneration.CfgScale,
			Steps:             newGeneration.Steps,
			Processed:         true,
			Checkpoint:        config.SDModelCheckpoint,
			VAE:               config.SDVae,
			Hypernetwork:      config.SDHypernetwork,
			AlwaysOnScripts:   newGeneration.AlwaysOnScripts,
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

	// TODO: Add ephemeral follow up to delete message
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

	//handlers.EphemeralFollowup(q.botSession, imagine.DiscordInteraction, "Delete generation", handlers.Components[handlers.DeleteAboveButton])

	return nil
}

func (q *queueImplementation) switchModel(generation *entities.ImageGeneration, config *stable_diffusion_api.APIConfig, c []stable_diffusion_api.Cacheable) (POST stable_diffusion_api.APIConfig) {
	for _, c := range c {
		var toLoad *string
		var loadedModel *string
		switch c.(type) {
		case *stable_diffusion_api.SDModels:
			toLoad = generation.Checkpoint
			loadedModel = config.SDModelCheckpoint
			//log.Printf("Checkpoint: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.VAEModels:
			toLoad = generation.VAE
			loadedModel = config.SDVae
			//log.Printf("VAE: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.HypernetworkModels:
			toLoad = generation.Hypernetwork
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

func (q *queueImplementation) processUpscaleImagine(imagine *QueueItem) {
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
	if generation.AlwaysOnScripts == nil {
		generation.NewScripts()
	}
	if generation.AlwaysOnScripts.ADetailer == nil {
		generation.AlwaysOnScripts.NewADetailerWithArgs()
		generation.AlwaysOnScripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", generation)
	}

	resp, err := q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:      0,
		UpscalingResize: 2,
		Upscaler1:       "R-ESRGAN 2x+",
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
			AlwaysOnScripts:   generation.AlwaysOnScripts,
		},
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

	if generation.AlwaysOnScripts != nil {
		scripts, err := json.Marshal(generation.AlwaysOnScripts)
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
