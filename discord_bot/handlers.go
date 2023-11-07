package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
	"log"
	"regexp"
	"stable_diffusion_bot/imagine_queue"
	"strconv"
	"strings"
)

const extraLoras = 6

const (
	promptOption       = "prompt"
	negativeOption     = "negative_prompt"
	samplerOption      = "sampler_name"
	aspectRatio        = "aspect_ratio"
	loraOption         = "lora"
	checkpointOption   = "checkpoint"
	hiresFixOption     = "use_hires_fix"
	hiresFixSize       = "hires_fix_size"
	restoreFacesOption = "restore_faces"
	adModelOption      = "ad_model"
	cfgScaleOption     = "cfg_scale"
)

func (b *botImpl) addImagineCommand() error {
	log.Printf("Adding command '%s'...", b.imagineCommandString())

	options := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        promptOption,
			Description: "The text prompt to imagine",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        negativeOption,
			Description: "Negative prompt",
			Required:    false,
		},
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         checkpointOption,
			Description:  "The checkpoint to change to when generating. Sets for the next person.",
			Required:     false,
			Autocomplete: true,
		},
		{
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
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         loraOption,
			Description:  "The lora(s) to apply",
			Required:     false,
			Autocomplete: true,
		},
		{
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
		{
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
		{
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
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        cfgScaleOption,
			Description: "upscale multiplier for cfg. default=7",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "7",
					Value: "7",
				},
			},
		},
		{
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
		{
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
	}

	for i := 0; i < extraLoras; i++ {
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         loraOption + fmt.Sprintf("%d", i+2),
			Description:  "The lora(s) to apply",
			Required:     false,
			Autocomplete: true,
		})
	}

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineCommandString(),
		Description: "Ask the bot to imagine something",
		Options:     options,
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineCommandString(), err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) addImagineSettingsCommand() error {
	log.Printf("Adding command '%s'...", b.imagineSettingsCommandString())

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineSettingsCommandString(),
		Description: "Change the default settings for the imagine command",
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineSettingsCommandString(), err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeReroll,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm reimagining that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeUpscale,
		InteractionIndex:   upscaleIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm upscaling that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeVariation,
		InteractionIndex:   variationIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm imagining more variations for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}

	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var position int
	var queueError error
	var prompt string
	negative := ""
	sampler := "Euler a"
	hiresFix := false
	restoreFaces := false
	var stringValue string
	var checkpoint string

	if option, ok := optionMap[promptOption]; ok {
		prompt = option.StringValue()

		if nopt, ok := optionMap[negativeOption]; ok {
			negative = nopt.StringValue()
		}

		if smpl, ok := optionMap[samplerOption]; ok {
			sampler = smpl.StringValue()
		}

		if hires, ok := optionMap[hiresFixOption]; ok {
			hiresFix, _ = strconv.ParseBool(hires.StringValue())
		}

		if hires, ok := optionMap[restoreFacesOption]; ok {
			restoreFaces, _ = strconv.ParseBool(hires.StringValue())
		}

		if aDetailOpt, ok := optionMap[adModelOption]; ok {
			stringValue = aDetailOpt.StringValue()
			// adModel = strings.Split(stringValue, ",")
			// use AppendSegModelByString instead
		}

		if cpkt, ok := optionMap[checkpointOption]; ok {
			checkpoint = cpkt.StringValue()
			log.Printf("user wants to change checkpoint to %v", checkpoint)
		}

		strength := regexp.MustCompile(`:([\d\.]+)$`)

		for i := 0; i < extraLoras+1; i++ {
			loraKey := loraOption
			if i != 0 {
				loraKey += fmt.Sprintf("%d", i+1)
			}

			if lora, ok := optionMap[loraKey]; ok {
				loraValue := lora.StringValue()
				if loraValue != "" {

					loraValue = sanitizeTooltip(loraValue)

					// add :1 if no strength is specified
					if !strength.MatchString(loraValue) {
						loraValue += ":1"
					}
					re := regexp.MustCompile(`.+\\|\.safetensors`)
					loraValue = re.ReplaceAllString(loraValue, "")
					lora := ", <lora:" + loraValue + ">"
					log.Println("Adding lora: ", lora)
					prompt += lora
				}
			}
		}

		if aspectRatio, ok := optionMap[aspectRatio]; ok {
			prompt += " --ar " + aspectRatio.StringValue()
		}

		if hiresFixSize, ok := optionMap[hiresFixSize]; ok {
			prompt += " --zoom " + hiresFixSize.StringValue()
		}

		if cfgScaleOption, ok := optionMap[cfgScaleOption]; ok {
			prompt += " --cfgscale " + fmt.Sprintf("%v", cfgScaleOption.IntValue())
		}

		if restoreFacesOption, ok := optionMap[restoreFacesOption]; ok {
			restoreFacesValue, err := strconv.ParseBool(restoreFacesOption.StringValue())
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			}
			restoreFaces = restoreFacesValue
		}

		imagine := &imagine_queue.QueueItem{
			Prompt:             prompt,
			NegativePrompt:     negative,
			SamplerName1:       sampler,
			Type:               imagine_queue.ItemTypeImagine,
			UseHiresFix:        hiresFix,
			RestoreFaces:       restoreFaces,
			DiscordInteraction: i.Interaction,
			ADetailerString:    stringValue,
			Checkpoint:         checkpoint,
		}

		position, queueError = b.imagineQueue.AddImagine(imagine)
		if queueError != nil {
			log.Printf("Error adding imagine to queue: %v\n", queueError)
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
		prompt,
	)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &queueString,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
	}
}

func (b *botImpl) processImagineAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	log.Printf("running autocomplete handler")
	var input string
	for index, opt := range data.Options {
		if !opt.Focused {
			continue
		}
		switch {
		case strings.HasPrefix(opt.Name, loraOption):
			log.Printf("Focused option (%v): %v", index, opt.Name)
			input = opt.StringValue()

			var choices []*discordgo.ApplicationCommandOptionChoice

			if input != "" {
				log.Printf("Autocompleting '%v'", input)

				input = sanitizeTooltip(input)

				cache, err := b.StableDiffusionApi.SDLorasCache()
				if err != nil {
					log.Printf("Error retrieving loras cache: %v", err)
				}

				re := regexp.MustCompile(`.+\\|\.safetensors.*|(:[\d.]+$)`)
				sanitized := re.ReplaceAllString(input, "")

				log.Printf("looking up lora: %v", sanitized)
				results := fuzzy.FindFrom(sanitized, cache)

				for index, result := range results {
					if index > 25 {
						break
					}
					regExp := regexp.MustCompile(`(?:models\\)?Lora\\(.*)`)

					alias := regExp.FindStringSubmatch(cache[result.Index].Path)

					var nameToUse string
					switch {
					case alias != nil && alias[1] != "":
						// replace double slash with single slash
						regExp := regexp.MustCompile(`\\{2,}`)
						nameToUse = regExp.ReplaceAllString(alias[1], `\`)
					default:
						nameToUse = cache[result.Index].Name
					}

					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  nameToUse,
						Value: cache[result.Index].Name,
					})
				}

				weight := re.FindStringSubmatch(input)
				log.Printf("weight: %v", weight)

				var tooltip string
				if len(results) > 0 {
					input = cache[results[0].Index].Name
					tooltip = fmt.Sprintf("✨%v", input)
				} else {
					input = sanitized
					tooltip = fmt.Sprintf("❌%v", input)
				}

				if weight != nil && weight[1] != "" {
					input += weight[1]
					tooltip += fmt.Sprintf(" 🪄%v", weight[1])
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
		case opt.Name == checkpointOption:
			log.Printf("Focused option (%v): %v", index, opt.Name)
			input = opt.StringValue()

			var choices []*discordgo.ApplicationCommandOptionChoice

			if input != "" {
				log.Printf("Autocompleting '%v'", input)
				cache, err := b.StableDiffusionApi.SDModelsCache()
				if err != nil {
					log.Printf("Error retrieving checkpoint cache: %v", err)
				}
				results := fuzzy.FindFrom(input, cache)

				for index, result := range results {
					if index > 25 {
						break
					}
					//regExp := regexp.MustCompile(`(?:models\\)?Stable-diffusion\\(.*)`)
					//
					//alias := regExp.FindStringSubmatch(cache[result.Index].Filename)
					//
					//var nameToUse string
					//switch {
					//case alias != nil && alias[1] != "":
					//	// replace double slash with single slash
					//	regExp := regexp.MustCompile(`\\{2,}`)
					//	nameToUse = regExp.ReplaceAllString(alias[1], `\`)
					//default:
					//	nameToUse = cache[result.Index].Title
					//}

					// Match against String() method according to fuzzy docs
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  cache[result.Index].Title,
						Value: cache[result.Index].Title,
					})
				}

				//choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
				//	Name:  input,
				//	Value: input,
				//})
			} else {
				choices = []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Type a checkpoint name. You can also attempt to fuzzy match a checkpoint name.",
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
		}
		break
	}
}

func sanitizeTooltip(input string) string {
	tooltipRegex := regexp.MustCompile(`(?:✨|❌)(.+) 🪄:([\d\.]+)$|(?:✨|❌)(.+)`)
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
	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error getting default settings for settings command: %v", err)

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:      "Settings",
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
