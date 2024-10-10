package main

import (
	"strings"
	"strconv"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
)

const PLAY_BAGH_ID = "1291052523439783977"

type SessionState interface {
	isSessionState()
}

type SessionStateAwaitingChallengeResponse struct {
	Challenger *discordgo.User
	Challengee *discordgo.User
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
	Token string
	SendRules bool
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
				actionLog += playerMention + "'s shield is still broken. Its damage is at "
				actionLog += strconv.Itoa(player.ShieldBreakCounter) + ". "
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
					actionLog += patientMention + "'s counterattack renders " + agentMention + "'s " + attackString + " impotent. "
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
						// roll for shield break
						shieldBreakProbability := float32(boostDifferential) / float32(boostDifferential + 1)

						roll := rand.Float32()
						if roll < shieldBreakProbability {
							patient.ShieldBreakCounter = boostDifferential
							shieldJustBroke[patient] = true
							actionLog += patientMention + "'s shield **breaks**! Its damage is at " + strconv.Itoa(patient.ShieldBreakCounter) + ". "
						}
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
				actionLog += strconv.Itoa(damage) + " damage. "
			}
			break
		case Guard:
			if patientAction != Attack {
				// no effect
				actionLog += agentMention + " " + actionStrings[Guard] + "s to no effect. "
			}
			break
		case Heal:
			if patientAction != Attack {
				// heal not interrupted
				maxOverheal := BASE_MAX_HEALTH + 1 + agent.Boost

				newHP := min(agent.HP + 1 + agent.Boost, maxOverheal)

				actionLog += agentMention + " " + actionStrings[Heal] + "s"

				if agent.HP <= newHP { // no effect
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
			actionLog += playerMention + "'s advantage falls to " + strconv.Itoa(player.Advantage) + ". "
		}

		if !isOver && player.ShieldBreakCounter > 0 && !shieldJustBroke[player] {
			player.ShieldBreakCounter--
			if player.ShieldBreakCounter == 0 {
				actionLog += playerMention + "'s shield is **mended**!"
			}
		}

		player.UnlockAction() // TODO move this out of the scope of next state
		player.ClearAction()
	}

	if isOver {
		if winner == nil {
			actionLog += " Both players have lost all health in the same turn, resulting in a **draw**."
		} else {
			actionLog += " " + winner.User.Mention() + " secures **victory**!"
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

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.BoolVar(&SendRules, "rules", false, "Set if you want to send a message with the rules")
	flag.Parse()
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

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

func ready(s *discordgo.Session, r *discordgo.Ready) {
	if SendRules {
		dat1, err1 := os.ReadFile("rules-1.md")
		if err1 != nil {
			fmt.Println(err1)
			return
		}
		dat2, err2 := os.ReadFile("rules-2.md")
		if err2 != nil {
			fmt.Println(err2)
			return
		}
		dat3, err3 := os.ReadFile("rules-3.md")
		if err3 != nil {
			fmt.Println(err3)
			return
		}
		s.ChannelMessageSend(PLAY_BAGH_ID, string(dat1))
		s.ChannelMessageSend(PLAY_BAGH_ID, string(dat2))
		s.ChannelMessageSend(PLAY_BAGH_ID, string(dat3))
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
		case "!challenge":
			if len(commandAndArgs) != 2 || len(m.Mentions) != 1 {
				return
			}
			challenger := m.Author
			challengee := m.Mentions[0]

			if challenger.ID == challengee.ID {
				s.ChannelMessageSendReply(m.ChannelID, challenger.Mention() + ", you can't challenge yourself!", m.Reference())
				return
			}

			_, hasChallenger := Games[challenger.ID]
			_, hasChallengee := Games[challengee.ID]

			if hasChallengee {
				s.ChannelMessageSendReply(m.ChannelID, challengee.Mention() + " is busy. Try challenging them later.", m.Reference())
				return
			}

			if hasChallenger {
				s.ChannelMessageSendReply(m.ChannelID, challenger.Mention() + ", you're already busy. Try again after your game is done.", m.Reference())
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
				s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + ", no one is challenging you.", m.Reference())
				return
			}

			challengeAsChallenge, isChallenge := challenge.(*SessionStateAwaitingChallengeResponse)
			if !isChallenge {
				s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + ", you're in the middle of a game already.", m.Reference())
				return
			}

			if challengeAsChallenge.Challenger.ID == acceptor.ID {
				s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + ", you can't accept your own challenge.", m.Reference())
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

			s.ChannelMessageSend(thread.ID, newGame.ToString())
			
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
				s.ChannelMessageSend(game.Thread.ID, game.ToString()) // TODO change to edit message
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
