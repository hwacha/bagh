package main

import (
	"github.com/bwmarrin/discordgo"
)

const (
	acceptorNotBAGHerErrorMessage       = "You are not a `bagher`! Use the `/join` command to become a `bagher` and accept this challenge."
	acceptOutdatedChallengeErrorMessage = "You've tried to accept an outdated challenge."
	alreadyBAGHerErrorMessage           = "You're already a `bagher`!"
	alreadyNotBAGHerErrorMessage        = "You're not a `bagher` already!"
	baghOptions                         = "Welcome to BAGH! You have access to the following slash commands:\n" +
		"- `/join`: adds the `bagher` role and allows you to issue and accept challenges from other BAGH players.\n" +
		"- `/leave`: removes the `bagher` role. You won't be able to issue challenges, and other player's can't challenge you.\n" +
		"- `/rules`: enumerates the rules of BAGH.\n" +
		"- `/bagh`: gives help and instructions.\n" +
		"You can also use the following user commands. To use a user command, right-click on a user (in this server's members list), and go to Apps.\n" +
		"- `challenge`: challenges someone to a game of BAGH."
	challengeAcceptedWhileInGameErrorMessage            = "You're in the middle of a game already."
	challengeRefusedWhileInGameErrorMessage             = "You're in the middle of a game already."
	challengerIssuesChallengeWhileInSessionErrorMessage = "You're already busy. Try again after your game is done."
	challengerNotBAGHerErrorMessage                     = "You are not a `bagher`! Use the `/join` command to become a `bagher` and issue challenges."
	checkPermissionsErrorMessage                        = "There was a problem restoring the BAGH channel.\n" +
		"- Check that the BAGH App has the correct permissions.\n" +
		"- If the `bagher` role exists, make sure that it's lower on the role heirarchy than the BAGH role.\n" +
		"- The `play-bagh` channel should give the BAGH app the following permissions:\n" +
		"  - green viewing.\n" +
		"  - default for everything else."
	chooseAnActionPrompt          = "Choose one of the following actions."
	exitGamePrompt                = "Exit the game by selecting one of the following options."
	forfeitConfirmation           = "You have chosen to forfeit this game."
	gameThreadMissingErrorMessage = "You're in the middle of a game, but the thread has been deleted. Ask an admin to run `/restore` to bring it back."
	goodbyeMessage                = "You can no longer play BAGH in this server. Goodbye!"
	issueChallengePrompt          = "Issue someone a challenge by right-clicking on their name in the server, going to Apps," +
		" and clicking the `challenge` option with my icon next to it."
	leaveWhenInSessionErrorMessage         = "You can't leave BAGH while you're in a game session. `refuse`, `rescind`, or `forfeit` to enable leaving."
	nonPlayerUsesInGameCommandErrorMessage = "You are not a player in this game of BAGH."
	playBAGHChannelMissingErrorMessage     = "The `play-bagh` channel is missing. Ask an admin to run `/restore` to bring it back."
	playerNotBAGHerJoinPrompt              = "Use the `/join` command to view the BAGH channel and start playing BAGH."
	refuseOutdatedChallengeErrorMessage    = "You've tried to refuse an outdated challenge."
	resendLastRoundNotification            = "The message for the current round got deleted. It will now be re-sent."
	rescindOutdatedChallengeErrorMessage   = "You've tried to rescind an outdated challenge."
	restoreConfirmation                    = "`play-bagh` channel, `bagher` role, and all ongoing game threads have been restored."
	roleMissingErrorMessage                = "The `bagher` role is missing from the server. Ask an admin to run `/restore` to bring it back."
	selfAcceptChallengeErrorMessage        = "You can't accept your own challenge!"
	selfChallengeErrorMessage              = "You can't challenge yourself!"
	undoneSelectionChooseAnActionPrompt    = "You have undone your selection. " + chooseAnActionPrompt
	votedToDrawConfirmation                = "You have voted to end the game this round in a draw."
	voteToDrawPassesNotification           = "By unanimous consent, the game ends this round in a **draw**.\n# Draw."
	voteToDrawWithdrawnConfirmation        = "You have withdrawn your vote to end the game this round in a draw."
	welcomeMessage                         = "Welcome to BAGH! You can now play in this server."
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
	return challengee.Mention() + " is not a BAGHer! They cannot be challenged to a game of BAGH. Check for a `bagher` role."
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

func forfeitNotification(forfeiter *discordgo.User, otherPlayer *discordgo.User) string {
	return forfeiter.Mention() + " has forfeited. " + otherPlayer.Mention() + " **wins** by default!\n# Congratulations, " + otherPlayer.Mention() + "!"
}

func gameThreadTitle(challenger *discordgo.Member, challengee *discordgo.Member) string {
	challengeeNick := "BAGH-Bot"
	if challengee != nil {
		challengeeNick = challengee.DisplayName()
	}

	return challenger.DisplayName() + "'s BAGH Game Against " + challengeeNick
}

func memberRemovedNotification(removedPlayer *discordgo.User) string {
	return removedPlayer.Mention() + " has been removed from the server you were playing BAGH in. The session has been terminated."
}

func playerAcceptOrRefuseChallengePrompt(challenger *discordgo.User, dm *discordgo.Channel, message *discordgo.Message) string {
	return challenger.Mention() + "'s challenge is awaiting your response.\nAccept or refuse here: " +
		"https://discord.com/channels/@me/" + dm.ID + "/" + message.ID
}

func playerInGameRedirectToGameThread(thread *discordgo.Channel) string {
	return "You're in the middle of a BAGH game.\nJoin back in here: " + thread.Mention()
}

func votedToDrawNotification(voter *discordgo.User) string {
	return voter.Mention() + " has voted to end the game this round in a draw."
}

func voteToDrawWithdrawnNotification(voter *discordgo.User) string {
	return voter.Mention() + " has withdrawn their vote to end the game this round in a draw."
}
