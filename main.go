package main

import (
	"strings"
	"strconv"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"unicode/utf8"
	"syscall"
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
)

const (
	APPLICATION_ID = "1291027616702402632"
	PLAY_BAGH_ID = "1291052523439783977"
	GUILD_ID = "320038510570504192"
)

type SessionState interface {
	isSessionState()
}

type SessionStateAwaitingChallengeResponse struct {
	Challenger *discordgo.User
	Challengee *discordgo.User

	ChallengerInteraction *discordgo.Interaction
	ChallengeeMessage     *discordgo.Message
}
func (a *SessionStateAwaitingChallengeResponse) isSessionState() {}

type Action int

const (
	Boost Action = iota
	Attack
	Guard
	Heal
	Unchosen
)

var actionStrings = map[Action]string{
	Boost:  "⬆️ **BOOST** ⬆️",
	Attack: "⚔️ **ATTACK** ⚔️",
	Guard:  "🛡️ **GUARD** 🛡️",
	Heal:   "✨ **HEAL** ✨",
}

var actionCommands = map[Action]string{
	Boost:  "!boost",
	Attack: "!attack",
	Guard:  "!guard",
	Heal:   "!heal",
}

func MakeActionOptionList () string {
	var list = "You can choose any of the following actions:\n"
	for _, action := range [4]Action{Boost, Attack, Guard, Heal} {
		list += "- " + actionStrings[action] + " (`" + actionCommands[action] + "`)\n"
	}
	return list
}

type Player struct {
	User *discordgo.User
	HP int
	ShieldBreakCounter int
	Advantage int
	Boost int
	currentAction Action
	actionLocked bool
}

func NewPlayer(u *discordgo.User) Player {
	return Player{ User:u, HP:BASE_MAX_HEALTH, Advantage:0, Boost:0, currentAction:Unchosen, actionLocked: false }
}

func (p Player) GetAction() Action {
	return p.currentAction
}

func (p *Player) SetAction(a Action) bool {
	if p.currentAction != Unchosen {
		return false
	}
	p.currentAction = a
	return true
}

func (p *Player) UnlockAction() {
	p.actionLocked = false
}

func (p *Player) ClearAction() bool {
	if !p.actionLocked && p.currentAction != Unchosen {
		p.currentAction = Unchosen
		return true
	}
	return false
}

type SessionStateGameOngoing struct {
	Thread *discordgo.Channel
	DiscordState *discordgo.State
	Challenger Player
	Challengee Player
	Round int
}
func (o *SessionStateGameOngoing) isSessionState() {}

type SessionStateGameOver struct {}
func (c *SessionStateGameOver) isSessionState() {}

var (
	CommandLine bool
	Secret bool
	Token string
	Games = make(map[string]SessionState)
)

func (game *SessionStateGameOngoing) GetPlayer(userID string) *Player {
	if game.Challenger.User.ID == userID {
		return &game.Challenger
	}
	if game.Challengee.User.ID == userID {
		return &game.Challengee
	}
	return nil
}

// returns whether the game ended, if it was a draw,
// and if not, who the winner and loser was
func (game *SessionStateGameOngoing) IsGameOver() (bool, *Player) {
	if game.Challenger.HP > 0 && game.Challengee.HP > 0 {
		return false, nil
	}
	if game.Challenger.HP <= 0 && game.Challengee.HP <= 0 {
		return true, nil
	}
	if game.Challenger.HP > 0 {
		return true, &game.Challenger
	}
	if game.Challengee.HP > 0 {
		return true, &game.Challengee
	}

	// this code shouldn't be reached. If it is, end the game in a draw.
	return true, nil
}

const (
	BASE_MAX_HEALTH int = 3
	MAX_BOOST int = 6
)

func (game *SessionStateGameOngoing) NextStateFromActions() (string, bool, *Player) {
	gainedOrRetainedAdvantage := make(map[*Player]bool)
	shieldJustBroke := make(map[*Player]bool)

	players := [2]*Player{ &game.Challenger, &game.Challengee }

	actionLog := ""

	// Initial Phase
	for _, player := range players {
		playerAction  := player.GetAction()
		playerMention := player.User.Mention()

		if player.ShieldBreakCounter > 0 {
			roll := rand.Float32()
			if roll < 1.0 / float32(player.ShieldBreakCounter + 1) {
				player.ShieldBreakCounter = 0
			}

			if player.ShieldBreakCounter == 0 {
				actionLog += playerMention + "'s shield is **mended**! "
			} else {
				actionLog += playerMention + "'s shield is still broken. "
			}
		}

		if playerAction == Boost {
			if player.Boost < MAX_BOOST {
				player.Boost += 1
				actionLog += playerMention + " " + actionStrings[Boost] + "s to " + strconv.Itoa(player.Boost) + ". "	
			} else {
				actionLog += playerMention + " " + actionStrings[Boost] + "s, preserving a boost of " + strconv.Itoa(player.Boost) + ". "
			}
		}
	}

	type ActionInfo struct {
		Agent   *Player
		Patient *Player
	}

	playerRelations := [2]ActionInfo{
		ActionInfo{
			Agent:   &game.Challenger,
			Patient: &game.Challengee,
		},
		ActionInfo{
			Agent:   &game.Challengee,
			Patient: &game.Challenger,
		},
	}

	delayString := ""

	// Middle Phase
	for _, playerRelation := range playerRelations {
		agent   := playerRelation.Agent
		patient := playerRelation.Patient
		
		agentAction   :=   agent.GetAction()
		patientAction := patient.GetAction()

		agentMention   :=   agent.User.Mention()
		patientMention := patient.User.Mention()

		agentHasAdvantage   :=   agent.Advantage > patient.Advantage
		patientHasAdvantage := patient.Advantage > agent.Advantage
		// positive if agent has more boost
		// negative if patient has more boost
		// 0 if equal boost
		boostDifferential := agent.Boost - patient.Boost

		switch agentAction {
		case Attack:
			attackGoesThrough := true
			switch patientAction {
			case Attack:
				if patientHasAdvantage { // attack has no effect
					attackGoesThrough = false

					attackString := actionStrings[Attack]
					if agent.Boost > 0 {
						attackString = "boosted " + attackString
					}
					delayString += patientMention + "'s counterattack renders " + agentMention + "'s " + attackString + " impotent. "
				}
				break
			case Guard:
				if patient.ShieldBreakCounter > 0 { // shield is broken
					actionLog += agentMention + " attacks, and " + patientMention + " " + actionStrings[Guard] + "s, but the shield is broken. "
				} else { // shield not broken
					attackGoesThrough = false
					attackString := actionStrings[Attack] + "s"
					if agent.Boost > 0 {
						attackString += " with a boost of " + strconv.Itoa(agent.Boost)
					}
					guardString := actionStrings[Guard] + "s"
					if patient.Boost > 0 {
						guardString += " with a boost of " + strconv.Itoa(patient.Boost)
					}
					actionLog += agentMention + " " + attackString + ", but " + patientMention + " " + guardString + " and prevents damage. "
					// agent has higher boost
					if boostDifferential > 0 {
						// actionLog += "Because " + agentMention + " has higher boost than " + patientMention + ", " + patientMention + " gains no advantage. "

						patient.ShieldBreakCounter = boostDifferential
						shieldJustBroke[patient] = true
						actionLog += patientMention + "'s shield **breaks**! Its damage is at " + strconv.Itoa(patient.ShieldBreakCounter) + ". "
					} else if agentHasAdvantage { // agent has advantage
						// actionLog += "Because " + agentMention + " has advantage over " + patientMention + ", " + patientMention + " gains no advantage. "
					} else { // patient gains or retains advantage
						oldAdvantage := patient.Advantage
						patient.Advantage = max(patient.Advantage, (-1 * boostDifferential) + 1)

						actionLog += patientMention
						if oldAdvantage == patient.Advantage {
							actionLog += " retains advantage at "
						} else {
							actionLog += " gains advantage up to "
						}
						actionLog += strconv.Itoa(patient.Advantage) + ". "
						gainedOrRetainedAdvantage[patient] = true
					}
				}
				break
			case Heal:
				// heal is interrupted
				delayString += patientMention + "'s " + actionStrings[Heal] + "ing is **interrupted** by " + agentMention + "'s attack. "
				break
			}
			if attackGoesThrough {
				damage := 1 + agent.Boost

				patient.HP -= damage
				patient.HP = max(patient.HP, 0)

				actionLog += agentMention + " " + actionStrings[Attack] + "s for "
				if agent.Boost > 0 {
					actionLog += "a boosted "
				}
				actionLog += strconv.Itoa(damage) + " damage"
				if agentHasAdvantage {
					actionLog += " with advantage"
				}

				actionLog += ". "
			}
			break
		case Guard:
			if patientAction != Attack {
				// no effect
				actionLog += agentMention + " " + actionStrings[Guard] + "s to no effect. "
			}
			break
		case Heal:
			if patientAction != Attack { // heal not interrupted
				maxOverheal := BASE_MAX_HEALTH + 1 + agent.Boost
				newHP := min(agent.HP + 1 + agent.Boost, maxOverheal)

				actionLog += agentMention + " " + actionStrings[Heal] + "s"

				if agent.HP >= newHP { // no effect
					actionLog += " to no effect. "
				} else {
					diff := newHP - agent.HP
					agent.HP = newHP

					actionLog += " by " + strconv.Itoa(diff) + " to "

					if newHP > BASE_MAX_HEALTH {
						actionLog += "an overheal of "
					}

					actionLog += strconv.Itoa(newHP) + ". "
				}
			}
		}
	}

	actionLog += delayString

	// determine end game
	isOver, winner := game.IsGameOver()

	secondString := ""
	thirdString := ""

	// End Phase
	for _, player := range players {
		playerAction := player.GetAction()
		playerMention := player.User.Mention()
		if playerAction != Boost {
			if player.Boost > 0 {
				player.Boost = 0
				if !isOver {
					actionLog += playerMention + "'s boost is expended to 0. "	
				}
			}
		}

		if !isOver && !gainedOrRetainedAdvantage[player] && player.Advantage > 0 {
			player.Advantage--
			secondString += playerMention + "'s advantage falls to " + strconv.Itoa(player.Advantage) + ". "
		}

		if !isOver && player.ShieldBreakCounter > 0 {
			if !shieldJustBroke[player] {
				player.ShieldBreakCounter--
			}
			if player.ShieldBreakCounter == 0 {
				thirdString += playerMention + "'s shield is **mended**! "
			} else {
				thirdString += "The chance of " + playerMention + "'s shield mending next turn is **1 in " + strconv.Itoa(player.ShieldBreakCounter + 1) + "**. "
			}
		}

		player.UnlockAction() // TODO move this out of the scope of next state
		player.ClearAction()
	}
	actionLog += secondString
	actionLog += thirdString

	if isOver {
		if winner == nil {
			actionLog += "Both players have lost all health in the same turn, resulting in a **draw**."
		} else {
			actionLog += winner.User.Mention() + " secures **victory**!"
		}
	} else {
		game.Round++
	}
	
	return actionLog, isOver, winner
}

func (game *SessionStateGameOngoing) ToString() string {
	gameString := "# Round " + strconv.Itoa(game.Round) + "\n"
	for _, player := range [2]Player{game.Challenger, game.Challengee} {
		shield := " 🛡️"
		if player.ShieldBreakCounter == 0 {
			shield += "✔️ "
		} else {
			shield += "❌ "
		}
		boost := ""
		if player.Boost > 0 {
			boost = " ⬆️"
			if player.Boost > 1 {
				boost += "x" + strconv.Itoa(player.Boost)
			}
		}
		advantage := ""
		if player.Advantage > 0 {
			advantage = " [Adv."
			if player.Advantage > 1 {
				advantage += "x" + strconv.Itoa(player.Advantage)
			}
			advantage += "]"
		}
		gameString += "🤺 " + player.User.Mention() + ": ❤️x" + strconv.Itoa(player.HP) + shield + boost + advantage + "\n"
	}
	return gameString
}

func (game *SessionStateGameOngoing) PromptActionString(s *discordgo.Session) string {
	str := "You may now choose an action for **Round " + strconv.Itoa(game.Round) + "** in your DMs.\n"

	challengerDMChannel, _ := s.UserChannelCreate(game.Challenger.User.ID)
	challengeeDMChannel, _ := s.UserChannelCreate(game.Challengee.User.ID)

	str += game.Challenger.User.Mention() + ", click here: <#" + challengerDMChannel.ID + ">\n"
	str += game.Challengee.User.Mention() + ", click here: <#" + challengeeDMChannel.ID + ">\n"

	return str
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.BoolVar(&CommandLine, "c", false, "Play on command line")
	flag.BoolVar(&Secret, "s", false, "Make command line action inputs secret")
	flag.Parse()
}

func runGameCommandLine() {
	var p1ActionString, p2ActionString string
	var p1Action, p2Action Action = Unchosen, Unchosen

	p1 := NewPlayer(&discordgo.User{ID: "1"})
	p2 := NewPlayer(&discordgo.User{ID: "2"})
	game := SessionStateGameOngoing{Thread: nil, Challenger: p1, Challengee: p2, Round: 1}

	redact := func () {
		fmt.Print("\033[A")
		fmt.Print("\033[4C")
		fmt.Print("[action]")
		fmt.Println()
	}

	for {
		fmt.Println(game.ToString())
		for {
			fmt.Print("p1: ")
			fmt.Scanln(&p1ActionString)
			switch p1ActionString {
			case "b", "boost":
				p1Action = Boost
				break
			case "a", "attack":
				p1Action = Attack
				break
			case "g", "guard":
				p1Action = Guard
				break
			case "h", "heal":
				p1Action = Heal
				break
			default:
				fmt.Println("Invalid.")
				continue
			}
			if Secret {
				redact()
			}
			break
		}

		for {
			fmt.Print("p2: ")
			fmt.Scanln(&p2ActionString)
			switch p2ActionString {
			case "b", "boost":
				p2Action = Boost
				break
			case "a", "attack":
				p2Action = Attack
				break
			case "g", "guard":
				p2Action = Guard
				break
			case "h", "heal":
				p2Action = Heal
				break
			default:
				fmt.Println("Invalid.")
				continue
			}
			if Secret {
				redact()
			}
			break
		}

		game.Challenger.SetAction(p1Action)
		game.Challengee.SetAction(p2Action)

		actionLog, isOver, _ := game.NextStateFromActions()

		fmt.Println(actionLog)

		if isOver {
			break
		}
	}

}

func main() {
	if CommandLine {
		runGameCommandLine()
		return
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)
	dg.AddHandler(handleApplicationCommand)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsGuildMembers

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

func sendRules(s *discordgo.Session, userID string) {
	data, err := os.ReadFile("rules.md")
	if err != nil {
		fmt.Println(err)
		return
	}

	lines := strings.Split(string(data), "\n")
	
	dm, _ := s.UserChannelCreate(userID)
	sendChunk := func (chunk string) {
		s.ChannelMessageSend(dm.ID, chunk)
	}

	chunk := ""
	numCharsInChunk := 0
	
	for _, line := range lines {
		line += "\n"
		numCharsInNextLine := utf8.RuneCountInString(line)
		if numCharsInChunk + numCharsInNextLine > 2000 {
			sendChunk(chunk)
			chunk = ""
			numCharsInChunk = 0
		}
		chunk += line
		numCharsInChunk += numCharsInNextLine
	}

	if chunk != "" {
		sendChunk(chunk)
	}
}

func ready(s *discordgo.Session, ready *discordgo.Ready) {
	// const me = "186296587914313728"
	// myDM, _ := s.UserChannelCreate(me)

	s.ApplicationCommandCreate(APPLICATION_ID, GUILD_ID, &discordgo.ApplicationCommand{
		Type: 2,
		Name: "challenge",
	})

	s.ApplicationCommandCreate(APPLICATION_ID, GUILD_ID, &discordgo.ApplicationCommand{
		Type:        1,
		Name:        "action",
		Description: "Choose an action for the current round",
	})
}

func handleApplicationCommand (s *discordgo.Session, i *discordgo.InteractionCreate) {
	ir := func (content string) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}

	actionOptionsResponseData := discordgo.InteractionResponseData{
		Content: "Choose one of the following actions.",
		Flags: discordgo.MessageFlagsEphemeral,
		Components: []discordgo.MessageComponent{
			// ActionRow is a container of all buttons within the same row.
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "Boost",
						Style: discordgo.SecondaryButton,
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
		},
	}

	user := i.User
	if user == nil {
		user = i.Member.User
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "challenge":
			challenger := user
			challengee, _ := s.User(i.ApplicationCommandData().TargetID)

			if challenger.ID == challengee.ID {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You can't challenge yourself!",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})

				return
			}

			_, hasChallenger := Games[challenger.ID]
			_, hasChallengee := Games[challengee.ID]

			if hasChallengee {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: challengee.Mention() + " is busy. Try challenging them later.",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			if hasChallenger {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You're already busy. Try again after your game is done.",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You have challenged " + challengee.Mention() + ".",
					Flags: discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label: "Rescind",
									Style: discordgo.SecondaryButton,
									Disabled: false,
									CustomID: "challenge_rescind",
								},
							},
						},
					},
				},
			})

			acceptOrRefuseRow := func (prefix string) (discordgo.ActionsRow) {
				return discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label: "Accept",
							Style: discordgo.PrimaryButton,
							Disabled: false,
							CustomID: prefix + "_accept",
						},
						discordgo.Button{
							Label:    "Refuse",
							Style:    discordgo.SecondaryButton,
							Disabled: false,
							CustomID: prefix + "_refuse",
						},
					},
				}
			}

			challengeeDM, _ := s.UserChannelCreate(challengee.ID)
			challengeeMessage, _ := s.ChannelMessageSendComplex(challengeeDM.ID, &discordgo.MessageSend{
				Content: challenger.Mention() + " has challenged you to a game of BAGH.",
				Flags: discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					acceptOrRefuseRow("challenge"),
				},
			})

			newGameSession := SessionStateAwaitingChallengeResponse{
				Challenger: challenger,
				Challengee: challengee,
				ChallengerInteraction: i.Interaction,
				ChallengeeMessage: challengeeMessage,
			}

			Games[challenger.ID] = &newGameSession
			Games[challengee.ID] = &newGameSession

			return
		case "action":
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &actionOptionsResponseData,
			})
		}
	case discordgo.InteractionMessageComponent:
		buttonID := i.MessageComponentData().CustomID
		if strings.HasPrefix(buttonID, "action_") {
			action := Unchosen
			switch buttonID {
			case "action_boost":
				action = Boost
			case "action_attack":
				action = Attack
			case "action_guard":
				action = Guard
			case "action_heal":
				action = Heal
			case "action_undo":
				action = Unchosen
			default:
				fmt.Println("error: action button ID not recognized.")
				return
			}

			if action == Unchosen {
				actionOptionsResponseDataCopy := actionOptionsResponseData
				actionOptionsResponseDataCopy.Content = "You have undone your selection. " + actionOptionsResponseDataCopy.Content
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &actionOptionsResponseDataCopy,
				})

				return
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You have chosen to " + actionStrings[action] + ".",
					Flags: discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label: "Undo",
									Style: discordgo.DangerButton,
									Disabled: false,
									CustomID: "action_undo",
								},
							},
						},
					},
				},
			})
		} else {
			switch buttonID {
			case "challenge_accept":
				acceptor := i.Interaction.User
				challenge, hasAcceptor := Games[acceptor.ID]

				if !hasAcceptor {
					ir("No one is challenging you.")
					return
				}

				challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
				if !isChallenge {
					ir("You're in the middle of a game already.")
					return
				}

				if challengeAsChallenge.Challenger.ID == acceptor.ID {
					ir("You can't accept your own challenge.")
					return
				}

				challenger := challengeAsChallenge.Challenger
				
				// // start a new thread for a game
				// thread, err := s.ThreadStart(m.ChannelID,
				// 	challenger.Username + "'s BAGH Game Against " + acceptor.Username,
				// 	discordgo.ChannelTypeGuildPrivateThread, 60)
				// if err != nil {
				// 	s.ChannelMessageSendReply(m.ChannelID, "There was a problem starting a thread.", m.Reference())
				// 	fmt.Println(err)
				// 	return
				// }
				// s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + " has accepted " + challenger.Mention() + "'s challenge. Check for a new game thread and your DMs.", m.Reference())

				// make a game object and put the thread reference there
				newGame := SessionStateGameOngoing{Thread: nil, Challenger: NewPlayer(challenger), Challengee: NewPlayer(acceptor), Round: 1}

				Games[challenger.ID] = &newGame
				Games[acceptor.ID] = &newGame

				// s.ChannelMessageSend(thread.ID, newGame.ToString() + newGame.PromptActionString(s))
				
				// DM each player and ask them for an action
				// const dmIntroString = "Welcome to BAGH! Your chosen action is hidden until both players have made a move. So, you can type your action for the round here."
				// linkToGame := "You can view the game here: <#" + thread.ID + ">"

				// challengerDMChannel, _ := s.UserChannelCreate(challenger.ID)
				
				// s.ChannelMessageSend(challengerDMChannel.ID, dmIntroString + "\n\n" + linkToGame + "\n\n" + MakeActionOptionList())

				// acceptorDMChannel, _ := s.UserChannelCreate(acceptor.ID)
				// s.ChannelMessageSend(acceptorDMChannel.ID, dmIntroString + "\n\n" + linkToGame + "\n\n" + MakeActionOptionList())

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You have accepted " + challenger.Mention() + "'s challenge!",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				s.ChannelMessageDelete(i.Interaction.ChannelID, i.Interaction.Message.ID)
				return
			case "challenge_refuse":
				refuser := i.Interaction.User
				challenge, hasRefuser := Games[refuser.ID]

				if !hasRefuser {
					fmt.Println("assertion failure: challenge_refuse called with refuser not challenged.")
					return
				}

				challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
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

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "You have refused " + challenger.Mention() + "'s challenge.",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})

				challengerContent := refuser.Mention() + " has refused your challenge."

				s.InteractionResponseEdit(challengeAsChallenge.ChallengerInteraction, &discordgo.WebhookEdit{
					Content: &challengerContent,
					Components: &[]discordgo.MessageComponent{},
				})

				return
			case "challenge_rescind":
				rescinder := i.Interaction.User
				if rescinder == nil {
					rescinder = i.Interaction.Member.User
				}
				challenge, hasRetractor := Games[rescinder.ID]

				if !hasRetractor {
					fmt.Println("assertion failure: challenge_rescind called with rescinder not issuing challenge.")
					return
				}

				challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
				if !isChallenge {
					fmt.Println("assertion failure: challenge_rescind called during ongoing game.")
					return
				}

				challengee := challengeAsChallenge.Challengee

				if challengee.ID == rescinder.ID {
					fmt.Println("assertion failure: challenge_rescind called by challengee.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Content: "You have rescinded your challenge to " + challengee.Mention() + ".",
						Flags: discordgo.MessageFlagsEphemeral,
					},
				})
				challengeeContent := rescinder.Mention() + " has rescinded their challenge."
				challengeeDMChannel, _ := s.UserChannelCreate(challengee.ID)
				_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
					Content: &challengeeContent,
					Components: &[]discordgo.MessageComponent{},
					ID: challengeAsChallenge.ChallengeeMessage.ID,
					Channel: challengeeDMChannel.ID,
				})
				if err != nil {
					fmt.Println(err)
				}

				delete(Games, rescinder.ID)
				delete(Games, challengee.ID)
				return
			}
		}
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.ChannelID == PLAY_BAGH_ID {
		commandAndArgs := strings.Split(m.Content, " ")
		command := commandAndArgs[0]

		switch command {
		case "!rules":
			if len(commandAndArgs) != 1 {
				return
			}
			sendRules(s, m.Author.ID)
			ch, _ := s.UserChannelCreate(m.Author.ID)
			s.ChannelMessageSendReply(m.ChannelID, "I've sent you a DM with the rules. See it here: <#" + ch.ID + ">", m.Reference())
		case "!challenge":
			if len(commandAndArgs) != 2 || len(m.Mentions) != 1 {
				return
			}
			challenger := m.Author
			challengee := m.Mentions[0]

			if challenger.ID == challengee.ID {
				s.ChannelMessageSendReply(m.ChannelID, "You can't challenge yourself!", m.Reference())
				return
			}

			_, hasChallenger := Games[challenger.ID]
			_, hasChallengee := Games[challengee.ID]

			if hasChallengee {
				s.ChannelMessageSendReply(m.ChannelID, challengee.Mention() + " is busy. Try challenging them later.", m.Reference())
				return
			}

			if hasChallenger {
				s.ChannelMessageSendReply(m.ChannelID, "You're already busy. Try again after your game is done.", m.Reference())
				return
			}

			newGameSession := SessionStateAwaitingChallengeResponse{Challenger: challenger, Challengee: challengee}

			Games[challenger.ID] = &newGameSession
			Games[challengee.ID] = &newGameSession
			s.ChannelMessageSendReply(m.ChannelID, challengee.Mention() + ", you have been challenged to play BAGH. Type `!accept` to accept the challenge, or `!refuse` to refuse it.", m.Reference())

			return
		
		case "!accept":
			if len(commandAndArgs) > 1 {
				return
			}

			acceptor := m.Author
			challenge, hasAcceptor := Games[acceptor.ID]

			if !hasAcceptor {
				s.ChannelMessageSendReply(m.ChannelID, "No one is challenging you.", m.Reference())
				return
			}

			challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
			if !isChallenge {
				s.ChannelMessageSendReply(m.ChannelID, "You're in the middle of a game already.", m.Reference())
				return
			}

			if challengeAsChallenge.Challenger.ID == acceptor.ID {
				s.ChannelMessageSendReply(m.ChannelID, "You can't accept your own challenge.", m.Reference())
				return
			}

			challenger := challengeAsChallenge.Challenger
			
			// start a new thread for a game
			thread, err := s.ThreadStart(m.ChannelID,
				challenger.Username + "'s BAGH Game Against " + acceptor.Username,
				discordgo.ChannelTypeGuildPrivateThread, 60)
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "There was a problem starting a thread.", m.Reference())
				fmt.Println(err)
				return
			}
			s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + " has accepted " + challenger.Mention() + "'s challenge. Check for a new game thread and your DMs.", m.Reference())

			// make a game object and put the thread reference there
			newGame := SessionStateGameOngoing{Thread: thread, Challenger: NewPlayer(challenger), Challengee: NewPlayer(acceptor), Round: 1}

			Games[challenger.ID] = &newGame
			Games[acceptor.ID] = &newGame

			s.ChannelMessageSend(thread.ID, newGame.ToString() + newGame.PromptActionString(s))
			
			// DM each player and ask them for an action
			const dmIntroString = "Welcome to BAGH! Your chosen action is hidden until both players have made a move. So, you can type your action for the round here."
			linkToGame := "You can view the game here: <#" + thread.ID + ">"

			challengerDMChannel, _ := s.UserChannelCreate(challenger.ID)
			
			s.ChannelMessageSend(challengerDMChannel.ID, dmIntroString + "\n\n" + linkToGame + "\n\n" + MakeActionOptionList())

			acceptorDMChannel, _ := s.UserChannelCreate(acceptor.ID)
			s.ChannelMessageSend(acceptorDMChannel.ID, dmIntroString + "\n\n" + linkToGame + "\n\n" + MakeActionOptionList())

			return
		case "!refuse":
			if len(commandAndArgs) > 1 {
				return
			}

			refuser := m.Author
			challenge, hasRefuser := Games[refuser.ID]

			if !hasRefuser {
				s.ChannelMessageSendReply(m.ChannelID, refuser.Mention() + ", no one is challenging you.", m.Reference())
				return
			}

			challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
			if !isChallenge {
				s.ChannelMessageSendReply(m.ChannelID, refuser.Mention() + ", you're in the middle of a game already.", m.Reference())
				return
			}

			challenger := challengeAsChallenge.Challenger

			if challenger.ID == refuser.ID {
				s.ChannelMessageSendReply(m.ChannelID, refuser.Mention() + ", you're the challenger. Type `!retract` to retract your challenge.", m.Reference())
				return
			}

			delete(Games, refuser.ID)
			delete(Games, challenger.ID)

			s.ChannelMessageSendReply(m.ChannelID, refuser.Mention() + " has refused " + challenger.Mention() + "'s challenge.", m.Reference())
			return
		case "!retract":
			if len(commandAndArgs) > 1 {
				return
			}

			retractor := m.Author
			challenge, hasRetractor := Games[retractor.ID]

			if !hasRetractor {
				s.ChannelMessageSendReply(m.ChannelID, retractor.Mention() + ", you haven't challenged anyone.", m.Reference())
				return
			}

			challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
			if !isChallenge {
				s.ChannelMessageSendReply(m.ChannelID, retractor.Mention() + ", you're in the middle of a game already.", m.Reference())
				return
			}

			challengee := challengeAsChallenge.Challengee

			if challengee.ID == retractor.ID {
				s.ChannelMessageSendReply(m.ChannelID, challengee.Mention() + ", you're the one who's been challenged. Type `!refuse` to refuse the challenge.", m.Reference())
				return
			}

			delete(Games, retractor.ID)
			delete(Games, challengee.ID)

			s.ChannelMessageSendReply(m.ChannelID, retractor.Mention() + " has retracted their challenge to " + challengee.Mention() + ".", m.Reference())
			return
		}
	}

	speakerDMChannel, _ := s.UserChannelCreate(m.Author.ID)
	if m.ChannelID == speakerDMChannel.ID { // we're in a DM
		speakerGameSession, speakerIsInGameSession := Games[m.Author.ID]

		game, gameIsOngoing := speakerGameSession.(*SessionStateGameOngoing)
		_, gameIsChallenge := speakerGameSession.(*SessionStateAwaitingChallengeResponse)

		reportIfActionEnteredButGameIsntOngoing := func () bool {
			if !speakerIsInGameSession {
				s.ChannelMessageSendReply(m.ChannelID, "You're not in a game session right now.", m.Reference())
				return true
			}

			if !gameIsOngoing {
				if gameIsChallenge {
					s.ChannelMessageSendReply(m.ChannelID, "Your game hasn't started yet.", m.Reference())	
				}
				return true
			}

			return false
		}

		setActionAndReportIfActionLocked := func (a Action) bool {
			if !game.GetPlayer(m.Author.ID).SetAction(a) {
				s.ChannelMessageSendReply(m.ChannelID, "You've already chosen an action for this round. Message `!reconsider` to withdraw your current selection.", m.Reference())
				return false
			}
			return true
		}

		action := Unchosen

		switch m.Content {
		case "!boost":
			action = Boost
			break
		case "!attack":
			action = Attack
			break
		case "!guard":
			action = Guard
			break
		case "!heal":
			action = Heal
			break
		case "!reconsider":
			if game.GetPlayer(m.Author.ID).ClearAction() {
				s.ChannelMessageSendReply(m.ChannelID, "You have withdrawn you selection. Select a new action.", m.Reference())
				if m.Author.ID == game.Challenger.User.ID {
					s.ChannelMessageSend(game.Challengee.User.ID, "Your opponent has withdrawn their selected move.")
				} else {
					s.ChannelMessageSend(game.Challenger.User.ID, "Your opponent has withdrawn their selected move.")
				}
			} else {
				s.ChannelMessageSendReply(game.Challenger.User.ID, "Either no action has been selected for this round yet, or your action was already committed.", m.Reference())
			}
			return
		case "!rules":
			sendRules(s, m.Author.ID)
		default:
			s.ChannelMessageSendReply(m.ChannelID, "Invalid command.", m.Reference())
			return
		}

		if reportIfActionEnteredButGameIsntOngoing() || !setActionAndReportIfActionLocked(action) {
			return
		}

		s.ChannelMessageSendReply(m.ChannelID, "You chose to " + actionStrings[action] + ".", m.Reference())

		if game.Challenger.GetAction() != Unchosen && game.Challengee.GetAction() != Unchosen {
			game.Challenger.actionLocked = true
			game.Challengee.actionLocked = true
			actionLog, isGameOver, winner := game.NextStateFromActions()
			s.ChannelMessageSend(game.Thread.ID, actionLog)

			if isGameOver {
				delete(Games, game.Challenger.User.ID)
				delete(Games, game.Challengee.User.ID)

				if winner == nil {
					s.ChannelMessageSend(game.Thread.ID, "# Draw.")
				} else {
					s.ChannelMessageSend(game.Thread.ID, "# Congratulations, " + winner.User.Mention() + "!")
				}
			} else {
				s.ChannelMessageSend(game.Thread.ID, game.ToString() + game.PromptActionString(s))
			}
			
			for _, player := range [2]Player{game.Challenger, game.Challengee} {
				playerDMChannel, _ := s.UserChannelCreate(player.User.ID)
				s.ChannelMessageSend(playerDMChannel.ID, "Both players have now chosen an action for this round. See the results here: <#" + game.Thread.ID + ">")
			}
		} else {
			for _, player := range [2]Player{game.Challenger, game.Challengee} {
				playerDMChannel, _ := s.UserChannelCreate(player.User.ID)
				if player.User.ID == m.Author.ID {
					s.ChannelMessageSend(playerDMChannel.ID, "Waiting for your opponent's action for this round...")
				} else {
					s.ChannelMessageSend(playerDMChannel.ID, "Your opponent has chosen an action for this round.")
				}
			}
		}
	} else {
		gameSession, sessionFound := Games[m.Author.ID]
		if sessionFound {
			game, isOngoing := gameSession.(*SessionStateGameOngoing)
			if isOngoing {
				if m.ChannelID == game.Thread.ID {
					switch m.Content {
					case "!forfeit":
						var forfeiter *Player  = nil
						var winner *Player = nil
						if game.Challenger.User.ID == m.Author.ID {
							forfeiter = &game.Challenger
							winner = &game.Challengee
						} else {
							forfeiter = &game.Challengee
							winner = &game.Challenger
						}
						s.ChannelMessageSendReply(m.ChannelID, forfeiter.User.Mention() + " has forfeited the game to " + winner.User.Mention() + ". " + winner.User.Mention() + " **wins**!", m.Reference())
						return
					case "!boost", "!attack", "!guard", "!heal":
						s.ChannelMessageSendReply(m.ChannelID, "Pssst, " + m.Author.Mention() + ", your action is supposed to be a secret. DM me your final choice.", m.Reference())
						return
					}
				}
			}
		}
	}
}
