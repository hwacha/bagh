# BAGH: **B**oost, **A**ttack, **G**uard, **H**eal
BAGH is a simple turn-based combat game for two players. The objective is to defeat your opponent by lowering their HP to 0. If both players lose all their HP in a single turn, the game ends in a draw.

Both players choose an action each round: **Boost**, **Attack**, **Guard**, or **Heal**.
## Actions
### Boost
**Boost**ing increases a player's **boost** by 1. Successive boosts increase the boost even higher. There is no limit on how high a player's boost can be. Boost makes every other action increasingly more effective. Once any other action is performed, all of a player's boost is expended to 0.

A player can boost up to 6. Any successive boosts will preserve but not increase boost.

**NOTE**: *boost is expended even if an action is unsuccessful or has no effect. The only way to keep boost is to preserve the boost streaks.*
### Attack
**Attack**ing does a point of damage to the opposing player. If the opposing player's HP drops to 0, the the attack will win. If both players attack each other on the same turn, they will each damage the other. This may result in a draw if both players lose all their HP.

A boosted attack will do one more point of damage for each boost. For example, if a player has a boost of 2 and attacks, they will do 3 points of damage.
### Guard
**Guard**ing only has an effect if the other player attacks. The guarding player will not take any damage that turn.

The boost of the guarder and attacker are compared. If the guarder has the same boost as the attacker, or if neither player has any boost, then the guarder gains **advantage**. A guarder can only gain advantage if the attacker does not have advantage over the guarder. If the guarder has *less* boost than the attacker, there is a chance of a **shield break**.

_**Advantage**_
When both players attack each other, usually they will both deal damage. However, if one or both players have advantage, then their advantage is compared. If one player has higher advantage than the other, or if only one player has advantage, then only that player deals damage from the attack.

If the guarder has more boost than the attacker, they gain a point more of advantage for every point of boost higher than the attacker's boost. For example, if the guarder has a boost of 5 and the attacker has a boost of 2, then the guarder will gain 1 advantage from a successful guard, plus 3 advantage gained from the boost difference, for a total of 4 gained advantage. If the gaurder already has advantage, then the maximum advantage between their old advantage and the new advantage becomes their advantage next turn.

Unless a player had gained advantage in the current turn, advantage drops by 1 every turn until it reaches 0.

_**Shield Breaking**_
If a player's shield is broken, **Guard**ing will not stop an attack from dealing damage.

The chance of a shield break gets higher the larger the boost difference between attacker and guarder. Also, a shield will be more badly broken when the boost difference is higher. Every turn, after actions are chosen but before they are performed, there is a chance the shield will **mend**. That turn, a **Guard** action will successfully prevent damage from an attack. As more turns pass, there is a better chance the shield will mend.

**NOTE**: *It is guaranteed the shield will mend after as many turns as the original boost difference has passed.*
### Heal
**Heal**ing restores HP by a point. Players start with 3HP, and can overheal up to 4HP. Successive unboosted heals will have no effect until a player's HP drops below 4.

A boosted heal will heal one more point for each boost, and can overheal one more point past 4HP. For example, a player with 4HP and a boost of 2 will heal by 3 points, but will be capped at a max overheal of 4 base limit + 2 boost = 6HP max overheal.

If an attacker attacks on the same turn as a player tries to heal, they will be **interrupted** before healing.
## Controls
Use `!command`s to interact with the `BAGH` Bot. To start a game type `!challenge @User` in the `play-bagh` channel, where @User is a mention of someone in the channel. Type `!retract` if you wish to rescind your challenge. If you've been challenged, you can type `!accept` to accept the challenge, or `!refuse` to refuse it. Once a challenge has been accepted, a new thread will appear. In that thread, you will see the state of the game as well as a log of each action taken. In this thread you may `!forfeit` the game to end the game. This will count as a win by default for the other player.

To submit actions for each round, you will be prompted to check your DMs with the `BAGH` Bot. You can type either `!boost`, `!attack`, `!guard`, or `!heal` to choose an action. If you regret your choice for a round, you can type `!reconsider` to withdraw your current action. Note that this is only possible if the other player hasn't made their move yet.
