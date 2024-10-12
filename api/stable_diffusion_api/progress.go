// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    progress, err := UnmarshalProgress(bytes)
//    bytes, err = progress.Marshal()

package stable_diffusion_api

import "encoding/json"

func UnmarshalProgress(data []byte) (Progress, error) {
	var r Progress
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Progress) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Progress struct {
	// The current image in base64 format. opts.show_progress_every_n_steps is required for this to work.
	CurrentImage *string `json:"current_image,omitempty"`
	EtaRelative  float64 `json:"eta_relative"`
	// The progress with a range of 0 to 1
	Progress float64 `json:"progress"`
	// The current state snapshot
	State State `json:"state"`
	// Info text used by WebUI.
	Textinfo *string `json:"textinfo,omitempty"`
}

type State struct {
	Skipped       bool   `json:"skipped"`
	Interrupted   bool   `json:"interrupted"`
	Job           string `json:"job"`
	JobCount      int64  `json:"job_count"`
	JobTimestamp  string `json:"job_timestamp"`
	JobNo         int64  `json:"job_no"`
	SamplingStep  int64  `json:"sampling_step"`
	SamplingSteps int64  `json:"sampling_steps"`
}

func (api *apiImplementation) GetProgress() (*Progress, error) {
	progress, err := GET[Progress](api.Client(), api.Host("/progress"))
	if err != nil {
		return nil, err
	}
	return progress, nil
}
