// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    sDModels, err := UnmarshalSDModels(bytes)
//    bytes, err = sDModels.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
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
	postURL := api.Host("/sdapi/v1/refresh-checkpoints")

	err := POST[error](api.Client(), postURL, nil, nil)
	if err != nil {
		return nil, err
	}

	return c.apiGET(api)
}

func (c *SDModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	getURL := api.Host("/sdapi/v1/sd-models")

	cache, err := GET[SDModels](api.Client(), getURL)
	if err != nil {
		return nil, err
	}
	CheckpointCache = cache

	return CheckpointCache, nil
}
