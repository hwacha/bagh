package main

import (
	"github.com/bwmarrin/discordgo"
)

var acceptOrRefuseButtonRow = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Accept",
				Style:    discordgo.PrimaryButton,
				Disabled: false,
				CustomID: "challenge_accept",
			},
			discordgo.Button{
				Label:    "Refuse",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "challenge_refuse",
			},
		},
	},
}

var actionButtonGrid = []discordgo.MessageComponent{
	// ActionRow is a container of all buttons within the same row.
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Boost",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "action_boost",
				Emoji: &discordgo.ComponentEmoji{
					Name: "⬆️",
				},
			},
			discordgo.Button{
				Label:    "Guard",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "action_guard",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🛡️",
				},
			},
		},
	},
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Attack",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "action_attack",
				Emoji: &discordgo.ComponentEmoji{
					Name: "⚔️",
				},
			},
			discordgo.Button{
				Label:    "Heal",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "action_heal",
				Emoji: &discordgo.ComponentEmoji{
					Name: "✨",
				},
			},
		},
	},
}

var actionUndoButton = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Undo",
				Style:    discordgo.DangerButton,
				Disabled: false,
				CustomID: "action_undo",
			},
		},
	},
}

var chooseActionButton = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Choose Action",
				Style:    discordgo.PrimaryButton,
				Disabled: false,
				CustomID: "choose_action",
			},
		},
	},
}

var emptyActionGrid = []discordgo.MessageComponent{}

var rescindButton = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Rescind",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "challenge_rescind",
			},
		},
	},
}
