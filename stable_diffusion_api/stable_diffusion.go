package stable_diffusion_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type jsonTextToImageResponse struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
}

type jsonInfoResponse struct {
	Seed        int64   `json:"seed"`
	AllSeeds    []int64 `json:"all_seeds"`
	AllSubseeds []int   `json:"all_subseeds"`
}

type TextToImageResponse struct {
	Images   []string `json:"images"`
	Seeds    []int64  `json:"seeds"`
	Subseeds []int    `json:"subseeds"`
}

type StableDiffusionModel struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
	Hash      string `json:"hash"`
	Sha256    string `json:"sha256"`
	Filename  string `json:"filename"`
	Config    string `json:"config"`
}

type TextToImageRequest struct {
	Prompt            string            `json:"prompt"`
	NegativePrompt    string            `json:"negative_prompt"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	RestoreFaces      bool              `json:"restore_faces"`
	EnableHR          bool              `json:"enable_hr"`
	HRUpscaleRate     float64           `json:"hr_scale"`
	HRUpscaler        string            `json:"hr_upscaler"`
	HRResizeX         int               `json:"hr_resize_x"`
	HRResizeY         int               `json:"hr_resize_y"`
	DenoisingStrength float64           `json:"denoising_strength"`
	BatchSize         int               `json:"batch_size"`
	Seed              int64             `json:"seed"`
	Subseed           int               `json:"subseed"`
	SubseedStrength   float64           `json:"subseed_strength"`
	SamplerName       string            `json:"sampler_name"`
	CfgScale          float64           `json:"cfg_scale"`
	Steps             int               `json:"steps"`
	NIter             int               `json:"n_iter"`
	AlwaysOnScripts   *entities.Scripts `json:"alwayson_scripts,omitempty"`
}

func (api *apiImplementation) PopulateCache() (errors []error) {
	if CheckpointCache == nil {
		_, err := api.checkpointsApi()
		if err != nil {
			errors = append(errors, err)
		}
		log.Printf("Successfully precached %v checkpoint models", len(CheckpointCache))
	}

	if LoraCache == nil {
		_, err := api.sdLoraApi()
		if err != nil {
			errors = append(errors, err)
		}
		log.Printf("Successfully precached %v lora models", len(LoraCache))
	}

	return nil
}

func (api *apiImplementation) SDModels() ([]StableDiffusionModel, error) {
	// Make an HTTP request to fetch the stable diffusion models
	handle, err := os.Open("available_models.json")
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	// Parse the response and create choices
	var sdModels []StableDiffusionModel
	err = json.NewDecoder(handle).Decode(&sdModels)
	if err != nil {
		return nil, err
	}

	return sdModels, nil
}

func (api *apiImplementation) TextToImage(req *TextToImageRequest) (*TextToImageResponse, error) {
	//fmt.Println("TextToImageRequest", req)
	if req == nil {
		return nil, errors.New("missing request")
	}

	postURL := api.host + "/sdapi/v1/txt2img"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if !handlers.CheckAPIAlive(api.host) {
		return nil, fmt.Errorf(handlers.DeadAPI)
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

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	respStruct := &jsonTextToImageResponse{}

	err = json.Unmarshal(body, respStruct)
	if err != nil {
		log.Printf("API URL: %s", postURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	infoStruct := &jsonInfoResponse{}

	err = json.Unmarshal([]byte(respStruct.Info), infoStruct)
	if err != nil {
		log.Printf("API URL: %s", postURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return &TextToImageResponse{
		Images:   respStruct.Images,
		Seeds:    infoStruct.AllSeeds,
		Subseeds: infoStruct.AllSubseeds,
	}, nil
}

type UpscaleRequest struct {
	ResizeMode         int                 `json:"resize_mode"`
	UpscalingResize    int                 `json:"upscaling_resize"`
	Upscaler1          string              `json:"upscaler_1"`
	TextToImageRequest *TextToImageRequest `json:"text_to_image_request"`
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
	if upscaleReq == nil {
		return nil, errors.New("missing request")
	}

	textToImageReq := upscaleReq.TextToImageRequest

	if textToImageReq == nil {
		return nil, errors.New("missing text to image request")
	}

	textToImageReq.NIter = 1

	regeneratedImage, err := api.TextToImage(textToImageReq)
	if err != nil {
		return nil, err
	}

	jsonReq := &upscaleJSONRequest{
		ResizeMode:      upscaleReq.ResizeMode,
		UpscalingResize: upscaleReq.UpscalingResize,
		Upscaler1:       upscaleReq.Upscaler1,
		Image:           regeneratedImage.Images[0],
	}

	jsonReqMessage, _ := json.MarshalIndent(jsonReq, "", "  ")
	// set image key to value of blank string
	jsonWithoutImage := make(map[string]any)
	_ = json.Unmarshal(jsonReqMessage, &jsonWithoutImage)
	delete(jsonWithoutImage, "image")
	jsonReqMessage, _ = json.MarshalIndent(jsonWithoutImage, "", "  ")
	log.Printf(string(jsonReqMessage))

	postURL := api.host + "/sdapi/v1/extra-single-image"

	jsonData, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	if !handlers.CheckAPIAlive(api.host) {
		return nil, fmt.Errorf(handlers.DeadAPI)
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

	defer response.Body.Close()

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
	getURL := api.host + "/sdapi/v1/progress"

	request, err := http.NewRequest("GET", getURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Error with API Request: %v", err)

		return nil, err
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	respStruct := &ProgressResponse{}

	err = json.Unmarshal(body, respStruct)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return respStruct, nil
}

type POSTCheckpoint struct {
	SdModelCheckpoint string `json:"sd_model_checkpoint,omitempty"`
}

func (api *apiImplementation) GET(getURL string) ([]byte, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, fmt.Errorf(handlers.DeadAPI)
	}

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

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	return body, nil
}

func (api *apiImplementation) UpdateConfiguration(post POSTCheckpoint) error {
	headers := map[string]string{
		"accept":       "application/json",
		"Content-Type": "application/json",
	}

	body, err := json.Marshal(post)
	if err != nil {
		return errors.New("error marshalling config: " + err.Error())
	}
	fmt.Printf("Passing '%v' to sdapi/v1/options", string(body))

	if !handlers.CheckAPIAlive(api.host) {
		return fmt.Errorf(handlers.DeadAPI)
	}

	req, err := http.NewRequest("POST", api.host+"/sdapi/v1/options", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	return nil
}

func (api *apiImplementation) GetConfig() (*APIConfig, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, fmt.Errorf(handlers.DeadAPI)
	}

	req, err := http.NewRequest("GET", api.host+"/sdapi/v1/options", nil)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiConfig APIConfig
	apiConfig, err = UnmarshalAPIConfig(body)
	if err != nil {
		return nil, err
	}

	return &apiConfig, nil
}

func (api *apiImplementation) GetCheckpoint() (string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return "", err
	}

	return apiConfig.Checkpoint(), nil
}
