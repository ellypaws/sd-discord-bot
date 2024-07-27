package stable_diffusion

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny-sd/llm"
	"github.com/sahilm/fuzzy"
	"log"
	"os"
	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	p "stable_diffusion_bot/gui/progress"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/repositories"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type SDQueueItem struct {
	Type ItemType

	*entities.ImageGenerationRequest

	LLMRequest *llm.Request
	LLMCreated time.Time

	AspectRatio        string
	InteractionIndex   int
	DiscordInteraction *discordgo.Interaction

	ADetailerString string // use AppendSegModelByString
	Attachments     map[string]*entities.MessageAttachment

	Img2ImgItem
	ControlnetItem

	Raw *entities.TextToImageRaw // raw JSON input

	Interrupt chan *discordgo.Interaction
}

func (q *SDQueueItem) Interaction() *discordgo.Interaction {
	return q.DiscordInteraction
}

type Img2ImgItem struct {
	*entities.MessageAttachment
	DenoisingStrength float64
}

type ControlnetItem struct {
	*entities.MessageAttachment
	ControlMode  entities.ControlMode
	ResizeMode   entities.ResizeMode
	Type         string
	Preprocessor string // also called the module in entities.ControlNetParameters
	Model        string
	Enabled      bool
}

type ItemType int

const (
	botID = "bot"

	initializedWidth      = 512
	initializedHeight     = 512
	initializedBatchCount = 4
	initializedBatchSize  = 1
)

type SDQueue struct {
	botSession          *discordgo.Session
	stableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	queue               chan *SDQueueItem
	currentImagine      *SDQueueItem
	mu                  sync.Mutex
	imageGenerationRepo image_generations.Repository
	compositor          composite_renderer.Renderer
	defaultSettingsRepo default_settings.Repository
	botDefaultSettings  *entities.DefaultSettings
	cancelledItems      map[string]bool

	llmConfig *llm.Config

	stop chan os.Signal
}

type Config struct {
	StableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	ImageGenerationRepo image_generations.Repository
	DefaultSettingsRepo default_settings.Repository
}

func New(cfg Config) (queue.Queue[*SDQueueItem], error) {
	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
	}

	if cfg.ImageGenerationRepo == nil {
		return nil, errors.New("missing image generation repository")
	}

	if cfg.DefaultSettingsRepo == nil {
		return nil, errors.New("missing default settings repository")
	}

	return &SDQueue{
		stableDiffusionAPI:  cfg.StableDiffusionAPI,
		imageGenerationRepo: cfg.ImageGenerationRepo,
		queue:               make(chan *SDQueueItem, 100),
		compositor:          composite_renderer.Compositor(),
		defaultSettingsRepo: cfg.DefaultSettingsRepo,
		cancelledItems:      make(map[string]bool),
	}, nil
}

const (
	ItemTypeImagine ItemType = iota
	ItemTypeReroll
	ItemTypeUpscale
	ItemTypeVariation
	ItemTypeImg2Img
	ItemTypeRaw // raw JSON
)

func (q *SDQueue) DefaultQueueItem() *SDQueueItem {
	defaultBatchCount, err := q.defaultBatchCount()
	if err != nil {
		log.Printf("Error getting default batch count: %v", err)
		defaultBatchCount = 1
	}

	defaultBatchSize, err := q.defaultBatchSize()
	if err != nil {
		log.Printf("Error getting default batch size: %v", err)
		defaultBatchSize = 4
	}

	defaultWidth, err := q.defaultWidth()
	if err != nil {
		log.Printf("Error getting default width: %v", err)
		defaultWidth = 512
	}

	defaultHeight, err := q.defaultHeight()
	if err != nil {
		log.Printf("Error getting default height: %v", err)
		defaultHeight = 512
	}

	return &SDQueueItem{
		Type: ItemTypeImagine,

		ImageGenerationRequest: &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				CreatedAt: time.Now(),
			},
			TextToImageRequest: &entities.TextToImageRequest{
				Width:             defaultWidth,
				Height:            defaultHeight,
				NegativePrompt:    DefaultNegative,
				Steps:             20,
				Seed:              -1,
				SamplerName:       "Euler a",
				EnableHr:          false,
				HrUpscaler:        "R-ESRGAN 2x+",
				HrSecondPassSteps: 20,
				HrScale:           1.0,
				DenoisingStrength: 0.7,
				CFGScale:          7.0,
				NIter:             defaultBatchCount,
				BatchSize:         defaultBatchSize,
			},
		},

		Img2ImgItem: Img2ImgItem{
			DenoisingStrength: 0.7,
		},
		ControlnetItem: ControlnetItem{
			ControlMode: entities.ControlModeBalanced,
			ResizeMode:  entities.ResizeModeScaleToFit,
		},
	}
}

func (q *SDQueue) NewItem(interaction *discordgo.Interaction, options ...func(*SDQueueItem)) *SDQueueItem {
	item := q.DefaultQueueItem()
	item.DiscordInteraction = interaction

	for _, option := range options {
		option(item)
	}

	return item
}

func WithPrompt(prompt string) func(*SDQueueItem) {
	return func(q *SDQueueItem) {
		q.Prompt = prompt
	}
}

func WithCurrentModels(api stable_diffusion_api.StableDiffusionAPI) func(*SDQueueItem) {
	return func(q *SDQueueItem) {
		config, err := api.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
		} else {
			q.ImageGenerationRequest.Checkpoint = config.SDModelCheckpoint
			q.VAE = config.SDVae
			q.Hypernetwork = config.SDHypernetwork
		}
	}
}

func (q *SDQueue) Add(queue *SDQueueItem) (int, error) {
	if len(q.queue) == cap(q.queue) {
		return -1, errors.New("queue is full")
	}

	q.queue <- queue

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *SDQueue) StartPollingWithLLM(botSession *discordgo.Session, config *llm.Config) {
	q.llmConfig = config
	q.Start(botSession)
}

func (q *SDQueue) Start(botSession *discordgo.Session) {
	q.botSession = botSession

	botDefaultSettings, err := q.initializeOrGetBotDefaults()
	if err != nil {
		log.Printf("Error getting/initializing bot default settings: %v", err)

		return
	}

	q.botDefaultSettings = botDefaultSettings

	var once bool

Polling:
	for {
		select {
		case <-q.stop:
			break Polling
		case <-time.After(1 * time.Second):
			if q.currentImagine == nil {
				if err := q.pullNextInQueue(); err != nil {
					log.Printf("Error processing next item: %v", err)
				}
				once = false
			} else if !once {
				log.Printf("Waiting for current imagine to finish...\n")
				once = true
			}
		}
	}

	log.Println("Polling stopped for Stable Diffusion")
}

func (q *SDQueue) Stop() {
	if q.stop == nil {
		q.stop = make(chan os.Signal)
	}
	q.stop <- os.Interrupt
	close(q.stop)
}

func (q *SDQueue) pullNextInQueue() error {
	for len(q.queue) > 0 {
		// Peek at the next item without blocking
		if q.currentImagine != nil {
			log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
			return errors.New("currentImagine is not nil")
		}

		var err error
		select {
		case q.currentImagine = <-q.queue:
			if q.currentImagine.DiscordInteraction == nil {
				// If the interaction is nil, we can't respond. Make sure to set the implementation before adding to the queue.
				// Example: queue.DiscordInteraction = i.Interaction
				log.Panicf("DiscordInteraction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", q.currentImagine)
			}
			if interaction := q.currentImagine.DiscordInteraction; interaction != nil && q.cancelledItems[q.currentImagine.DiscordInteraction.ID] {
				// If the item is cancelled, skip it
				delete(q.cancelledItems, interaction.ID)
				q.done()
				return nil
			}
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
				q.done()
				return handlers.ErrorEdit(q.botSession, q.currentImagine.DiscordInteraction, fmt.Errorf("unknown item type: %v", q.currentImagine.Type))
			}
		default:
			log.Printf("WARNING: we're trying to pull the next item in the queue, but the queue is empty")
			return nil // Queue is empty
		}

		if err != nil {
			if q.currentImagine.DiscordInteraction == nil {
				return err
			}
			return handlers.ErrorEdit(q.botSession, q.currentImagine.DiscordInteraction, fmt.Errorf("error processing current item: %w", err))
		}
	}

	return nil
}

func (q *SDQueue) done() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentImagine = nil
}

func (q *SDQueue) Remove(messageInteraction *discordgo.MessageInteraction) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Mark the item as cancelled
	q.cancelledItems[messageInteraction.ID] = true

	return nil
}

func (q *SDQueue) Interrupt(i *discordgo.Interaction) error {
	if q.currentImagine == nil {
		return errors.New("there is no generation currently in progress")
	}

	// Mark the item as cancelled
	log.Printf("Interrupting generation #%s\n", q.currentImagine.DiscordInteraction.ID)
	if q.currentImagine.Interrupt == nil {
		q.currentImagine.Interrupt = make(chan *discordgo.Interaction)
	}
	q.currentImagine.Interrupt <- i

	return nil
}

func (q *SDQueue) fillInBotDefaults(settings *entities.DefaultSettings) (*entities.DefaultSettings, bool) {
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

func (q *SDQueue) initializeOrGetBotDefaults() (*entities.DefaultSettings, error) {
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

func (q *SDQueue) GetBotDefaultSettings() (*entities.DefaultSettings, error) {
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

func (q *SDQueue) defaultWidth() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Width, nil
}

func (q *SDQueue) defaultHeight() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Height, nil
}

func (q *SDQueue) defaultBatchCount() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchCount, nil
}

func (q *SDQueue) defaultBatchSize() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchSize, nil
}

func (q *SDQueue) UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error) {
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

func (q *SDQueue) UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error) {
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
func (q *SDQueue) UpdateModelName(modelName string) (*entities.DefaultSettings, error) {
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

const DefaultNegative = "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
	"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
	"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy"

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

	if request.EnableHr == true {
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

	seedString := fmt.Sprintf("%d", request.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine", user.ID))

	out.WriteString(fmt.Sprintf(" `%d x %d`", request.Width, request.Height))

	if ram != nil {
		out.WriteString(fmt.Sprintf(" **RAM**: `%s`/`%s`", ram.Used, ram.Total))
	}

	if vram != nil {
		out.WriteString(fmt.Sprintf(" **VRAM**:`%s`/`%s`", vram.Used, vram.Total))
	}

	if request.EnableHr == true {
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
