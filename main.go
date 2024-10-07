package main

import (
	"strings"
	"strconv"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	Heal:   "🏥 **HEAL** 🏥",
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
	DisplayName string
	HP int
	Advantage int
	Boost int
	currentAction Action
}

func NewPlayer(u *discordgo.User, displayName string) Player {
	return Player{ User:u, DisplayName: displayName, HP:BASE_MAX_HEALTH, Advantage:0, Boost:0, currentAction:Unchosen }
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

func (p *Player) ClearAction() {
	p.currentAction = Unchosen
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
func (game *SessionStateGameOngoing) IsGameOver() (bool, bool, *Player, *Player) {
	if game.Challenger.HP > 0 && game.Challengee.HP > 0 {
		return false, false, nil, nil
	}
	if game.Challenger.HP <= 0 && game.Challengee.HP <= 0 {
		return true, true, nil, nil
	}
	if game.Challenger.HP > 0 {
		return true, false, &game.Challenger, &game.Challengee
	}
	if game.Challengee.HP > 0 {
		return true, false, &game.Challengee, &game.Challenger
	}

	// this code shouldn't be reached. If it is, end the game in a draw.
	return true, true, nil, nil
}

const BASE_MAX_HEALTH int = 3

func (game *SessionStateGameOngoing) NextStateFromActions() string {
	var challengerGainedAdvantageThisTurn bool = false
	var challengeeGainedAdvantageThisTurn bool = false

	challengerAction := game.Challenger.GetAction()
	challengeeAction := game.Challengee.GetAction()

	defer game.Challenger.ClearAction()
	defer game.Challengee.ClearAction()

	actionLog := ""

	// Attack
	if challengerAction == Attack {
		if challengeeAction != Attack || game.Challenger.Advantage >= game.Challengee.Advantage {
			actionLog += game.Challenger.User.Mention() + actionStrings[Attack] + "s"
			// Guard
			if challengeeAction == Guard {
				actionLog += ", but " + game.Challengee.User.Mention() + actionStrings[Guard] + "s and takes no damage. "
				boostDifference := game.Challengee.Boost - game.Challenger.Boost
				if boostDifference < 0 {
					boostDifference = -1
				}
				if game.Challengee.Advantage >= game.Challenger.Advantage && game.Challengee.Boost >= game.Challenger.Boost && game.Challengee.Advantage < boostDifference + 1 {
					game.Challengee.Advantage = boostDifference + 1
					actionLog += game.Challengee.DisplayName + " gains advantage"
					if boostDifference > 0 {
						actionLog += " boosted by " + strconv.Itoa(boostDifference)
					}
					actionLog += "."
					challengeeGainedAdvantageThisTurn = true
				} else {
					actionLog += "Because " + game.Challenger.DisplayName
					if game.Challengee.Advantage < game.Challenger.Advantage {
						actionLog += " had advantage over "
					} else {
						actionLog += " was more boosted than "
						
					}
					actionLog += game.Challengee.DisplayName + ", " + game.Challengee.DisplayName + " gains no advantage."
				}
			} else {
				if challengeeAction == Attack && game.Challenger.Advantage > game.Challengee.Advantage {
					actionLog += " with advantage over " + game.Challengee.User.Mention()
				}
				game.Challengee.HP -= 1 + game.Challenger.Boost
				game.Challengee.Advantage = 0
				actionLog += " for "
				if game.Challenger.Boost > 0 {
					actionLog += " a boosted "
				}
				actionLog += strconv.Itoa(1 + game.Challenger.Boost) + " damage"
			}
		}
	}
	if challengeeAction == Attack {
		if challengerAction != Attack || game.Challengee.Advantage >= game.Challenger.Advantage {
			if challengerAction == Attack {
				actionLog += " just as "
			}
			actionLog += game.Challengee.User.Mention() + actionStrings[Attack] + "s"
			// Guard
			if challengerAction == Guard {
				actionLog += ", but " + game.Challenger.User.Mention() + actionStrings[Guard] + "s and takes no damage. "
				boostDifference := game.Challenger.Boost - game.Challengee.Boost
				if boostDifference < 0 {
					boostDifference = -1
				}
				if game.Challenger.Advantage >= game.Challengee.Advantage && game.Challenger.Boost >= game.Challengee.Boost && game.Challenger.Advantage < boostDifference + 1 {
					game.Challenger.Advantage = boostDifference + 1
					actionLog += game.Challenger.DisplayName + " gains advantage"
					if boostDifference > 0 {
						actionLog += " boosted by " + strconv.Itoa(boostDifference)
					}
					actionLog += "."
					challengerGainedAdvantageThisTurn = true
				} else {
					actionLog += "Because " + game.Challengee.DisplayName
					if game.Challenger.Advantage < game.Challengee.Advantage {
						actionLog += " had advantage over "
					} else {
						actionLog += " was more boosted than "
						
					}
					actionLog += game.Challenger.DisplayName + ", " + game.Challenger.DisplayName + " gains no advantage."
				}
			} else {
				if challengerAction == Attack && game.Challengee.Advantage > game.Challenger.Advantage {
					actionLog += " with advantage over " + game.Challenger.User.Mention()
				}
				game.Challenger.HP -= 1 + game.Challengee.Boost
				game.Challenger.Advantage = 0
				actionLog += " for "
				if game.Challengee.Boost > 0 {
					actionLog += " a boosted "
				}
				actionLog += strconv.Itoa(1 + game.Challengee.Boost) + " damage"
			}
		}
	}

	isOver, isDraw, winner, _ := game.IsGameOver()

	incRound := func () {
		if !isOver {
			game.Round++	
		}
	}
	defer incRound()

	if isOver && isDraw {
		actionLog += game.Challenger.DisplayName + " and " + game.Challengee.DisplayName + " are defeated on the same turn, resulting in a **draw**."
		return actionLog
	}

	performAndReportBoostExpendature := func (p *Player, al *string) {
		if !isOver {
			if p.Boost > 0 {
				*al += p.DisplayName + "'s boost is expended to 0."
			}
			p.Boost = 0
		}
	}

	// Guard, no attack
	if challengerAction == Guard && challengeeAction != Attack {
		actionLog += game.Challenger.User.Mention() + " " + actionStrings[Guard] + "s to no effect"
	}

	if challengeeAction == Guard && challengerAction != Attack {
		if challengerAction == Guard {
			actionLog += " and "
		}
		actionLog += game.Challengee.User.Mention() + " " + actionStrings[Guard] + "s to no effect"
	}

	// Boost
	if challengerAction == Boost {
		game.Challenger.Boost += 1
		if (challengeeAction == Attack || challengeeAction == Guard) {
			actionLog += ", and then " + game.Challenger.User.Mention() + " " + actionStrings[Boost] + "s to " + strconv.Itoa(game.Challenger.Boost) + "."
		} else {
			actionLog += game.Challenger.User.Mention() + " " + actionStrings[Boost] + "s to " + strconv.Itoa(game.Challenger.Boost)
		}
	} else {
		defer performAndReportBoostExpendature(&game.Challenger, &actionLog)
	}

	// Heal
	if challengerAction == Heal {
		maxOverheal := BASE_MAX_HEALTH + game.Challenger.Boost

		if isOver && challengeeAction == Attack && winner == &game.Challengee {
			actionLog += " before " + game.Challenger.User.Mention() + " had a chance to " + actionStrings[Heal] + "."
			return actionLog
		}

		if challengeeAction == Attack || challengeeAction == Guard {
			actionLog += ", and then"
		}

		newHP := game.Challenger.HP + 1 + game.Challenger.Boost
		
		if newHP > maxOverheal {
			newHP = maxOverheal
		}

		actionLog += game.Challenger.User.Mention() + " " + actionStrings[Heal] + "s "

		if game.Challenger.Boost > 0 {
			actionLog += "boosted by " + strconv.Itoa(game.Challenger.Boost) + " "
		}

		if newHP == game.Challenger.HP {
			 actionLog += "to no effect"
		} else {
			actionLog += ", healing by " + strconv.Itoa(newHP - game.Challenger.HP) + " to "
			if newHP > BASE_MAX_HEALTH {
				actionLog += "an overheal of "
			}
			actionLog += strconv.Itoa(newHP)
			game.Challenger.HP = newHP
		}
	}

	if challengeeAction == Boost {
		game.Challengee.Boost += 1
		if (challengerAction == Attack || challengerAction == Guard) {
			actionLog += ", and then " + game.Challengee.User.Mention() + " " + actionStrings[Boost] + "s to " + strconv.Itoa(game.Challengee.Boost) + "."
		} else {
			actionLog += " and " + game.Challengee.User.Mention() + " " + actionStrings[Boost] + "s to " + strconv.Itoa(game.Challengee.Boost) + "."
		}
	} else {
		defer performAndReportBoostExpendature(&game.Challengee, &actionLog)
	}

	if challengeeAction == Heal {
		maxOverheal := BASE_MAX_HEALTH + game.Challengee.Boost

		if isOver && challengerAction == Attack && winner == &game.Challenger {
			actionLog += " before " + game.Challengee.User.Mention() + " had a chance to " + actionStrings[Heal] + "."
			return actionLog
		}

		if challengerAction == Attack || challengerAction == Guard {
			actionLog += ", and then"
		} else {
			actionLog += " and "
		}

		newHP := game.Challengee.HP + 1 + game.Challengee.Boost
		
		if newHP > maxOverheal {
			newHP = maxOverheal
		}

		actionLog += game.Challengee.User.Mention() + " " + actionStrings[Heal] + "s "

		if game.Challengee.Boost > 0 {
			actionLog += "boosted by " + strconv.Itoa(game.Challengee.Boost) + " "
		}

		if newHP == game.Challengee.HP {
			 actionLog += "to no effect"
		} else {
			actionLog += ", healing by " + strconv.Itoa(newHP - game.Challengee.HP) + " to "
			if newHP > BASE_MAX_HEALTH {
				actionLog += "an overheal of "
			}
			actionLog += strconv.Itoa(newHP)
			game.Challengee.HP = newHP
		}
	}

	// depreciate advantage
	if !isOver {
		if !challengerGainedAdvantageThisTurn && game.Challenger.Advantage > 0 {
			game.Challenger.Advantage -= 1
			actionLog += " " + game.Challenger.DisplayName + "'s advantage falls to " + strconv.Itoa(game.Challenger.Advantage) + "."
		}

		if !challengeeGainedAdvantageThisTurn && game.Challengee.Advantage > 0 {
			game.Challengee.Advantage -= 1
			actionLog += " " + game.Challengee.DisplayName + "'s advantage falls to " + strconv.Itoa(game.Challengee.Advantage) + "."
		}
	}

	return actionLog
}

func (game *SessionStateGameOngoing) ToString() string {
	gameString := "# Round " + strconv.Itoa(game.Round) + "\n"
	for _, player := range [2]Player{game.Challenger, game.Challengee} {
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
				boost += "x" + strconv.Itoa(player.Advantage)
			}
			advantage += "]"
		}
		gameString += player.User.Mention() + ": 😐 ❤️x" + strconv.Itoa(player.HP) + boost + advantage + "\n"
	}
	return gameString
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

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
				discordgo.ChannelTypeGuildPrivateThread, 1440)
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "There was a problem starting a thread.", m.Reference())
				fmt.Println(err)
				return
			}
			s.ChannelMessageSendReply(m.ChannelID, acceptor.Mention() + " has accepted " + challenger.Mention() + "'s challenge. Check for a new game thread and your DMs.", m.Reference())

			// make a game object and put the thread reference there
			newGame := SessionStateGameOngoing{Thread: thread, Challenger: NewPlayer(challenger, challenger.Username), Challengee: NewPlayer(acceptor, acceptor.Username), Round: 1}

			Games[challenger.ID] = &newGame
			Games[acceptor.ID] = &newGame

			s.ChannelMessageSend(thread.ID, newGame.ToString())
			
			// DM each player and ask them for an action
			const dmIntroString = "Welcome to BAGH! Your chosen action is hidden until both players have made a move. So, you can type your action for the round here."
			linkToGame := "You can view the game here: <#" + thread.ID + ">"

			challengerDMChannel, _ := s.UserChannelCreate(challenger.ID)
			
			s.ChannelMessageSend(challengerDMChannel.ID, dmIntroString)
			s.ChannelMessageSend(challengerDMChannel.ID, linkToGame)
			s.ChannelMessageSend(challengerDMChannel.ID, MakeActionOptionList())

			acceptorDMChannel, _ := s.UserChannelCreate(acceptor.ID)
			s.ChannelMessageSend(acceptorDMChannel.ID, dmIntroString)
			s.ChannelMessageSend(acceptorDMChannel.ID, linkToGame)
			s.ChannelMessageSend(acceptorDMChannel.ID, MakeActionOptionList())

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
				s.ChannelMessageSendReply(m.ChannelID, "You've already chosen an action for this round.", m.Reference())
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
		}

		if action == Unchosen {
			s.ChannelMessageSendReply(m.ChannelID, "Invalid command.", m.Reference())
			return
		}

		if reportIfActionEnteredButGameIsntOngoing() || !setActionAndReportIfActionLocked(action) {
			return
		}

		s.ChannelMessageSendReply(m.ChannelID, "You chose to " + actionStrings[action] + ".", m.Reference())

		if game.Challenger.GetAction() != Unchosen && game.Challengee.GetAction() != Unchosen {
			actionLog := game.NextStateFromActions()
			s.ChannelMessageSend(game.Thread.ID, actionLog)
			s.ChannelMessageSend(game.Thread.ID, game.ToString()) // TODO change to edit message
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
	}
}
