package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/entities"
)

const (
	checkpointSelect = "imagine_sd_model_name_menu"
	dimensionSelect  = "imagine_dimension_setting_menu"
	batchCountSelect = "imagine_batch_count_setting_menu"
	batchSizeSelect  = "imagine_batch_size_setting_menu"
)

const (
	rerollButton  = "imagine_reroll"
	upscaleButton = "imagine_upscale"
	variantButton = "imagine_variation"
)

const (
	deleteButton  = "delete_error_message"
	dismissButton = "dismiss_error_message"
	urlButton     = "url_button"
	urlDelete     = "url_delete"

	readmoreDismiss = "readmore_dismiss"

	paginationButtons = "pagination_button"
	okCancelButtons   = "ok_cancel_buttons"

	roleSelect = "role_select"
)

var minValues = 1

var components = map[string]discordgo.MessageComponent{
	deleteButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete this message",
				Style:    discordgo.DangerButton,
				CustomID: deleteButton,
			},
		},
	},
	urlButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Read more",
				Style: discordgo.LinkButton,
			},
		},
	},
	urlDelete: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: "Read more",
				Style: discordgo.LinkButton,
				Emoji: discordgo.ComponentEmoji{
					Name: "📜",
				},
			},
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				CustomID: deleteButton,
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	dismissButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Dismiss",
				Style:    discordgo.SecondaryButton,
				CustomID: deleteButton,
			},
		},
	},

	readmoreDismiss: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Read more",
				Style:    discordgo.LinkButton,
				CustomID: urlButton,
			},
			discordgo.Button{
				Label:    "Dismiss",
				Style:    discordgo.SecondaryButton,
				CustomID: deleteButton,
			},
		},
	},

	paginationButtons: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: paginationButtons + "_previous",
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: paginationButtons + "_next",
			},
		},
	},
	okCancelButtons: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "OK",
				Style:    discordgo.SuccessButton,
				CustomID: okCancelButtons + "_ok",
			},
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: okCancelButtons + "_cancel",
			},
		},
	},

	roleSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.RoleSelectMenu,
				CustomID:    roleSelect,
				Placeholder: "Pick a role",
			},
		},
	},

	checkpointSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    checkpointSelect,
				Placeholder: "Change SD Model",
				MinValues:   &minValues,
				MaxValues:   1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:       "Checkpoint",
						Value:       "Placeholder",
						Description: "Placeholder",
						Default:     false,
					},
				},
			},
		},
	},

	dimensionSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{discordgo.SelectMenu{
			CustomID:  dimensionSelect,
			MinValues: nil,
			MaxValues: 1,
			Options: []discordgo.SelectMenuOption{
				{
					Label:   "Size: 512x512",
					Value:   "512_512",
					Default: true,
				},
				{
					Label:   "Size: 768x768",
					Value:   "768_768",
					Default: false,
				},
			},
		},
		},
	},

	batchCountSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  batchCountSelect,
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch count: 1",
						Value:   "1",
						Default: false,
					},
					{
						Label:   "Batch count: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch count: 4",
						Value:   "4",
						Default: true,
					},
				},
			},
		},
	},

	batchSizeSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  batchSizeSelect,
				MinValues: &minValues,
				MaxValues: 1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:   "Batch size: 1",
						Value:   "1",
						Default: true,
					},
					{
						Label:   "Batch size: 2",
						Value:   "2",
						Default: false,
					},
					{
						Label:   "Batch size: 4",
						Value:   "4",
						Default: false,
					},
				},
			},
		},
	},
}

// patch from upstream
func (b *botImpl) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	models, err := b.StableDiffusionApi.SDModelsCache()

	// populate checkpoint dropdown and set default
	checkpointDropdown := components[checkpointSelect].(discordgo.ActionsRow)
	var modelOptions []discordgo.SelectMenuOption
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
	} else {
		for i, model := range models {
			if i > 20 {
				break
			}
			modelOptions = append(modelOptions, discordgo.SelectMenuOption{
				Label:   shortenString(model.ModelName),
				Value:   shortenString(model.Title),
				Default: settings.SDModelName == model.Title,
			})
		}

		checkpointDropdown.Components[0] = discordgo.SelectMenu{
			Options: modelOptions,
		}
		components[checkpointSelect] = checkpointDropdown
	}

	// set default dimension from config
	dimensions := components[dimensionSelect].(discordgo.ActionsRow)
	dimensions.Components[0].(discordgo.SelectMenu).Options[0].Default = settings.Width == 512 && settings.Height == 512
	dimensions.Components[0].(discordgo.SelectMenu).Options[1].Default = settings.Width == 768 && settings.Height == 768
	components[dimensionSelect] = dimensions

	// set default batch count from config
	batchCount := components[batchCountSelect].(discordgo.ActionsRow)
	for i, option := range batchCount.Components[0].(discordgo.SelectMenu).Options {
		if i == settings.BatchCount {
			option.Default = true
		} else {
			option.Default = false
		}
		batchCount.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	components[batchCountSelect] = batchCount

	// set the default batch size from config
	batchSize := components[batchSizeSelect].(discordgo.ActionsRow)
	for i, option := range batchSize.Components[0].(discordgo.SelectMenu).Options {
		if i == settings.BatchSize {
			option.Default = true
		} else {
			option.Default = false
		}
		batchSize.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	components[batchSizeSelect] = batchSize

	return []discordgo.MessageComponent{
		components[checkpointSelect],
		components[dimensionSelect],
		components[batchCountSelect],
		components[batchSizeSelect],
	}
}

func (b *botImpl) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) {
	botSettings, err := b.imagineQueue.UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}
}

func (b *botImpl) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) {
	botSettings, err := b.imagineQueue.UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineSDModelNameSetting(s *discordgo.Session, i *discordgo.InteractionCreate, newModelName string) {
	botSettings, err := b.imagineQueue.UpdateModelName(newModelName)
	if err != nil {
		log.Printf("error updating sd model name settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating sd model name settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
