package discord_bot

import (
	"github.com/bwmarrin/discordgo"
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
