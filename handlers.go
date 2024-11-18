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
	return roles[slices.IndexFunc(roles, func(role *discordgo.Role) bool { return role.Name == "bagher" })]
}

func userHasBAGHerRoleInGuild(s *discordgo.Session, i *discordgo.Interaction, user *discordgo.User) bool {
	member, err := s.GuildMember(i.GuildID, user.ID)
	if err != nil {
		panic(err)
	}
	return slices.ContainsFunc(member.Roles, func(role string) bool {
		return bagherRoleInGuild(s, i).ID == role
	})
}

func findBAGHChannelInGuild(s *discordgo.Session, i *discordgo.Interaction) *discordgo.Channel {
	channels, _ := s.GuildChannels(i.GuildID)
	return channels[slices.IndexFunc(channels, func(ch *discordgo.Channel) bool { return ch.Name == "play-bagh" })]
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

			for _, chooseActionInteraction := range actor.ChooseActionInteractions {
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

			for _, chooseActionInteraction := range actor.ChooseActionInteractions {
				s.InteractionResponseEdit(chooseActionInteraction, &discordgo.WebhookEdit{
					Content:    &asrd.Content,
					Components: &asrd.Components,
				})
			}
		}
		if !slices.Contains(actor.ChooseActionInteractions, i.Interaction) {
			actor.ChooseActionInteractions = append(actor.ChooseActionInteractions, i.Interaction)
		}

		if game.Challenger.GetAction() != Unchosen && game.Challengee.GetAction() != Unchosen {

			// remove the "choose action" button from the last message
			s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         game.LastRoundMessageID,
				Channel:    game.Thread.ID,
				Components: &emptyActionGrid,
			})

			game.Challenger.actionLocked = true
			game.Challengee.actionLocked = true

			// remove any buttons from outdated interactions from the previous round
			for _, player := range [2]*Player{&game.Challenger, &game.Challengee} {
				for _, chooseActionInteraction := range player.ChooseActionInteractions {
					s.InteractionResponseEdit(chooseActionInteraction, &discordgo.WebhookEdit{
						Components: &emptyActionGrid,
					})
				}
				player.ChooseActionInteractions = nil
			}

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
					Components: chooseActionButton,
				})

				game.LastRoundMessageID = msg.ID

				if game.Challengee.User.ID == APPLICATION_ID {
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

func makeChannelAndRoleForGuild(s *discordgo.Session, guild *discordgo.Guild) {
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

	s.GuildMemberRoleAdd(guild.ID, APPLICATION_ID, bagherRole.ID)

	if playBAGHChannel == nil {
		// make the channel private, but allow anyone with an opt-in role
		s.GuildChannelCreateComplex(guild.ID, discordgo.GuildChannelCreateData{
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
			},
		})
	} else {
		s.ChannelEdit(playBAGHChannel.ID, &discordgo.ChannelEdit{
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
	var cahs = [...]ApplicationCommandAndHandler{
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "bagh",
				Description: "brings up any salient interaction for a user.",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
					// case 5: member is in-game, in the thread
					if i.Interaction.ChannelID == game.Thread.ID {
						messageComponentHandlers["choose_action"](s, i)
					} else {
						// case 6: member is in-game, outside of the thread
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
				s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, bagherRoleInGuild(s, i.Interaction).ID)
				ir(s, i, welcomeMessage)
			},
		},
		{
			Command: discordgo.ApplicationCommand{
				Type:        discordgo.ChatApplicationCommand,
				Name:        "leave",
				Description: "removes bagher role",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, bagherRoleInGuild(s, i.Interaction).ID)
				ir(s, i, goodbyeMessage)
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

				// challenge BAGH
				if challengee.ID == APPLICATION_ID {
					// start a new thread for a game
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
						Components: chooseActionButton,
					})

					newGame.LastRoundMessageID = msg.ID

					Games[challenger.ID] = &newGame

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: challengeAcceptNotificationForChallenger(nil, thread),
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
	"challenge_accept": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		acceptor := i.Interaction.User
		challenge, hasAcceptor := Games[acceptor.ID]

		if !hasAcceptor {
			ir(s, i, challengeAcceptedNoChallengeErrorMessage)
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			ir(s, i, challengeAcceptedWhileInGameErrorMessage)
			return
		}

		if challengeAsChallenge.Challenger.ID == acceptor.ID {
			ir(s, i, selfAcceptChallengeErrorMessage)
			return
		}

		challenger := challengeAsChallenge.Challenger
		challengerMember, _ := s.GuildMember(challengeAsChallenge.ChallengerInteractions[0].GuildID, challenger.ID)
		challengeeMember, _ := s.GuildMember(challengeAsChallenge.ChallengerInteractions[0].GuildID, acceptor.ID)

		thread, _ := s.ThreadStart(challengeAsChallenge.Channel.ID,
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
			Components: chooseActionButton,
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
			fmt.Println("assertion failure: challenge_refuse called with refuser not challenged.")
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			fmt.Println("assertion failure: challenge_refuse called during ongoing game.")
			return
		}

		challenger := challengeAsChallenge.Challenger

		if challenger.ID == refuser.ID {
			fmt.Println("assertion failure: challenge_refuse called by challenger.")
			return
		}

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
		challenge, hasRetractor := Games[rescinder.ID]

		if !hasRetractor {
			fmt.Println("assertion failure: challenge_rescind called with rescinder not issuing challenge.")
			return
		}

		challengeAsChallenge, isChallenge := challenge.(*AwaitingChallengeResponse)
		if !isChallenge {
			fmt.Println("assertion failure: challenge_rescind called during ongoing game.")
			return
		}

		challengee := challengeAsChallenge.Challengee

		if challengee.ID == rescinder.ID {
			fmt.Println("assertion failure: challenge_rescind called by challengee.")
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
		_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content:    &challengeeContent,
			Components: &clearNotificationButton,
			ID:         challengeAsChallenge.ChallengeeMessage.ID,
			Channel:    challengeeDMChannel.ID,
		})
		if err != nil {
			fmt.Println(err)
		}

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

		player.ChooseActionInteractions = append(player.ChooseActionInteractions, i.Interaction)

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
	makeChannelAndRoleForGuild(s, gc.Guild)
}

func handleReady(s *discordgo.Session, ready *discordgo.Ready) {
	for _, guild := range ready.Guilds {
		// create a text channel for bagh if it doesn't exist
		makeChannelAndRoleForGuild(s, guild)

		// register application commands
		for _, commandAndHandler := range applicationCommandsAndHandlers {
			s.ApplicationCommandCreate(APPLICATION_ID, guild.ID, &commandAndHandler.Command)
		}
	}
}
