package discord_bot

import (
	"fmt"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
	"log"
	"regexp"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"
)

var commandHandlers = map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	helloCommand: func(b *botImpl, bot *discordgo.Session, i *discordgo.InteractionCreate) {
		handlers.Responses[handlers.HelloResponse].(handlers.NewResponseType)(bot, i)
	},
	imagineCommand:         (*botImpl).processImagineCommand,
	imagineSettingsCommand: (*botImpl).processImagineSettingsCommand,
	refreshCommand:         (*botImpl).processRefreshCommand,
}

var autocompleteHandlers = map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	imagineCommand: (*botImpl).processImagineAutocomplete,
}

func getOpts(data discordgo.ApplicationCommandInteractionData) map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption {
	options := data.Options
	optionMap := make(map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[CommandOption(opt.Name)] = opt
	}
	return optionMap
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)

	optionMap := getOpts(i.ApplicationCommandData())

	var position int

	var queue *imagine_queue.QueueItem

	if option, ok := optionMap[promptOption]; !ok {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a prompt.")
		return
	} else {
		queue = imagine_queue.NewQueueItem(imagine_queue.WithPrompt(option.StringValue()))

		queue.Type = imagine_queue.ItemTypeImagine
		queue.DiscordInteraction = i.Interaction

		if option, ok := optionMap[negativeOption]; ok {
			queue.NegativePrompt = option.StringValue()
		}

		if option, ok := optionMap[samplerOption]; ok {
			queue.SamplerName1 = option.StringValue()
		}

		if option, ok := optionMap[stepOption]; ok {
			queue.Steps = int(option.IntValue())
		}

		if option, ok := optionMap[seedOption]; ok {
			queue.Seed = option.IntValue()
		}

		if option, ok := optionMap[restoreFacesOption]; ok {
			restoreFaces, err := strconv.ParseBool(option.StringValue())
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				queue.RestoreFaces = restoreFaces
			}

		}

		if option, ok := optionMap[adModelOption]; ok {
			queue.ADetailerString = option.StringValue()
			// adModel = strings.Split(stringValue, ",")
			// use AppendSegModelByString instead
		}

		if option, ok := optionMap[checkpointOption]; ok {
			value := option.StringValue()
			queue.Checkpoint = &value
			log.Printf("user wants to change checkpoint to %v", value)
		}

		if option, ok := optionMap[vaeOption]; ok {
			value := option.StringValue()
			queue.VAE = &value
			log.Printf("user wants to use vae %v", value)
		}

		if option, ok := optionMap[hypernetworkOption]; ok {
			value := option.StringValue()
			queue.Hypernetwork = &value
			log.Printf("user wants to use hypernetwork %v", value)
		}

		if option, ok := optionMap[embeddingOption]; ok {
			queue.Prompt += " " + option.StringValue()
			log.Printf("Adding embedding: %v", option.StringValue())
		}

		for i := 0; i < extraLoras+1; i++ {
			loraKey := loraOption
			if i != 0 {
				loraKey += CommandOption(fmt.Sprintf("%d", i+1))
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
					queue.Prompt += lora
				}
			}
		}

		if option, ok := optionMap[aspectRatio]; ok {
			//queue.Prompt += " --ar " + queue.AspectRatio
			queue.AspectRatio = option.StringValue()
		}

		if option, ok := optionMap[hiresFixSize]; ok {
			//queue.Prompt += " --zoom " + option.StringValue()
			hiresUpscaleRate, err := strconv.ParseFloat(option.StringValue(), 64)
			if err != nil {
				log.Printf("Error parsing hiresUpscaleRate: %v", err)
			} else {
				queue.HiresUpscaleRate = hiresUpscaleRate
				queue.UseHiresFix = true
			}
		}

		if option, ok := optionMap[hiresFixOption]; ok {
			hiresFix, err := strconv.ParseBool(option.StringValue())
			if err != nil {
				log.Printf("Error parsing hiresFix value: %v.", err)
			} else {
				queue.UseHiresFix = hiresFix
			}
		}

		if option, ok := optionMap[cfgScaleOption]; ok {
			//prompt += " --cfgscale " + fmt.Sprintf("%v", option.IntValue())
			queue.CfgScale = option.FloatValue()
		}

		if option, ok := optionMap[restoreFacesOption]; ok {
			restoreFacesValue, err := strconv.ParseBool(option.StringValue())
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				queue.RestoreFaces = restoreFacesValue
			}
		}

		if i.ApplicationCommandData().Resolved != nil {
			if attachments := i.ApplicationCommandData().Resolved.Attachments; attachments != nil {
				if queue.Attachments == nil {
					queue.Attachments = make(map[string]*entities.MessageAttachment, len(attachments))
				}
				for snowflake, attachment := range attachments {
					queue.Attachments[snowflake] = &entities.MessageAttachment{
						MessageAttachment: *attachment,
					}
					log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
					if !strings.HasPrefix(attachment.ContentType, "image") {
						log.Printf("Attachment[%v] is not an image, removing from queue.", snowflake)
						delete(queue.Attachments, snowflake)
					}

					image, err := utils.GetImageBase64(attachment.URL)
					if err != nil {
						log.Printf("Error getting image from URL: %v", err)
						handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error getting image from URL.", err)
						return
					}
					queue.Attachments[snowflake].Image = image
				}
			}
		}

		if option, ok := optionMap[img2imgOption]; ok {
			if attachment, ok := queue.Attachments[option.Value.(string)]; !ok {
				handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide an image to img2img.")
				return
			} else {
				queue.Type = imagine_queue.ItemTypeImg2Img

				queue.Img2ImgItem.Image = attachment

				if option, ok := optionMap[denoisingOption]; ok {
					queue.DenoisingStrength = option.FloatValue()
				}
			}
		}

		var controlnet bool

		if option, ok := optionMap[controlnetImage]; ok {
			if attachment, ok := queue.Attachments[option.Value.(string)]; !ok {
				handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide an image to controlnet.")
				return
			} else {
				queue.ControlnetItem.Image = attachment
			}
			controlnet = true
		}

		if option, ok := optionMap[controlnetControlMode]; ok {
			queue.ControlnetItem.ControlMode = option.StringValue()
			controlnet = true
		}

		if option, ok := optionMap[controlnetResizeMode]; ok {
			queue.ControlnetItem.ResizeMode = option.StringValue()
			controlnet = true
		}

		if option, ok := optionMap[controlnetType]; ok {
			queue.ControlnetItem.Type = option.StringValue()
			controlnet = true
		}

		if option, ok := optionMap[controlnetPreprocessor]; ok {
			queue.ControlnetItem.Preprocessor = option.StringValue()
			controlnet = true
		}

		if option, ok := optionMap[controlnetModel]; ok {
			queue.ControlnetItem.Model = option.StringValue()
			controlnet = true
		}

		// TODO: Finish implementation of processing controlnet JSON object to be passed to the API
		if controlnet {
			// Debugging for now
			log.Printf("Controlnet item: %#v", queue.ControlnetItem)
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Controlnet is not yet implemented.")
			return
		}

		var err error
		position, err = b.imagineQueue.AddImagine(queue)
		if err != nil {
			log.Printf("Error adding imagine to queue: %v\n", err)
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error adding imagine to queue.", err)
		} else {
			// TODO: Remove debug message
			//log.Printf("Added imagine %#v to queue. Position: %v\n", queue, position)
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
		queue.Prompt,
	)

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, queueString, handlers.Components[handlers.CancelButton])
}

var weightRegex = regexp.MustCompile(`.+\\|\.(?:safetensors|ckpt|pth?)|(:[\d.]+$)`)

func (b *botImpl) processImagineAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	log.Printf("running autocomplete handler")
	for optionIndex, opt := range data.Options {
		if !opt.Focused {
			continue
		}
		input := opt.StringValue()
		switch {
		case strings.HasPrefix(opt.Name, string(loraOption)):
			log.Printf("Focused option (%v): %v", optionIndex, opt.Name)

			var choices []*discordgo.ApplicationCommandOptionChoice

			if input != "" {
				log.Printf("Autocompleting '%v'", input)

				input = sanitizeTooltip(input)

				cache, err := b.StableDiffusionApi.SDLorasCache()
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

					alias := regExp.FindStringSubmatch((*cache)[result.Index].Path)

					var nameToUse string
					switch {
					case alias != nil && alias[1] != "":
						// replace double slash with single slash
						regExp := regexp.MustCompile(`\\{2,}`)
						nameToUse = regExp.ReplaceAllString(alias[1], `\`)
					default:
						nameToUse = (*cache)[result.Index].Name
					}

					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  nameToUse,
						Value: (*cache)[result.Index].Name,
					})
				}

				weightMatches := weightRegex.FindAllStringSubmatch(input, -1)
				log.Printf("weightMatches: %v", weightMatches)

				var tooltip string
				if len(results) > 0 {
					input = (*cache)[results[0].Index].Name
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

				//choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
				//	Name:  input,
				//	Value: input,
				//})
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
					// TODO: check if discord counts bytes or chars
					choices[i].Name = choice.Name[:100]
				}
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices[:min(25, len(choices))], // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
				},
			})
		default:
			switch CommandOption(opt.Name) {
			case checkpointOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.CheckpointCache)
			case vaeOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.VAECache)
			case hypernetworkOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.HypernetworkCache)
			case embeddingOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.EmbeddingCache)
			case controlnetPreprocessor:
				b.autocompleteControlnet(s, i, optionIndex, opt, input, stable_diffusion_api.ControlnetModulesCache)
			case controlnetModel:
				b.autocompleteControlnet(s, i, optionIndex, opt, input, stable_diffusion_api.ControlnetModelsCache)
			}
		}
		break
	}
}

func (b *botImpl) autocompleteModels(s *discordgo.Session, i *discordgo.InteractionCreate, index int, opt *discordgo.ApplicationCommandInteractionDataOption, input string, c stable_diffusion_api.Cacheable) {
	log.Printf("Focused option (%v): %v", index, opt.Name)
	input = opt.StringValue()

	var choices []*discordgo.ApplicationCommandOptionChoice

	if input != "" {
		if c == nil {
			log.Printf("Cacheable interface is nil")
		}
		log.Printf("Autocompleting '%v'", input)

		cache, err := c.GetCache(b.StableDiffusionApi)
		if err != nil {
			log.Printf("Error retrieving %v cache: %v", opt.Name, err)
		}
		results := fuzzy.FindFrom(input, cache)
		//log.Printf("Finding from %v: %v", input, cache)
		//log.Printf("Cache: %v, cache.len(): %v", cache, cache.Len())
		//log.Printf("Results: %v", results)

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
			// TODO: check if discord counts bytes or chars
			choices[i].Name = choice.Name[:100]
		}
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
}

func (b *botImpl) autocompleteControlnet(s *discordgo.Session, i *discordgo.InteractionCreate, index int, opt *discordgo.ApplicationCommandInteractionDataOption, input string, c stable_diffusion_api.Cacheable) {
	input = opt.StringValue()

	// check the Type first
	optionMap := getOpts(i.ApplicationCommandData())

	cache, err := stable_diffusion_api.ControlnetTypesCache.GetCache(b.StableDiffusionApi)
	if err != nil {
		log.Printf("Error retrieving %v cache: %v", opt.Name, err)
		return
	}
	controlnets := cache.(*stable_diffusion_api.ControlnetTypes)

	log.Printf("Focused option (%v): %v", index, opt.Name)

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
	if input != "" {
		if len(toSearch) == 0 {
			log.Printf("No controlnet types found for %v", opt.Name)
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

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
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

func (b *botImpl) processImagineSettingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)
	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error getting default settings for settings command: %v", err)

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction,
		"Choose default settings for the imagine command:",
		messageComponents,
	)
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

func (b *botImpl) processRefreshCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)

	var errors []error
	var content = strings.Builder{}

	var toRefresh []stable_diffusion_api.Cacheable

	switch CommandOption("refresh_" + i.ApplicationCommandData().Options[0].Name) {
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
		newCache, err := b.StableDiffusionApi.RefreshCache(cache)
		if err != nil || newCache == nil {
			errors = append(errors, err)
			content.WriteString(fmt.Sprintf("`%T` cache refresh failed.\n", cache))
			continue
		}
		content.WriteString(fmt.Sprintf("`%T` cache refreshed. %v items loaded.\n", cache, cache.Len()))
		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, content.String())
	}

	if errors != nil {
		log.Printf("Error refreshing cache: %v", errors)
		handlers.Errors[handlers.ErrorFollowup](s, i.Interaction, "Error refreshing cache.", errors)
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, content.String())
}
