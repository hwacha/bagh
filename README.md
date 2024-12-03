# BAGH: Boost, Attack, Guard, Heal

BAGH is a simple two-player game implemented as a Discord Bot. Each player secretly chooses one of four moves—Boost, Attack, Guard, or Heal. Both moves are applied once both players have chosen. Boosting makes every other move stronger. Attacking deals damage to the opponent's health. Guarding prevents damage from an attack. Healing regains HP but is interrupted by an attack. For more details on the game, check out the [ruleset](https://github.com/hwacha/bagh/blob/main/rules.md).

The application and game logic is implemented using Go. Discord API calls are made using [discordgo](https://github.com/bwmarrin/discordgo), a Go wrapper for the Discord API. The user interface is implemented using Discord's message components, application commands, and ephemeral messages for secrecy within the game thread.

Click [this link](https://discord.com/oauth2/authorize?client_id=1291027616702402632&permissions=397552921648&integration_type=0&scope=bot+applications.commands) to add BAGH to your Discord server.
