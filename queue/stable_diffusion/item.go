package stable_diffusion

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/entities"
	"time"
)

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
