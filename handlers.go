package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

var actionOptionsResponseData = discordgo.InteractionResponseData{
	Content:    chooseAnActionPrompt,
	Flags:      discordgo.MessageFlagsEphemeral,
	Components: actionButtonGrid,
}

var actionSelectedResponseData = func(action Action) discordgo.InteractionResponseData {
	return discordgo.InteractionResponseData{
		Content:    actionSelectedConfirmation(action),
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: actionUndoButton,
	}
}

func bagherRoleInGuild(s *discordgo.Session, i *discordgo.Interaction) *discordgo.Role {
	roles, _ := s.GuildRoles(i.GuildID)
	roleIndex := slices.IndexFunc(roles, func(role *discordgo.Role) bool {
		return role.Name == "bagher"
	})
	if roleIndex == -1 {
		return nil
	}
	return roles[roleIndex]
}

func userHasBAGHerRoleInGuild(s *discordgo.Session, i *discordgo.Interaction, user *discordgo.User) bool {
	member, _ := s.GuildMember(i.GuildID, user.ID)

	brig := bagherRoleInGuild(s, i)

	return brig != nil && slices.ContainsFunc(member.Roles, func(roleID string) bool {
		return brig.ID == roleID
	})
}

func findBAGHChannelInGuild(s *discordgo.Session, i *discordgo.Interaction) *discordgo.Channel {
	channels, _ := s.GuildChannels(i.GuildID)
	index := slices.IndexFunc(channels, func(ch *discordgo.Channel) bool { return ch.Name == "play-bagh" })
	if index == -1 {
		return nil
	}
	return channels[index]
}

func cleanupButtons(s *discordgo.Session, game *GameOngoing) {
	// remove the game buttons from the last round message
	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         game.LastRoundMessageID,
		Channel:    game.Thread.ID,
		Components: &emptyActionGrid,
	})

	// remove any buttons from outdated interactions from the previous round
	for _, player := range [2]*Player{&game.Challenger, &game.Challengee} {
		for _, chooseActionInteraction := range player.Interactions.ChooseAction {
			s.InteractionResponseEdit(chooseActionInteraction, &discordgo.WebhookEdit{
				Components: &emptyActionGrid,
			})
		}
		player.Interactions.ChooseAction = nil

		for _, exitGameInteraction := range player.Interactions.ExitGame {
			s.InteractionResponseEdit(exitGameInteraction, &discordgo.WebhookEdit{
				Components: &emptyActionGrid,
			})
		}
		player.Interactions.ExitGame = nil
	}
}

func handleGameActionSelection(action Action) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		actor := game.GetPlayer(presserID)

		if action == Unchosen {
			actor.ClearAction()

			actionOptionsResponseDataCopy := actionOptionsResponseData
			actionOptionsResponseDataCopy.Content = undoneSelectionChooseAnActionPrompt

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &actionOptionsResponseDataCopy,
			})

			for _, chooseActionInteraction := range actor.Interactions.ChooseAction {
				s.InteractionResponseEdit(chooseActionInteraction, &discordgo.WebhookEdit{
					Content:    &actionOptionsResponseDataCopy.Content,
					Components: &actionOptionsResponseDataCopy.Components,
				})
			}
			return
		} else {
			actor.SetAction(action)

			asrd := actionSelectedResponseData(action)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &asrd,
			})

			for _, chooseActionInteraction := range actor.Interactions.ChooseAction {
				s.InteractionResponseEdit(chooseActionInteraction, &discordgo.WebhookEdit{
					Content:    &asrd.Content,
					Components: &asrd.Components,
				})
			}
		}
		if !slices.Contains(actor.Interactions.ChooseAction, i.Interaction) {
			actor.Interactions.ChooseAction = append(actor.Interactions.ChooseAction, i.Interaction)
		}

		if game.Challenger.GetAction() != Unchosen && game.Challengee.GetAction() != Unchosen {
			game.Challenger.actionLocked = true
			game.Challengee.actionLocked = true

			cleanupButtons(s, game)

			actionLog, isGameOver, winner := game.NextStateFromActions()
			s.ChannelMessageSend(game.Thread.ID, actionLog)

			if isGameOver {
				delete(Games, game.Challenger.User.ID)
				delete(Games, game.Challengee.User.ID)

				if winner == nil {
					s.ChannelMessageSend(game.Thread.ID, "# Draw.")
				} else {
					s.ChannelMessageSend(game.Thread.ID, "# Congratulations, "+winner.User.Mention()+"!")
				}
			} else {
				msg, _ := s.ChannelMessageSendComplex(game.Thread.ID, &discordgo.MessageSend{
					Content:    game.ToString(),
					Components: chooseActionOrExitGameButtonRow,
				})

				game.LastRoundMessageID = msg.ID

				if game.Challengee.User.ID == ApplicationID {
					game.ChooseAIMove()
				}
			}
		}
	}
}

func ir(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func makeChannelAndRoleForGuild(s *discordgo.Session, guild *discordgo.Guild) (*discordgo.Channel, error) {
	// create a text channel for bagh, if it doesn't exist
	channels, _ := s.GuildChannels(guild.ID)

	var playBAGHChannel *discordgo.Channel = nil
	for _, channel := range channels {
		if channel.Name == "play-bagh" {
			playBAGHChannel = channel
			break
		}
	}

	var bagherRole *discordgo.Role = nil
	roles, _ := s.GuildRoles(guild.ID)
	for _, role := range roles {
		if role.Name == "bagher" {
			bagherRole = role
			break
		}
	}

	if bagherRole == nil {
		lightPurple := 9859481
		role, _ := s.GuildRoleCreate(guild.ID, &discordgo.RoleParams{
			Name:  "bagher",
			Color: &lightPurple, // light purple
		})

		bagherRole = role
	}
	s.GuildMemberRoleAdd(guild.ID, ApplicationID, bagherRole.ID)

	if playBAGHChannel == nil {
		// make the channel private, but allow anyone with an opt-in role
		return s.GuildChannelCreateComplex(guild.ID, discordgo.GuildChannelCreateData{
			Name: "play-bagh",
			Type: discordgo.ChannelTypeGuildText,
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{
					ID:   guild.ID,
					Type: discordgo.PermissionOverwriteTypeRole,
					Deny: discordgo.PermissionViewChannel,
				},
				{
					ID:    bagherRole.ID,
					Type:  discordgo.PermissionOverwriteTypeRole,
					Allow: discordgo.PermissionViewChannel,
				},
				{
					ID:    ApplicationID,
					Type:  discordgo.PermissionOverwriteTypeMember,
					Allow: discordgo.PermissionViewChannel,
				},
			},
		})
	} else {
		return s.ChannelEdit(playBAGHChannel.ID, &discordgo.ChannelEdit{
			PermissionOverwrites: []*discordgo.PermissionOverwrite{
				{
					ID:   guild.ID,
					Type: discordgo.PermissionOverwriteTypeRole,
					Deny: discordgo.PermissionViewChannel,
				},
				{
					ID:    bagherRole.ID,
					Type:  discordgo.PermissionOverwriteTypeRole,
					Allow: discordgo.PermissionViewChannel,
				},
				{
					ID:    ApplicationID,
					Type:  discordgo.PermissionOverwriteTypeMember,
					Allow: discordgo.PermissionViewChannel,
				},
			},
		})
	}
}

func sendRules(s *discordgo.Session, interaction *discordgo.Interaction) {
	const CHAR_LIMIT int = 2000
	data, err := os.ReadFile("rules.md")
	if err != nil {
		fmt.Println(err)
		return
	}

	lines := strings.Split(string(data), "\n")

	sendChunk := func(chunk string, once *bool) {
		if !*once {
			s.InteractionRespond(interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: chunk,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		} else {
			s.FollowupMessageCreate(interaction, false, &discordgo.WebhookParams{
				Content: chunk,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
		*once = true
	}

	chunk := ""
	numCharsInChunk := 0
	once := false
	for _, line := range lines {
		line += "\n"
		numCharsInNextLine := utf8.RuneCountInString(line)
		if numCharsInChunk+numCharsInNextLine > CHAR_LIMIT {
			sendChunk(chunk, &once)
			chunk = ""
			numCharsInChunk = 0
		}
		chunk += line
		numCharsInChunk += numCharsInNextLine
	}

	if chunk != "" {
		sendChunk(chunk, &once)
	}
}

type ApplicationCommandAndHandler struct {
	Command discordgo.ApplicationCommand
	Handler func(*discordgo.Session, *discordgo.InteractionCreate)
}

var applicationCommandsAndHandlers = func() map[string]ApplicationCommandAndHandler {
	manageServerPermission := int64(discordgo.PermissionManageServer)
	var cahs = [...]ApplicationCommandAndHandler{
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "bagh",
				Description: "brings up any salient interaction for a user.",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				// case 0: bagher role is missing.
				if bagherRoleInGuild(s, i.Interaction) == nil {
					ir(s, i, roleMissingErrorMessage)
					return
				}

				// case 1: member is not a BAGHer.
				if !userHasBAGHerRoleInGuild(s, i.Interaction, i.Interaction.Member.User) {
					ir(s, i, challengerNotBAGHerErrorMessage)
					return
				}

				session, memberHasSession := Games[i.Interaction.Member.User.ID]

				// case 2: member is not in a session
				if !memberHasSession {
					ir(s, i, issueChallengePrompt)
					return
				}

				challenge, sessionIsChallenge := session.(*AwaitingChallengeResponse)
				if sessionIsChallenge {
					// case 3: member has issued someone a challenge
					if i.Interaction.Member.User.ID == challenge.Challenger.ID {
						if !slices.ContainsFunc(challenge.ChallengerInteractions, func(ci *discordgo.Interaction) bool {
							return i.Interaction.ID == ci.ID
						}) {
							challenge.ChallengerInteractions = append(challenge.ChallengerInteractions, i.Interaction)
						}
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content:    challengeIssuedConfirmationToChallenger(challenge.Challengee),
								Flags:      discordgo.MessageFlagsEphemeral,
								Components: rescindButton,
							},
						})
					} else {
						// case 4: member has been issued a challenge by someone else
						challengeeDMChannel, _ := s.UserChannelCreate(challenge.Challengee.ID)
						ir(s, i, playerAcceptOrRefuseChallengePrompt(challenge.Challenger, challengeeDMChannel, challenge.ChallengeeMessage))
					}
				} else {
					game, _ := session.(*GameOngoing)
					if i.Interaction.ChannelID == game.Thread.ID {
						// case 5: member is in-game, in the thread, but the message has been deleted.
						if !slices.ContainsFunc(game.Thread.Messages, func(m *discordgo.Message) bool { return m.ID == game.LastRoundMessageID }) {
							s.ChannelMessageSendComplex(game.Thread.ID, &discordgo.MessageSend{
								Content:    game.ToString(),
								Components: chooseActionOrExitGameButtonRow,
							})
							ir(s, i, resendLastRoundNotification)
						} else {
							// case 6: member is in-game, in the thread.
							messageComponentHandlers["choose_action"](s, i)
						}
					} else {
						threadToConfirm, _ := s.Channel(game.Thread.ID)
						if threadToConfirm == nil {
							// case 6: member is in-game, but the thread no longer exists
							ir(s, i, gameThreadMissingErrorMessage)
						}
						// case 7: member is in-game and the thread exists, but they are outside of the thread
						ir(s, i, playerInGameRedirectToGameThread(game.Thread))
					}
				}
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "join",
				Description: "adds bagher role",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				brig := bagherRoleInGuild(s, i.Interaction)
				if brig == nil {
					ir(s, i, roleMissingErrorMessage)
				} else if userHasBAGHerRoleInGuild(s, i.Interaction, i.Member.User) {
					ir(s, i, alreadyBAGHerErrorMessage)
				} else {
					s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, brig.ID)
					ir(s, i, welcomeMessage)
				}
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "leave",
				Description: "removes bagher role",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				_, inSession := Games[i.Interaction.Member.User.ID]

				if inSession {
					ir(s, i, leaveWhenInSessionErrorMessage)
					return
				}

				brig := bagherRoleInGuild(s, i.Interaction)
				if brig == nil {
					ir(s, i, roleMissingErrorMessage)
				} else if !userHasBAGHerRoleInGuild(s, i.Interaction, i.Member.User) {
					ir(s, i, alreadyNotBAGHerErrorMessage)
				} else {
					s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, brig.ID)
					ir(s, i, goodbyeMessage)
				}
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type:                     discordgo.ChatApplicationCommand,
				Name:                     "restore",
				Description:              "restores the `play-bagh` channel, `bagher` role, and any ongoing game threads",
				DefaultMemberPermissions: &manageServerPermission,
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				guild, _ := s.Guild(i.GuildID)
				ch, err := makeChannelAndRoleForGuild(s, guild)

				if err != nil {
					ir(s, i, checkPermissionsErrorMessage)
					return
				}

				for _, session := range Games {
					challenge, isChallenge := session.(*AwaitingChallengeResponse)
					if isChallenge {
						challenge.Channel = ch
					} else {
						game, _ := session.(*GameOngoing)

						threadToConfirm, _ := s.Channel(game.Thread.ID)

						if threadToConfirm != nil {
							continue
						}

						challengerMember, _ := s.GuildMember(guild.ID, game.Challenger.User.ID)
						var challengeeMember *discordgo.Member = nil
						if game.Challengee.User.ID != ApplicationID {
							challengeeMember, _ = s.GuildMember(guild.ID, game.Challengee.User.ID)
						}

						newThread, _ := s.ThreadStart(ch.ID,
							gameThreadTitle(challengerMember, challengeeMember),
							discordgo.ChannelTypeGuildPrivateThread, 60)

						game.Thread = newThread
						msg, _ := s.ChannelMessageSendComplex(newThread.ID, &discordgo.MessageSend{
							Content:    game.ToString(),
							Components: chooseActionOrExitGameButtonRow,
						})

						game.LastRoundMessageID = msg.ID
					}

				}
				ir(s, i, restoreConfirmation)
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "rules",
				Description: "enumerates the rules of BAGH",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				sendRules(s, i.Interaction)
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type: discordgo.UserApplicationCommand,
				Name: "challenge",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				user := i.Member.User
				challenger := user
				challengee, _ := s.User(i.ApplicationCommandData().TargetID)

				if !userHasBAGHerRoleInGuild(s, i.Interaction, challenger) {
					ir(s, i, challengerNotBAGHerErrorMessage)
					return
				}

				if !userHasBAGHerRoleInGuild(s, i.Interaction, challengee) {
					ir(s, i, challengeeNotBAGHerError(challengee))
					return
				}

				if challenger.ID == challengee.ID {
					ir(s, i, selfChallengeErrorMessage)
					return
				}

				_, hasChallenger := Games[challenger.ID]

				if hasChallenger {
					ir(s, i, challengerIssuesChallengeWhileInSessionErrorMessage)
					return
				}

				playBAGHChannel := findBAGHChannelInGuild(s, i.Interaction)

				if playBAGHChannel == nil {
					ir(s, i, playBAGHChannelMissingErrorMessage)
					return
				}

				// challenge BAGH
				if challengee.ID == ApplicationID {
					// start a new thread for a game
					playBAGHChannel := findBAGHChannelInGuild(s, i.Interaction)
					if playBAGHChannel == nil {
						ir(s, i, playBAGHChannelMissingErrorMessage)
						return
					}

					challengerMember, _ := s.GuildMember(i.GuildID, challenger.ID)
					thread, _ := s.ThreadStart(playBAGHChannel.ID, gameThreadTitle(challengerMember, nil),
						discordgo.ChannelTypeGuildPrivateThread, 60)

					newGame := GameOngoing{
						Thread:             thread,
						LastRoundMessageID: "",
						Challenger:         NewPlayer(challenger),
						Challengee:         NewPlayer(challengee),
						Round:              1,
					}
					newGame.ChooseAIMove()

					msg, _ := s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
						Content:    newGame.ToString(),
						Components: chooseActionOrExitGameButtonRow,
					})

					newGame.LastRoundMessageID = msg.ID

					Games[challenger.ID] = &newGame

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: challengeAcceptNotificationForChallenger(challengee, thread),
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				_, hasChallengee := Games[challengee.ID]
				if hasChallengee {
					ir(s, i, challengeIssuedWhileChallengeeInSessionErrorMessage(challengee))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content:    challengeIssuedConfirmationToChallenger(challengee),
						Flags:      discordgo.MessageFlagsEphemeral,
						Components: rescindButton,
					},
				})

				challengeeDM, _ := s.UserChannelCreate(challengee.ID)
				challengeeMessage, _ := s.ChannelMessageSendComplex(challengeeDM.ID, &discordgo.MessageSend{
					Content:    challengeIssuedNotificationToChallengee(challenger),
					Flags:      discordgo.MessageFlagsEphemeral,
					Components: acceptOrRefuseButtonRow,
				})

				newGameSession := AwaitingChallengeResponse{
					Challenger:             challenger,
					Challengee:             challengee,
					Channel:                playBAGHChannel,
					ChallengerInteractions: []*discordgo.Interaction{i.Interaction},
					ChallengeeMessage:      challengeeMessage,
				}

				Games[challenger.ID] = &newGameSession
				Games[challengee.ID] = &newGameSession
			},
		},
	}

	res := make(map[string]ApplicationCommandAndHandler)
	for _, cah := range cahs {
		res[cah.Command.Name] = cah
	}

	return res
}()

var messageComponentHandlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
	"action_boost":  handleGameActionSelection(Boost),
	"action_attack": handleGameActionSelection(Attack),
	"action_guard":  handleGameActionSelection(Guard),
	"action_heal":   handleGameActionSelection(Heal),
	"action_undo":   handleGameActionSelection(Unchosen),
	"challenge_accept": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		acceptor := i.Interaction.User
		challenge, hasAcceptor := Games[acceptor.ID]

		if !hasAcceptor {
			ir(s, i, acceptOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			ir(s, i, acceptOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		if challengeAsChallenge.ChallengeeMessage.ID != i.Interaction.Message.ID {
			ir(s, i, acceptOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		challenger := challengeAsChallenge.Challenger
		challengerMember, _ := s.GuildMember(challengeAsChallenge.ChallengerInteractions[0].GuildID, challenger.ID)
		challengeeMember, _ := s.GuildMember(challengeAsChallenge.ChallengerInteractions[0].GuildID, acceptor.ID)

		if bagherRoleInGuild(s, challengeAsChallenge.ChallengerInteractions[0]) == nil {
			ir(s, i, roleMissingErrorMessage)
			return
		}

		if !userHasBAGHerRoleInGuild(s, challengeAsChallenge.ChallengerInteractions[0], acceptor) {
			ir(s, i, acceptorNotBAGHerErrorMessage)
			return
		}

		playBAGHChannel := findBAGHChannelInGuild(s, challengeAsChallenge.ChallengerInteractions[0])

		if playBAGHChannel == nil {
			ir(s, i, playBAGHChannelMissingErrorMessage)
			return
		}

		thread, _ := s.ThreadStart(playBAGHChannel.ID,
			gameThreadTitle(challengerMember, challengeeMember),
			discordgo.ChannelTypeGuildPrivateThread, 60)

		// make a game object and put the thread reference there
		newGame := GameOngoing{
			Thread:             thread,
			LastRoundMessageID: "",
			Challenger:         NewPlayer(challenger),
			Challengee:         NewPlayer(acceptor),
			Round:              1,
		}

		Games[challenger.ID] = &newGame
		Games[acceptor.ID] = &newGame

		msg, _ := s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
			Content:    newGame.ToString(),
			Components: chooseActionOrExitGameButtonRow,
		})

		newGame.LastRoundMessageID = msg.ID

		ir(s, i, challengeAcceptConfirmationForChallengee(challenger, thread))
		s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)

		challengerContent := challengeAcceptNotificationForChallenger(acceptor, thread)
		for _, ci := range challengeAsChallenge.ChallengerInteractions {
			s.InteractionResponseEdit(ci, &discordgo.WebhookEdit{
				Content:    &challengerContent,
				Components: &[]discordgo.MessageComponent{},
			})
		}
	},
	"challenge_refuse": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		refuser := i.Interaction.User
		challenge, hasRefuser := Games[refuser.ID]

		if !hasRefuser {
			ir(s, i, refuseOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			ir(s, i, refuseOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		if challengeAsChallenge.ChallengeeMessage.ID != i.Interaction.Message.ID {
			ir(s, i, refuseOutdatedChallengeErrorMessage)
			s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
			return
		}

		challenger := challengeAsChallenge.Challenger

		delete(Games, refuser.ID)
		delete(Games, challenger.ID)

		ir(s, i, challengeRefusedConfirmationToChallengee(challenger))
		s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)

		challengerContent := challengeRefusedNotificationToChallenger(refuser)
		challengerDMChannel, _ := s.UserChannelCreate(challenger.ID)
		s.ChannelMessageSendComplex(challengerDMChannel.ID, &discordgo.MessageSend{
			Content:    challengerContent,
			Components: clearNotificationButton,
		})

		for _, ci := range challengeAsChallenge.ChallengerInteractions {
			s.InteractionResponseEdit(ci, &discordgo.WebhookEdit{
				Content:    &challengerContent,
				Components: &emptyActionGrid,
			})
		}
	},
	"challenge_rescind": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		rescinder := i.Interaction.User
		if rescinder == nil {
			rescinder = i.Interaction.Member.User
		}
		challenge, hasRescinder := Games[rescinder.ID]

		if !hasRescinder {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content: rescindOutdatedChallengeErrorMessage,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content: rescindOutdatedChallengeErrorMessage,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		challengee := challengeAsChallenge.Challengee

		if challengee.ID == rescinder.ID {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content: rescindOutdatedChallengeErrorMessage,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		challengeRiscindedConfirmation := challengeRescindedConfirmationToChallenger(challengee)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: challengeRiscindedConfirmation,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		for _, ci := range challengeAsChallenge.ChallengerInteractions {
			s.InteractionResponseEdit(ci, &discordgo.WebhookEdit{
				Content:    &challengeRiscindedConfirmation,
				Components: &[]discordgo.MessageComponent{},
			})
		}

		challengeeContent := challengeRescindedNotificationToChallengee(rescinder)
		challengeeDMChannel, _ := s.UserChannelCreate(challengee.ID)
		s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content:    &challengeeContent,
			Components: &clearNotificationButton,
			ID:         challengeAsChallenge.ChallengeeMessage.ID,
			Channel:    challengeeDMChannel.ID,
		})

		delete(Games, rescinder.ID)
		delete(Games, challengee.ID)
	},
	"choose_action": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		player := game.GetPlayer(presserID)

		player.Interactions.ChooseAction = append(player.Interactions.ChooseAction, i.Interaction)

		var responseData discordgo.InteractionResponseData

		if player.GetAction() == Unchosen {
			responseData = actionOptionsResponseData
		} else {
			responseData = actionSelectedResponseData(player.GetAction())
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &responseData,
		})
	},
	"exit_game": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		player := game.GetPlayer(presserID)

		player.Interactions.ExitGame = append(player.Interactions.ExitGame, i.Interaction)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    exitGamePrompt,
				Components: voteToDrawOrForfeitButtonRow,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	},
	"forfeit": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		forfeiter := game.GetPlayer(presserID).User
		winner := game.GetOtherPlayer(presserID).User

		cleanupButtons(s, game)

		// confirm forfeit
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    forfeitConfirmation,
				Components: emptyActionGrid,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})

		// remove game session
		delete(Games, forfeiter.ID)
		delete(Games, winner.ID)

		// notify thread of forfeit and winner
		s.ChannelMessageSend(game.Thread.ID, forfeitNotification(forfeiter, winner))
	},
	"vote_to_draw": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		voter := game.GetPlayer(presserID)
		voter.votedToDraw = true

		otherPlayer := game.GetOtherPlayer(presserID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    votedToDrawConfirmation,
				Components: withdrawVoteOrForfeitButtonRow,
			},
		})

		votedToDrawConfirmationVar := votedToDrawConfirmation

		for _, exitGameInteraction := range voter.Interactions.ExitGame {
			s.InteractionResponseEdit(exitGameInteraction, &discordgo.WebhookEdit{
				Content:    &votedToDrawConfirmationVar,
				Components: &withdrawVoteOrForfeitButtonRow,
			})
		}

		s.ChannelMessageSend(game.Thread.ID, votedToDrawNotification(voter.User))

		if otherPlayer.votedToDraw {
			cleanupButtons(s, game)
			delete(Games, voter.User.ID)
			delete(Games, otherPlayer.User.ID)
			s.ChannelMessageSend(game.Thread.ID, voteToDrawPassesNotification)
		}
	},
	"withdraw_vote_to_draw": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		presserID := i.Interaction.Member.User.ID
		game, found := Games[presserID].(*GameOngoing)

		if !(found && game.Thread.ID == i.Interaction.ChannelID) {
			ir(s, i, nonPlayerUsesInGameCommandErrorMessage)
			return
		}

		voter := game.GetPlayer(presserID)
		voter.votedToDraw = false

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    voteToDrawWithdrawnConfirmation,
				Components: voteToDrawOrForfeitButtonRow,
			},
		})

		voteToDrawWithdrawnConfirmationVar := voteToDrawWithdrawnConfirmation

		for _, exitGameInteraction := range voter.Interactions.ExitGame {
			s.InteractionResponseEdit(exitGameInteraction, &discordgo.WebhookEdit{
				Content:    &voteToDrawWithdrawnConfirmationVar,
				Components: &voteToDrawOrForfeitButtonRow,
			})
		}

		s.ChannelMessageSend(game.Thread.ID, voteToDrawWithdrawnNotification(voter.User))
	},
	"clear_notification": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
	},
}

func handleApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		applicationCommandsAndHandlers[i.ApplicationCommandData().Name].Handler(s, i)
	case discordgo.InteractionMessageComponent:
		buttonID := i.MessageComponentData().CustomID
		messageComponentHandlers[buttonID](s, i)
	}
}

func handleGuildCreate(s *discordgo.Session, gc *discordgo.GuildCreate) {
	ch, err := makeChannelAndRoleForGuild(s, gc.Guild)

	// register application commands
	for _, commandAndHandler := range applicationCommandsAndHandlers {
		s.ApplicationCommandCreate(ApplicationID, gc.Guild.ID, &commandAndHandler.Command)
	}

	// pin a message about BAGH options to the play-bagh channel
	if err == nil {
		msg, messageError := s.ChannelMessageSend(ch.ID, baghOptions)

		if messageError != nil {
			pinErr := s.ChannelMessagePin(ch.ID, msg.ID)
			if pinErr != nil {
				fmt.Println(pinErr)
			}
		}
	} else {
		fmt.Println(err)
	}

}

func handleReady(s *discordgo.Session, ready *discordgo.Ready) {
}

func handleGuildMemberRemove(s *discordgo.Session, gmr *discordgo.GuildMemberRemove) {
	session, hasSession := Games[gmr.Member.User.ID]
	if hasSession {
		delete(Games, gmr.Member.User.ID)
		challenge, isChallenge := session.(*AwaitingChallengeResponse)

		var dmChannel *discordgo.Channel

		if isChallenge {
			if challenge.Challenger.ID == gmr.Member.User.ID {
				delete(Games, challenge.Challengee.ID)
				dmChannel, _ = s.UserChannelCreate(challenge.Challengee.ID)
				s.ChannelMessageDelete(dmChannel.ID, challenge.ChallengeeMessage.ID)
				s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
					Content:    memberRemovedNotification(challenge.Challenger),
					Components: clearNotificationButton,
				})
			} else {
				delete(Games, challenge.Challenger.ID)
				dmChannel, _ = s.UserChannelCreate(challenge.Challenger.ID)
				s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
					Content:    memberRemovedNotification(challenge.Challengee),
					Components: clearNotificationButton,
				})
			}
		} else {
			game, _ := session.(*GameOngoing)
			leaver := game.GetPlayer(gmr.Member.User.ID)
			stayer := game.GetOtherPlayer(gmr.Member.User.ID)
			delete(Games, stayer.User.ID)
			cleanupButtons(s, game)
			dmChannel, _ = s.UserChannelCreate(stayer.User.ID)
			s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
				Content:    memberRemovedNotification(leaver.User),
				Components: clearNotificationButton,
			})
			s.ChannelMessageSend(game.Thread.ID, memberRemovedNotification(leaver.User))
		}
	}
}

func handleGuildLeave(_ *discordgo.Session, gd *discordgo.GuildDelete) {
	for id, session := range Games {
		challenge, isChallenge := session.(*AwaitingChallengeResponse)
		if isChallenge && challenge.Channel.GuildID == gd.Guild.ID {
			defer delete(Games, id)
		} else {
			game, _ := session.(*GameOngoing)
			if game.Thread.GuildID == gd.Guild.ID {
				defer delete(Games, id)
			}
		}
	}
}
