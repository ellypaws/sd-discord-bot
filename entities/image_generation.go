package entities

import (
	"encoding/json"
	"fmt"
	"time"
)

type ImageGeneration struct {
	ID                int64     `json:"id"`
	InteractionID     string    `json:"interaction_id"`
	MessageID         string    `json:"message_id"`
	MemberID          string    `json:"member_id"`
	SortOrder         int       `json:"sort_order"`
	Prompt            string    `json:"prompt"`
	NegativePrompt    string    `json:"negative_prompt"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	RestoreFaces      bool      `json:"restore_faces"`
	EnableHR          bool      `json:"enable_hr"`
	HRUpscaleRate     float64   `json:"hr_scale"`
	HRUpscaler        string    `json:"hr_upscaler"`
	HiresWidth        int       `json:"hr_resize_x"`
	HiresHeight       int       `json:"hr_resize_y"`
	DenoisingStrength float64   `json:"denoising_strength"`
	BatchCount        int       `json:"batch_count"`
	BatchSize         int       `json:"batch_size"`
	Seed              int64     `json:"seed"`
	Subseed           int       `json:"subseed"`
	SubseedStrength   float64   `json:"subseed_strength"`
	SamplerName       string    `json:"sampler_name"`
	CfgScale          float64   `json:"cfg_scale"`
	Steps             int       `json:"steps"`
	Processed         bool      `json:"processed"`
	CreatedAt         time.Time `json:"created_at"`
	AlwaysOnScripts   *Scripts  `json:"alwayson_scripts,omitempty"`
	Checkpoint        *string   `json:"checkpoint,omitempty"`
	VAE               *string   `json:"vae,omitempty"`
	Hypernetwork      *string   `json:"hypernetwork,omitempty"`
}

func NewGeneration() *ImageGeneration {
	return &ImageGeneration{}
}

func (g *ImageGeneration) PrintJson() {
	p, _ := json.MarshalIndent(g, "", "    ")
	fmt.Println("generation: ", string(p))
}
