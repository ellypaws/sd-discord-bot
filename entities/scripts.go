package entities

type Scripts struct {
	ADetailer  *ADetailer  `json:"ADetailer,omitempty"`
	ControlNet *ControlNet `json:"controlnet,omitempty"`
}

func (g *ImageGeneration) NewScripts() {
	g.AlwaysOnScripts = &Scripts{}
}

type ADetailer struct {
	Args []*ADetailerParameters `json:"args,omitempty"`
}

func (g *ImageGeneration) NewADetailer() {
	if g.AlwaysOnScripts == nil {
		g.NewScripts()
	}
	g.AlwaysOnScripts.NewADetailerWithArgs()
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
