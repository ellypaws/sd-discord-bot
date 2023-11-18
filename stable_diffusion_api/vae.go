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

type VAEs []Vae

func UnmarshalVAEs(data []byte) (VAEs, error) {
	var r VAEs
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *VAEs) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Vae struct {
	ModelName string `json:"model_name"`
	Filename  string `json:"filename"`
}

func (c VAEs) String(i int) string {
	return c[i].ModelName
}

func (c VAEs) Len() int {
	return len(c)
}

var VAECache VAEs

func (api *apiImplementation) VAECache() (VAEs, error) {
	if VAECache != nil {
		log.Println("Using cached VAEs")
		return VAECache, nil
	}
	return api.vaeApi()
}

func (api *apiImplementation) vaeApi() (VAEs, error) {
	getURL := api.host + "/sdapi/v1/sd-vae"

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

	if len(VAECache) > 2 {
		log.Printf("Successfully cached %v vaes from api: %v...", len(VAECache), VAECache[:2])
	}
	return VAECache, nil
}
