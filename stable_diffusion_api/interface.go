package stable_diffusion_api

import (
	"net/http"
	"stable_diffusion_bot/entities"
)

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error) // Deprecated: use SDCheckpointsCache instead

	PopulateCache() (errors []error)
	RefreshCache(cache Cacheable) (Cacheable, error)
	CachePreview(c Cacheable) (Cacheable, error)

	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error) // Deprecated: use TextToImageRequest instead
	TextToImageRequest(req *entities.TextToImageRequest) (*TextToImageResponse, error)
	ImageToImageRequest(req *entities.ImageToImageRequest) (*entities.ImageToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)

	UpdateConfiguration(config APIConfig) error

	GetConfig() (*APIConfig, error)
	GetCheckpoint() (*string, error)
	GetVAE() (*string, error)
	GetHypernetwork() (*string, error)

	GET(string) ([]byte, error)
	POST(postURL string, jsonData []byte) (*http.Response, error)
	Host() string

	// invidual caches TODO: use Cacheable interface
	SDCheckpointsCache() (*SDModels, error)            // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDLorasCache() (*LoraModels, error)                // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDVAECache() (*VAEModels, error)                   // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDHypernetworkCache() (*HypernetworkModels, error) // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDEmbeddingCache() (*EmbeddingModels, error)       // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method

	Interrupt() error
}

type Cacheable interface {
	String(int) string
	Len() int

	// GetCache uses each implementation's apiGET method to fetch the cache.
	// Make sure to check which type assertion is required, usually *Type
	GetCache(StableDiffusionAPI) (Cacheable, error)
	Refresh(StableDiffusionAPI) (Cacheable, error)

	apiGET(StableDiffusionAPI) (Cacheable, error)
}
