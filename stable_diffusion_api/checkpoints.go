// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    sDModels, err := UnmarshalSDModels(bytes)
//    bytes, err = sDModels.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
	"log"
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

var CheckpointCache SDModels

// TODO: SDModelsCache and SDLorasCache are identical except for the endpoint they hit and the cache they write to.
func (api *apiImplementation) SDModelsCache() (SDModels, error) {
	if CheckpointCache != nil {
		log.Println("Using cached SD models")
		return CheckpointCache, nil
	}
	return api.checkpointsApi()
}

func (api *apiImplementation) checkpointsApi() (SDModels, error) {
	// Make an HTTP request to fetch the stable diffusion models
	getURL := api.host + "/sdapi/v1/sd-models"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	CheckpointCache, err = UnmarshalSDModels(body)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	if len(CheckpointCache) > 2 {
		log.Printf("Successfully cached %v checkpoints from api: %v...", len(CheckpointCache), CheckpointCache[:2])
	}
	return CheckpointCache, nil
}
