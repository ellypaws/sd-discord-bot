// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    loraModels, err := UnmarshalLoraModels(bytes)
//    bytes, err = loraModels.Marshal()

package stable_diffusion_api

import (
	"bytes"
	"log"
	"regexp"
)
import "errors"
import "encoding/json"

type LoraModels []LoraModel

func UnmarshalLoraModels(data []byte) (LoraModels, error) {
	var r LoraModels
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *LoraModels) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type LoraModel struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
	Path  string `json:"path"`
	//Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	SsSDModelName                *string                     `json:"ss_sd_model_name,omitempty"`
	SsResolution                 *string                     `json:"ss_resolution,omitempty"`
	SsClipSkip                   *string                     `json:"ss_clip_skip,omitempty"`
	SsNumTrainImages             *string                     `json:"ss_num_train_images,omitempty"`
	SsTagFrequency               map[string]map[string]int64 `json:"ss_tag_frequency,omitempty"`
	SsBatchSizePerDevice         *string                     `json:"ss_batch_size_per_device,omitempty"`
	SsBucketInfo                 *SsBucketInfoUnion          `json:"ss_bucket_info"`
	SsCacheLatents               *SsBucketNoUpscale          `json:"ss_cache_latents,omitempty"`
	SsColorAug                   *SsBucketNoUpscale          `json:"ss_color_aug,omitempty"`
	SsDatasetDirs                map[string]SsDatasetDir     `json:"ss_dataset_dirs,omitempty"`
	SsEnableBucket               *SsBucketNoUpscale          `json:"ss_enable_bucket,omitempty"`
	SsEpoch                      *string                     `json:"ss_epoch,omitempty"`
	SsFlipAug                    *SsBucketNoUpscale          `json:"ss_flip_aug,omitempty"`
	SsFullFp16                   *SsBucketNoUpscale          `json:"ss_full_fp16,omitempty"`
	SsGradientAccumulationSteps  *string                     `json:"ss_gradient_accumulation_steps,omitempty"`
	SsGradientCheckpointing      *SsBucketNoUpscale          `json:"ss_gradient_checkpointing,omitempty"`
	SsKeepTokens                 *string                     `json:"ss_keep_tokens,omitempty"`
	SsLearningRate               *string                     `json:"ss_learning_rate,omitempty"`
	SsLrScheduler                *SsLrScheduler              `json:"ss_lr_scheduler,omitempty"`
	SsLrWarmupSteps              *string                     `json:"ss_lr_warmup_steps,omitempty"`
	SsMaxBucketReso              *string                     `json:"ss_max_bucket_reso,omitempty"`
	SsMaxTokenLength             *string                     `json:"ss_max_token_length,omitempty"`
	SsMaxTrainSteps              *string                     `json:"ss_max_train_steps,omitempty"`
	SsMinBucketReso              *string                     `json:"ss_min_bucket_reso,omitempty"`
	SsMixedPrecision             *SsMixedPrecision           `json:"ss_mixed_precision,omitempty"`
	SsNetworkAlpha               *string                     `json:"ss_network_alpha,omitempty"`
	SsNetworkDim                 *string                     `json:"ss_network_dim,omitempty"`
	SsNetworkModule              *SsNetworkModule            `json:"ss_network_module,omitempty"`
	SsNewSDModelHash             *string                     `json:"ss_new_sd_model_hash,omitempty"`
	SsNoiseOffset                *SsNetworkDropout           `json:"ss_noise_offset,omitempty"`
	SsNumBatchesPerEpoch         *string                     `json:"ss_num_batches_per_epoch,omitempty"`
	SsNumEpochs                  *string                     `json:"ss_num_epochs,omitempty"`
	SsNumRegImages               *string                     `json:"ss_num_reg_images,omitempty"`
	SsOptimizer                  *string                     `json:"ss_optimizer,omitempty"`
	SsOutputName                 *string                     `json:"ss_output_name,omitempty"`
	SsRandomCrop                 *SsBucketNoUpscale          `json:"ss_random_crop,omitempty"`
	SsRegDatasetDirs             map[string]SsDatasetDir     `json:"ss_reg_dataset_dirs,omitempty"`
	SsSDModelHash                *string                     `json:"ss_sd_model_hash,omitempty"`
	SsSDScriptsCommitHash        *string                     `json:"ss_sd_scripts_commit_hash,omitempty"`
	SsSeed                       *string                     `json:"ss_seed,omitempty"`
	SsSessionID                  *string                     `json:"ss_session_id,omitempty"`
	SsShuffleCaption             *SsBucketNoUpscale          `json:"ss_shuffle_caption,omitempty"`
	SsTextEncoderLr              *string                     `json:"ss_text_encoder_lr,omitempty"`
	SsTotalBatchSize             *string                     `json:"ss_total_batch_size,omitempty"`
	SsTrainingComment            *string                     `json:"ss_training_comment,omitempty"`
	SsTrainingStartedAt          *string                     `json:"ss_training_started_at,omitempty"`
	SsUnetLr                     *string                     `json:"ss_unet_lr,omitempty"`
	SsV2                         *SsBucketNoUpscale          `json:"ss_v2,omitempty"`
	SshsLegacyHash               *string                     `json:"sshs_legacy_hash,omitempty"`
	SshsModelHash                *string                     `json:"sshs_model_hash,omitempty"`
	SsmdAuthor                   *string                     `json:"ssmd_author,omitempty"`
	SsmdDescription              *string                     `json:"ssmd_description,omitempty"`
	SsmdDisplayName              *string                     `json:"ssmd_display_name,omitempty"`
	SsmdKeywords                 *string                     `json:"ssmd_keywords,omitempty"`
	SsmdRating                   *string                     `json:"ssmd_rating,omitempty"`
	SsmdSource                   *string                     `json:"ssmd_source,omitempty"`
	SsmdTags                     *SsmdTags                   `json:"ssmd_tags,omitempty"`
	SsBucketNoUpscale            *SsBucketNoUpscale          `json:"ss_bucket_no_upscale,omitempty"`
	SsCaptionDropoutEveryNEpochs *string                     `json:"ss_caption_dropout_every_n_epochs,omitempty"`
	SsCaptionDropoutRate         *string                     `json:"ss_caption_dropout_rate,omitempty"`
	SsCaptionTagDropoutRate      *string                     `json:"ss_caption_tag_dropout_rate,omitempty"`
	SsFaceCropAugRange           *Ss                         `json:"ss_face_crop_aug_range,omitempty"`
	SsLowram                     *SsBucketNoUpscale          `json:"ss_lowram,omitempty"`
	SsMaxGradNorm                *string                     `json:"ss_max_grad_norm,omitempty"`
	SsPriorLossWeight            *string                     `json:"ss_prior_loss_weight,omitempty"`
	SsTrainingFinishedAt         *string                     `json:"ss_training_finished_at,omitempty"`
	SsNetworkDropout             *SsNetworkDropout           `json:"ss_network_dropout,omitempty"`
	SsMultiresNoiseDiscount      *string                     `json:"ss_multires_noise_discount,omitempty"`
	SsBaseModelVersion           *SsBaseModelVersion         `json:"ss_base_model_version,omitempty"`
	SsScaleWeightNorms           *SsScaleWeightNorms         `json:"ss_scale_weight_norms,omitempty"`
	SsMultiresNoiseIterations    *string                     `json:"ss_multires_noise_iterations,omitempty"`
	SsSteps                      *string                     `json:"ss_steps,omitempty"`
	SsMinSnrGamma                *string                     `json:"ss_min_snr_gamma,omitempty"`
	SsAdaptiveNoiseScale         *Ss                         `json:"ss_adaptive_noise_scale,omitempty"`
	SsDatasets                   *string                     `json:"ss_datasets,omitempty"`
	SsNewVaeHash                 *SsNewVaeHash               `json:"ss_new_vae_hash,omitempty"`
	SsVaeHash                    *SsVaeHash                  `json:"ss_vae_hash,omitempty"`
	SsVaeName                    *string                     `json:"ss_vae_name,omitempty"`
	SsNetworkArgs                *SsNetworkArgs              `json:"ss_network_args,omitempty"`
	SsZeroTerminalSnr            *SsBucketNoUpscale          `json:"ss_zero_terminal_snr,omitempty"`
	ModelspecArchitecture        *string                     `json:"modelspec.architecture,omitempty"`
	ModelspecResolution          *string                     `json:"modelspec.resolution,omitempty"`
	ModelspecImplementation      *string                     `json:"modelspec.implementation,omitempty"`
	ModelspecSaiModelSpec        *string                     `json:"modelspec.sai_model_spec,omitempty"`
	ModelspecTitle               *string                     `json:"modelspec.title,omitempty"`
	ModelspecPredictionType      *string                     `json:"modelspec.prediction_type,omitempty"`
	ModelspecDate                *string                     `json:"modelspec.date,omitempty"`
	ModelspecEncoderLayer        *string                     `json:"modelspec.encoder_layer,omitempty"`
	SsIPNoiseGamma               *Ss                         `json:"ss_ip_noise_gamma,omitempty"`
	Format                       *string                     `json:"format,omitempty"`
	LoraKeyEncoding              *string                     `json:"lora_key_encoding,omitempty"`
	LoraTeRank                   *string                     `json:"lora_te_rank,omitempty"`
	LoraUnetRank                 *string                     `json:"lora_unet_rank,omitempty"`
	TextEncoder                  *string                     `json:"text_encoder,omitempty"`
	Unet                         *string                     `json:"unet,omitempty"`
}

type SsBucketInfoClass struct {
	Buckets        map[string]Bucket `json:"buckets"`
	MeanImgArError float64           `json:"mean_img_ar_error"`
}

type Bucket struct {
	Resolution []int64 `json:"resolution"`
	Count      int64   `json:"count"`
}

type SsDatasetDir struct {
	NRepeats int64 `json:"n_repeats"`
	ImgCount int64 `json:"img_count"`
}

type SsNetworkArgs struct {
	ConvDim              *string            `json:"conv_dim,omitempty"`
	ConvAlpha            *string            `json:"conv_alpha,omitempty"`
	Algo                 *Algo              `json:"algo,omitempty"`
	UseCp                *SsBucketNoUpscale `json:"use_cp,omitempty"`
	RankDropout          *SsNetworkDropout  `json:"rank_dropout,omitempty"`
	ModuleDropout        *SsNetworkDropout  `json:"module_dropout,omitempty"`
	Dropout              *string            `json:"dropout,omitempty"`
	TrainOnInput         *SsBucketNoUpscale `json:"train_on_input,omitempty"`
	UseConvCp            *SsBucketNoUpscale `json:"use_conv_cp,omitempty"`
	DownLrWeight         *string            `json:"down_lr_weight,omitempty"`
	MidLrWeight          *string            `json:"mid_lr_weight,omitempty"`
	UpLrWeight           *string            `json:"up_lr_weight,omitempty"`
	BlockLrZeroThreshold *string            `json:"block_lr_zero_threshold,omitempty"`
	DisableConvCp        *SsBucketNoUpscale `json:"disable_conv_cp,omitempty"`
}

type Ss string

const (
	SsNone    Ss = "None"
	The000357 Ss = "0.00357"
	The0005   Ss = "0.005"
)

type SsBaseModelVersion string

const (
	SDV1        SsBaseModelVersion = "sd_v1"
	SDV1V       SsBaseModelVersion = "sd_v1_v"
	SdxlBaseV09 SsBaseModelVersion = "sdxl_base_v0-9"
	SdxlBaseV10 SsBaseModelVersion = "sdxl_base_v1-0"
)

type SsBucketInfoEnum string

const (
	Null SsBucketInfoEnum = "null"
)

type SsBucketNoUpscale string

const (
	SsBucketNoUpscaleFalse SsBucketNoUpscale = "False"
	True                   SsBucketNoUpscale = "True"
)

type SsLrScheduler string

const (
	Adafactor00001     SsLrScheduler = "adafactor:0.0001"
	Adafactor00009     SsLrScheduler = "adafactor:0.0009"
	Adafactor0001      SsLrScheduler = "adafactor:0.001"
	Adafactor5E05      SsLrScheduler = "adafactor:5e-05"
	Constant           SsLrScheduler = "constant"
	ConstantWithWarmup SsLrScheduler = "constant_with_warmup"
	Cosine             SsLrScheduler = "cosine"
	CosineWithRestarts SsLrScheduler = "cosine_with_restarts"
	Linear             SsLrScheduler = "linear"
	Polynomial         SsLrScheduler = "polynomial"
)

type SsMixedPrecision string

const (
	Bf16 SsMixedPrecision = "bf16"
	Fp16 SsMixedPrecision = "fp16"
	No   SsMixedPrecision = "no"
)

type Algo string

const (
	Ia3   Algo = "ia3"
	Locon Algo = "locon"
	Loha  Algo = "loha"
	Lora  Algo = "lora"
)

type SsNetworkDropout string

const (
	SsNetworkDropoutNone SsNetworkDropout = "None"
	The00                SsNetworkDropout = "0.0"
	The00357             SsNetworkDropout = "0.0357"
	The005               SsNetworkDropout = "0.05"
	The008               SsNetworkDropout = "0.08"
	The01                SsNetworkDropout = "0.1"
	The015               SsNetworkDropout = "0.15"
	The03                SsNetworkDropout = "0.3"
)

type SsNetworkModule string

const (
	LoconLoconKohya       SsNetworkModule = "locon.locon_kohya"
	LycorisKohya          SsNetworkModule = "lycoris.kohya"
	NetworksLora          SsNetworkModule = "networks.lora"
	SDScriptsNetworksLora SsNetworkModule = "sd_scripts.networks.lora"
)

type SsNewVaeHash string

const (
	C6A580B13A5Bc05A5E16E4Dbb80608Ff2Ec251A162311590C1F34C013D7F3Dab    SsNewVaeHash = "c6a580b13a5bc05a5e16e4dbb80608ff2ec251a162311590c1f34c013d7f3dab"
	F921Fb3F29891D2A77A6571E56B8B5052420D2884129517A333C60B1B4816Cdf    SsNewVaeHash = "f921fb3f29891d2a77a6571e56b8b5052420d2884129517a333c60b1b4816cdf"
	The2F11C4A99Ddc28D0Ad8Bce0Acc38Bed310B45D38A3Fe4Bb367Dc30F3Ef1A4868 SsNewVaeHash = "2f11c4a99ddc28d0ad8bce0acc38bed310b45d38a3fe4bb367dc30f3ef1a4868"
	The63Aeecb90Ff7Bc1C115395962D3E803571385B61938377Bc7089B36E81E92E2E SsNewVaeHash = "63aeecb90ff7bc1c115395962d3e803571385b61938377bc7089b36e81e92e2e"
)

type SsScaleWeightNorms string

const (
	SsScaleWeightNormsFalse SsScaleWeightNorms = "False"
	SsScaleWeightNormsNone  SsScaleWeightNorms = "None"
	The10                   SsScaleWeightNorms = "1.0"
)

type SsVaeHash string

const (
	D636E597    SsVaeHash = "d636e597"
	F458B5C6    SsVaeHash = "f458b5c6"
	The223531C6 SsVaeHash = "223531c6"
	The975B2546 SsVaeHash = "975b2546"
)

type SsmdTags string

const (
	ConceptNsfw SsmdTags = "concept, nsfw"
	Empty       SsmdTags = ""
	Style       SsmdTags = "style"
)

type SsBucketInfoUnion struct {
	Enum              *SsBucketInfoEnum
	SsBucketInfoClass *SsBucketInfoClass
}

func (x *SsBucketInfoUnion) UnmarshalJSON(data []byte) error {
	x.SsBucketInfoClass = nil
	x.Enum = nil
	var c SsBucketInfoClass
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.SsBucketInfoClass = &c
	}
	return nil
}

func (x *SsBucketInfoUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.SsBucketInfoClass != nil, x.SsBucketInfoClass, false, nil, x.Enum != nil, x.Enum, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")

}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}

// String uses the path if it's available to append the folder the lora is located in
func (c LoraModels) String(i int) string {

	regExp := regexp.MustCompile(`(?:models\\)?Lora\\(.*)`)

	alias := regExp.FindStringSubmatch(c[i].Path)

	var nameToUse string
	switch {
	case alias != nil && alias[1] != "":
		// replace double slash with single slash
		regExp := regexp.MustCompile(`\\{2,}`)
		nameToUse = regExp.ReplaceAllString(alias[1], `\`)
	default:
		nameToUse = c[i].Name
	}

	return nameToUse
}

func (c LoraModels) Len() int {
	return len(c)
}

var LoraCache *LoraModels

// GetCache returns var LoraCache *LoraModels as a Cacheable. Assert using cache.(*LoraModels)
func (c LoraModels) GetCache(api StableDiffusionAPI) (Cacheable, error) {
	if LoraCache != nil {
		return LoraCache, nil
	}
	return c.apiGET(api)
}

func (c LoraModels) apiGET(api StableDiffusionAPI) (Cacheable, error) {
	getURL := "/sdapi/v1/loras"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	cache, err := UnmarshalLoraModels(body)
	LoraCache = &cache
	if err != nil {
		log.Printf("API URL: %s", getURL)
		log.Printf("Unexpected API response: %s", string(body))

		return nil, err
	}

	return LoraCache, nil
}

func (api *apiImplementation) SDLorasCache() (*LoraModels, error) {
	cache, err := LoraCache.GetCache(api)
	return cache.(*LoraModels), err
}
