# BAGH: **B**oost, **A**ttack, **G**uard, **H**eal
BAGH is a simple turn-based combat game for two players. The objective is to defeat your opponent by lowering their HP to 0. If both players lose all their HP in a single turn, the game ends in a draw.

A BAGH match win is given to the first player to win 3 BAGH games.

Both players privately choose an action each round: **Boost**, **Attack**, **Guard**, or **Heal**. Actions are then revealed and performed simultaneously.
## Actions
### Boost
**Boost**ing increases a player's **boost** stat by 1. Successive boosts increase the boost even higher. Boost makes every other action increasingly more effective. Once any other action is performed, all of a player's boost is expended to 0.

A player can accumulate up to 6 total boost. Any boosts beyond this will not increase the boost but preserve it at 6.

**NOTE**: *Boost is expended even if an action is unsuccessful or has no effect. The only way to keep boost is to preserve the boost streaks.*
### Attack
**Attack**ing does a point of damage to the opposing player. If the opposing player's HP drops to 0, then the attacker will win. If both players attack each other on the same turn, they will each damage the other. This may result in a draw if both players lose all their HP in one turn.

A boosted attack will do one more point of damage for each boost. For example, if a player has a boost of 2 and attacks, they will do 3 points of damage.
### Guard
**Guard**ing only has an effect if the other player attacks. The guarding player will not take any damage that turn.

The boost of the guarder and attacker are compared. If the guarder has the same boost as the attacker, or if neither player has any boost, then the guarder gains **priority**. A guarder can only gain priority if the attacker does not already have a higher priority.

If the guarder has *less* boost than the attacker, they will prevent damage but their **shield** will **break**.

_**Priority**_
When both players attack each other, usually they will both deal damage. However, if one or both players have priority, then their priority is compared. If one player has higher priority than the other, or if only one player has priority, then only that player deals damage from the attack.

**NOTE**: *Priority and boost are separate values. Priority comparisons are separate from boost comparisons.*

If the guarder has more boost than the attacker, they gain a point more of priority for every point of boost higher than the attacker's boost. For example, if the guarder has a boost of 5 and the attacker has a boost of 2, then the guarder will gain 1 priority from a successful guard, plus 3 priority gained from the boost difference, for a total of 4 gained priority. If the guarder already has priority, en they will gain only the boost difference, without the extra 1 base priority. You can think of this as losing one priority from depreciation after gaining some that turn.

If an attacker has priority, then the priority a guarder gains will be dampened by 1.

For example, a player with priority of 1 and a boost of 2 guards against a player with priority of 2 and no boost. The priority conferred to the guarding player is calculated as follows: no base priority, since the guarder already had priority that turn, plus 2 from the boost difference, minus 1 from the attacking player's priority (although their priority is at 2, only 1 priority is discounted). The total priority gained by the guarder is therefore 1.

Unless a player made an effective guard previously in the current turn (i.e., they guarded an attack and their shield did not break), priority drops by 1 every turn until it reaches 0.

_**Shield Breaking**_
If a player's shield is broken, **Guard**ing will not prevent damage from an attack.

A shield will be more badly broken when the boost difference is higher. Its damage is equal to the boost difference. Every turn, after actions are chosen but before they are performed, there is a **1 in (damage + 1)** chance the shield will **mend**. That turn, a **Guard** action will successfully prevent damage from an attack. Otherwise, the shield remains broken, and its damage decreases by 1. When its damage falls to 0, it is guaranteed to mend.

If one player attacks with a boost of 1 and the other player guards with no boost, their shield will break with a damage of 1. At the beginning of the next turn, there is a 1 in 2 chance it will mend. If it's still broken at the end of the turn, its damage falls to 0 and is guaranteed to mend the next turn.

As another example, if one player attacks with a boost of 5 and the other player guards with a boost of 2, their shield will break with a damage of 3. The next turn, there is a 1 in 4 chance it will mend. If it's still broken at the end of the turn, its damage will falls to 2. The next turn, there is a 1 in 3 chance it will mend. Every turn the shield remains broken, its damage will fall by 1, making it more likely to mend.

**NOTE**: *It is guaranteed the shield will mend after as many turns as the original boost difference has passed.*
### Heal
**Heal**ing restores HP by a point. Players start with 3HP, and can overheal up to 10HP. Healing beyond 10HP will have no effect.

A boosted heal will heal one more point for each boost. For example, a player with 3HP and a boost of 2 will heal 1 base HP plus 2 boosted for a total of 3 gained HP to 6.

If an attacker attacks on the same turn as a player tries to heal, they will be **interrupted** before healing. However, if the healer has priority over the attacker, then the healing will go through along with the attack. The healer's resultant health will be there original health minus damage plus health. For example, if a player with priority of 1 heals while a player with no priority attacks, the healing player will take 1 damage but heal by 1, resulting in no net change of HP.
