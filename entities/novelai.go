package entities

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/novelai-metadata/pkg/meta"
	"image"
	"image/png"
	"io"
	"math"
	"math/rand"
	"reflect"
	"stable_diffusion_bot/utils"
)

type NovelAIRequest struct {
	Action     actions    `json:"action,omitempty"`
	Input      string     `json:"input,omitempty"`
	Model      models     `json:"model,omitempty"`
	Parameters Parameters `json:"parameters"`
	URL        string     `json:"url,omitempty"`
}

type Parameters struct {
	// Deprecated: Use NovelAIRequest.Input instead.
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`

	ResolutionPreset *resolutionPreset `json:"resolution_preset,omitempty"`

	Width   int64   `json:"width,omitempty"`
	Height  int64   `json:"height,omitempty"`
	Steps   int64   `json:"steps,omitempty"`
	Seed    int64   `json:"seed,omitempty"`
	Sampler string  `json:"sampler,omitempty"`
	Smea    bool    `json:"sm,omitempty"`     // Smea versions of samplers are modified to perform better at high resolutions.
	SmeaDyn bool    `json:"sm_dyn,omitempty"` // Dyn variants of Smea samplers often lead to more varied output, but may fail at very high resolutions.
	Scale   float64 `json:"scale,omitempty"`  // Prompt guidance, also known as CFG Scale
	// Whether to enable [Decrisper].
	//
	// [Decrisper]: https://docs.novelai.net/image/stepsguidance.html#decrisper
	Decrisper bool `json:"dynamic_thresholding,omitempty"`

	// QualityToggle adds AdditionalPositive to the prompt if set to true.
	QualityToggle bool `json:"qualityToggle,omitempty"`

	// UcPreset aka Undesired Content preset.
	// The presets are as follows:
	// 0: HeavyNegative
	// 1: LightNegative
	// 2: HumanFocusNegative
	// 3: None
	UcPreset *int64 `json:"ucPreset,omitempty"`

	// ImageCount is the number of images to generate.
	// The maximum value is 8 up to 360,448 px, 6 up to 409,600 px.
	ImageCount uint8 `json:"n_samples,omitempty"`

	ExtraNoiseSeed int64    `json:"extra_noise_seed,omitempty"`
	NoiseSchedule  schedule `json:"noise_schedule,omitempty"`

	Img2Img             *async  `json:"image,omitempty"` // used by Img2Img
	Noise               float64 `json:"noise,omitempty"`
	Strength            float64 `json:"strength,omitempty"`
	ControlnetCondition string  `json:"controlnet_condition,omitempty"`
	ControlnetModel     string  `json:"controlnet_model,omitempty"`
	ControlnetStrength  float64 `json:"controlnet_strength,omitempty"`

	// AddOriginalImage is used for inpainting
	AddOriginalImage bool   `json:"add_original_image,omitempty"`
	Mask             string `json:"mask,omitempty"`

	// VibeTransferImage is used for Vibe Transfer
	VibeTransferImage                     *async    `json:"reference_image,omitempty"`
	ReferenceImageMultiple                []*async  `json:"reference_image_multiple,omitempty"`
	ReferenceInformationExtracted         float64   `json:"reference_information_extracted,omitempty"`
	ReferenceInformationExtractedMultiple []float64 `json:"reference_information_extracted_multiple,omitempty"`
	ReferenceStrength                     float64   `json:"reference_strength,omitempty"`
	ReferenceStrengthMultiple             []float64 `json:"reference_strength_multiple,omitempty"`

	ParamsVersion  int64 `json:"params_version,omitempty"`
	Legacy         bool  `json:"legacy,omitempty"`
	LegacyV3Extend bool  `json:"legacy_v3_extend,omitempty"`
}

type NovelAIResponse struct {
	Images []io.Reader `json:"images"`
}

type async = utils.Image

type sampler = string
type schedule = string

const (
	UCHeavy = iota
	UCLight
	UCHumanFocus

	HeavyNegative      = ", lowres, {bad}, error, fewer, extra, missing, worst quality, jpeg artifacts, bad quality, watermark, unfinished, displeasing, chromatic aberration, signature, extra digits, artistic error, username, scan, [abstract]"
	LightNegative      = ", lowres, jpeg artifacts, worst quality, watermark, blurry, very displeasing"
	HumanFocusNegative = ", lowres, {bad}, error, fewer, extra, missing, worst quality, jpeg artifacts, bad quality, watermark, unfinished, displeasing, chromatic aberration, signature, extra digits, artistic error, username, scan, [abstract], bad anatomy, bad hands, @_@, mismatched pupils, heart-shaped pupils, glowing eyes"

	HeavyNegativeFurry = ", {{worst quality}}, [displeasing], {unusual pupils}, guide lines, {{unfinished}}, {bad}, url, artist name, {{tall image}}, mosaic, {sketch page}, comic panel, impact (font), [dated], {logo}, ych, {what}, {where is your god now}, {distorted text}, repeated text, {floating head}, {1994}, {widescreen}, absolutely everyone, sequence, {compression artifacts}, hard translated, {cropped}, {commissioner name}, unknown text, high contrast"
	LightNegativeFurry = ", {worst quality}, guide lines, unfinished, bad, url, tall image, widescreen, compression artifacts, unknown text"

	AdditionalPositive = ", best quality, amazing quality, very aesthetic, absurdres"
)

func UnmarshalNovelAIRequest(data []byte) (NovelAIRequest, error) {
	var r NovelAIRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *NovelAIRequest) Reader() (io.Reader, error) {
	r.Init()

	maxSamples := r.GetMaxNSamples()
	if r.Parameters.ImageCount > maxSamples {
		return nil, &json.MarshalerError{
			Type: reflect.TypeFor[uint8](),
			Err: fmt.Errorf(
				"max value of n_samples is %d under current resolution (%dx%d). Got %d",
				maxSamples,
				r.Parameters.Width,
				r.Parameters.Height,
				r.Parameters.ImageCount,
			),
		}
	}

	if r.Parameters.Width < 64 || r.Parameters.Width > 49152 {
		return nil, fmt.Errorf("width out of range (64-49152): %d", r.Parameters.Width)
	}
	if r.Parameters.Height < 64 || r.Parameters.Height > 49152 {
		return nil, fmt.Errorf("height out of range (64-49152): %d", r.Parameters.Height)
	}
	if r.Parameters.Steps < 1 || r.Parameters.Steps > 50 {
		return nil, fmt.Errorf("steps out of range (1-50): %d", r.Parameters.Steps)
	}
	if r.Parameters.Scale < 0 || r.Parameters.Scale > 10 {
		return nil, fmt.Errorf("scale out of range (0-10): %f", r.Parameters.Scale)
	}
	if r.Parameters.Seed < 0 || r.Parameters.Seed > 4294967295-7 {
		return nil, fmt.Errorf("seed out of range (0-4294967295-7): %d", r.Parameters.Seed)
	}
	if r.Parameters.ExtraNoiseSeed != 0 && (r.Parameters.ExtraNoiseSeed < 0 || r.Parameters.ExtraNoiseSeed > 4294967295-7) {
		return nil, fmt.Errorf("extraNoiseSeed out of range (0-4294967295-7): %d", r.Parameters.ExtraNoiseSeed)
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(r)
	return &buf, err
}

func DefaultNovelAIRequest() *NovelAIRequest {
	uc := int64(2)
	return &NovelAIRequest{
		Action: ActionGenerate,
		Model:  ModelV4Full,
		Parameters: Parameters{
			ResolutionPreset: &ResolutionNormalSquare,
			Steps:            28,
			Scale:            7.0,
			Smea:             false,
			QualityToggle:    true,
			UcPreset:         &uc,
			ImageCount:       1,
			Sampler:          SamplerDefault,
			NoiseSchedule:    ScheduleDefault,
			Seed:             rand.Int63n(4294967295 - 7),

			ReferenceInformationExtracted: 1.0,
			ReferenceStrength:             0.6,
			Strength:                      0.7,
		},
	}
}

func (r *NovelAIRequest) Init() {
	if r.Parameters.ResolutionPreset != nil {
		r.Parameters.Width = r.Parameters.ResolutionPreset[0]
		r.Parameters.Height = r.Parameters.ResolutionPreset[1]
		r.Parameters.ResolutionPreset = nil
	}

	if r.Input == "" {
		r.Input = r.Parameters.Prompt
		r.Parameters.Prompt = ""
	}

	if r.Parameters.QualityToggle {
		r.Input += AdditionalPositive
	}

	if r.Parameters.Seed <= 0 {
		r.Parameters.Seed = rand.Int63n(4294967295 - 7)
	}

	if r.Parameters.UcPreset != nil {
		switch r.Model {
		case ModelFurryV3, MovelFurryV3Inp:
			switch *r.Parameters.UcPreset {
			case UCHeavy:
				r.Parameters.NegativePrompt += HeavyNegativeFurry
			case UCLight:
				r.Parameters.NegativePrompt += LightNegativeFurry
			case UCHumanFocus:
			default:
			}
		case ModelV4Full, ModelV4Preview, ModelV3, ModelV3Inp:
			fallthrough
		default:
			switch *r.Parameters.UcPreset {
			case UCHeavy:
				r.Parameters.NegativePrompt += HeavyNegative
			case UCLight:
				r.Parameters.NegativePrompt += LightNegative
			case UCHumanFocus:
				r.Parameters.NegativePrompt += HumanFocusNegative
			default:
			}
		}
	}

	if r.Parameters.VibeTransferImage != nil {
		ifUnset(&r.Parameters.ReferenceInformationExtracted, 1.0)
		ifUnset(&r.Parameters.ReferenceStrength, 0.6)
	}

	if r.Parameters.Img2Img != nil {
		ifUnset(&r.Parameters.Strength, 0.7)
	}
}

func ifUnset[T interface{ ~float64 | int }](a *T, b T) {
	if a == nil {
		return
	}

	if *a == 0 {
		*a = b
	}
}

func (r *NovelAIRequest) GetMaxNSamples() uint8 {
	w, h := r.Parameters.Width, r.Parameters.Height

	if w*h <= 512*704 {
		return 8
	}

	if w*h <= 640*640 {
		return 6
	}

	if w*h <= 1024*3072 {
		return 4
	}

	return 0
}

type models = string

const (
	ModelV3         models = "nai-diffusion-3"
	ModelV4Preview  models = "nai-diffusion-4-curated-preview"
	ModelV4Full     models = "nai-diffusion-4-full"
	ModelV3Inp      models = "nai-diffusion-3-inpainting"
	ModelFurryV3    models = "nai-diffusion-furry-3"
	MovelFurryV3Inp models = "nai-diffusion-furry-3-inpainting"
	ModelFurryV1    models = "nai-diffusion-furry"
	ModelV1         models = "nai-diffusion"
	ModelV1Curated  models = "safe-diffusion"
	ModelV2         models = "nai-diffusion-2"
)

type actions = string

const (
	ActionGenerate actions = "generate"
	ActionInpaint  actions = "infill"
	ActionImg2Img  actions = "img2img"
)

// Image represents either a base64 encoded image, an image.Image, or an io.Reader.
// Deprecated: use utils.Image
type Image struct {
	Base64   *string
	Image    *image.Image
	Reader   io.Reader
	Metadata *meta.Metadata
}

func (b *Image) UnmarshalJSON(data []byte) error {
	d := string(bytes.Trim(data, `"`))
	b.Base64 = &d

	reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))
	img, _, err := image.Decode(reader)
	if err != nil {
		return err
	}
	b.Image = &img
	return nil
}

func (b *Image) MarshalJSON() ([]byte, error) {
	if b.Reader != nil {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, b.Reader)
		if err != nil {
			return nil, err
		}

		var bin bytes.Buffer
		err = json.NewEncoder(&bin).Encode(buf.String())
		if err != nil {
			return nil, err
		}

		return bin.Bytes(), nil
	}
	if b.Base64 != nil {
		return json.Marshal(b.Base64)
	}
	if b.Image == nil {
		return []byte(`null`), nil
	}

	buf := new(bytes.Buffer)
	b64Writer := base64.NewEncoder(base64.StdEncoding, buf)
	err := png.Encode(b64Writer, *b.Image)
	if err != nil {
		return nil, err
	}

	return json.Marshal(buf.String())
}

func (b *Image) ImageBytes(w io.Writer) error {
	if b.Image == nil {
		return errors.New("no image data")
	}
	return png.Encode(w, *b.Image)
}

const (
	SamplerDefault = SamplerEuler // Euler

	SamplerEuler          sampler = "k_euler"              // Euler
	SamplerEulerAncestral sampler = "k_euler_ancestral"    // Euler Ancestral
	SamplerDPM2SAncestral sampler = "k_dpmpp_2s_ancestral" // DPM++ 2S Ancestral
	SamplerDPM2M          sampler = "k_dpmpp_2m"           // DPM++ 2M
	SamplerDPMSDE         sampler = "k_dpmpp_sde"          // DPM++ SDE
	SamplerDDIM           sampler = "ddim"                 // DDIM
)

const (
	ScheduleDefault = ScheduleNative

	ScheduleNative          schedule = "native"
	ScheduleKarras          schedule = "karras"
	ScheduleExponential     schedule = "exponential"
	SchedulePolyexponential schedule = "polyexponential"
)

type resolutionPreset [2]int64

var (
	ResolutionSmallPortrait      resolutionPreset = [2]int64{512, 768}
	ResolutionSmallLandscape     resolutionPreset = [2]int64{768, 512}
	ResolutionSmallSquare        resolutionPreset = [2]int64{640, 640}
	ResolutionNormalPortrait     resolutionPreset = [2]int64{832, 1216}
	ResolutionNormalLandscape    resolutionPreset = [2]int64{1216, 832}
	ResolutionNormalSquare       resolutionPreset = [2]int64{1024, 1024}
	ResolutionLargePortrait      resolutionPreset = [2]int64{1024, 1536}
	ResolutionLargeLandscape     resolutionPreset = [2]int64{1536, 1024}
	ResolutionLargeSquare        resolutionPreset = [2]int64{1472, 1472}
	ResolutionWallpaperPortrait  resolutionPreset = [2]int64{1088, 1920}
	ResolutionWallpaperLandscape resolutionPreset = [2]int64{1920, 1088}
)

const (
	OptionSmallPortrait      = "Small Portrait"
	OptionSmallLandscape     = "Small Landscape"
	OptionSmallSquare        = "Small Square"
	OptionNormalPortrait     = "Normal Portrait"
	OptionNormalLandscape    = "Normal Landscape"
	OptionNormalSquare       = "Normal Square"
	OptionLargePortrait      = "Large Portrait"
	OptionLargeLandscape     = "Large Landscape"
	OptionLargeSquare        = "Large Square"
	OptionWallpaperPortrait  = "Wallpaper Portrait"
	OptionWallpaperLandscape = "Wallpaper Landscape"
)

func GetDimensions(option string) resolutionPreset {
	switch option {
	case OptionSmallPortrait:
		return ResolutionSmallPortrait
	case OptionSmallLandscape:
		return ResolutionSmallLandscape
	case OptionSmallSquare:
		return ResolutionSmallSquare
	case OptionNormalPortrait:
		return ResolutionNormalPortrait
	case OptionNormalLandscape:
		return ResolutionNormalLandscape
	case OptionNormalSquare:
		return ResolutionNormalSquare
	case OptionLargePortrait:
		return ResolutionLargePortrait
	case OptionLargeLandscape:
		return ResolutionLargeLandscape
	case OptionLargeSquare:
		return ResolutionLargeSquare
	case OptionWallpaperPortrait:
		return ResolutionWallpaperPortrait
	case OptionWallpaperLandscape:
		return ResolutionWallpaperLandscape
	default:
		return ResolutionNormalSquare
	}
}
func (r *NovelAIRequest) CalculateCost(opus bool) int64 {
	steps := r.Parameters.Steps
	nSamples := r.Parameters.ImageCount
	uncondScale := r.Parameters.Scale
	strength := 1.0
	if r.Action == ActionImg2Img {
		strength = r.Parameters.Strength
	}
	smeaFactor := 1.0
	if r.Parameters.SmeaDyn {
		smeaFactor = 1.4
	} else if r.Parameters.Smea {
		smeaFactor = 1.2
	}

	resolution := max(r.Parameters.Width*r.Parameters.Height, 65536)
	if resolution > ResolutionNormalPortrait[0]*ResolutionNormalPortrait[1] && resolution <= ResolutionNormalSquare[0]*ResolutionNormalSquare[1] {
		resolution = ResolutionNormalPortrait[0] * ResolutionNormalPortrait[1]
	}

	perSample := int64(math.Ceil(2.951823174884865e-6*float64(resolution) + 5.753298233447344e-7*float64(resolution)*float64(steps)))
	perSample = int64(math.Max(math.Ceil(float64(perSample)*strength*smeaFactor), 2))

	if uncondScale != 1.0 {
		perSample = int64(math.Ceil(float64(perSample) * 1.3))
	}

	if opus && steps <= 28 && resolution <= ResolutionNormalSquare[0]*ResolutionNormalSquare[1] {
		nSamples -= 1
	}
	return perSample * int64(max(1, nSamples))
}
