package main

import (
	"math/rand/v2"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type SessionState interface {
	isSessionState()
}

type AwaitingChallengeResponse struct {
	Challenger *discordgo.User
	Challengee *discordgo.User

	Channel *discordgo.Channel

	ChallengerInteractions []*discordgo.Interaction
	ChallengeeMessage      *discordgo.Message
}

func (a *AwaitingChallengeResponse) isSessionState() {}

type GameOngoing struct {
	Thread             *discordgo.Channel
	LastRoundMessageID string
	Challenger         Player
	Challengee         Player
	Round              int
}

func (o *GameOngoing) isSessionState() {}

func (game *GameOngoing) GetPlayer(userID string) *Player {
	if game.Challenger.User.ID == userID {
		return &game.Challenger
	}
	if game.Challengee.User.ID == userID {
		return &game.Challengee
	}
	return nil
}

func (game *GameOngoing) GetOtherPlayer(userID string) *Player {
	if game.Challenger.User.ID == userID {
		return &game.Challengee
	}
	if game.Challengee.User.ID == userID {
		return &game.Challenger
	}
	return nil
}

func (game *GameOngoing) ChooseAIMove() {
	r := rand.IntN(4)
	game.Challengee.currentAction = Action(r)
}

// returns whether the game ended, if it was a draw,
// and if not, who the winner and loser was
func (game *GameOngoing) IsGameOver() (bool, *Player) {
	if game.Challenger.HP > 0 && game.Challengee.HP > 0 {
		return false, nil
	}
	if game.Challenger.HP > game.Challengee.HP {
		return true, &game.Challenger
	}
	if game.Challengee.HP > game.Challenger.HP {
		return true, &game.Challengee
	}

	// this code shouldn't be reached. If it is, end the game in a draw.
	return true, nil
}

const (
	BASE_MAX_HEALTH int = 3
	MAX_BOOST       int = 6
)

func (game *GameOngoing) NextStateFromActions() (string, bool, *Player) {
	gainedOrRetainedAdvantage := make(map[*Player]bool)
	shieldJustBroke := make(map[*Player]bool)

	players := [2]*Player{&game.Challenger, &game.Challengee}

	actionLog := ""

	// Initial Phase
	for _, player := range players {
		playerAction := player.GetAction()
		playerMention := player.User.Mention()

		if player.ShieldBreakCounter > 0 {
			roll := rand.Float32()
			if roll < 1.0/float32(player.ShieldBreakCounter+1) {
				player.ShieldBreakCounter = 0
			}

			if player.ShieldBreakCounter == 0 {
				actionLog += "- " + playerMention + "'s shield is **mended**!\n"
			} else {
				actionLog += "- " + playerMention + "'s shield remains **broken**.\n"
			}
		}

		if playerAction == Boost {
			if player.Boost < MAX_BOOST {
				player.Boost += 1
				actionLog += "- " + playerMention + " " + actionStrings[Boost] + "s to **" + strconv.Itoa(player.Boost) + "**.\n"
			} else {
				actionLog += "- " + playerMention + " " + actionStrings[Boost] + "s, preserving a boost of **" + strconv.Itoa(player.Boost) + "**.\n"
			}
		}
	}

	type ActionInfo struct {
		Agent   *Player
		Patient *Player
	}

	playerRelations := [2]ActionInfo{
		{
			Agent:   &game.Challenger,
			Patient: &game.Challengee,
		},
		{
			Agent:   &game.Challengee,
			Patient: &game.Challenger,
		},
	}

	delayString := ""

	// Middle Phase
	for _, playerRelation := range playerRelations {
		agent := playerRelation.Agent
		patient := playerRelation.Patient

		agentAction := agent.GetAction()
		patientAction := patient.GetAction()

		agentMention := agent.User.Mention()
		patientMention := patient.User.Mention()

		agentHasAdvantage := agent.Advantage > patient.Advantage
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
					delayString += "- " + patientMention + "'s counterattack renders " + agentMention + "'s " + attackString + " **impotent**.\n"
				}
			case Guard:
				if patient.ShieldBreakCounter > 0 { // shield is broken
					actionLog += "- " + agentMention + " attacks, and " + patientMention + " " + actionStrings[Guard] + "s, but the shield is **broken**.\n"
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
					actionLog += "- " + agentMention + " " + attackString + ", but " + patientMention + " " + guardString + " and **prevents damage**.\n"
					// agent has higher boost
					if boostDifferential > 0 {
						patient.ShieldBreakCounter = boostDifferential
						shieldJustBroke[patient] = true
						actionLog += "- " + patientMention + "'s shield **breaks**! Its damage is at " + strconv.Itoa(patient.ShieldBreakCounter) + ".\n"
					} else if agentHasAdvantage { // agent has advantage
						actionLog += "- Because " + agentMention + " has advantage over " + patientMention + ", " + patientMention + " gains no advantage.\n"
					} else { // patient gains or retains advantage
						oldAdvantage := patient.Advantage
						patient.Advantage = max(patient.Advantage, (-1*boostDifferential)+1)

						actionLog += "- " + patientMention
						if oldAdvantage == patient.Advantage {
							actionLog += " retains advantage at **"
						} else {
							actionLog += " gains advantage up to **"
						}
						actionLog += strconv.Itoa(patient.Advantage) + "**.\n"
						gainedOrRetainedAdvantage[patient] = true
					}
				}
			case Heal:
				if !patientHasAdvantage {
					// heal is interrupted
					delayString += "- " + patientMention + "'s " + actionStrings[Heal] + "ing is **interrupted** by " + agentMention + "'s attack.\n"
				}

			}
			if attackGoesThrough {
				damage := 1 + agent.Boost

				patient.HP -= damage

				actionLog += "- " + agentMention + " " + actionStrings[Attack] + "s for "
				if agent.Boost > 0 {
					actionLog += "a boosted "
				}
				actionLog += "**" + strconv.Itoa(damage) + "** damage"
				if agentHasAdvantage {
					actionLog += " with advantage"
				}

				actionLog += ".\n"
			}
		case Guard:
			if patientAction != Attack {
				// no effect
				actionLog += "- " + agentMention + " " + actionStrings[Guard] + "s to **no effect**.\n"
			}
		case Heal:
			if patientAction != Attack || agentHasAdvantage { // heal not interrupted
				maxOverheal := BASE_MAX_HEALTH + 1 + agent.Boost
				newHP := min(agent.HP+1+agent.Boost, maxOverheal)

				actionLog += "- " + agentMention + " " + actionStrings[Heal] + "s"

				if patientAction == Attack && agentHasAdvantage {
					actionLog += ", with **advantage preventing interruption** from " + patient.User.Mention() + "'s attack,"
				}

				if agent.HP >= newHP { // no effect
					actionLog += " to no effect.\n"
				} else {
					diff := newHP - agent.HP
					agent.HP = newHP

					actionLog += " by **" + strconv.Itoa(diff) + "** to "

					if newHP > BASE_MAX_HEALTH {
						actionLog += "an overheal of "
					}

					actionLog += "**" + strconv.Itoa(newHP) + "**.\n"
				}
			}
		}
	}

	// endure/sudden death to prevent potential draw
	if game.Challenger.HP <= 0 && game.Challengee.HP <= 0 {
		actionLog += "- Both players have lost all their HP the same turn, "
		if game.Challenger.HP == game.Challengee.HP {
			game.Challenger.HP = 1
			game.Challengee.HP = 1
			actionLog += "and have the same final HP. They endure with 1HP each."
		} else if game.Challenger.HP > game.Challengee.HP {
			actionLog += "but " + game.Challenger.User.Mention() + "'s health was utlimately higher, securing **victory**!"
		} else {
			actionLog += "but " + game.Challengee.User.Mention() + "'s health was utlimately higher, securing **victory**!"
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
					actionLog += "- " + playerMention + "'s boost is **expended to 0**.\n"
				}
			}
		}

		if !isOver && !gainedOrRetainedAdvantage[player] && player.Advantage > 0 {
			player.Advantage--
			secondString += "- " + playerMention + "'s advantage **falls to " + strconv.Itoa(player.Advantage) + "**.\n"
		}

		if !isOver && player.ShieldBreakCounter > 0 {
			if !shieldJustBroke[player] {
				player.ShieldBreakCounter--
			}
			if player.ShieldBreakCounter == 0 {
				thirdString += "- " + playerMention + "'s shield is **mended**! "
			} else {
				thirdString += "- The chance of " + playerMention + "'s shield mending next turn is **1 in " + strconv.Itoa(player.ShieldBreakCounter+1) + "**.\n"
			}
		}

		player.UnlockAction() // TODO move this out of the scope of next state
		player.ClearAction()
	}
	actionLog += secondString
	actionLog += thirdString

	if isOver {
		if winner == nil {
			actionLog += "- Both players have lost all health in the same turn, resulting in a **draw**."
		} else {
			actionLog += "- " + winner.User.Mention() + " secures **victory**!"
		}
	} else {
		game.Challenger.votedToDraw = false
		game.Challengee.votedToDraw = false
		game.Round++
	}

	return actionLog, isOver, winner
}

func (game *GameOngoing) ToString() string {
	gameString := "# Round " + strconv.Itoa(game.Round) + "\n"
	for _, player := range [2]Player{game.Challenger, game.Challengee} {
		shield := ""
		if player.ShieldBreakCounter > 0 {
			shield += "- 🛡️❌ (chance of mending: 1 in " + strconv.Itoa(player.ShieldBreakCounter+1) + ")\n"
		}
		boost := ""
		if player.Boost > 0 {
			boost = "- ⬆️"
			if player.Boost > 1 {
				boost += "x" + strconv.Itoa(player.Boost)
			}
			boost += "\n"
		}
		advantage := ""
		if player.Advantage > 0 {
			advantage = "- [Adv."
			if player.Advantage > 1 {
				advantage += "x" + strconv.Itoa(player.Advantage)
			}
			advantage += "]\n"
		}
		gameString += "🤺 " + player.User.Mention() + "\n- ❤️x" + strconv.Itoa(player.HP) + "\n" + shield + boost + advantage + "\n"
	}
	return gameString
}
