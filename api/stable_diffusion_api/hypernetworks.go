// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    hypernetworkModels, err := UnmarshalHypernetworkModels(bytes)
//    bytes, err = hypernetworkModels.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
	"log"
)

type HypernetworkModels []HypernetworkModel

func UnmarshalHypernetworkModels(data []byte) (HypernetworkModels, error) {
	var r HypernetworkModels
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *HypernetworkModels) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type HypernetworkModel struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (c HypernetworkModels) String(i int) string {
	return c[i].Name
}

func (c HypernetworkModels) Len() int {
	return len(c)
}

var HypernetworkCache *HypernetworkModels

// GetCache returns var HypernetworkCache *HypernetworkModels as a Cacheable. Assert using cache.(*HypernetworkModels)
func (c *HypernetworkModels) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if HypernetworkCache != nil {
		return HypernetworkCache, nil
	}
	return c.apiGET(api)
}

func (c *HypernetworkModels) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	log.Println("No endpoint to refresh hypernetworks cache")
	return c.GetCache(api)
}

func (c *HypernetworkModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	getURL := api.Host("/sdapi/v1/hypernetworks")

	cache, err := GET[HypernetworkModels](api.Client(), getURL)
	if err != nil {
		return nil, err
	}
	HypernetworkCache = cache

	return HypernetworkCache, nil
}
