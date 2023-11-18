// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    vAEs, err := UnmarshalVAEs(bytes)
//    bytes, err = vAEs.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
	"log"
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

var VAECache VAEModels

func (api *apiImplementation) SDVAECache() (VAEModels, error) {
	if VAECache != nil {
		return VAECache, nil
	}
	return api.vaeApi()
}

func (api *apiImplementation) vaeApi() (VAEModels, error) {
	getURL := "/sdapi/v1/sd-vae"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	VAECache, err = UnmarshalVAEs(body)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return VAECache, nil
}
