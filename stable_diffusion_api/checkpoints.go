// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    sDModels, err := UnmarshalSDModels(bytes)
//    bytes, err = sDModels.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

type SDModels []SDModel

func UnmarshalSDModels(data []byte) (SDModels, error) {
	var r SDModels
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *SDModels) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type SDModel struct {
	Title     string  `json:"title"`
	ModelName string  `json:"model_name"`
	Hash      *string `json:"hash"`
	Sha256    *string `json:"sha256"`
	Filename  string  `json:"filename"`
	Config    *string `json:"config"`
}

// String is what we fuzzy match against
func (c SDModels) String(i int) string {
	return c[i].Title
}

func (c SDModels) Len() int {
	return len(c)
}

var CheckpointCache *SDModels

// GetCache returns var CheckpointCache *SDModels as a Cacheable. Assert using cache.(*SDModels)
func (c *SDModels) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if CheckpointCache != nil {
		return CheckpointCache, nil
	}
	return c.apiGET(api)
}

func (c *SDModels) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	postURL := "/sdapi/v1/refresh-checkpoints"

	response, err := api.POST(postURL, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response)

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		log.Printf("API URL: %s", postURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, errors.New("unexpected API response")
	}

	return c.apiGET(api)
}

func (c *SDModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	// Make an HTTP request to fetch the stable diffusion models
	getURL := "/sdapi/v1/sd-models"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	cache, err := UnmarshalSDModels(body)
	CheckpointCache = &cache
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return CheckpointCache, nil
}

func (api *apiImplementation) SDCheckpointsCache() (*SDModels, error) {
	cache, err := CheckpointCache.GetCache(api)
	if err != nil {
		return nil, err
	}
	return cache.(*SDModels), err
}
