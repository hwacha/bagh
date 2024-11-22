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

var chooseActionOrExitGameButtonRow = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Choose Action",
				Style:    discordgo.PrimaryButton,
				Disabled: false,
				CustomID: "choose_action",
			},
			discordgo.Button{
				Label:    "Exit Match",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "exit_match",
			},
		},
	},
}

var clearNotificationButton = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Clear Notification",
				Style:    discordgo.DangerButton,
				Disabled: false,
				CustomID: "clear_notification",
			},
		},
	},
}

var emptyActionGrid = []discordgo.MessageComponent{}

var voteToDrawOrForfeitButtonRow = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Vote to Draw",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "vote_to_draw",
			},
			discordgo.Button{
				Label:    "Forfeit",
				Style:    discordgo.DangerButton,
				Disabled: false,
				CustomID: "forfeit",
			},
		},
	},
}

var withdrawVoteOrForfeitButtonRow = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Withdraw Vote",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "withdraw_vote_to_draw",
			},
			discordgo.Button{
				Label:    "Forfeit",
				Style:    discordgo.DangerButton,
				Disabled: false,
				CustomID: "forfeit",
			},
		},
	},
}

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
