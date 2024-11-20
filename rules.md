# BAGH: **B**oost, **A**ttack, **G**uard, **H**eal
BAGH is a simple turn-based combat game for two players. The objective is to defeat your opponent by lowering their HP to 0. If both players lose all their HP in a single turn, the game ends in a draw.

Both players privately choose an action each round: **Boost**, **Attack**, **Guard**, or **Heal**. Actions are then revealed and performed simultaneously.
## Actions
### Boost
**Boost**ing increases a player's **boost** stat by 1. Successive boosts increase the boost even higher. Boost makes every other action increasingly more effective. Once any other action is performed, all of a player's boost is expended to 0.

A player can accumulate up to 6 total boost. Any boosts beyond this will not increase the boost but preserve it at 6.

**NOTE**: *Boost is expended even if an action is unsuccessful or has no effect. The only way to keep boost is to preserve the boost streaks.*
### Attack
**Attack**ing does a point of damage to the opposing player. If the opposing player's HP drops to 0, then the attacker will win. If both players attack each other on the same turn, they will each damage the other. This may result in a draw if both players lose all their HP.

A boosted attack will do one more point of damage for each boost. For example, if a player has a boost of 2 and attacks, they will do 3 points of damage.
### Guard
**Guard**ing only has an effect if the other player attacks. The guarding player will not take any damage that turn.

The boost of the guarder and attacker are compared. If the guarder has the same boost as the attacker, or if neither player has any boost, then the guarder gains the condition of **advantage**. A guarder can only gain the condition of advantage if the attacker does not already have a higher advantage.

If the guarder has *less* boost than the attacker, they will prevent damage but their **shield** will **break**.

_**Advantage**_
When both players attack each other, usually they will both deal damage. However, if one or both players have advantage, then their advantage is compared. If one player has higher advantage than the other, or if only one player has advantage, then only that player deals damage from the attack.

**NOTE**: *Advantage and boost are separate values. Advantage comparisons are separate from boost comparisons.*

If the guarder has more boost than the attacker, they gain a point more of advantage for every point of boost higher than the attacker's boost. For example, if the guarder has a boost of 5 and the attacker has a boost of 2, then the guarder will gain 1 advantage from a successful guard, plus 3 advantage gained from the boost difference, for a total of 4 gained advantage. If the guarder already has advantage, then the maximum advantage between their old advantage and the new advantage becomes their advantage next turn.

For example, a player with advantage of 1 and a boost of 2 guards against a player with advantage of 2 and no boost. The guarding player will gain an advantage of 2.

In contrast, a player with advantage of 3 guards against a player with advantage of 1. The gained advantage is less than the guarder's old advantage, so the guarding player will retain advantage of 3.

Unless a player had gained advantage in the current turn, advantage drops by 1 every turn until it reaches 0.

_**Shield Breaking**_
If a player's shield is broken, **Guard**ing will not prevent damage from an attack.

A shield will be more badly broken when the boost difference is higher. Its damage is equal to the boost difference. Every turn, after actions are chosen but before they are performed, there is a **1 in (damage + 1)** chance the shield will **mend**. That turn, a **Guard** action will successfully prevent damage from an attack. Otherwise, the shield remains broken, and its damage decreases by 1. When its damage falls to 0, it is guaranteed to mend.

If one player attacks with a boost of 1 and the other player guards with no boost, their shield will break with a damage of 1. At the beginning of the next turn, there is a 1 in 2 chance it will mend. If it's still broken at the end of the turn, its damage falls to 0 and is guaranteed to mend the next turn.

As another example, if one player attacks with a boost of 5 and the other player guards with a boost of 2, their shield will break with a damage of 3. The next turn, there is a 1 in 4 chance it will mend. If it's still broken at the end of the turn, its damage will falls to 2. The next turn, there is a 1 in 3 chance it will mend. Every turn the shield remains broken, its damage will fall by 1, making it more likely to mend.

**NOTE**: *It is guaranteed the shield will mend after as many turns as the original boost difference has passed.*
### Heal
**Heal**ing restores HP by a point. Players start with 3HP, and can overheal up to 4HP. Successive unboosted heals will have no effect until a player's HP drops below 4.

A boosted heal will heal one more point for each boost, and can overheal one more point past 4HP. For example, a player with 4HP and a boost of 2 will heal by 3 points, but will be capped at a max overheal of 4 base limit + 2 boost = 6HP max overheal.

If an attacker attacks on the same turn as a player tries to heal, they will be **interrupted** before healing. However, if the healer has advantage over the attacker, then the healing will go through along with the attack. For example, if a player with advantage of 1 heals while a player with no advantage attacks, the healing player will take 1 damage but heal by 1, resulting in no net change of HP.

## Controls
Use application commands to interact with the `BAGH` Bot. To play, type `/join` to gain the `bagher` role. To start a game, right click on a server member, and click on `Apps->challenge` on someone else with the `bagher` role. Once a challenge has been accepted, a new thread will appear. In that thread, you will see the state of the game as well as a log of each action taken.
