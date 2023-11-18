package stable_diffusion_api

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error)
	PopulateCache() (errors []error)
	SDCheckpointsCache() (SDModels, error)
	SDLorasCache() (LoraModels, error)
	SDVAECache() (VAEModels, error)
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	UpdateConfiguration(configuration POSTCheckpoint) error
	GetConfig() (*APIConfig, error)
	GetCheckpoint() (string, error)
}
