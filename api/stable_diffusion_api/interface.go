package stable_diffusion_api

import (
	"github.com/sahilm/fuzzy"
	"net/http"
	"stable_diffusion_bot/entities"
)

type StableDiffusionAPI interface {
	PopulateCache() (errors []error)
	RefreshCache(cache Cacheable) (Cacheable, error)
	CachePreview(c Cacheable) (Cacheable, error)

	TextToImageRequest(req *entities.TextToImageRequest) (*entities.TextToImageResponse, error)
	TextToImageRaw(req []byte) (*entities.TextToImageResponse, error)
	ImageToImageRequest(req *entities.ImageToImageRequest) (*entities.ImageToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	GetProgress() (*Progress, error)

	UpdateConfiguration(config entities.Config) error

	GetConfig() (*entities.Config, error)
	GetCheckpoint() (*string, error)
	GetVAE() (*string, error)
	GetHypernetwork() (*string, error)

	GetMemory() (*entities.Memory, error)
	GetMemoryReadable() (*entities.ReadableMemory, error)
	GetVRAMReadable() (*entities.ReadableMemory, error)

	GET(string) ([]byte, error)
	POST(postURL string, jsonData []byte) (*http.Response, error)
	Host() string

	Interrupt() error
}

type Cacheable interface {
	fuzzy.Source

	// GetCache uses each implementation's apiGET method to fetch the cache.
	// Make sure to check which type assertion is required, usually *Type
	GetCache(StableDiffusionAPI) (Cacheable, error)
	Refresh(StableDiffusionAPI) (Cacheable, error)

	apiGET(StableDiffusionAPI) (Cacheable, error)
}
