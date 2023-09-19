package entities

// ControlNetParameters represents the parameters for a ControlNet processing unit.
type ControlNetParameters struct {
	InputImage    *string `json:"input_image,omitempty"` // InputImage represents the image used in this unit. The default is nil.
	Mask          *string `json:"mask,omitempty"`        // Mask represents the pixel_perfect filter for the image. The default is nil.
	Module        string  `json:"module,omitempty"`      // Module defines the preprocessor for the image. Defaults to "none".
	Model         string  `json:"model,omitempty"`       // Model defines the name of the model for conditioning. Defaults to "None".
	Weight        int     `json:"weight"`                // Weight denotes this unit's weight. Defaults to 1.
	ResizeMode    string  `json:"resize_mode"`           // ResizeMode determines how to resize the input image. Defaults to "Scale to Fit (Inner Fit)".
	Lowvram       bool    `json:"lowvram"`               // Lowvram flag indicating if the system should compensate for low GPU memory. Defaults to false.
	ProcessorRes  int     `json:"processor_res"`         // ProcessorRes is the resolution for the preprocessor. Defaults to 64.
	ThresholdA    int     `json:"threshold_a"`           // ThresholdA is the first parameter of the preprocessor when it accepts arguments. Defaults to 64.
	ThresholdB    int     `json:"threshold_b"`           // ThresholdB, like ThresholdA, is a preprocessor parameter and defaults to 64.
	GuidanceStart float64 `json:"guidance_start"`        // GuidanceStart is the generation ratio at which this unit starts having an impact. Defaults to 0.0.
	GuidanceEnd   float64 `json:"guidance_end"`          // GuidanceEnd is the generation ratio at which this unit discontinues its impact. Defaults to 1.0.
	ControlMode   string  `json:"control_mode"`          // ControlMode determines the balance between the prompt and control model. Defaults to "Balanced".
	PixelPerfect  bool    `json:"pixel_perfect"`         // PixelPerfect flag enables the pixel-perfect preprocessor. Defaults to false.
}
