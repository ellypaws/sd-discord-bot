package entities

// ControlNetParameters represents the parameters for a ControlNet processing unit. Use /controlnet/control_types endpoint to get both the module and model lists.
// By using (*stable_diffusion_api.ControlnetTypes).GetCache or stable_diffusion_api.ControlnetTypesCache to access the endpoint and get the lists of modules and models.
type ControlNetParameters struct {
	InputImage    *string     `json:"input_image,omitempty"`    // InputImage represents the image used in this unit. The default is nil.
	Mask          *string     `json:"mask,omitempty"`           // Mask represents the pixel_perfect filter for the image. The default is nil.
	Module        string      `json:"module,omitempty"`         // Module defines the preprocessor for the image. Defaults to "none". Use /controlnet/module_list in the api
	Model         string      `json:"model,omitempty"`          // Model defines the name of the model for conditioning. Defaults to "None". Use /controlnet/model_list in the api
	Weight        float64     `json:"weight,omitempty"`         // Weight denotes this unit's weight. Defaults to 1.
	ResizeMode    ResizeMode  `json:"resize_mode,omitempty"`    // ResizeMode determines how to resize the input image. Defaults to "Scale to Fit (Inner Fit)".
	Lowvram       bool        `json:"lowvram,omitempty"`        // Lowvram flag indicating if the system should compensate for low GPU memory. Defaults to false.
	ProcessorRes  int         `json:"processor_res,omitempty"`  // ProcessorRes is the resolution for the preprocessor. Defaults to 64.
	ThresholdA    int         `json:"threshold_a,omitempty"`    // ThresholdA is the first parameter of the preprocessor when it accepts arguments. Defaults to 64.
	ThresholdB    int         `json:"threshold_b,omitempty"`    // ThresholdB, like ThresholdA, is a preprocessor parameter and defaults to 64.
	GuidanceStart float64     `json:"guidance_start,omitempty"` // GuidanceStart is the generation ratio at which this unit starts having an impact. Defaults to 0.0.
	GuidanceEnd   float64     `json:"guidance_end,omitempty"`   // GuidanceEnd is the generation ratio at which this unit discontinues its impact. Defaults to 1.0.
	ControlMode   ControlMode `json:"control_mode,omitempty"`   // ControlMode determines the balance between the prompt and control model. Defaults to "Balanced".
	PixelPerfect  bool        `json:"pixel_perfect,omitempty"`  // PixelPerfect flag enables the pixel-perfect preprocessor. Defaults to false.
}

type ControlMode string

const (
	ControlModeBalanced ControlMode = "Balanced"
	ControlModePrompt   ControlMode = "My prompt is more important"
	ControlModeControl  ControlMode = "ControlNet is more important"
)

type ResizeMode string

const (
	ResizeModeJustResize ResizeMode = "Just Resize"
	ResizeModeScaleToFit ResizeMode = "Scale to Fit (Inner Fit)"
	ResizeModeEnvelope   ResizeMode = "Envelope (Outer Fit)"
)

type ControlNet struct {
	Args []*ControlNetParameters `json:"args,omitempty"`
}

func (s *Scripts) NewControlNet() {
	s.ControlNet = &ControlNet{}
}
