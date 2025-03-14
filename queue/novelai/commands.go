package novelai

import (
	"github.com/bwmarrin/discordgo"

	"stable_diffusion_bot/entities"
)

func (q *NAIQueue) commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        NovelAICommand,
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
				// commandOptions[cfgRescaleOption],
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
	}
}

var commandOptions = map[string]*discordgo.ApplicationCommandOption{
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
	cfgScaleOption: {
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        cfgScaleOption,
		Description: "value for cfg. default=7.0",
		Required:    false,
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

	novelaiModelOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiModelOption,
		Description: "The model to use for NovelAI. Default is V3. Older versions are not recommended.",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "NAI Diffusion Anime V4 (Default)",
				Value: entities.ModelV4Full,
			},
			{
				Name:  "NAI Diffusion Anime V4 Curated Preview",
				Value: entities.ModelV4Preview,
			},
			{
				Name:  "NAI Diffusion Anime V3",
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
			// {
			//	Name:  "Large Portrait (1024x1536)",
			//	Value: entities.OptionLargePortrait,
			// },
			// {
			//	Name:  "Large Landscape (1536x1024)",
			//	Value: entities.OptionLargeLandscape,
			// },
			// {
			//	Name:  "Large Square (1472x1472)",
			//	Value: entities.OptionLargeSquare,
			// },
			// {
			//	Name:  "Wallpaper Portrait (1088x1920)",
			//	Value: entities.OptionWallpaperPortrait,
			// },
			// {
			//	Name:  "Wallpaper Landscape (1920x1088)",
			//	Value: entities.OptionWallpaperLandscape,
			// },
		},
	},

	novelaiSamplerOption: {
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        novelaiSamplerOption,
		Description: "The method to use for sampling. Default is Euler ancestral",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Default (Euler a)",
				Value: entities.SamplerDefault,
			},
			{
				Name:  "Euler",
				Value: entities.SamplerEuler,
			},
			{
				Name:  "Euler ancestral",
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
		Description: "The scheduler when sampling. Default is Karras.",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Default (Karras)",
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
