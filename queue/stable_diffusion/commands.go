package stable_diffusion

import (
	"cmp"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/entities"
)

func (q *SDQueue) commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        ImagineCommand,
			Description: "Ask the bot to imagine something",
			Options:     imagineOptions(),
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        ImagineSettingsCommand,
			Description: "Change the default settings for the imagine command",
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        RefreshCommand,
			Description: "Refresh the loaded models from the API",
			Options: []*discordgo.ApplicationCommandOption{
				commandOptions[refreshLoraOption],
				commandOptions[refreshCheckpoint],
				commandOptions[refreshVAEOption],
				commandOptions[refreshAllOption],
			},
		},
		{
			Name:        RawCommand,
			Description: "Send a raw json request to the API. ",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				commandOptions[jsonFile],
				commandOptions[useDefaults],
				commandOptions[unsafeOption],
			},
		},
	}
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
		// commandOptions[hiresFixOption],
		commandOptions[hiresFixSize],
		commandOptions[cfgScaleOption],
		// commandOptions[restoreFacesOption],
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
				Name:  entities.ControlModeBalanced,
				Value: entities.ControlModeBalanced,
			},
			{
				Name:  entities.ControlModePrompt,
				Value: entities.ControlModePrompt,
			},
			{
				Name:  entities.ControlModeControl,
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
				Name:  entities.ResizeModeJustResize,
				Value: entities.ResizeModeJustResize,
			},
			{
				Name:  entities.ResizeModeScaleToFit,
				Value: entities.ResizeModeScaleToFit,
			},
			{
				Name:  entities.ResizeModeEnvelope,
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

// Deprecated: If we want to dynamically update the controlnet types, we can do it here
func (q *SDQueue) controlnetTypes() {
	if false {
		controlnet, err := stable_diffusion_api.ControlnetTypesCache.GetCache(q.stableDiffusionAPI)
		if err != nil {
			log.Printf("Error getting controlnet types: %v", err)
			panic(err)
		}
		// modify the choices of controlnetType by using the controlnetTypes cache
		var keys map[string]bool = make(map[string]bool)
		for key := range controlnet.(*stable_diffusion_api.ControlnetTypes).ControlTypes {
			if keys[key] {
				continue
			}
			keys[key] = true

			commandOptions[controlnetType].Choices = append(commandOptions[controlnetType].Choices,
				&discordgo.ApplicationCommandOptionChoice{
					Name:  key,
					Value: key,
				})
			if len(commandOptions[controlnetType].Choices) >= 25 {
				break
			}
		}
		sort.Slice(commandOptions[controlnetType].Choices, func(i, j int) bool {
			return cmp.Less(commandOptions[controlnetType].Choices[i].Name, commandOptions[controlnetType].Choices[j].Name)
		})
	}
}
