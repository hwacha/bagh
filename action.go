package main

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
