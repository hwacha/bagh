package main

import "github.com/bwmarrin/discordgo"

type Player struct {
	User                     *discordgo.User
	ChooseActionInteractions []*discordgo.Interaction
	HP                       int
	ShieldBreakCounter       int
	Advantage                int
	Boost                    int
	currentAction            Action
	actionLocked             bool
}

func NewPlayer(u *discordgo.User) Player {
	return Player{User: u, HP: BASE_MAX_HEALTH, Advantage: 0, Boost: 0, currentAction: Unchosen, actionLocked: false}
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
