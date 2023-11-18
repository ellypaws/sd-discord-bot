// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    hypernetworkModels, err := UnmarshalHypernetworkModels(bytes)
//    bytes, err = hypernetworkModels.Marshal()

package stable_diffusion_api

import "encoding/json"

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

var HypernetworkCache HypernetworkModels

func (api *apiImplementation) SDHypernetworkCache() (HypernetworkModels, error) {
	if HypernetworkCache != nil {
		return HypernetworkCache, nil
	}
	return api.hypernetworkApi()
}

func (api *apiImplementation) hypernetworkApi() (HypernetworkModels, error) {
	getURL := "/sdapi/v1/hypernetworks"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	HypernetworkCache, err = UnmarshalHypernetworkModels(body)
	if err != nil {
		return nil, err
	}

	return HypernetworkCache, nil
}
