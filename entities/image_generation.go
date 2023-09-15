package entities

import (
	"time"
)

type ImageGeneration struct {
	ID                int64                 `json:"id"`
	InteractionID     string                `json:"interaction_id"`
	MessageID         string                `json:"message_id"`
	MemberID          string                `json:"member_id"`
	SortOrder         int                   `json:"sort_order"`
	Prompt            string                `json:"prompt"`
	NegativePrompt    string                `json:"negative_prompt"`
	Width             int                   `json:"width"`
	Height            int                   `json:"height"`
	RestoreFaces      bool                  `json:"restore_faces"`
	EnableHR          bool                  `json:"enable_hr"`
	HRUpscaleRate     float64               `json:"hr_scale"`
	HRUpscaler        string                `json:"hr_upscaler"`
	HiresWidth        int                   `json:"hr_resize_x"`
	HiresHeight       int                   `json:"hr_resize_y"`
	DenoisingStrength float64               `json:"denoising_strength"`
	BatchCount        int                   `json:"batch_count"`
	BatchSize         int                   `json:"batch_size"`
	Seed              int64                 `json:"seed"`
	Subseed           int                   `json:"subseed"`
	SubseedStrength   float64               `json:"subseed_strength"`
	SamplerName       string                `json:"sampler_name"`
	CfgScale          float64               `json:"cfg_scale"`
	Steps             int                   `json:"steps"`
	Processed         bool                  `json:"processed"`
	CreatedAt         time.Time             `json:"created_at"`
	ExtraSDModelName  string                `json:"-"`
	AlwaysonScripts   map[string]*ADetailer `json:"alwayson_scripts"`
}

type ADetailer struct {
	Args []ADetailerParameters `json:"args,omitempty"`
}

type ADetailerParameters struct {
	AdModel                    string  `json:"ad_model,omitempty"`
	AdPrompt                   string  `json:"ad_prompt,omitempty"`
	AdNegativePrompt           string  `json:"ad_negative_prompt,omitempty"`
	AdConfidence               float64 `json:"ad_confidence,omitempty"`
	AdMaskKLargest             int     `json:"ad_mask_k_largest,omitempty"`
	AdMaskMinRatio             float64 `json:"ad_mask_min_ratio,omitempty"`
	AdMaskMaxRatio             float64 `json:"ad_mask_max_ratio,omitempty"`
	AdDilateErode              int     `json:"ad_dilate_erode,omitempty"`
	AdXOffset                  int     `json:"ad_x_offset,omitempty"`
	AdYOffset                  int     `json:"ad_y_offset,omitempty"`
	AdMaskMergeInvert          string  `json:"ad_mask_merge_invert,omitempty"`
	AdMaskBlur                 int     `json:"ad_mask_blur,omitempty"`
	AdDenoisingStrength        float64 `json:"ad_denoising_strength,omitempty"`
	AdInpaintOnlyMasked        bool    `json:"ad_inpaint_only_masked,omitempty"`
	AdInpaintOnlyMaskedPadding int     `json:"ad_inpaint_only_masked_padding,omitempty"`
	AdUseInpaintWidthHeight    bool    `json:"ad_use_inpaint_width_height,omitempty"`
	AdInpaintWidth             int     `json:"ad_inpaint_width,omitempty"`
	AdInpaintHeight            int     `json:"ad_inpaint_height,omitempty"`
	AdUseSteps                 bool    `json:"ad_use_steps,omitempty"`
	AdSteps                    int     `json:"ad_steps,omitempty"`
	AdUseCfgScale              bool    `json:"ad_use_cfg_scale,omitempty"`
	AdCfgScale                 float64 `json:"ad_cfg_scale,omitempty"`
	AdUseSampler               bool    `json:"ad_use_sampler,omitempty"`
	AdSampler                  string  `json:"ad_sampler,omitempty"`
	AdUseNoiseMultiplier       bool    `json:"ad_use_noise_multiplier,omitempty"`
	AdNoiseMultiplier          float64 `json:"ad_noise_multiplier,omitempty"`
	AdUseClipSkip              bool    `json:"ad_use_clip_skip,omitempty"`
	AdClipSkip                 int     `json:"ad_clip_skip,omitempty"`
	AdRestoreFace              bool    `json:"ad_restore_face,omitempty"`
	AdControlnetModel          string  `json:"ad_controlnet_model,omitempty"`
	AdControlnetModule         *string `json:"ad_controlnet_module,omitempty"`
	AdControlnetWeight         float64 `json:"ad_controlnet_weight,omitempty"`
	AdControlnetGuidanceStart  float64 `json:"ad_controlnet_guidance_start,omitempty"`
	AdControlnetGuidanceEnd    float64 `json:"ad_controlnet_guidance_end,omitempty"`
}

// AppendSegModel is a method for the ADetailer struct that takes in
// an ADetailerParameters instance as an argument. It appends the provided
// segmentation model to the existing list of segmentation models (Args)
// maintained within the ADetailer instance. This enables dynamic addition
// of segmentation models to an ADetailer without modifying pre-existing data.
func (detailer *ADetailer) AppendSegModel(parameters ADetailerParameters) {
	detailer.Args = append(detailer.Args, parameters)
}
