package stable_diffusion

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"stable_diffusion_bot/api/stable_diffusion_api"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/queue"
	"stable_diffusion_bot/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
)

const (
	ImagineCommand         Command = "imagine"
	ImagineSettingsCommand Command = "imagine_settings"
	RefreshCommand         Command = "refresh"
	RawCommand             Command = JSONInput
)

const (
	// Command options
	promptOption       = "prompt"
	negativeOption     = "negative_prompt"
	samplerOption      = "sampler_name"
	aspectRatio        = "aspect_ratio"
	loraOption         = "lora"
	checkpointOption   = "checkpoint"
	vaeOption          = "vae"
	hypernetworkOption = "hypernetwork"
	embeddingOption    = "embedding"
	hiresFixOption     = "use_hires_fix"
	hiresFixSize       = "hires_fix_size"
	restoreFacesOption = "restore_faces"
	adModelOption      = "ad_model"
	cfgScaleOption     = "cfg_scale"
	stepOption         = "step"
	seedOption         = "seed"
	batchCountOption   = "batch_count"
	batchSizeOption    = "batch_size"
	clipSkipOption     = "clip_skip"
	cfgRescaleOption   = "cfg_rescale"

	img2imgOption   = "img2img"
	denoisingOption = "denoising"

	refreshLoraOption = "refresh_lora"
	refreshCheckpoint = "refresh_checkpoint"
	refreshVAEOption  = "refresh_vae"
	//refreshHypernetworkOption CommandOption = "refresh_hypernetwork"
	//refreshEmbeddingOption    CommandOption = "refresh_embedding"
	refreshAllOption = "refresh_all"

	controlnetImage        = "controlnet_image"
	controlnetType         = "controlnet_type"
	controlnetControlMode  = "controlnet_control_mode"
	controlnetResizeMode   = "controlnet_resize_mode"
	controlnetPreprocessor = "controlnet_preprocessor"
	controlnetModel        = "controlnet_model"

	jsonFile     = "json_file"
	useDefaults  = "use_defaults"
	unsafeOption = "unsafe"

	extraLoras = 2
)

func (q *SDQueue) handlers() map[discordgo.InteractionType]map[string]queue.Handler {
	return queue.CommandHandlers{
		discordgo.InteractionApplicationCommand: {
			ImagineCommand:         q.processImagineCommand,
			ImagineSettingsCommand: q.processImagineSettingsCommand,
			RefreshCommand:         q.processRefreshCommand,
			RawCommand:             q.processRawCommand,
		},
		discordgo.InteractionApplicationCommandAutocomplete: {
			ImagineCommand: q.processImagineAutocomplete,
		},
		discordgo.InteractionModalSubmit: {
			RawCommand: q.processRawModal,
		},
	}
}

func (q *SDQueue) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	optionMap := utils.GetOpts(i.ApplicationCommandData())

	var position int
	var item *SDQueueItem

	if option, ok := optionMap[promptOption]; !ok {
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a prompt.")
	} else {
		parameters, sanitized := utils.ExtractKeyValuePairsFromPrompt(option.StringValue())
		item = q.NewItem(i.Interaction, WithPrompt(sanitized))
		item.Type = ItemTypeImagine

		if _, ok := interfaceConvertAuto[string, string](&item.NegativePrompt, negativeOption, optionMap, parameters); ok {
			item.NegativePrompt = strings.ReplaceAll(item.NegativePrompt, "{DEFAULT}", DefaultNegative)
		}

		interfaceConvertAuto[string, string](&item.SamplerName, samplerOption, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[int, float64](&item.Steps, stepOption, optionMap, parameters); ok {
			item.Steps = int(*floatVal)
		}

		if floatVal, ok := interfaceConvertAuto[int64, float64](&item.Seed, seedOption, optionMap, parameters); ok {
			item.Seed = int64(*floatVal)
		}

		if boolVal, ok := interfaceConvertAuto[bool, string](&item.RestoreFaces, restoreFacesOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				item.RestoreFaces = boolean
			}
		}

		interfaceConvertAuto[string, string](&item.ADetailerString, adModelOption, optionMap, parameters)

		if config, err := q.stableDiffusionAPI.GetConfig(); err != nil {
			_ = handlers.ErrorEdit(s, i.Interaction, "Error retrieving config.", err)
		} else {
			item.Checkpoint = config.SDModelCheckpoint
			item.VAE = config.SDVae
			item.Hypernetwork = config.SDHypernetwork
		}

		interfaceConvertAuto[string, string](item.Checkpoint, checkpointOption, optionMap, parameters)
		interfaceConvertAuto[string, string](item.VAE, vaeOption, optionMap, parameters)
		interfaceConvertAuto[string, string](item.Hypernetwork, hypernetworkOption, optionMap, parameters)

		if option, ok := optionMap[embeddingOption]; ok {
			item.Prompt += " " + option.StringValue()
			log.Printf("Adding embedding: %v", option.StringValue())
		}

		for i := 0; i < extraLoras+1; i++ {
			loraKey := loraOption
			if i != 0 {
				loraKey += fmt.Sprintf("%d", i+1)
			}

			if option, ok := optionMap[loraKey]; ok {
				loraValue := option.StringValue()
				if loraValue != "" {

					loraValue = sanitizeTooltip(loraValue)

					// add :1 if no strength is specified
					strength := regexp.MustCompile(`:([\d.]+)$`)
					if !strength.MatchString(loraValue) {
						loraValue += ":1"
					}
					re := regexp.MustCompile(`.+\\|\.safetensors`)
					loraValue = re.ReplaceAllString(loraValue, "")
					lora := ", <lora:" + loraValue + ">"
					log.Println("Adding lora: ", lora)
					item.Prompt += lora
				}
			}
		}

		interfaceConvertAuto[string, string](&item.AspectRatio, aspectRatio, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[float64, string](&item.HrScale, hiresFixSize, optionMap, parameters); ok {
			float, err := strconv.ParseFloat(*floatVal, 64)
			if err != nil {
				log.Printf("Error parsing hiresUpscaleRate: %v", err)
			} else {
				item.HrScale = between(float, 1.0, 4.0)
				item.EnableHr = true
			}
		}

		if boolVal, ok := interfaceConvertAuto[bool, string](&item.EnableHr, hiresFixOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing hiresFix value: %v.", err)
			} else {
				item.EnableHr = boolean
			}
		}

		interfaceConvertAuto[float64, float64](&item.CFGScale, cfgScaleOption, optionMap, parameters)

		// calculate batch count and batch size. prefer batch size to be the bigger number, both numbers should add up to 4.
		// if batch size is 4, then batch count should be 1. if both are 4, set batch size to 4 and batch count to 1.
		// if batch size is 1, then batch count *can* be 4, but it can also be 1.

		if floatVal, ok := interfaceConvertAuto[int, float64](&item.NIter, batchCountOption, optionMap, parameters); ok {
			item.NIter = int(*floatVal)
		}

		if intVal, ok := interfaceConvertAuto[int, float64](&item.BatchSize, batchSizeOption, optionMap, parameters); ok {
			item.BatchSize = int(*intVal)
		}

		const maxImages = 4
		item.BatchSize = between(item.BatchSize, 1, maxImages)
		item.NIter = min(maxImages/item.BatchSize, item.NIter)

		if boolVal, ok := interfaceConvertAuto[bool, string](&item.RestoreFaces, restoreFacesOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				item.RestoreFaces = boolean
			}
		}

		attachments, err := utils.GetAttachments(i)
		if err != nil {
			return handlers.ErrorEdit(s, i.Interaction, "Error getting attachments.", err)
		}

		if option, ok := optionMap[img2imgOption]; ok {
			if attachment, ok := attachments[option.Value.(string)]; !ok {
				return handlers.ErrorEdit(s, i.Interaction, "You need to provide an image to img2img.")
			} else {
				item.Type = ItemTypeImg2Img

				item.Img2ImgItem.Image = attachment.Image

				if option, ok := optionMap[denoisingOption]; ok {
					item.TextToImageRequest.DenoisingStrength = option.FloatValue()
					item.Img2ImgItem.DenoisingStrength = option.FloatValue()
				}
			}
		}

		if option, ok := optionMap[controlnetImage]; ok {
			if attachment, ok := attachments[option.Value.(string)]; ok {
				item.ControlnetItem.Image = attachment.Image
			} else {
				return handlers.ErrorEdit(s, i.Interaction, "You need to provide an image to controlnet.")
			}
			item.ControlnetItem.Enabled = true
		}

		if controlVal, ok := interfaceConvertAuto[entities.ControlMode, string](&item.ControlnetItem.ControlMode, controlnetControlMode, optionMap, parameters); ok {
			item.ControlnetItem.ControlMode = entities.ControlMode(*controlVal)
			item.ControlnetItem.Enabled = true
		}

		if resizeVal, ok := interfaceConvertAuto[entities.ResizeMode, string](&item.ControlnetItem.ResizeMode, controlnetResizeMode, optionMap, parameters); ok {
			item.ControlnetItem.ResizeMode = entities.ResizeMode(*resizeVal)
			item.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&item.ControlnetItem.Type, controlnetType, optionMap, parameters); ok {
			log.Printf("Controlnet type: %v", item.ControlnetItem.Type)
			cache, err := stable_diffusion_api.ControlnetTypesCache.GetCache(q.stableDiffusionAPI)
			if err != nil {
				log.Printf("Error retrieving controlnet types cache: %v", err)
			} else {
				// set default preprocessor and model
				if types, ok := cache.(*stable_diffusion_api.ControlnetTypes).ControlTypes[item.ControlnetItem.Type]; ok {
					item.ControlnetItem.Preprocessor = types.DefaultOption
					item.ControlnetItem.Model = types.DefaultModel
				}
			}
			item.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&item.ControlnetItem.Preprocessor, controlnetPreprocessor, optionMap, parameters); ok {
			//queue.ControlnetItem.Preprocessor = *preprocessor
			item.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&item.ControlnetItem.Model, controlnetModel, optionMap, parameters); ok {
			//queue.ControlnetItem.Model = *model
			item.ControlnetItem.Enabled = true
		}

		interfaceConvertAuto[float64, float64](&item.OverrideSettings.CLIPStopAtLastLayers, clipSkipOption, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[float64, float64](nil, cfgRescaleOption, optionMap, parameters); ok {
			item.CFGRescale = &entities.CFGRescale{
				Args: entities.CFGRescaleParameters{
					CfgRescale:   *floatVal,
					AutoColorFix: false,
					FixStrength:  0,
					KeepOriginal: false,
				},
			}
		}

		position, err = q.Add(item)
		if err != nil {
			return handlers.ErrorEdit(s, i.Interaction, "Error adding imagine to queue.", err)
		}
	}

	var snowflake string

	switch {
	case i.Member != nil:
		snowflake = i.Member.User.ID
	case i.User != nil:
		snowflake = i.User.ID
	}

	queueString := fmt.Sprintf(
		"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \n```\n%s\n```",
		position,
		snowflake,
		item.Prompt,
	)

	message, err := handlers.EditInteractionResponse(s, i.Interaction, queueString, handlers.Components[handlers.Cancel])
	if err != nil {
		return err
	}
	if item.DiscordInteraction != nil && item.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting message ID for interaction %v", item.DiscordInteraction.ID)
		item.DiscordInteraction.Message = message
	}

	return nil
}

var weightRegex = regexp.MustCompile(`.+\\|\.(?:safetensors|ckpt|pth?)|(:[\d.]+$)`)

func (q *SDQueue) processImagineAutocomplete(_ *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData()
	log.Printf("running autocomplete handler")
	for optionIndex, opt := range data.Options {
		if !opt.Focused {
			continue
		}
		log.Printf("Focused option (%v): %v", optionIndex, opt.Name)

		if strings.HasPrefix(opt.Name, loraOption) {
			return q.autocompleteLora(i, opt)
		}
		switch opt.Name {
		case checkpointOption:
			return q.autocompleteModels(i, opt, stable_diffusion_api.CheckpointCache)
		case vaeOption:
			return q.autocompleteModels(i, opt, stable_diffusion_api.VAECache)
		case hypernetworkOption:
			return q.autocompleteModels(i, opt, stable_diffusion_api.HypernetworkCache)
		case embeddingOption:
			return q.autocompleteModels(i, opt, stable_diffusion_api.EmbeddingCache)
		case controlnetPreprocessor:
			return q.autocompleteControlnet(i, opt, stable_diffusion_api.ControlnetModulesCache)
		case controlnetModel:
			return q.autocompleteControlnet(i, opt, stable_diffusion_api.ControlnetModelsCache)
		}

		break
	}

	return nil
}

func (q *SDQueue) autocompleteLora(i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) error {
	var choices []*discordgo.ApplicationCommandOptionChoice

	input := opt.StringValue()
	if input != "" {
		log.Printf("Autocompleting '%v'", input)

		input = sanitizeTooltip(input)

		cache, err := stable_diffusion_api.LoraCache.GetCache(q.stableDiffusionAPI)
		if err != nil {
			log.Printf("Error retrieving loras cache: %v", err)
		}

		sanitized := weightRegex.ReplaceAllString(input, "")

		log.Printf("looking up lora: %v", sanitized)
		results := fuzzy.FindFrom(sanitized, cache)

		for index, result := range results {
			if index > 25 {
				break
			}
			regExp := regexp.MustCompile(`(?:models\\)?Lora\\(.*)`)

			alias := regExp.FindStringSubmatch((*cache.(*stable_diffusion_api.LoraModels))[result.Index].Path)

			var nameToUse string
			switch {
			case alias != nil && alias[1] != "":
				// replace double slash with single slash
				regExp := regexp.MustCompile(`\\{2,}`)
				nameToUse = regExp.ReplaceAllString(alias[1], `\`)
			default:
				nameToUse = (*cache.(*stable_diffusion_api.LoraModels))[result.Index].Name
			}

			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  nameToUse,
				Value: (*cache.(*stable_diffusion_api.LoraModels))[result.Index].Name,
			})
		}

		weightMatches := weightRegex.FindAllStringSubmatch(input, -1)
		log.Printf("weightMatches: %v", weightMatches)

		var tooltip string
		if len(results) > 0 {
			input = (*cache.(*stable_diffusion_api.LoraModels))[results[0].Index].Name
			tooltip = fmt.Sprintf("✨%v", input)
		} else {
			input = sanitized
			tooltip = fmt.Sprintf("❌%v", input)
		}

		if weightMatches != nil && weightMatches[len(weightMatches)-1][1] != "" {
			weight := weightMatches[len(weightMatches)-1][1]
			input += weight
			tooltip += fmt.Sprintf(" 🪄%v", weight)
		} else {
			tooltip += " 🪄1 (𝗱𝗲𝗳𝗮𝘂𝗹𝘁)"
		}

		log.Printf("Name: (tooltip) %v\nValue: (input) %v", tooltip, input)
		choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
			Name:  tooltip,
			Value: input,
		})
	} else {
		choices = []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  "Type a lora name. Add a colon after to specify the strength. (e.g. \"clay:0.5\")",
				Value: "placeholder",
			},
		}
	}

	// make sure we're under 100 char limit and under 25 choices
	for i, choice := range choices {
		if len(choice.Name) > 100 {
			choices[i].Name = choice.Name[:100]
		}
	}

	err := q.botSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))], // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
		},
	})
	return handlers.Wrap(err)
}

func (q *SDQueue) autocompleteModels(i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption, c stable_diffusion_api.Cacheable) error {
	var choices []*discordgo.ApplicationCommandOptionChoice

	input := opt.StringValue()
	if input != "" {
		if c == nil {
			return errors.New("cacheable interface is nil")
		}
		log.Printf("Autocompleting '%v'", input)

		cache, err := c.GetCache(q.stableDiffusionAPI)
		if err != nil {
			return fmt.Errorf("error retrieving %v cache: %w", opt.Name, err)
		}
		results := fuzzy.FindFrom(input, cache)

		for index, result := range results {
			// Match against String() method according to fuzzy docs
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  cache.String(result.Index),
				Value: cache.String(result.Index),
			})
			if index >= 25 {
				break
			}
		}
	} else {
		choices = []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  fmt.Sprintf("Type the %[1]v name. You can also attempt to fuzzy match the %[1]v.", opt.Name),
				Value: "placeholder",
			},
		}
	}

	// make sure we're under 100 char limit and under 25 choices
	for i, choice := range choices {
		if len(choice.Name) > 100 {
			choices[i].Name = choice.Name[:100]
		}
	}

	if len(choices) == 0 {
		return nil
	}

	err := q.botSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
	return handlers.Wrap(err)
}

func (q *SDQueue) autocompleteControlnet(i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption, c stable_diffusion_api.Cacheable) error {
	// check the Type first
	optionMap := utils.GetOpts(i.ApplicationCommandData())

	cache, err := stable_diffusion_api.ControlnetTypesCache.GetCache(q.stableDiffusionAPI)
	if err != nil {
		return fmt.Errorf("error retrieving %s cache: %w", opt.Name, err)
	}
	controlnets := cache.(*stable_diffusion_api.ControlnetTypes)

	var toSearch []string
	var controlType = "All"
	option, ok := optionMap[controlnetType]
	if ok {
		controlType = option.StringValue()
	}

	if types, ok := controlnets.ControlTypes[controlType]; ok {
		switch c.(type) {
		case *stable_diffusion_api.ControlnetModules:
			toSearch = types.ModuleList
		case *stable_diffusion_api.ControlnetModels:
			toSearch = types.ModelList
		}
	} else {
		log.Printf("No controlnet types found for %v: %v", opt.Name, option.StringValue())
	}

	var choices []*discordgo.ApplicationCommandOptionChoice

	input := opt.StringValue()
	if input != "" {
		if len(toSearch) == 0 {
			return fmt.Errorf("no controlnet types found for %v", opt.Name)
		}
		log.Printf("Autocompleting '%v'", input)

		results := fuzzy.Find(input, toSearch)
		//log.Printf("Finding from %v: %v", input, cache)
		//log.Printf("Cache: %v, cache.len(): %v", cache, cache.Len())
		//log.Printf("Results: %v", results)

		for index, result := range results {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  toSearch[result.Index],
				Value: toSearch[result.Index],
			})
			if index >= 25 {
				break
			}
		}
	} else {
		for index, item := range toSearch {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  item,
				Value: item,
			})
			if index >= 25 {
				break
			}
		}
	}

	if len(choices) == 0 {
		choices = []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  fmt.Sprintf("Type the %[1]v name. You can also attempt to fuzzy match the %[1]v.", opt.Name),
				Value: "placeholder",
			},
		}
	}

	err = q.botSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
	return handlers.Wrap(err)
}

func sanitizeTooltip(input string) string {
	tooltipRegex := regexp.MustCompile(`[✨❌](.+) 🪄:([\d.]+)$|[✨❌](.+)`)
	sanitizedTooltip := tooltipRegex.FindStringSubmatch(input)

	if sanitizedTooltip != nil {
		log.Printf("Removing tooltip: %#v", sanitizedTooltip)

		switch {
		case sanitizedTooltip[1] != "":
			input = sanitizedTooltip[1] + ":" + sanitizedTooltip[2]
		case sanitizedTooltip[3] != "":
			input = sanitizedTooltip[3]
		}
		log.Printf("Sanitized input: %v", input)
	}
	return input
}

func (q *SDQueue) processImagineSettingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	botSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return fmt.Errorf("error getting default settings for settings command: %w", err)
	}

	messageComponents := q.settingsMessageComponents(botSettings)

	_, err = handlers.EditInteractionResponse(s, i.Interaction,
		"Choose default settings for the imagine command:",
		messageComponents,
	)
	return err
	//err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	//	Type: discordgo.InteractionResponseChannelMessageWithSource,
	//	Data: &discordgo.InteractionResponseData{
	//		Title:      "Settings",
	//		Content:    "Choose default settings for the imagine command:",
	//		Components: messageComponents,
	//	},
	//})
	//if err != nil {
	//	log.Printf("Error responding to interaction: %v", err)
	//}
}

func (q *SDQueue) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) error {
	botSettings, err := q.UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			return handlers.Wrap(err)
		}

		return nil
	}

	messageComponents := q.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	return handlers.Wrap(err)
}

func (q *SDQueue) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) error {
	botSettings, err := q.UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			return handlers.Wrap(err)
		}

		return nil
	}

	messageComponents := q.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose default settings for the imagine command:",
			Components: messageComponents,
		},
	})
	return handlers.Wrap(err)
}

func (q *SDQueue) processImagineModelSetting(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Type != discordgo.InteractionApplicationCommandAutocomplete {
		return nil
	}
	if len(i.MessageComponentData().Values) == 0 {
		return fmt.Errorf("no values for %v", i.MessageComponentData().CustomID)
	}
	newModelName := i.MessageComponentData().Values[0]

	var config entities.Config
	var modelType string
	switch i.MessageComponentData().CustomID {
	case CheckpointSelect:
		config = entities.Config{SDModelCheckpoint: &newModelName}
		modelType = "checkpoint"
	case VAESelect:
		config = entities.Config{SDVae: &newModelName}
		modelType = "vae"
	case HypernetworkSelect:
		config = entities.Config{SDHypernetwork: &newModelName}
		modelType = "hypernetwork"
	}

	err := handlers.UpdateFromComponent(s, i.Interaction,
		fmt.Sprintf("Updating [**%v**] model to `%v`...", modelType, newModelName),
		i.Interaction.Message.Components,
	)
	if err != nil {
		return err
	}

	err = q.stableDiffusionAPI.UpdateConfiguration(config)
	if err != nil {
		log.Printf("error updating sd model name settings: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction,
			fmt.Sprintf("Error updating [%v] model name settings...", modelType))
	}

	botSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error retrieving bot settings: %v", err)
		return handlers.ErrorEphemeral(s, i.Interaction, "Error retrieving bot settings...")
	}

	newComponents := q.settingsMessageComponents(botSettings)
	_, err = handlers.EditInteractionResponse(s, i.Interaction,
		fmt.Sprintf("Updated [**%v**] model to `%v`", modelType, newModelName),
		newComponents,
	)
	if err != nil {
		return err
	}

	time.AfterFunc(5*time.Second, func() {
		_, _ = handlers.EditInteractionResponse(s, i.Interaction,
			"Choose default settings for the imagine command:",
			newComponents,
		)
	})

	return nil
}

// patch from upstream
func (q *SDQueue) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	config, err := q.stableDiffusionAPI.GetConfig()
	if err != nil {
		log.Printf("Error retrieving config: %v", err)
	} else {
		populateOption(q.stableDiffusionAPI, CheckpointSelect, stable_diffusion_api.CheckpointCache, config)
		populateOption(q.stableDiffusionAPI, VAESelect, stable_diffusion_api.VAECache, config)
		populateOption(q.stableDiffusionAPI, HypernetworkSelect, stable_diffusion_api.HypernetworkCache, config)
	}

	// set default dimension from config
	dimensions := components[DimensionSelect].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	dimensions.Options[0].Default = settings.Width == 512 && settings.Height == 512
	dimensions.Options[1].Default = settings.Width == 768 && settings.Height == 768
	dimensions.Options[2].Default = settings.Width == 1024 && settings.Height == 1024
	dimensions.Options[3].Default = settings.Width == 832 && settings.Height == 1216
	components[DimensionSelect].(discordgo.ActionsRow).Components[0] = dimensions

	batchSlice := []int{1, 2, 4}
	// set default batch count from config
	batchCount := components[BatchCountSelect].(discordgo.ActionsRow)
	for i, option := range batchCount.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchCount {
			option.Default = true
		} else {
			option.Default = false
		}
		batchCount.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	components[BatchCountSelect] = batchCount

	// set the default batch size from config
	batchSize := components[BatchSizeSelect].(discordgo.ActionsRow)
	for i, option := range batchSize.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchSize {
			option.Default = true
		} else {
			option.Default = false
		}
		batchSize.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	components[BatchSizeSelect] = batchSize

	return []discordgo.MessageComponent{
		components[CheckpointSelect],
		components[VAESelect],
		components[HypernetworkSelect],
		components[DimensionSelect],
		//Components[BatchCountSelect],
		components[BatchSizeSelect],
	}
}

// populateOption will fill in the options for a given dropdown component that implements stable_diffusion_api.Cacheable
func populateOption(api stable_diffusion_api.StableDiffusionAPI, handler handlers.Component, cache stable_diffusion_api.Cacheable, config *entities.Config) {
	checkpointDropdown := components[handler].(discordgo.ActionsRow)
	var modelOptions []discordgo.SelectMenuOption

	models, err := cache.GetCache(api)
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
		return
	} else {
		var modelNames []string
		var currentModel *string

		switch toRange := models.(type) {
		case *stable_diffusion_api.SDModels:
			currentModel = config.SDModelCheckpoint
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.ModelName),
					Value: shortenString(model.Title),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.ModelName)
				}
				if model.Hash != nil {
					modelOptions[i].Description = fmt.Sprintf("[%v]", *model.Hash)
				}
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.VAEModels:
			currentModel = config.SDVae
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.ModelName),
					Value: shortenString(model.ModelName),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.ModelName)
				}
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.HypernetworkModels:
			currentModel = config.SDHypernetwork
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label: shortenString(model.Name),
					Value: shortenString(model.Name),
				})
				if currentModel != nil {
					modelOptions[i].Default = strings.Contains(*currentModel, model.Name)
				}
				modelNames = append(modelNames, model.Name)
			}
		}

		var Default bool
		for i, model := range modelOptions {
			if model.Default {
				modelOptions[i].Emoji = &discordgo.ComponentEmoji{
					Name: "✨",
				}
				Default = true
				break
			}
		}

		if currentModel != nil && *currentModel != "" && *currentModel != "None" && !Default {
			modelOptions = append([]discordgo.SelectMenuOption{{
				Label:   shortenString(*currentModel),
				Value:   shortenString(*currentModel),
				Default: true,
				Emoji: &discordgo.ComponentEmoji{
					Name: "✨",
				},
			}}, modelOptions...)
		}

		if len(modelOptions) == 0 {
			modelOptions = append(modelOptions, discordgo.SelectMenuOption{
				Label:       "No models found",
				Value:       "None",
				Description: "Are you sure you have the right API URL?",
				Default:     false,
			})
		} else {
			modelOptions = append([]discordgo.SelectMenuOption{{
				Label:       "None",
				Value:       "None",
				Description: "Unset the model",
				Emoji: &discordgo.ComponentEmoji{
					Name: "❌",
				},
			}}, modelOptions...)
		}
		component := checkpointDropdown.Components[0].(discordgo.SelectMenu)
		component.Options = modelOptions

		components[handler].(discordgo.ActionsRow).Components[0] = component
	}
}

func shortenString(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}

func (q *SDQueue) processRefreshCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	var errs []error
	var content = strings.Builder{}

	var toRefresh []stable_diffusion_api.Cacheable

	switch "refresh_" + i.ApplicationCommandData().Options[0].Name {
	case refreshLoraOption:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.LoraCache}
	case refreshCheckpoint:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.CheckpointCache}
	case refreshVAEOption:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.VAECache}
	case refreshAllOption:
		toRefresh = []stable_diffusion_api.Cacheable{
			stable_diffusion_api.LoraCache,
			stable_diffusion_api.CheckpointCache,
			stable_diffusion_api.VAECache,
		}
	}

	for _, cache := range toRefresh {
		newCache, err := q.stableDiffusionAPI.RefreshCache(cache)
		if err != nil || newCache == nil {
			errs = append(errs, err)
			content.WriteString(fmt.Sprintf("`%T` cache refresh failed.\n", cache))
			continue
		}
		content.WriteString(fmt.Sprintf("`%T` cache refreshed. %v items loaded.\n", newCache, newCache.Len()))
		_, err = handlers.EditInteractionResponse(s, i.Interaction, content.String())
		if err != nil {
			return err
		}
	}

	if errs != nil {
		return handlers.ErrorFollowup(s, i.Interaction, "Error refreshing cache.", errs)
	}

	_, err := handlers.EditInteractionResponse(s, i.Interaction, content.String())
	return err
}

// processRawCommand responds with a Modal to receive a json blob from the user to pass to the api
func (q *SDQueue) processRawCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	optionMap := utils.GetOpts(i.ApplicationCommandData())

	params := entities.RawParams{
		UseDefault: true,
		Unsafe:     false,
	}
	if option, ok := optionMap[useDefaults]; ok {
		params.UseDefault = option.BoolValue()
	}

	if option, ok := optionMap[unsafeOption]; ok {
		params.Unsafe = option.BoolValue()
	}

	if interactionBytes, err := json.Marshal(i.Interaction); err != nil {
		log.Printf("Error marshalling interaction: %v", err)
	} else {
		log.Printf("Interaction: %v", string(interactionBytes))
	}

	var snowflake string
	if option, ok := optionMap[jsonFile]; !ok {
		// if no json file is provided, we need to respond with a modal to get the json blob from the user
		modalDefault[i.ID] = params
		log.Printf("modalDefault: %v", modalDefault)
		interactionResponse := discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: RawCommand,
				Title:    "Raw JSON",
				Components: []discordgo.MessageComponent{
					components[JSONInput],
				},
			},
		}
		err := s.InteractionRespond(i.Interaction, &interactionResponse)
		if err != nil {
			delete(modalDefault, i.ID)
			log.Printf("Error responding to interaction: %v", err)

			byteArr, err := json.Marshal(interactionResponse)
			if err != nil {
				log.Printf("Error marshalling interaction response data: %v", err)
			}
			log.Printf("Raw JSON: %v", string(byteArr))
			return handlers.Wrap(err)
		}

		return nil
	} else {
		snowflake = option.Value.(string)
	}

	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}
	attachments := i.ApplicationCommandData().Resolved.Attachments
	if attachments == nil {
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a JSON file.")
	}

	for snowflake, attachment := range attachments {
		log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
		if !strings.HasPrefix(attachment.ContentType, "application/json") {
			log.Printf("Attachment[%v] is not a json file, removing from queue.", snowflake)
			return handlers.ErrorEdit(s, i.Interaction, "You need to provide a JSON file.")
		}
	}

	attachment, ok := attachments[snowflake]
	if !ok {
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a JSON file.")
	}

	// download attachment url using http and convert to []byte
	resp, err := http.Get(attachment.URL)
	if err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error downloading attachment.", err)
	}
	defer resp.Body.Close()
	if params.Blob, err = io.ReadAll(resp.Body); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error reading attachment.", err)
	}

	params.Debug = strings.Contains(attachment.Filename, "DEBUG")
	if err := q.jsonToQueue(i, params); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error adding imagine to queue.", err)
	}

	return nil
}

var modalDefault = make(map[string]entities.RawParams)

//	type TextInput struct {
//	   CustomID    string         `json:"custom_id"`
//	   Label       string         `json:"label"`
//	   Style       TextInputStyle `json:"style"`
//	   Placeholder string         `json:"placeholder,omitempty"`
//	   Value       string         `json:"value,omitempty"`
//	   Required    bool           `json:"required"`
//	   MinLength   int            `json:"min_length,omitempty"`
//	   MaxLength   int            `json:"max_length,omitempty"`
//	}
func getModalData(data discordgo.ModalSubmitInteractionData) map[handlers.Component]*discordgo.TextInput {
	var options = make(map[handlers.Component]*discordgo.TextInput)
	for _, actionRow := range data.Components {
		for _, c := range actionRow.(*discordgo.ActionsRow).Components {
			switch c := c.(type) {
			case *discordgo.TextInput:
				options[handlers.Component(c.CustomID)] = c
			default:
				log.Fatalf("Wrong component type: %T, skipping...", c)
			}
		}
	}
	return options
}

func (q *SDQueue) processRawModal(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := handlers.ThinkResponse(s, i); err != nil {
		return err
	}

	modalData := getModalData(i.ModalSubmitData())

	var params entities.RawParams
	if message, err := q.botSession.InteractionResponse(i.Interaction); err != nil {
		return handlers.ErrorEdit(s, i.Interaction, "Error retrieving modal data.", err)
	} else {
		if p, ok := modalDefault[message.Interaction.ID]; ok {
			params = p
			delete(modalDefault, message.Interaction.ID)
		}
	}

	if data, ok := modalData[JSONInput]; !ok || data == nil || data.Value == "" {
		log.Printf("modalData: %#v\n", modalData)
		log.Printf("i.ModalSubmitData(): %#v\n", i.ModalSubmitData())
		return handlers.ErrorEdit(s, i.Interaction, "You need to provide a JSON blob.")
	} else {
		params.Debug = strings.Contains(data.Value, "{DEBUG}")
		params.Blob = []byte(strings.ReplaceAll(data.Value, "{DEBUG}", ""))
		if err := q.jsonToQueue(i, params); err != nil {
			return handlers.ErrorEdit(s, i.Interaction, "Error adding imagine to queue.", err)
		}
	}

	return nil
}

func (q *SDQueue) jsonToQueue(i *discordgo.InteractionCreate, params entities.RawParams) error {
	item := &SDQueueItem{
		ImageGenerationRequest: &entities.ImageGenerationRequest{GenerationInfo: entities.GenerationInfo{CreatedAt: time.Now()}},
		DiscordInteraction:     i.Interaction,
	}
	if params.UseDefault {
		item = q.NewItem(i.Interaction)
	}

	item.Type = ItemTypeRaw
	item.Raw = &entities.TextToImageRaw{TextToImageRequest: item.ImageGenerationRequest.TextToImageRequest, RawParams: params}

	// Override Scripts by unmarshalling to Raw
	err := json.Unmarshal(params.Blob, &item.Raw)
	if err != nil {
		return err
	}

	item.ImageGenerationRequest.TextToImageRequest = item.Raw.TextToImageRequest

	position, err := q.Add(item)
	if err != nil {
		return err
	}
	message, err := handlers.EditInteractionResponse(q.botSession, i.Interaction,
		fmt.Sprintf("I'm dreaming something up for you. You are currently #%d in line. Defaults: %v", position, params.UseDefault),
		handlers.Components[handlers.Cancel],
	)
	if item.DiscordInteraction != nil && item.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting message ID for interaction %v", item.DiscordInteraction.ID)
		item.DiscordInteraction.Message = message
	}

	return err
}

type Command = string
type CommandOption = string

// If FieldType and ValueType are the same, then we attempt to assert FieldType to value.Value
// Otherwise, we return the interface conversion to the caller to do manual type conversion
//
// Example:
//
//	if int64Val, ok := interfaceConvertAuto[int, int64](&queue.Steps, stepOption, optionMap, parameters); ok {
//		queue.Steps = int(*int64Val)
//	}
//
// (*discordgo.ApplicationCommandInteractionDataOption).IntValue() actually uses float64 for the interface conversion, so use float64 for integers, numbers, etc.
// and then convert to the desired type.
// Only string and float64 are supported for V as that's what the discordgo API returns.
// If the field is nil, then we don't assign the value to the field.
// Instead, we return *V and bool to indicate whether the conversion was successful.
// This is useful for when we want to convert to a type that is not the same as the field type.
func interfaceConvertAuto[F any, V string | float64](field *F, option CommandOption, optionMap map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption, parameters map[CommandOption]string) (*V, bool) {
	if value, ok := optionMap[option]; ok {
		vToField, ok := value.Value.(F)
		if ok && field != nil {
			*field = vToField
		}
		valueType, ok := value.Value.(V)
		return &valueType, ok
	}
	if value, ok := parameters[option]; ok {
		if field != nil {
			_, err := fmt.Sscanf(value, "%v", field)
			if err != nil {
				return nil, false
			}
		}
		var out V
		_, err := fmt.Sscanf(value, "%v", &out)
		if err != nil {
			return nil, false
		}
		return &out, true
	}
	return nil, false
}
