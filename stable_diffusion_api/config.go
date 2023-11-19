// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    aPIConfig, err := UnmarshalAPIConfig(bytes)
//    bytes, err = aPIConfig.Marshal()

package stable_diffusion_api

import (
	"encoding/json"
)

func UnmarshalAPIConfig(data []byte) (APIConfig, error) {
	var r APIConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (config *APIConfig) Marshal() ([]byte, error) {
	return json.Marshal(config)
}

type APIConfig struct {
	SamplesSave                           bool     `json:"samples_save"`
	SamplesFormat                         string   `json:"samples_format"`
	SamplesFilenamePattern                string   `json:"samples_filename_pattern"`
	SaveImagesAddNumber                   bool     `json:"save_images_add_number"`
	SaveImagesReplaceAction               string   `json:"save_images_replace_action"`
	GridSave                              bool     `json:"grid_save"`
	GridFormat                            string   `json:"grid_format"`
	GridExtendedFilename                  bool     `json:"grid_extended_filename"`
	GridOnlyIfMultiple                    bool     `json:"grid_only_if_multiple"`
	GridPreventEmptySpots                 bool     `json:"grid_prevent_empty_spots"`
	GridZipFilenamePattern                string   `json:"grid_zip_filename_pattern"`
	NRows                                 float64  `json:"n_rows"`
	Font                                  string   `json:"font"`
	GridTextActiveColor                   string   `json:"grid_text_active_color"`
	GridTextInactiveColor                 string   `json:"grid_text_inactive_color"`
	GridBackgroundColor                   string   `json:"grid_background_color"`
	EnablePnginfo                         bool     `json:"enable_pnginfo"`
	SaveTxt                               bool     `json:"save_txt"`
	SaveImagesBeforeFaceRestoration       bool     `json:"save_images_before_face_restoration"`
	SaveImagesBeforeHighresFix            bool     `json:"save_images_before_highres_fix"`
	SaveImagesBeforeColorCorrection       bool     `json:"save_images_before_color_correction"`
	SaveMask                              bool     `json:"save_mask"`
	SaveMaskComposite                     bool     `json:"save_mask_composite"`
	JPEGQuality                           float64  `json:"jpeg_quality"`
	WebpLossless                          bool     `json:"webp_lossless"`
	ExportFor4Chan                        bool     `json:"export_for_4chan"`
	ImgDownscaleThreshold                 float64  `json:"img_downscale_threshold"`
	TargetSideLength                      float64  `json:"target_side_length"`
	ImgMaxSizeMp                          float64  `json:"img_max_size_mp"`
	UseOriginalNameBatch                  bool     `json:"use_original_name_batch"`
	UseUpscalerNameAsSuffix               bool     `json:"use_upscaler_name_as_suffix"`
	SaveSelectedOnly                      bool     `json:"save_selected_only"`
	SaveInitImg                           bool     `json:"save_init_img"`
	TempDir                               string   `json:"temp_dir"`
	CleanTempDirAtStart                   bool     `json:"clean_temp_dir_at_start"`
	SaveIncompleteImages                  bool     `json:"save_incomplete_images"`
	NotificationAudio                     bool     `json:"notification_audio"`
	NotificationVolume                    float64  `json:"notification_volume"`
	OutdirSamples                         string   `json:"outdir_samples"`
	OutdirTxt2ImgSamples                  string   `json:"outdir_txt2img_samples"`
	OutdirImg2ImgSamples                  string   `json:"outdir_img2img_samples"`
	OutdirExtrasSamples                   string   `json:"outdir_extras_samples"`
	OutdirGrids                           string   `json:"outdir_grids"`
	OutdirTxt2ImgGrids                    string   `json:"outdir_txt2img_grids"`
	OutdirImg2ImgGrids                    string   `json:"outdir_img2img_grids"`
	OutdirSave                            string   `json:"outdir_save"`
	OutdirInitImages                      string   `json:"outdir_init_images"`
	SaveToDirs                            bool     `json:"save_to_dirs"`
	GridSaveToDirs                        bool     `json:"grid_save_to_dirs"`
	UseSaveToDirsForUI                    bool     `json:"use_save_to_dirs_for_ui"`
	DirectoriesFilenamePattern            string   `json:"directories_filename_pattern"`
	DirectoriesMaxPromptWords             float64  `json:"directories_max_prompt_words"`
	ESRGANTile                            float64  `json:"ESRGAN_tile"`
	ESRGANTileOverlap                     float64  `json:"ESRGAN_tile_overlap"`
	RealesrganEnabledModels               []string `json:"realesrgan_enabled_models"`
	UpscalerForImg2Img                    string   `json:"upscaler_for_img2img"`
	FaceRestoration                       bool     `json:"face_restoration"`
	FaceRestorationModel                  string   `json:"face_restoration_model"`
	CodeFormerWeight                      float64  `json:"code_former_weight"`
	FaceRestorationUnload                 bool     `json:"face_restoration_unload"`
	AutoLaunchBrowser                     string   `json:"auto_launch_browser"`
	EnableConsolePrompts                  bool     `json:"enable_console_prompts"`
	ShowWarnings                          bool     `json:"show_warnings"`
	ShowGradioDeprecationWarnings         bool     `json:"show_gradio_deprecation_warnings"`
	MemmonPollRate                        float64  `json:"memmon_poll_rate"`
	SamplesLogStdout                      bool     `json:"samples_log_stdout"`
	MultipleTqdm                          bool     `json:"multiple_tqdm"`
	PrintHypernetExtra                    bool     `json:"print_hypernet_extra"`
	ListHiddenFiles                       bool     `json:"list_hidden_files"`
	DisableMmapLoadSafetensors            bool     `json:"disable_mmap_load_safetensors"`
	HideLdmPrints                         bool     `json:"hide_ldm_prints"`
	DumpStacksOnSignal                    bool     `json:"dump_stacks_on_signal"`
	APIEnableRequests                     bool     `json:"api_enable_requests"`
	APIForbidLocalRequests                bool     `json:"api_forbid_local_requests"`
	APIUseragent                          string   `json:"api_useragent"`
	UnloadModelsWhenTraining              bool     `json:"unload_models_when_training"`
	PinMemory                             bool     `json:"pin_memory"`
	SaveOptimizerState                    bool     `json:"save_optimizer_state"`
	SaveTrainingSettingsToTxt             bool     `json:"save_training_settings_to_txt"`
	DatasetFilenameWordRegex              string   `json:"dataset_filename_word_regex"`
	DatasetFilenameJoinString             string   `json:"dataset_filename_join_string"`
	TrainingImageRepeatsPerEpoch          float64  `json:"training_image_repeats_per_epoch"`
	TrainingWriteCSVEvery                 float64  `json:"training_write_csv_every"`
	TrainingXattentionOptimizations       bool     `json:"training_xattention_optimizations"`
	TrainingEnableTensorboard             bool     `json:"training_enable_tensorboard"`
	TrainingTensorboardSaveImages         bool     `json:"training_tensorboard_save_images"`
	TrainingTensorboardFlushEvery         float64  `json:"training_tensorboard_flush_every"`
	SDModelCheckpoint                     string   `json:"sd_model_checkpoint"`
	SDCheckpointsLimit                    float64  `json:"sd_checkpoints_limit"`
	SDCheckpointsKeepInCPU                bool     `json:"sd_checkpoints_keep_in_cpu"`
	SDCheckpointCache                     float64  `json:"sd_checkpoint_cache"`
	SDUnet                                string   `json:"sd_unet"`
	EnableQuantization                    bool     `json:"enable_quantization"`
	EnableEmphasis                        bool     `json:"enable_emphasis"`
	EnableBatchSeeds                      bool     `json:"enable_batch_seeds"`
	CommaPaddingBacktrack                 float64  `json:"comma_padding_backtrack"`
	CLIPStopAtLastLayers                  float64  `json:"CLIP_stop_at_last_layers"`
	UpcastAttn                            bool     `json:"upcast_attn"`
	RandnSource                           string   `json:"randn_source"`
	Tiling                                bool     `json:"tiling"`
	HiresFixRefinerPass                   string   `json:"hires_fix_refiner_pass"`
	SdxlCropTop                           float64  `json:"sdxl_crop_top"`
	SdxlCropLeft                          float64  `json:"sdxl_crop_left"`
	SdxlRefinerLowAestheticScore          float64  `json:"sdxl_refiner_low_aesthetic_score"`
	SdxlRefinerHighAestheticScore         float64  `json:"sdxl_refiner_high_aesthetic_score"`
	SDVaeExplanation                      string   `json:"sd_vae_explanation"`
	SDVaeCheckpointCache                  float64  `json:"sd_vae_checkpoint_cache"`
	SDVae                                 string   `json:"sd_vae"`
	SDVaeOverridesPerModelPreferences     bool     `json:"sd_vae_overrides_per_model_preferences"`
	AutoVaePrecision                      bool     `json:"auto_vae_precision"`
	SDVaeEncodeMethod                     string   `json:"sd_vae_encode_method"`
	SDVaeDecodeMethod                     string   `json:"sd_vae_decode_method"`
	InpaintingMaskWeight                  float64  `json:"inpainting_mask_weight"`
	InitialNoiseMultiplier                float64  `json:"initial_noise_multiplier"`
	Img2ImgExtraNoise                     float64  `json:"img2img_extra_noise"`
	Img2ImgColorCorrection                bool     `json:"img2img_color_correction"`
	Img2ImgFixSteps                       bool     `json:"img2img_fix_steps"`
	Img2ImgBackgroundColor                string   `json:"img2img_background_color"`
	Img2ImgEditorHeight                   float64  `json:"img2img_editor_height"`
	Img2ImgSketchDefaultBrushColor        string   `json:"img2img_sketch_default_brush_color"`
	Img2ImgInpaintMaskBrushColor          string   `json:"img2img_inpaint_mask_brush_color"`
	Img2ImgInpaintSketchDefaultBrushColor string   `json:"img2img_inpaint_sketch_default_brush_color"`
	ReturnMask                            bool     `json:"return_mask"`
	ReturnMaskComposite                   bool     `json:"return_mask_composite"`
	CrossAttentionOptimization            string   `json:"cross_attention_optimization"`
	SMinUncond                            float64  `json:"s_min_uncond"`
	TokenMergingRatio                     float64  `json:"token_merging_ratio"`
	TokenMergingRatioImg2Img              float64  `json:"token_merging_ratio_img2img"`
	TokenMergingRatioHr                   float64  `json:"token_merging_ratio_hr"`
	PadCondUncond                         bool     `json:"pad_cond_uncond"`
	PersistentCondCache                   bool     `json:"persistent_cond_cache"`
	BatchCondUncond                       bool     `json:"batch_cond_uncond"`
	UseOldEmphasisImplementation          bool     `json:"use_old_emphasis_implementation"`
	UseOldKarrasSchedulerSigmas           bool     `json:"use_old_karras_scheduler_sigmas"`
	NoDpmppSdeBatchDeterminism            bool     `json:"no_dpmpp_sde_batch_determinism"`
	UseOldHiresFixWidthHeight             bool     `json:"use_old_hires_fix_width_height"`
	DontFixSecondOrderSamplersSchedule    bool     `json:"dont_fix_second_order_samplers_schedule"`
	HiresFixUseFirstpassConds             bool     `json:"hires_fix_use_firstpass_conds"`
	UseOldScheduling                      bool     `json:"use_old_scheduling"`
	InterrogateKeepModelsInMemory         bool     `json:"interrogate_keep_models_in_memory"`
	InterrogateReturnRanks                bool     `json:"interrogate_return_ranks"`
	InterrogateClipNumBeams               float64  `json:"interrogate_clip_num_beams"`
	InterrogateClipMinLength              float64  `json:"interrogate_clip_min_length"`
	InterrogateClipMaxLength              float64  `json:"interrogate_clip_max_length"`
	InterrogateClipDictLimit              float64  `json:"interrogate_clip_dict_limit"`
	InterrogateClipSkipCategories         []string `json:"interrogate_clip_skip_categories"`
	InterrogateDeepbooruScoreThreshold    float64  `json:"interrogate_deepbooru_score_threshold"`
	DeepbooruSortAlpha                    bool     `json:"deepbooru_sort_alpha"`
	DeepbooruUseSpaces                    bool     `json:"deepbooru_use_spaces"`
	DeepbooruEscape                       bool     `json:"deepbooru_escape"`
	DeepbooruFilterTags                   string   `json:"deepbooru_filter_tags"`
	ExtraNetworksShowHiddenDirectories    bool     `json:"extra_networks_show_hidden_directories"`
	ExtraNetworksHiddenModels             string   `json:"extra_networks_hidden_models"`
	ExtraNetworksDefaultMultiplier        float64  `json:"extra_networks_default_multiplier"`
	ExtraNetworksCardWidth                float64  `json:"extra_networks_card_width"`
	ExtraNetworksCardHeight               float64  `json:"extra_networks_card_height"`
	ExtraNetworksCardTextScale            float64  `json:"extra_networks_card_text_scale"`
	ExtraNetworksCardShowDesc             bool     `json:"extra_networks_card_show_desc"`
	ExtraNetworksCardOrderField           string   `json:"extra_networks_card_order_field"`
	ExtraNetworksCardOrder                string   `json:"extra_networks_card_order"`
	ExtraNetworksAddTextSeparator         string   `json:"extra_networks_add_text_separator"`
	UIExtraNetworksTabReorder             string   `json:"ui_extra_networks_tab_reorder"`
	TextualInversionPrintAtLoad           bool     `json:"textual_inversion_print_at_load"`
	TextualInversionAddHashesToInfotext   bool     `json:"textual_inversion_add_hashes_to_infotext"`
	SDHypernetwork                        string   `json:"sd_hypernetwork"`
	Localization                          string   `json:"localization"`
	GradioTheme                           string   `json:"gradio_theme"`
	GradioThemesCache                     bool     `json:"gradio_themes_cache"`
	GalleryHeight                         string   `json:"gallery_height"`
	ReturnGrid                            bool     `json:"return_grid"`
	DoNotShowImages                       bool     `json:"do_not_show_images"`
	SendSeed                              bool     `json:"send_seed"`
	SendSize                              bool     `json:"send_size"`
	JSModalLightbox                       bool     `json:"js_modal_lightbox"`
	JSModalLightboxInitiallyZoomed        bool     `json:"js_modal_lightbox_initially_zoomed"`
	JSModalLightboxGamepad                bool     `json:"js_modal_lightbox_gamepad"`
	JSModalLightboxGamepadRepeat          float64  `json:"js_modal_lightbox_gamepad_repeat"`
	ShowProgressInTitle                   bool     `json:"show_progress_in_title"`
	SamplersInDropdown                    bool     `json:"samplers_in_dropdown"`
	DimensionsAndBatchTogether            bool     `json:"dimensions_and_batch_together"`
	KeyeditPrecisionAttention             float64  `json:"keyedit_precision_attention"`
	KeyeditPrecisionExtra                 float64  `json:"keyedit_precision_extra"`
	KeyeditDelimiters                     string   `json:"keyedit_delimiters"`
	KeyeditDelimitersWhitespace           []string `json:"keyedit_delimiters_whitespace"`
	KeyeditMove                           bool     `json:"keyedit_move"`
	QuicksettingsList                     []string `json:"quicksettings_list"`
	UITabOrder                            []string `json:"ui_tab_order"`
	HiddenTabs                            []string `json:"hidden_tabs"`
	UIReorderList                         []string `json:"ui_reorder_list"`
	SDCheckpointDropdownUseShort          bool     `json:"sd_checkpoint_dropdown_use_short"`
	HiresFixShowSampler                   bool     `json:"hires_fix_show_sampler"`
	HiresFixShowPrompts                   bool     `json:"hires_fix_show_prompts"`
	DisableTokenCounters                  bool     `json:"disable_token_counters"`
	CompactPromptBox                      bool     `json:"compact_prompt_box"`
	AddModelHashToInfo                    bool     `json:"add_model_hash_to_info"`
	AddModelNameToInfo                    bool     `json:"add_model_name_to_info"`
	AddUserNameToInfo                     bool     `json:"add_user_name_to_info"`
	AddVersionToInfotext                  bool     `json:"add_version_to_infotext"`
	DisableWeightsAutoSwap                bool     `json:"disable_weights_auto_swap"`
	InfotextStyles                        string   `json:"infotext_styles"`
	ShowProgressbar                       bool     `json:"show_progressbar"`
	LivePreviewsEnable                    bool     `json:"live_previews_enable"`
	LivePreviewsImageFormat               string   `json:"live_previews_image_format"`
	ShowProgressGrid                      bool     `json:"show_progress_grid"`
	ShowProgressEveryNSteps               float64  `json:"show_progress_every_n_steps"`
	ShowProgressType                      string   `json:"show_progress_type"`
	LivePreviewAllowLowvramFull           bool     `json:"live_preview_allow_lowvram_full"`
	LivePreviewContent                    string   `json:"live_preview_content"`
	LivePreviewRefreshPeriod              float64  `json:"live_preview_refresh_period"`
	LivePreviewFastInterrupt              bool     `json:"live_preview_fast_interrupt"`
	HideSamplers                          []string `json:"hide_samplers"`
	EtaDdim                               float64  `json:"eta_ddim"`
	EtaAncestral                          float64  `json:"eta_ancestral"`
	DdimDiscretize                        string   `json:"ddim_discretize"`
	SChurn                                float64  `json:"s_churn"`
	STmin                                 float64  `json:"s_tmin"`
	STmax                                 float64  `json:"s_tmax"`
	SNoise                                float64  `json:"s_noise"`
	KSchedType                            string   `json:"k_sched_type"`
	SigmaMin                              float64  `json:"sigma_min"`
	SigmaMax                              float64  `json:"sigma_max"`
	Rho                                   float64  `json:"rho"`
	EtaNoiseSeedDelta                     float64  `json:"eta_noise_seed_delta"`
	AlwaysDiscardNextToLastSigma          bool     `json:"always_discard_next_to_last_sigma"`
	SgmNoiseMultiplier                    bool     `json:"sgm_noise_multiplier"`
	UniPCVariant                          string   `json:"uni_pc_variant"`
	UniPCSkipType                         string   `json:"uni_pc_skip_type"`
	UniPCOrder                            float64  `json:"uni_pc_order"`
	UniPCLowerOrderFinal                  bool     `json:"uni_pc_lower_order_final"`
	PostprocessingEnableInMainUI          []string `json:"postprocessing_enable_in_main_ui"`
	PostprocessingOperationOrder          []string `json:"postprocessing_operation_order"`
	UpscalingMaxImagesInCache             float64  `json:"upscaling_max_images_in_cache"`
	DisabledExtensions                    []string `json:"disabled_extensions"`
	DisableAllExtensions                  string   `json:"disable_all_extensions"`
	RestoreConfigStateFile                string   `json:"restore_config_state_file"`
	SDCheckpointHash                      string   `json:"sd_checkpoint_hash"`
}

func (api *apiImplementation) GetConfig() (*APIConfig, error) {
	getURL := "/sdapi/v1/options"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	var apiConfig APIConfig
	apiConfig, err = UnmarshalAPIConfig(body)
	if err != nil {
		return nil, err
	}

	return &apiConfig, nil
}

func (api *apiImplementation) GetCheckpoint() (string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return "", err
	}

	return apiConfig.SDModelCheckpoint, nil
}

func (api *apiImplementation) GetVAE() (string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return "", err
	}

	return apiConfig.SDVae, nil
}

func (api *apiImplementation) GetHypernetwork() (string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return "", err
	}

	return apiConfig.SDHypernetwork, nil
}
