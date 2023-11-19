package stable_diffusion_api

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error) // Deprecated: use SDCheckpointsCache instead
	PopulateCache() (errors []error)
	CachePreview(c Cacheable) (Cacheable, error)
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	UpdateConfiguration(configuration POSTCheckpoint) error
	GetConfig() (*APIConfig, error)
	GetCheckpoint() (string, error)
	GET(string) ([]byte, error)
	Host() string

	// invidual caches TODO: use Cacheable interface
	SDCheckpointsCache() (SDModels, error)            // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDLorasCache() (LoraModels, error)                // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDVAECache() (VAEModels, error)                   // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDHypernetworkCache() (HypernetworkModels, error) // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
	SDEmbeddingCache() (EmbeddingModels, error)       // Deprecated: use Cacheable interface instead with Cacheable.GetCache() method
}

type Cacheable interface {
	String(int) string
	Len() int
	GetCache(StableDiffusionAPI) (Cacheable, error)
	apiGET(StableDiffusionAPI) (Cacheable, error)
}
