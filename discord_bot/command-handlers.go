package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
	"log"
	"regexp"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/imagine_queue"
	"strconv"
	"strings"
)

var commandHandlers = map[string]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	helloCommand: func(b *botImpl, bot *discordgo.Session, i *discordgo.InteractionCreate) {
		handlers.Responses[handlers.HelloResponse].(handlers.NewResponseType)(bot, i)
	},
	imagineCommand:         (*botImpl).processImagineCommand,
	imagineSettingsCommand: (*botImpl).processImagineSettingsCommand,
}

var autocompleteHandlers = map[string]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	imagineCommand: (*botImpl).processImagineAutocomplete,
}

// addImagineCommand is now inside the commands map as imagineCommand: commands[imagineCommand]
// It also uses imagineOptions() to build the necessary commandOptions
// Deprecated: use commands[imagineCommand]
func (b *botImpl) addImagineCommand(name string, command *discordgo.ApplicationCommand) (error, *discordgo.ApplicationCommand) {
	log.Printf("Adding command '%s'...", name)

	commands[imagineCommand].Options = imagineOptions()

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, commands[imagineCommand])
	if err != nil {
		log.Printf("Error creating '%s' command: %v", name, err)

		return err, nil
	}

	return nil, cmd
}

// Deprecated: use commandHandlers[imagineCommand]
func (b *botImpl) addImagineSettingsCommand(command string) (error, *discordgo.ApplicationCommand) {
	log.Printf("Adding command '%s'...", command)

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineSettingsCommandString(),
		Description: "Change the default settings for the imagine command",
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineSettingsCommandString(), err)

		return err, nil
	}

	//b.registeredCommands[command] = cmd

	return nil, cmd
}

func getOpts(data discordgo.ApplicationCommandInteractionData) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	options := data.Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}

	optionMap := getOpts(i.ApplicationCommandData())

	var position int
	var queueError error
	var prompt string
	negative := ""
	sampler := "Euler a"
	hiresFix := false
	restoreFaces := false
	var stringValue string
	var vae string
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

		if vaeOpt, ok := optionMap[vaeOption]; ok {
			vae = vaeOpt.StringValue()
			log.Printf("user wants to use vae %v", vae)
		}

		if hypernetwork, ok := optionMap[hypernetworkOption]; ok {
			prompt += " " + hypernetwork.StringValue()
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
		default:
			switch opt.Name {
			case checkpointOption:
				log.Printf("Focused option (%v): %v", index, opt.Name)
				input = opt.StringValue()

				var choices []*discordgo.ApplicationCommandOptionChoice

				if input != "" {
					log.Printf("Autocompleting '%v'", input)
					cache, err := b.StableDiffusionApi.SDCheckpointsCache()
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
			case vaeOption:
				log.Printf("Focused option (%v): %v", index, opt.Name)
				input = opt.StringValue()

				var choices []*discordgo.ApplicationCommandOptionChoice

				if input != "" {
					log.Printf("Autocompleting '%v'", input)
					cache, err := b.StableDiffusionApi.SDVAECache()
					if err != nil {
						log.Printf("Error retrieving vae cache: %v", err)
					}
					results := fuzzy.FindFrom(input, cache)

					for index, result := range results {
						if index > 25 {
							break
						}

						// Match against String() method according to fuzzy docs
						choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
							Name:  cache[result.Index].ModelName,
							Value: cache[result.Index].ModelName,
						})
					}
				} else {
					choices = []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Type a vae name. You can also attempt to fuzzy match a vae.",
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
			case hypernetworkOption:
				log.Printf("Focused option (%v): %v", index, opt.Name)
				input = opt.StringValue()

				var choices []*discordgo.ApplicationCommandOptionChoice

				if input != "" {
					log.Printf("Autocompleting '%v'", input)
					cache, err := b.StableDiffusionApi.SDHypernetworkCache()
					if err != nil {
						log.Printf("Error retrieving hypernetwork cache: %v", err)
					}
					results := fuzzy.FindFrom(input, cache)

					for index, result := range results {
						if index > 25 {
							break
						}

						// Match against String() method according to fuzzy docs
						choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
							Name:  cache[result.Index].Name,
							Value: cache[result.Index].Name,
						})
					}
				} else {
					choices = []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Type a hypernetwork name. You can also attempt to fuzzy match a hypernetwork.",
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
			Content:    "Choose default settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
