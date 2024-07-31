package handlers

import (
	"github.com/bwmarrin/discordgo"
)

type Component = string

const (
	DeleteButton      Component = "delete_error_message"
	DeleteAboveButton Component = "delete_above"
	DeleteGeneration  Component = "delete_generation"

	dismissButton Component = "dismiss_error_message"
	urlButton     Component = "url_button"
	urlDelete     Component = "url_delete"

	readmoreDismiss Component = "readmore_dismiss"

	paginationButtons Component = "pagination_button"
	okCancelButtons   Component = "ok_cancel_buttons"

	Cancel    Component = "cancel"
	Interrupt Component = "interrupt"

	CancelDisabled    Component = "cancel_disabled"
	InterruptDisabled Component = "interrupt_disabled"

	roleSelect = "role_select"
)

var Components = map[Component]discordgo.MessageComponent{
	DeleteButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete this message",
				Style:    discordgo.DangerButton,
				CustomID: DeleteButton,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🗑️",
				},
			},
		},
	},
	DeleteAboveButton: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Delete above",
				Style:    discordgo.DangerButton,
				CustomID: DeleteAboveButton,
				Emoji: &discordgo.ComponentEmoji{
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
				Emoji: &discordgo.ComponentEmoji{
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
				Emoji: &discordgo.ComponentEmoji{
					Name: "📜",
				},
			},
			discordgo.Button{
				Label:    "Delete",
				Style:    discordgo.DangerButton,
				CustomID: DeleteButton,
				Emoji: &discordgo.ComponentEmoji{
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
	Cancel: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: Cancel,
			},
		},
	},
	CancelDisabled: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				CustomID: Cancel,
				Disabled: true,
			},
		},
	},
	Interrupt: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Interrupt",
				Style:    discordgo.DangerButton,
				CustomID: Interrupt,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⚠️",
				},
				Disabled: false,
			},
		},
	},
	InterruptDisabled: discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Interrupt",
				Style:    discordgo.DangerButton,
				CustomID: Interrupt,
				Emoji: &discordgo.ComponentEmoji{
					Name: "⚠️",
				},
				Disabled: true,
			},
		},
	},
}
