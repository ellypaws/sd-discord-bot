package entities

type Scripts struct {
	ADetailer  *ADetailer  `json:"ADetailer,omitempty"`
	ControlNet *ControlNet `json:"controlnet,omitempty"`
}

// Deprecated: use ImageGenerationRequest.NewScripts() instead
func (g *ImageGeneration) NewScripts() {
	g.AlwaysOnScripts = &Scripts{}
}

func (g *ImageGenerationRequest) NewScripts() {
	g.AlwaysonScripts = &Scripts{}
}

type ADetailer struct {
	Args []*ADetailerParameters `json:"args,omitempty"`
}

// Deprecated: use ImageGenerationRequest.NewADetailer() instead
func (g *ImageGeneration) NewADetailer() {
	if g.AlwaysOnScripts == nil {
		g.NewScripts()
	}
	g.AlwaysOnScripts.NewADetailerWithArgs()
}

func (g *ImageGenerationRequest) NewADetailer() {
	if g.TextToImageRequest.AlwaysonScripts == nil {
		g.NewScripts()
	}
	g.TextToImageRequest.AlwaysonScripts.NewADetailerWithArgs()
}

func (s *Scripts) NewADetailerWithArgs() {
	s.ADetailer = &ADetailer{}
	s.ADetailer.CreateArgs()
}

type ControlNet struct {
	Args []*ControlNetParameters `json:"args,omitempty"`
}

func (s *Scripts) NewControlNet() {
	s.ControlNet = &ControlNet{}
}
