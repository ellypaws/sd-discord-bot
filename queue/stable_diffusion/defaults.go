package stable_diffusion

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/repositories"
)

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

const (
	botID = "bot"

	initializedWidth      = 512
	initializedHeight     = 512
	initializedBatchCount = 4
	initializedBatchSize  = 1
)

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
