package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func runGameCommandLine() {
	var p1ActionString, p2ActionString string
	var p1Action, p2Action Action = Unchosen, Unchosen

	p1 := NewPlayer(&discordgo.User{ID: "1"})
	p2 := NewPlayer(&discordgo.User{ID: "2"})
	game := MatchOngoing{Thread: nil, LastRoundMessageID: "", Challenger: p1, Challengee: p2, Round: 1}

	redact := func() {
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
			case "a", "attack":
				p1Action = Attack
			case "g", "guard":
				p1Action = Guard
			case "h", "heal":
				p1Action = Heal
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
			case "a", "attack":
				p2Action = Attack
			case "g", "guard":
				p2Action = Guard
			case "h", "heal":
				p2Action = Heal
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
