package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"strings"
)

type (
	Command       = string
	CommandOption = string
)

var (
	// Command names
	imagineCommand         Command = "imagine"
	imagineSettingsCommand Command = "imagine_settings"
)

const (
	helloCommand   Command = "hello"
	refreshCommand Command = "refresh"
	llmCommand     Command = "llm"
	novelAICommand Command = "novelai"
	rawCommand     Command = Command(handlers.JSONInput)
)

const (
	// Command options
	promptOption       CommandOption = "prompt"
	negativeOption     CommandOption = "negative_prompt"
	samplerOption      CommandOption = "sampler_name"
	aspectRatio        CommandOption = "aspect_ratio"
	loraOption         CommandOption = "lora"
	checkpointOption   CommandOption = "checkpoint"
	vaeOption          CommandOption = "vae"
	hypernetworkOption CommandOption = "hypernetwork"
	embeddingOption    CommandOption = "embedding"
	hiresFixOption     CommandOption = "use_hires_fix"
	hiresFixSize       CommandOption = "hires_fix_size"
	restoreFacesOption CommandOption = "restore_faces"
	adModelOption      CommandOption = "ad_model"
	cfgScaleOption     CommandOption = "cfg_scale"
	stepOption         CommandOption = "step"
	seedOption         CommandOption = "seed"
	batchCountOption   CommandOption = "batch_count"
	batchSizeOption    CommandOption = "batch_size"
	clipSkipOption     CommandOption = "clip_skip"
	cfgRescaleOption   CommandOption = "cfg_rescale"

	systemPromptOption CommandOption = "system_prompt"
	maxTokensOption    CommandOption = "max_tokens"
	llmModelOption     CommandOption = "model" // TODO: Retrieve /v1/models from endpoint

	novelaiModelOption    CommandOption = "model"
	novelaiSizeOption     CommandOption = "size"
	novelaiSamplerOption  CommandOption = "sampler"
	novelaiUCPresetOption CommandOption = "uc_preset"
	novelaiQualityOption  CommandOption = "quality"
	novelaiScheduleOption CommandOption = "schedule"
	novelaiSMEAOption     CommandOption = "smea"
	novelaiSMEADynOption  CommandOption = "smea_dyn"

	novelaiVibeTransfer CommandOption = "vibe_transfer"
	novelaiInformation  CommandOption = "information_extracted"
	novelaiReference    CommandOption = "reference_strength"
	novelaiImg2ImgStr   CommandOption = "img2img_strength"

	img2imgOption   CommandOption = "img2img"
	denoisingOption CommandOption = "denoising"

	refreshLoraOption CommandOption = "refresh_lora"
	refreshCheckpoint CommandOption = "refresh_checkpoint"
	refreshVAEOption  CommandOption = "refresh_vae"
	//refreshHypernetworkOption CommandOption = "refresh_hypernetwork"
	//refreshEmbeddingOption    CommandOption = "refresh_embedding"
	refreshAllOption CommandOption = "refresh_all"

	controlnetImage        CommandOption = "controlnet_image"
	controlnetType         CommandOption = "controlnet_type"
	controlnetControlMode  CommandOption = "controlnet_control_mode"
	controlnetResizeMode   CommandOption = "controlnet_resize_mode"
	controlnetPreprocessor CommandOption = "controlnet_preprocessor"
	controlnetModel        CommandOption = "controlnet_model"

	jsonFile     CommandOption = "json_file"
	useDefaults  CommandOption = "use_defaults"
	unsafeOption CommandOption = "unsafe"

	extraLoras = 2
)

var commands = map[Command]*discordgo.ApplicationCommand{
	helloCommand: {
		Name: helloCommand,
		// All commands and options must have a description
		// Commands/options without description will fail the registration
		// of the command.
		Description: "Say hello to the bot",
		Type:        discordgo.ChatApplicationCommand,
	},
	imagineCommand: {
		Name:        imagineCommand,
		Description: "Ask the bot to imagine something",
		Options:     imagineOptions(),
		Type:        discordgo.ChatApplicationCommand,
	},
	imagineSettingsCommand: {
		Name:        imagineSettingsCommand,
		Description: "Change the default settings for the imagine command",
		Type:        discordgo.ChatApplicationCommand,
	},
	llmCommand: {
		Name:        llmCommand,
		Description: "Ask the bot to generate text using an LLM",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			commandOptions[promptOption],
			commandOptions[systemPromptOption],
			commandOptions[maxTokensOption],
		},
	},
	novelAICommand: {
		Name:        novelAICommand,
		Description: "Ask the bot to imagine something using NovelAI",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			commandOptions[promptOption],
			commandOptions[negativeOption],
			commandOptions[novelaiModelOption],
			commandOptions[novelaiSizeOption],
			commandOptions[novelaiSamplerOption],
			commandOptions[novelaiUCPresetOption],
			commandOptions[novelaiQualityOption],
			commandOptions[seedOption],
			commandOptions[cfgScaleOption],
			//commandOptions[cfgRescaleOption],
			commandOptions[novelaiScheduleOption],
			commandOptions[novelaiVibeTransfer],
			commandOptions[novelaiInformation],
			commandOptions[novelaiReference],
			commandOptions[img2imgOption],
			commandOptions[novelaiImg2ImgStr],
			commandOptions[novelaiSMEAOption],
			commandOptions[novelaiSMEADynOption],
		},
	},
	refreshCommand: {
		Name:        refreshCommand,
		Description: "Refresh the loaded models from the API",
		Options: []*discordgo.ApplicationCommandOption{
			commandOptions[refreshLoraOption],
			commandOptions[refreshCheckpoint],
			commandOptions[refreshVAEOption],
			commandOptions[refreshAllOption],
		},
	},
	rawCommand: {
		Name:        rawCommand,
		Description: "Send a raw json request to the API. ",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			commandOptions[jsonFile],
			commandOptions[useDefaults],
			commandOptions[unsafeOption],
		},
	},
}

func imagineOptions() (options []*discordgo.ApplicationCommandOption) {
	options = []*discordgo.ApplicationCommandOption{
		commandOptions[promptOption],
		commandOptions[negativeOption],
		commandOptions[stepOption],
		commandOptions[seedOption],
		commandOptions[checkpointOption],
		commandOptions[aspectRatio],
		commandOptions[loraOption],
		commandOptions[samplerOption],
		commandOptions[batchCountOption],
		commandOptions[batchSizeOption],
		//commandOptions[hiresFixOption],
		commandOptions[hiresFixSize],
		commandOptions[cfgScaleOption],
		//commandOptions[restoreFacesOption],
		commandOptions[adModelOption],
		commandOptions[vaeOption],
		commandOptions[hypernetworkOption],
		commandOptions[embeddingOption],
		commandOptions[img2imgOption],
		commandOptions[denoisingOption],
		commandOptions[controlnetImage],
		commandOptions[controlnetControlMode],
		commandOptions[controlnetType],
		commandOptions[controlnetResizeMode],
		commandOptions[controlnetPreprocessor],
		commandOptions[controlnetModel],
	}

	for i := 0; i < min(extraLoras, 25-len(options)); i++ {
		if len(options) > 25 {
			log.Printf("Max options reached, skipping extra lora options")
			break
		}
		loraOption := *commandOptions[loraOption]
		loraOption.Name += fmt.Sprintf("%d", i+2)
		options = append(options, &loraOption)
	}

	if len(options) > 25 {
		log.Printf("WARNING: Too many options (%d) for discord. Discord only allows 25 options per command. Some options will be skipped.", len(options))
		options = options[:25]
	}
	return
}

var commandOptions = map[CommandOption]*discordgo.ApplicationCommandOption{
	promptOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        promptOption,
		Description: "The text prompt to imagine",
		Required:    true,
	},
	negativeOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        negativeOption,
		Description: "Negative prompt",
		Required:    false,
	},
	stepOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        stepOption,
		Description: "Number of iterations to sample with. Default is 20",
		Required:    false,
	},
	seedOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        seedOption,
		Description: "Seed to use for sampling. Default is random (-1)",
	},
	checkpointOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         checkpointOption,
		Description:  "The checkpoint to change to when generating. Sets for the next person.",
		Required:     false,
		Autocomplete: true,
	},
	vaeOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         vaeOption,
		Description:  "The vae to use",
		Required:     false,
		Autocomplete: true,
	},
	hypernetworkOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         hypernetworkOption,
		Description:  "The hypernetwork to use",
		Required:     false,
		Autocomplete: true,
	},
	embeddingOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         embeddingOption,
		Description:  "The embedding to use",
		Required:     false,
		Autocomplete: true,
	},
	systemPromptOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        systemPromptOption,
		Description: "The system prompt to generate with",
		Required:    false,
	},
	maxTokensOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        maxTokensOption,
		Description: "The maximum number of tokens to generate. Use -1 for infinite (default: 1024)",
		Required:    false,
	},
	aspectRatio: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        aspectRatio,
		Description: "The aspect ratio to use. Default is 1:1 (note: you can specify your own aspect ratio)",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1:1",
				Value: "1:1",
			},
			{
				Name:  "2:3",
				Value: "2:3",
			},
			{
				Name:  "3:2",
				Value: "3:2",
			},
			{
				Name:  "3:4",
				Value: "3:4",
			},
			{
				Name:  "4:3",
				Value: "4:3",
			},
			{
				Name:  "16:9",
				Value: "16:9",
			},
			{
				Name:  "9:16",
				Value: "9:16",
			},
		},
	},
	loraOption: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         loraOption,
		Description:  "The lora(s) to apply",
		Required:     false,
		Autocomplete: true,
	},
	samplerOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        samplerOption,
		Description: "sampler",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Euler a",
				Value: "Euler a",
			},
			{
				Name:  "DDIM",
				Value: "DDIM",
			},
			{
				Name:  "UniPC",
				Value: "UniPC",
			},
			{
				Name:  "Euler",
				Value: "Euler",
			},
			{
				Name:  "DPM2 a Karras",
				Value: "DPM2 a Karras",
			},
			{
				Name:  "DPM++ 2S a Karras",
				Value: "DPM++ 2S a Karras",
			},
			{
				Name:  "DPM++ 2M Karras",
				Value: "DPM++ 2M Karras",
			},
			{
				Name:  "DPM++ 3M SDE Karras",
				Value: "DPM++ 3M SDE Karras",
			},
		},
	},
	batchCountOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        batchCountOption,
		Description: "Number of batches to generate. Default is 1 and max is 4",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1",
				Value: "1",
			},
			{
				Name:  "2",
				Value: "2",
			},
			{
				Name:  "3",
				Value: "3",
			},
			{
				Name:  "4",
				Value: "4",
			},
		},
	},
	batchSizeOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        batchSizeOption,
		Description: "Number of batches to generate. Default and max is 4",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1",
				Value: "1",
			},
			{
				Name:  "2",
				Value: "2",
			},
			{
				Name:  "3",
				Value: "3",
			},
			{
				Name:  "4",
				Value: "4",
			},
		},
	},
	hiresFixOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        hiresFixOption,
		Description: "use hires.fix or not. default=No for better performance",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Yes",
				Value: "true",
			},
			{
				Name:  "No",
				Value: "false",
			},
		},
	},
	hiresFixSize: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        hiresFixSize,
		Description: "upscale multiplier for hires.fix. default=2",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "1.5",
				Value: "1.5",
			},
			{
				Name:  "2",
				Value: "2",
			},
		},
	},
	cfgScaleOption: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        cfgScaleOption,
		Description: "value for cfg. default=7.0",
		Required:    false,
	},
	restoreFacesOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        restoreFacesOption,
		Description: "Use Codeformer to restore faces",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Yes",
				Value: "true",
			},
			{
				Name:  "No",
				Value: "false",
			},
		},
	},
	adModelOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        adModelOption,
		Description: "The model to use for adetailer",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Face",
				Value: "face_yolov8n.pt",
			},
			{
				Name:  "Body",
				Value: "person_yolov8n-seg.pt",
			},
			{
				Name:  "Both",
				Value: "person_yolov8n-seg.pt,face_yolov8n.pt",
			},
		},
	},
	refreshLoraOption: {
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        strings.TrimPrefix(refreshLoraOption, "refresh_"),
		Description: "Refresh the lora models from the API.",
	},
	refreshCheckpoint: {
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        strings.TrimPrefix(refreshCheckpoint, "refresh_"),
		Description: "Refresh the checkpoint models from the API.",
	},
	refreshVAEOption: {
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        strings.TrimPrefix(refreshVAEOption, "refresh_"),
		Description: "Refresh the vae models from the API.",
	},
	refreshAllOption: {
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        strings.TrimPrefix(refreshAllOption, "refresh_"),
		Description: "Refresh all models from the API.",
	},
	img2imgOption: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        img2imgOption,
		Description: "Attach an image to use as input for img2img",
	},
	denoisingOption: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        denoisingOption,
		Description: "Denoising level for img2img. Default is 0.7",
	},
	controlnetImage: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        controlnetImage,
		Description: "The image to use for controlnet. Img2img is used if not specified",
		Required:    false,
	},
	controlnetType: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        controlnetType,
		Description: "The type of controlnet to use. Default is All",
		Required:    false,
		Choices:     controlTypes(),
	},
	controlnetControlMode: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        controlnetControlMode,
		Description: "The control mode to use for controlnet. Defaults to Balanced",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  string(entities.ControlModeBalanced),
				Value: entities.ControlModeBalanced,
			},
			{
				Name:  string(entities.ControlModePrompt),
				Value: entities.ControlModePrompt,
			},
			{
				Name:  string(entities.ControlModeControl),
				Value: entities.ControlModeControl,
			},
		},
	},
	controlnetResizeMode: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        controlnetResizeMode,
		Description: "The resize mode to use for controlnet. Defaults to Scale to Fit (Inner Fit)",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  string(entities.ResizeModeJustResize),
				Value: entities.ResizeModeJustResize,
			},
			{
				Name:  string(entities.ResizeModeScaleToFit),
				Value: entities.ResizeModeScaleToFit,
			},
			{
				Name:  string(entities.ResizeModeEnvelope),
				Value: entities.ResizeModeEnvelope,
			},
		},
	},
	controlnetPreprocessor: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         controlnetPreprocessor,
		Description:  "The preprocessor to use for controlnet. Set the type to see the available modules. Defaults to None",
		Required:     false,
		Autocomplete: true,
	},
	controlnetModel: {
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         controlnetModel,
		Description:  "The model to use for controlnet. Set the type to see the available models. Defaults to None",
		Required:     false,
		Autocomplete: true,
	},

	jsonFile: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        jsonFile,
		Description: "The json file to use for the raw command. If not specified, a modal will be opened to paste the json",
		Required:    false,
	},
	useDefaults: {
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        useDefaults,
		Description: "Use the default values for the raw command. This is set to True by default",
		Required:    false,
	},
	unsafeOption: {
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        unsafeOption,
		Description: "Process the json file without validation. This is set to False by default",
		Required:    false,
	},

	novelaiModelOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiModelOption,
		Description: "The model to use for NovelAI. Default is V3. Older versions are not recommended.",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "NAI Diffusion Anime V3 (Default)",
				Value: entities.ModelV3,
			},
			{
				Name:  "NAI Diffusion Furry V3",
				Value: entities.ModelFurryV3,
			},
			{
				Name:  "(Old) NAI Diffusion Anime V2",
				Value: entities.ModelV2,
			},
			{
				Name:  "(Old) NAI Diffusion Anime V1 (Full)",
				Value: entities.ModelV1,
			},
			{
				Name:  "(Old) NAI Diffusion Anime V1 (Curated)",
				Value: entities.ModelV1Curated,
			},
			{
				Name:  "(Old) NAI Diffusion Furry",
				Value: entities.ModelFurryV1,
			},
		},
	},

	novelaiSizeOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiSizeOption,
		Description: "The size of the image to generate. Default is Normal Square (1024x1024)",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Small Portrait (512x768)",
				Value: entities.OptionSmallPortrait,
			},
			{
				Name:  "Small Landscape (768x512)",
				Value: entities.OptionSmallLandscape,
			},
			{
				Name:  "Small Square (640x640)",
				Value: entities.OptionSmallSquare,
			},
			{
				Name:  "Normal Portrait (832x1216)",
				Value: entities.OptionNormalPortrait,
			},
			{
				Name:  "Normal Landscape (1216x832)",
				Value: entities.OptionNormalLandscape,
			},
			{
				Name:  "Normal Square (1024x1024)",
				Value: entities.OptionNormalSquare,
			},
			//{
			//	Name:  "Large Portrait (1024x1536)",
			//	Value: entities.OptionLargePortrait,
			//},
			//{
			//	Name:  "Large Landscape (1536x1024)",
			//	Value: entities.OptionLargeLandscape,
			//},
			//{
			//	Name:  "Large Square (1472x1472)",
			//	Value: entities.OptionLargeSquare,
			//},
			//{
			//	Name:  "Wallpaper Portrait (1088x1920)",
			//	Value: entities.OptionWallpaperPortrait,
			//},
			//{
			//	Name:  "Wallpaper Landscape (1920x1088)",
			//	Value: entities.OptionWallpaperLandscape,
			//},
		},
	},

	novelaiSamplerOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiSamplerOption,
		Description: "The method to use for sampling. Default is Euler",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Default (Euler)",
				Value: entities.SamplerDefault,
			},
			{
				Name:  "Euler",
				Value: entities.SamplerEuler,
			},
			{
				Name:  "Euler a",
				Value: entities.SamplerEulerAncestral,
			},
			{
				Name:  "DPM++ 2S Ancestral",
				Value: entities.SamplerDPM2SAncestral,
			},
			{
				Name:  "DPM++ 2M",
				Value: entities.SamplerDPM2M,
			},
			{
				Name:  "DPM++ SDE",
				Value: entities.SamplerDPMSDE,
			},
			{
				Name:  "DDIM",
				Value: entities.SamplerDDIM,
			},
		},
	},

	novelaiUCPresetOption: {
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        novelaiUCPresetOption,
		Description: "The preset to use for Undesired Content",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Heavy",
				Value: 0,
			},
			{
				Name:  "Light",
				Value: 1,
			},
			{
				Name:  "Human Focus",
				Value: 2,
			},
			{
				Name:  "None",
				Value: 3,
			},
		},
	},

	novelaiQualityOption: {
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        novelaiQualityOption,
		Description: "Tags to increase quality will be prepended to the prompt. Default is true",
		Required:    false,
	},

	novelaiScheduleOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiScheduleOption,
		Description: "The scheduler when sampling. Default is native.",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Default (Native)",
				Value: entities.ScheduleDefault,
			},
			{
				Name:  "Native",
				Value: entities.ScheduleNative,
			},
			{
				Name:  "Karras",
				Value: entities.ScheduleKarras,
			},
			{
				Name:  "Exponential",
				Value: entities.ScheduleExponential,
			},
			{
				Name:  "Polyexponential",
				Value: entities.SchedulePolyexponential,
			},
		},
	},

	novelaiSMEAOption: {
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        novelaiSMEAOption,
		Description: "Smea versions of samplers are modified to perform better at high resolutions. Default is off",
		Required:    false,
	},

	novelaiSMEADynOption: {
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        novelaiSMEADynOption,
		Description: "Dyn variants of Smea often lead to more varied output, but may fail at very high resolutions.",
		Required:    false,
	},

	novelaiVibeTransfer: {
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Name:        novelaiVibeTransfer,
		Description: "Attach an image to use as input for vibe transfer",
		Required:    false,
	},
	novelaiInformation: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        novelaiInformation,
		Description: "The amount of information to extract from the image. Default is 1.0",
		Required:    false,
	},
	novelaiReference: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        novelaiReference,
		Description: "The strength of the reference. Default is 0.6",
		Required:    false,
	},
	novelaiImg2ImgStr: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        novelaiImg2ImgStr,
		Description: "The strength of the img2img. Default is 0.7",
		Required:    false,
	},
}

func controlTypes() []*discordgo.ApplicationCommandOptionChoice {
	// ControlType is an alias for string
	type ControlType = string

	// Constants for different control types
	const (
		All          ControlType = "All"
		Canny        ControlType = "Canny"
		Depth        ControlType = "Depth"
		NormalMap    ControlType = "NormalMap"
		OpenPose     ControlType = "OpenPose"
		MLSD         ControlType = "MLSD"
		Lineart      ControlType = "Lineart"
		SoftEdge     ControlType = "SoftEdge"
		Scribble     ControlType = "Scribble/Sketch"
		Segmentation ControlType = "Segmentation"
		Shuffle      ControlType = "Shuffle"
		TileBlur     ControlType = "Tile/Blur"
		Inpaint      ControlType = "Inpaint"
		InstructP2P  ControlType = "InstructP2P"
		Reference    ControlType = "Reference"
		Recolor      ControlType = "Recolor"
		Revision     ControlType = "Revision"
		T2IAdapter   ControlType = "T2I-Adapter"
		IPAdapter    ControlType = "IP-Adapter"
	)

	var ControlTypes = []ControlType{
		All,
		Canny,
		Depth,
		NormalMap,
		OpenPose,
		MLSD,
		Lineart,
		SoftEdge,
		Scribble,
		Segmentation,
		Shuffle,
		TileBlur,
		Inpaint,
		InstructP2P,
		Reference,
		Recolor,
		Revision,
		T2IAdapter,
		IPAdapter,
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, controlType := range ControlTypes {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  controlType,
			Value: controlType,
		})
		if len(choices) >= 25 {
			break
		}
	}

	return choices
}

const (
	maskedUser    = "user"
	maskedChannel = "channel"
	maskedForum   = "threads"
	maskedRole    = "role"
)

var maskedOptions = map[string]*discordgo.ApplicationCommandOption{
	maskedUser: {
		Type:        discordgo.ApplicationCommandOptionUser,
		Name:        maskedUser,
		Description: "Choose a user",
		Required:    false,
	},
	maskedChannel: {
		Type:        discordgo.ApplicationCommandOptionChannel,
		Name:        maskedChannel,
		Description: "Choose a channel to close",
		// Channel type mask
		ChannelTypes: []discordgo.ChannelType{
			discordgo.ChannelTypeGuildText,
			discordgo.ChannelTypeGuildVoice,
		},
		Required: false,
	},
	maskedForum: {
		Type:        discordgo.ApplicationCommandOptionChannel,
		Name:        maskedForum,
		Description: "Choose a thread to mark as solved",
		ChannelTypes: []discordgo.ChannelType{
			discordgo.ChannelTypeGuildForum,
			discordgo.ChannelTypeGuildNewsThread,
			discordgo.ChannelTypeGuildPublicThread,
			discordgo.ChannelTypeGuildPrivateThread,
		},
	},
	maskedRole: {
		Type:        discordgo.ApplicationCommandOptionRole,
		Name:        maskedRole,
		Description: "Choose a role to add",
		Required:    false,
	},
}
