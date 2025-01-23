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

type MatchOngoing struct {
	Thread             *discordgo.Channel
	LastRoundMessageID string
	Challenger         Player
	Challengee         Player
	Game               int
	Round              int
}

func (o *MatchOngoing) isSessionState() {}

func (game *MatchOngoing) GetPlayer(userID string) *Player {
	if game.Challenger.User.ID == userID {
		return &game.Challenger
	}
	if game.Challengee.User.ID == userID {
		return &game.Challengee
	}
	return nil
}

func (game *MatchOngoing) GetOtherPlayer(userID string) *Player {
	if game.Challenger.User.ID == userID {
		return &game.Challengee
	}
	if game.Challengee.User.ID == userID {
		return &game.Challenger
	}
	return nil
}

func (game *MatchOngoing) GetPlayers() [2]*Player {
	return [2]*Player{&game.Challenger, &game.Challengee}
}

func (game *MatchOngoing) ChooseAIMove() {
	r := rand.IntN(4)
	game.Challengee.currentAction = Action(r)
}

// returns whether the game ended, if it was a draw,
// and if not, who the winner and loser was
func (game *MatchOngoing) IsGameOver() (bool, *Player) {
	if game.Challenger.HP > 0 && game.Challengee.HP > 0 {
		return false, nil
	}
	if game.Challenger.HP > game.Challengee.HP {
		return true, &game.Challenger
	}
	if game.Challengee.HP > game.Challenger.HP {
		return true, &game.Challengee
	}

	// if both player have no health and both have the same HP, draw.
	return true, nil
}

func (game *MatchOngoing) IsMatchOver() (bool, *Player) {
	if game.Challenger.Wins >= GAMES_TO_WIN && game.Challengee.Wins < GAMES_TO_WIN {
		return true, &game.Challenger
	}
	if game.Challengee.Wins >= GAMES_TO_WIN && game.Challenger.Wins < GAMES_TO_WIN {
		return true, &game.Challengee
	}
	if game.Challenger.Wins >= GAMES_TO_WIN && game.Challengee.Wins >= GAMES_TO_WIN {
		return true, nil
	}
	return false, nil
}

const (
	BASE_MAX_HEALTH int = 3
	MAX_BOOST       int = 6
	GAMES_TO_WIN    int = 3
)

func (game *MatchOngoing) NextStateFromActions() (string, bool, *Player) {
	gainedOrRetainedPriority := make(map[*Player]bool)
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

		agentHasPriority := agent.Priority > patient.Priority
		patientHasPriority := patient.Priority > agent.Priority

		// positive if agent has more boost
		// negative if patient has more boost
		// 0 if equal boost
		boostDifferential := agent.Boost - patient.Boost

		switch agentAction {
		case Attack:
			attackGoesThrough := true
			switch patientAction {
			case Attack:
				if patientHasPriority { // attack has no effect
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
					} else {
						oldPriority := patient.Priority
						totalPriorityGain := 1 // base gain from effective guard

						priorityIsDampened := agent.Priority > 0
						if priorityIsDampened {
							// agent's priority dampens priority gain by 1,
							// which can happen for N potential turns,
							// preserving payoff equivalence
							totalPriorityGain -= 1
						}
						totalPriorityGain += -boostDifferential
						patient.Priority += max(0, totalPriorityGain)
						// patient gains or retains priority

						if oldPriority == patient.Priority {
							actionLog += "- Because of " + agentMention + "'s priority, " + patientMention + " gains no priority.\n"
						} else {
							// account for overcounted priority w/ depreciation
							// now instead of later, for the sake of log coherence
							if oldPriority > 0 {
								patient.Priority -= 1
							}

							extraPriorityIsPositive := boostDifferential < 0
							priorityIsRetained := oldPriority == patient.Priority

							actionLog += "-" + patientMention
							if priorityIsRetained {
								actionLog += " retains priority"
							} else {
								actionLog += " gains priority"
							}

							if extraPriorityIsPositive {
								actionLog += " boosted by " + strconv.Itoa(-boostDifferential)

								if priorityIsDampened {
									actionLog += " but"
								}
							}

							if priorityIsDampened {
								actionLog += " dampened by 1 by " + agentMention + "'s priority"
							}

							if priorityIsRetained {
								actionLog += " at **"
							} else {
								actionLog += " up to **"
							}

							actionLog += strconv.Itoa(patient.Priority) + "**.\n"
							gainedOrRetainedPriority[patient] = true
						}
					}
				}
			case Heal:
				if !patientHasPriority {
					// heal is interrupted
					delayString += "- " + patientMention + "'s " + actionStrings[Heal] + "ing is **interrupted** by " + agentMention + "'s attack.\n"
				}

			}
			if attackGoesThrough {
				damage := 1 + agent.Boost

				patient.HP -= damage
				patient.HP = max(patient.HP, 0)

				actionLog += "- " + agentMention + " " + actionStrings[Attack] + "s for "
				if agent.Boost > 0 {
					actionLog += "a boosted "
				}
				actionLog += "**" + strconv.Itoa(damage) + "** damage"
				if agentHasPriority {
					actionLog += " with priority"
				}

				actionLog += ".\n"
			}
		case Guard:
			if patientAction != Attack {
				// no effect
				actionLog += "- " + agentMention + " " + actionStrings[Guard] + "s to **no effect**.\n"
			}
		case Heal:
			if patientAction != Attack || agentHasPriority { // heal not interrupted
				maxOverheal := BASE_MAX_HEALTH + 1 + agent.Boost
				newHP := min(agent.HP+1+agent.Boost, maxOverheal)

				actionLog += "- " + agentMention + " " + actionStrings[Heal] + "s"

				if patientAction == Attack && agentHasPriority {
					actionLog += ", with **priority preventing interruption** from " + patient.User.Mention() + "'s attack,"
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

	actionLog += delayString

	// determine end game
	isGameOver, gameWinner := game.IsGameOver()

	secondString := ""
	thirdString := ""

	// End Phase
	for _, player := range players {
		playerAction := player.GetAction()
		playerMention := player.User.Mention()
		if playerAction != Boost {
			if player.Boost > 0 {
				player.Boost = 0
				if !isGameOver {
					actionLog += "- " + playerMention + "'s boost is **expended to 0**.\n"
				}
			}
		}

		if !isGameOver && !gainedOrRetainedPriority[player] && player.Priority > 0 {
			player.Priority--
			secondString += "- " + playerMention + "'s priority **falls to " + strconv.Itoa(player.Priority) + "**.\n"
		}

		if !isGameOver && player.ShieldBreakCounter > 0 {
			if !shieldJustBroke[player] {
				player.ShieldBreakCounter--
			}
			if player.ShieldBreakCounter == 0 {
				thirdString += "- " + playerMention + "'s shield is **mended**! "
			} else {
				thirdString += "- The chance of " + playerMention + "'s shield mending next turn is **1 in " + strconv.Itoa(player.ShieldBreakCounter+1) + "**.\n"
			}
		}
	}
	actionLog += secondString
	actionLog += thirdString

	if isGameOver {
		if gameWinner == nil {
			actionLog += "- Both players have lost all health in the same turn, resulting in a **draw**."
		} else {
			actionLog += "- " + gameWinner.User.Mention() + " secures **victory**!"
		}
	} else {
		game.Round++
	}

	isMatchOver := false
	var matchWinner *Player = nil

	if isGameOver {
		if gameWinner != nil {
			gameWinner.Wins += 1
		}
		actionLog += "\n- The score is | " +
			game.Challenger.User.Mention() + " **" + strconv.Itoa(game.Challenger.Wins) + "** | " +
			game.Challengee.User.Mention() + " **" + strconv.Itoa(game.Challengee.Wins) + "** |\n"
		isMatchOver, matchWinner = game.IsMatchOver()

		if isMatchOver {
			actionLog += "- The match has ended."
		} else {
			game.Game++

			game.Round = 1
			for _, player := range players {
				player.HP = 3
				player.Boost = 0
				player.Priority = 0
				player.ShieldBreakCounter = 0
				player.currentAction = Unchosen
				player.actionLocked = false
				player.votedToDraw = false
			}

			actionLog += game.GameNumberString()
		}
	}

	return actionLog, isMatchOver, matchWinner
}

func (game *MatchOngoing) GameNumberString() string {
	return "# Game " + strconv.Itoa(game.Game) + "\n"
}

func (game *MatchOngoing) ToString() string {
	gameString := "## Round " + strconv.Itoa(game.Round) + "\n"
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
		priority := ""
		if player.Priority > 0 {
			priority = "- [Priority"
			if player.Priority > 1 {
				priority += "x" + strconv.Itoa(player.Priority)
			}
			priority += "]\n"
		}
		gameString += "🤺 " + player.User.Mention() + "\n- ❤️x" + strconv.Itoa(player.HP) + "\n" + shield + boost + priority + "\n"
	}
	return gameString
}
