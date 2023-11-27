package image_generations

import (
	"context"
	"stable_diffusion_bot/entities"
)

type Repository interface {
	Create(ctx context.Context, generation *entities.ImageGenerationRequest) (*entities.ImageGenerationRequest, error)
	GetByMessage(ctx context.Context, messageID string) (*entities.ImageGenerationRequest, error)
	GetByMessageAndSort(ctx context.Context, messageID string, sortOrder int) (*entities.ImageGenerationRequest, error)
}
