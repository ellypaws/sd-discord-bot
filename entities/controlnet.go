package entities

// ControlNetParameters represents the parameters for a ControlNet processing unit.
type ControlNetParameters struct {
	InputImage    string  `json:"input_image"`    // InputImage is the image to use in this unit. Defaults to null.
	Mask          string  `json:"mask"`           // Mask is utilized to filter the image. Defaults to null.
	Module        string  `json:"module"`         // Module is the preprocessor to use on the image for conditioning. Accepts values returned by /controlnet/module_list route. Defaults to "none".
	Model         string  `json:"model"`          // Model is the name of the model for conditioning. Accepts values returned by /controlnet/model_list route. Defaults to "None".
	Weight        int     `json:"weight"`         // Weight of this unit. Defaults to 1.
	ResizeMode    string  `json:"resize_mode"`    // ResizeMode is used to fit the input image to the output resolution. Defaults to "Scale to Fit (Inner Fit)".
	Lowvram       bool    `json:"lowvram"`        // Lowvram indicates if low GPU memory is compensated with processing time. Defaults to false.
	ProcessorRes  int     `json:"processor_res"`  // ProcessorRes is the resolution of the preprocessor. Defaults to 64.
	ThresholdA    int     `json:"threshold_a"`    // ThresholdA is the first parameter of the preprocessor. Effective when preprocessor accepts arguments. Defaults to 64.
	ThresholdB    int     `json:"threshold_b"`    // ThresholdB is the second parameter of the preprocessor, same usage as ThresholdA. Defaults to 64.
	GuidanceStart float64 `json:"guidance_start"` // GuidanceStart is the ratio of generation where this unit starts effecting. Defaults to 0.0.
	GuidanceEnd   float64 `json:"guidance_end"`   // GuidanceEnd is the ratio of generation where this unit stops effecting. Defaults to 1.0.
	ControlMode   string  `json:"control_mode"`   // ControlMode determines the balance between prompt and control model. Defaults to 0. See the related issue for usage details.
	PixelPerfect  bool    `json:"pixel_perfect"`  // PixelPerfect flag enables pixel-perfect preprocessor. Defaults to false.
}
