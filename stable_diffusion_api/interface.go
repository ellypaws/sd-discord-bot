package stable_diffusion_api

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error)
	PopulateCache() (errors []error)
	SDModelsCache() (SDModels, error)
	SDLorasCache() (LoraModels, error)
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	UpdateConfiguration(configuration POSTCheckpoint) error
	GetConfig() (*APIConfig, error)
	GetCheckpoint() (string, error)
}
