// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    vAEs, err := UnmarshalVAEs(bytes)
//    bytes, err = vAEs.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
)

type VAEModels []Vae

func UnmarshalVAEs(data []byte) (VAEModels, error) {
	var r VAEModels
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *VAEModels) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Vae struct {
	ModelName string `json:"model_name"`
	Filename  string `json:"filename"`
}

func (c VAEModels) String(i int) string {
	return c[i].ModelName
}

func (c VAEModels) Len() int {
	return len(c)
}

var VAECache *VAEModels

// GetCache returns var VAECache *VAEModels as a Cacheable. Assert using cache.(*VAEModels)
func (c *VAEModels) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if c != nil {
		return c, nil
	}
	if VAECache != nil {
		return VAECache, nil
	}
	return c.apiGET(api)
}

func (c *VAEModels) Refresh(api StableDiffusionAPI) (Cacheable, error) {
	postURL := api.Host("/sdapi/v1/refresh-vae")

	err := POST[error](api.Client(), postURL, nil, nil)
	if err != nil {
		return nil, err
	}

	return c.apiGET(api)
}

func (c *VAEModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	getURL := api.Host("/sdapi/v1/sd-vae")

	vae, err := GET[VAEModels](api.Client(), getURL)
	if err != nil {
		return nil, err
	}
	VAECache = vae

	return VAECache, nil
}
