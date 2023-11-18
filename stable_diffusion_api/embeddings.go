// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    embeddingModels, err := UnmarshalEmbeddingModels(bytes)
//    bytes, err = embeddingModels.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
	"log"
)

type EmbeddingModels []Embedding

type Embedding struct {
	Name   string `json:"name"`
	Loaded bool   `json:"loaded"`
	EmbeddingInfo
}

func UnmarshalEmbeddingModels(data []byte) (EmbeddingResponse, error) {
	var r EmbeddingResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *EmbeddingResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type EmbeddingResponse struct {
	Loaded  map[string]EmbeddingInfo `json:"loaded"`
	Skipped map[string]EmbeddingInfo `json:"skipped"`
}

type EmbeddingInfo struct {
	Step             *int64  `json:"step"`
	SDCheckpoint     *string `json:"sd_checkpoint"`
	SDCheckpointName *string `json:"sd_checkpoint_name"`
	Shape            int64   `json:"shape"`
	Vectors          int64   `json:"vectors"`
}

func (c EmbeddingModels) String(i int) string {
	return c[i].Name
}

func (c EmbeddingModels) Len() int {
	return len(c)
}

var EmbeddingCache EmbeddingModels

func (api *apiImplementation) SDEmbeddingCache() (EmbeddingModels, error) {
	if EmbeddingCache != nil {
		return EmbeddingCache, nil
	}
	return api.embeddingApi()
}

func (api *apiImplementation) embeddingApi() (EmbeddingModels, error) {
	getURL := "/sdapi/v1/embeddings"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	embeddingResponse, err := UnmarshalEmbeddingModels(body)
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	for name, embedding := range embeddingResponse.Loaded {
		EmbeddingCache = append(EmbeddingCache, Embedding{
			Name:          name,
			Loaded:        true,
			EmbeddingInfo: embedding,
		})
	}

	for name, embedding := range embeddingResponse.Skipped {
		EmbeddingCache = append(EmbeddingCache, Embedding{
			Name:          name,
			Loaded:        false,
			EmbeddingInfo: embedding,
		})
	}

	return EmbeddingCache, nil
}
