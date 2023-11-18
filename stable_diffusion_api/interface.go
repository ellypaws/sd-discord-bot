package stable_diffusion_api

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error) // Deprecated: use SDCheckpointsCache instead
	PopulateCache() (errors []error)
	SDCheckpointsCache() (SDModels, error)
	SDLorasCache() (LoraModels, error)
	SDVAECache() (VAEModels, error)
	SDHypernetworkCache() (HypernetworkModels, error)
	SDEmbeddingCache() (EmbeddingModels, error)
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	UpdateConfiguration(configuration POSTCheckpoint) error
	GetConfig() (*APIConfig, error)
	GetCheckpoint() (string, error)
}
