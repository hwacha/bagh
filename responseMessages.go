package main

import (
	"github.com/bwmarrin/discordgo"
)

const (
	challengeAcceptedNoChallengeErrorMessage            = "No one is challenging you."
	challengeAcceptedWhileInGameErrorMessage            = "You're in the middle of a game already."
	challengerIssuesChallengeWhileInSessionErrorMessage = "You're already busy. Try again after your game is done."
	challengerNotBAGHerErrorMessage                     = "You are not a BAGHer! Use the `/join` command to become a BAGHer and issue challenges."
	chooseAnActionPrompt                                = "Choose one of the following actions."
	undoneSelectionChooseAnActionPrompt                 = "You have undone your selection. " + chooseAnActionPrompt
	goodbyeMessage                                      = "You can no longer play BAGH in this server. Goodbye!"
	nonPlayerUsesInGameCommandErrorMessage              = "You are not a player in this game of BAGH."
	selfAcceptChallengeErrorMessage                     = "You can't accept your own challenge!"
	selfChallengeErrorMessage                           = "You can't challenge yourself!"
	welcomeMessage                                      = "Welcome to BAGH! You can now play in this server."
)

func actionSelectedConfirmation(action Action) string {
	return "You have chosen to " + actionStrings[action] + "."
}

func challengeAcceptConfirmationForChallengee(challenger *discordgo.User, thread *discordgo.Channel) string {
	return "You have accepted " + challenger.Mention() + "'s challenge!\nYou can play the game here: " + thread.Mention()
}

func challengeAcceptNotificationForChallenger(challengee *discordgo.User, thread *discordgo.Channel) string {
	return challengee.Mention() + " has accepted your challenge!\nYou can play the game here: " + thread.Mention()
}

func challengeeNotBAGHerError(challengee *discordgo.User) string {
	return challengee.Mention() + " is not a BAGHer! They cannot be challenged to a game of BAGH."
}

func challengeIssuedConfirmationToChallenger(challengee *discordgo.User) string {
	return "You have challenged " + challengee.Mention() + "."
}

func challengeIssuedNotificationToChallengee(challenger *discordgo.User) string {
	return challenger.Mention() + " has challenged you to a game of BAGH."
}

func challengeRefusedConfirmationToChallengee(challenger *discordgo.User) string {
	return "You have refused " + challenger.Mention() + "'s challenge."
}

func challengeRefusedNotificationToChallenger(challengee *discordgo.User) string {
	return challengee.Mention() + " has refused your challenge."
}

func challengeRescindedConfirmationToChallenger(challengee *discordgo.User) string {
	return "You have rescinded your challenge to " + challengee.Mention() + "."
}

func challengeRescindedNotificationToChallengee(challenger *discordgo.User) string {
	return challenger.Mention() + " has rescinded their challenge."
}

func challengeIssuedWhileChallengeeInSessionErrorMessage(challengee *discordgo.User) string {
	return challengee.Mention() + " is busy. Try challenging them later."
}

func gameThreadTitle(challenger *discordgo.Member, challengee *discordgo.Member) string {
	challengeeNick := "BAGH-Bot"
	if challengee != nil {
		challengeeNick = challengee.DisplayName()
	}

	return challenger.DisplayName() + "'s BAGH Game Against " + challengeeNick
}