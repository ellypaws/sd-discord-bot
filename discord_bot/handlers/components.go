package handlers

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

const (
	CheckpointSelect   = "imagine_sd_model_name_menu"
	VAESelect          = "imagine_vae_model_name_menu"
	HypernetworkSelect = "imagine_hypernetwork_model_name_menu"
	DimensionSelect    = "imagine_dimension_setting_menu"
	BatchCountSelect   = "imagine_batch_count_setting_menu"
	BatchSizeSelect    = "imagine_batch_size_setting_menu"
)

const (
	RerollButton  = "imagine_reroll"
	UpscaleButton = "imagine_upscale"
	VariantButton = "imagine_variation"
)

const (
	DeleteButton      = "delete_error_message"
	DeleteAboveButton = "delete_above"
	DeleteGeneration  = "delete_generation"

	dismissButton = "dismiss_error_message"
	urlButton     = "url_button"
	urlDelete     = "url_delete"

	readmoreDismiss = "readmore_dismiss"

	paginationButtons = "pagination_button"
	okCancelButtons   = "ok_cancel_buttons"

	roleSelect = "role_select"
)

var minValues = 1

var Components = map[string]discordgo.MessageComponent{
	DeleteButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete this message",
				Style:    discordgo.DangerButton,
				CustomID: DeleteButton,
			},
		},
	},
	DeleteAboveButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete above",
				Style:    discordgo.DangerButton,
				CustomID: DeleteAboveButton,
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	DeleteGeneration: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				CustomID: DeleteGeneration,
				Emoji: discordgo.ComponentEmoji{
					Name: "🗑️",
				},
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
				CustomID: DeleteButton,
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
				CustomID: DeleteButton,
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
				CustomID: DeleteButton,
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

	CheckpointSelect:   ModelSelectMenu(CheckpointSelect),
	VAESelect:          ModelSelectMenu(VAESelect),
	HypernetworkSelect: ModelSelectMenu(HypernetworkSelect),

	DimensionSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  DimensionSelect,
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
	BatchCountSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  BatchCountSelect,
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
	BatchSizeSelect: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:  BatchSizeSelect,
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

func ModelSelectMenu(ID string) discordgo.ActionsRow {
	display := strings.TrimPrefix(ID, "imagine_")
	display = strings.TrimSuffix(ID, "_model_name_menu")
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    ID,
				Placeholder: fmt.Sprintf("Change %s Model", display),
				MinValues:   &minValues,
				MaxValues:   1,
				Options: []discordgo.SelectMenuOption{
					{
						Label:       display,
						Value:       "Placeholder",
						Description: "Placeholder",
						Default:     false,
					},
				},
			},
		},
	}
}
