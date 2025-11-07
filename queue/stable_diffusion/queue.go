package stable_diffusion

import (
	"errors"
	"os"
	"sync"
	"time"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
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
	logger              *log.Logger

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
		logger: log.NewWithOptions(os.Stdout, log.Options{
			Level:           log.DebugLevel,
			Prefix:          "[SD]",
			ReportTimestamp: true,
			ReportCaller:    true,
		}),
	}, nil
}

func (q *SDQueue) Commands() []*discordgo.ApplicationCommand { return q.commands() }

func (q *SDQueue) Handlers() queue.CommandHandlers { return q.handlers() }

func (q *SDQueue) Components() queue.Components { return q.components() }

const (
	ItemTypeImagine ItemType = iota
	ItemTypeReroll
	ItemTypeUpscale
	ItemTypeVariation
	ItemTypeImg2Img
	ItemTypeRaw // raw JSON
)

func (q *SDQueue) Add(queue *SDQueueItem) (int, error) {
	if len(q.queue) == cap(q.queue) {
		return -1, errors.New("queue is full")
	}

	q.queue <- queue

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *SDQueue) Start(botSession *discordgo.Session) {
	q.botSession = botSession

	botDefaultSettings, err := q.initializeOrGetBotDefaults()
	if err != nil {
		q.logger.Error("Failed to get or initialize bot default settings", "error", err)
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
				if err := q.next(); err != nil {
					q.logger.Error("Failed to process next item", "error", err)
				}
				once = false
			} else if !once {
				q.logger.Info("Waiting for current imagine to finish...")
				once = true
			}
		}
	}

	q.logger.Info("Polling stopped for Stable Diffusion")
}

func (q *SDQueue) Stop() {
	if q.stop == nil {
		q.stop = make(chan os.Signal)
	}
	q.stop <- os.Interrupt
	close(q.stop)
}

func (q *SDQueue) Remove(messageInteraction *discordgo.MessageInteractionMetadata) error {
	q.mu.Lock()
	q.cancelledItems[messageInteraction.ID] = true
	q.mu.Unlock()

	return nil
}

func (q *SDQueue) Interrupt(i *discordgo.Interaction) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.currentImagine == nil {
		return errors.New("there is no generation currently in progress")
	}

	// Mark the item as cancelled
	q.logger.Info("Interrupting generation", "id", q.currentImagine.DiscordInteraction.ID)
	if q.currentImagine.Interrupt == nil {
		q.currentImagine.Interrupt = make(chan *discordgo.Interaction)
	}
	q.currentImagine.Interrupt <- i
	close(q.currentImagine.Interrupt)

	return nil
}
