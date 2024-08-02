package stable_diffusion_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
)

type apiImplementation struct {
	host string
}

type Config struct {
	Host string
}

func New(cfg Config) (StableDiffusionAPI, error) {
	if cfg.Host == "" {
		return nil, errors.New("missing host")
	}

	return &apiImplementation{
		host: cfg.Host,
	}, nil
}

func (api *apiImplementation) Host() string {
	return api.host
}

// Deprecated: Use the entities.ImageToImageResponse instead
type ImageToImageResponse struct {
	Images     []string          `json:"images,omitempty"`
	Info       string            `json:"info"`
	Parameters map[string]string `json:"parameters"`
}

type StableDiffusionModel struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
	Hash      string `json:"hash"`
	Sha256    string `json:"sha256"`
	Filename  string `json:"filename"`
	Config    string `json:"config"`
}

// Deprecated: Use the entities.TextToImageRequest in entities.ImageGeneration instead
type TextToImageRequest struct {
	Prompt            string            `json:"prompt"`
	NegativePrompt    string            `json:"negative_prompt"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	RestoreFaces      bool              `json:"restore_faces"`
	EnableHR          bool              `json:"enable_hr"`
	HRUpscaleRate     float64           `json:"hr_scale"`
	HRUpscaler        string            `json:"hr_upscaler"`
	HRSteps           int64             `json:"hr_second_pass_steps"`
	HRResizeX         int               `json:"hr_resize_x"`
	HRResizeY         int               `json:"hr_resize_y"`
	DenoisingStrength float64           `json:"denoising_strength"`
	BatchSize         int               `json:"batch_size"`
	Seed              int64             `json:"seed"`
	Subseed           int64             `json:"subseed"`
	SubseedStrength   float64           `json:"subseed_strength"`
	SamplerName       string            `json:"sampler_name"`
	CfgScale          float64           `json:"cfg_scale"`
	Steps             int               `json:"steps"`
	NIter             int               `json:"n_iter"`
	AlwaysOnScripts   *entities.Scripts `json:"alwayson_scripts,omitempty"`
}

func (api *apiImplementation) CachePreview(c Cacheable) (Cacheable, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}
	//_, err := c.GetCache(api)
	//if err != nil {
	//	return c, err
	//}
	if c.Len() > 2 {
		log.Printf("Successfully cached %v %T from api: %v...", c.Len(), c, c.String(0))
	}
	//return cache, nil

	return c, nil
}

func (api *apiImplementation) PopulateCache() (errors []error) {
	var caches = []Cacheable{
		CheckpointCache,
		LoraCache,
		VAECache,
		HypernetworkCache,
		EmbeddingCache,
	}
	for _, cache := range caches {
		cache, err := cache.GetCache(api)
		if err != nil {
			errors = append(errors, err)
		}
		_, err = api.CachePreview(cache)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}

func (api *apiImplementation) RefreshCache(cache Cacheable) (Cacheable, error) {
	if cache == nil {
		return cache.GetCache(api)
	}
	return cache.Refresh(api)
}

func (api *apiImplementation) TextToImageRequest(req *entities.TextToImageRequest) (*entities.TextToImageResponse, error) {
	jsonData, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	return api.TextToImageRaw(jsonData)
}

func (api *apiImplementation) TextToImageRaw(req []byte) (*entities.TextToImageResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if req == nil {
		return nil, errors.New("missing request")
	}

	response, err := api.POST("/sdapi/v1/txt2img", req)
	if err != nil {
		return nil, fmt.Errorf("error with POST request: %w", err)
	}
	defer closeResponseBody(response)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return entities.JSONToTextToImageResponse(body)
}

func (api *apiImplementation) ImageToImageRequest(req *entities.ImageToImageRequest) (*entities.ImageToImageResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if req == nil {
		return nil, errors.New("missing request")
	}

	jsonData, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	response, err := api.POST("/sdapi/v1/img2img", jsonData)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return entities.UnmarshalImageToImageResponse(body)
}

type UpscaleRequest struct {
	ResizeMode         int                          `json:"resize_mode"`
	UpscalingResize    int                          `json:"upscaling_resize"`
	Upscaler1          string                       `json:"upscaler_1"`
	TextToImageRequest *entities.TextToImageRequest `json:"text_to_image_request"`
}

type upscaleJSONRequest struct {
	ResizeMode      int    `json:"resize_mode"`
	UpscalingResize int    `json:"upscaling_resize"`
	Upscaler1       string `json:"upscaler_1"`
	Image           string `json:"image"`
}

type UpscaleResponse struct {
	Image string `json:"image"`
}

func (api *apiImplementation) UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if upscaleReq == nil {
		return nil, errors.New("missing request")
	}

	textToImageReq := upscaleReq.TextToImageRequest

	if textToImageReq == nil {
		return nil, errors.New("missing text to image request")
	}

	textToImageReq.NIter = 1

	regeneratedImage, err := api.TextToImageRequest(textToImageReq)
	if err != nil {
		return nil, err
	}

	if len(regeneratedImage.Images) == 0 {
		return nil, errors.New("no images returned from text to image request to upscale")
	}

	jsonReq := &upscaleJSONRequest{
		ResizeMode:      upscaleReq.ResizeMode,
		UpscalingResize: upscaleReq.UpscalingResize,
		Upscaler1:       upscaleReq.Upscaler1,
		Image:           regeneratedImage.Images[0],
	}

	postURL := api.host + "/sdapi/v1/extra-single-image"

	jsonData, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("API URL: %s", postURL)
		log.Printf("Error with API Request: %s", string(jsonData))

		return nil, err
	}

	defer closeResponseBody(response)
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("unexpected status code: %v", response.Status))
	}

	body, _ := io.ReadAll(response.Body)

	respStruct := &UpscaleResponse{}

	err = json.Unmarshal(body, respStruct)
	if err != nil {
		log.Printf("API URL: %s", postURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return respStruct, nil
}

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	EtaRelative float64 `json:"eta_relative"`
}

func (api *apiImplementation) GetCurrentProgress() (*ProgressResponse, error) {
	getURL := "/sdapi/v1/progress"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	respStruct := &ProgressResponse{}

	err = json.Unmarshal(body, respStruct)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return respStruct, nil
}

// Deprecated: Use entities.Config instead
type POSTConfig struct {
	SdModelCheckpoint string `json:"sd_model_checkpoint,omitempty"`
}

func (api *apiImplementation) GET(getURL string) ([]byte, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	getURL = api.host + getURL

	request, err := http.NewRequest("GET", getURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Error with API Request: %s", getURL)

		return nil, err
	}

	defer closeResponseBody(response)
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		errorString := "(unknown error)"
		if len(body) > 0 {
			errorString = fmt.Sprintf("\n```json\n%v\n```", string(body))
		}
		return nil, fmt.Errorf("unexpected status code: `%v` %v", response.Status, errorString)
	}

	body, _ := io.ReadAll(response.Body)
	return body, nil
}

func (api *apiImplementation) POST(postURL string, jsonData []byte) (*http.Response, error) {
	// Create a new POST request
	request, err := http.NewRequest("POST", api.host+postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Set headers
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	// Create an HTTP client
	client := &http.Client{}

	// Send the POST request
	response, err := client.Do(request)
	if err != nil {
		log.Printf("API URL: %s", api.host+postURL)
		log.Printf("Error with API Request: %v", err)
		log.Printf("Body: %v", string(jsonData))
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		errorString := "(unknown error)"
		if len(body) > 0 {
			errorString = fmt.Sprintf("\n```json\n%v\n```", string(body))
		}
		return nil, fmt.Errorf("unexpected status code: `%v` %v", response.Status, errorString)
	}

	return response, nil
}

func (api *apiImplementation) UpdateConfiguration(config entities.Config) error {
	if !handlers.CheckAPIAlive(api.host) {
		return errors.New(handlers.DeadAPI)
	}

	body, err := config.Marshal()
	if err != nil {
		return errors.New("error marshalling config: " + err.Error())
	}
	log.Printf("Passing '%v' to sdapi/v1/options", string(body))

	response, err := api.POST("/sdapi/v1/options", body)
	defer closeResponseBody(response)
	if err != nil {
		return err
	}

	log.Printf("Response status: %v", response)

	return nil
}

func closeResponseBody(response *http.Response) {
	if response != nil {
		if err := response.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}
}

// interrupt by posting to /sdapi/v1/interrupt using the POST() function
func (api *apiImplementation) Interrupt() error {
	if !handlers.CheckAPIAlive(api.host) {
		return errors.New(handlers.DeadAPI)
	}
	_, err := api.POST("/sdapi/v1/interrupt", nil)
	if err != nil {
		return err
	}

	return nil
}
