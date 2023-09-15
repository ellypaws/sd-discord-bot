package stable_diffusion_api

type StableDiffusionAPI interface {
	SDModels() ([]StableDiffusionModel, error)
	TextToImage(req *TextToImageRequest) (*TextToImageResponse, error)
	UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error)
	GetCurrentProgress() (*ProgressResponse, error)
	UpdateConfiguration(key, value string) error
}
