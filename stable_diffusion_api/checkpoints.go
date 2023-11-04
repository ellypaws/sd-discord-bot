// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    sDModels, err := UnmarshalSDModels(bytes)
//    bytes, err = sDModels.Marshal()

package stable_diffusion_api

import "encoding/json"

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

func (c SDModels) String(i int) string {
	return c[i].Title
}

func (c SDModels) Len() int {
	return len(c)
}
