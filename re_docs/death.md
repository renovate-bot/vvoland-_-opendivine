# Agent death (`.\AGENTS\agentfight.cpp`)

What happens when a living [`CAgent`](agent.md) dies: the state transition
that takes an NPC or party member from fighting to a lootable corpse. This
is the victim-side sequence — distinct from the attacker-side **XP reward**
([`progression.md`](progression.md)) and the spell-triggered death visuals
`CDeathEffect{1..4}Magic` ([`skills-magic.md`](skills-magic.md)).

## The "dead" flag — `agent+0x220 & 0x100`

Death state lives in the agent flag word **`+0x220`**. Bit **`0x100`** is
the universal *is-dead* predicate: combat, [perception/AI](ai-behaviour.md),
and the [damage-apply](combat.md) path all early-out on it. For example the
HP-apply `fcn.00417550` opens with
`test [esi+0x220], 0x180` / `test … 0x100` / `test al, 0x10` and bails if
any is set — a dead (or dying/disabled) agent takes no further hits.

`Die` additionally raises bits **`0x80`** and **`0x100000`**, and the
post-death callback raises **`0x200`**, so the flag word encodes the stage
of the death sequence.

## `Die` — `CNpc`/`CPartyMember` vtable `+0x34` (`fcn.004188e0`)

The kill handler is a **virtual** (slot `+0x34`, no direct xref — invoked
polymorphically from the per-frame update / damage path once Hp reaches 0).
It tears down the combatant:

1. Clears the active attack/aggro state (`[+0x2ac] = 0`) and stops motion
   and the current fight via `fcn.004273e0`.
2. Looks up a death entry by id in the global list **`[0x7467fc]`**
   (`fcn.005438c0` by-id search, ids `0x5b`/`0x35` — the death
   sound/effect descriptors).
3. Rolls **`rand()`** once (a death variation — e.g. which death animation
   / gib outcome), so death is not fully deterministic.
4. **Resets the `CAgentFight` sub-object** (re-stamps vtable `0x609020`,
   zeroes the attack-state fields `+0x254`/`+0x258`/`+0x260`/`+0x2e0`) —
   the agent stops being an attacker.
5. Raises the death flags `+0x220 |= 0x80`, `|= 0x100000`.
6. **Schedules the post-death callback** `fcn.00414fe0` (twice, via the
   timer helper `fcn.00414ef0` with delay `0x10`) — the finalisation runs
   after the death animation rather than inline.

## Post-death callback — `fcn.00414fe0`

Runs when the death animation completes:

- **Re-entry guard** — if the agent is already fully dead
  (`+0x220 & 0x100`) it logs the developer warning
  `"Npc dead ahead of callback"`.
- Raises `+0x220 |= 0x200` (death finalised).
- **Recomputes the cell-grid cell** (`+0x234`/`+0x238`) from the body's
  position (`fcn.0057bf30`, registry `[0x750d2c]`), so the corpse occupies
  the [cell grid](cell-grid.md) as a lootable, walk-over-able body rather
  than a live combatant.
- Tail-calls the agent's own vtable `+0x1c` to complete the transition.

The corpse representation surfaces in the data as the `Corpse` /
`Hardened Corpse` object states; the body keeps its
[inventory](inventory.md), which is what makes it lootable.

## How it connects

- **XP** — awarding experience is the *killer's* job, the separate
  party-XP path `fcn.0042aaf0` ([`progression.md`](progression.md)); `Die`
  here is only the victim's state change.
- **Loot** — the corpse retains its [inventory](inventory.md); the
  treasure a monster carries comes from its [egg](formats/eggs.md) /
  [treasure table](formats/treasure.md), populated at spawn, not at death.
- **Death visuals** — natural death plays the agent's death animation
  ([`animation.md`](animation.md)); spell-kills can additionally trigger
  `CDeathEffect{1..4}Magic` ([`skills-magic.md`](skills-magic.md)).
- **Story** — Osiris reacts to deaths through its event manager
  ([`osiris.md`](osiris.md)) (a `CReteEvent` fires on the kill), which is
  how quests advance "kill X".
- **FeignDeath** — the `FeignDeath` skill (`FeignDeathManaLost` prop) fakes
  this state without the real transition.

## Status

- Dead flag ✅ — `agent+0x220 & 0x100` is the *is-dead* predicate
  (early-out in `fcn.00417550` and across combat/AI); `Die` also raises
  `0x80`/`0x100000`, the callback raises `0x200`.
- `Die` handler ✅ — `CNpc`/`CPartyMember` vtable `+0x34` (`fcn.004188e0`,
  `agentfight.cpp`): stop combat (`fcn.004273e0`), reset `CAgentFight`
  sub-object (vtable `0x609020`), `rand()` roll, schedule the post-death
  callback.
- Post-death callback ✅ — `fcn.00414fe0`: re-entry guard, sets `0x200`,
  re-registers the corpse in the cell grid (`+0x234`/`+0x238`), tail-call
  vtable `+0x1c`.
- XP / loot / Osiris links ✅ (cross-referenced) — XP is the attacker-side
  `fcn.0042aaf0`; loot is the carried inventory; Osiris fires a kill event.
- `rand()` roll meaning ✅ (pinned) — it is a **percentage death-outcome
  gate**, not random damage: `Die` rolls `rand() % 100` and compares it
  against a **per-agent-type threshold at `[[agent+0x228] + 0x314]`**; when
  `roll < threshold` it sets the agent flag `+0x100000` and calls
  `fcn.00449380`. So `[agent+0x228]` is the agent's type/descriptor record
  and `+0x314` a percent chance (e.g. leave-a-corpse / death-effect
  probability) evaluated once on death.
- Death descriptor list `[0x7467fc]` 🟡 (narrowed) — a **global id-keyed
  descriptor registry**, initialised at boot (`main` → `fcn.00542340`) and
  shared by the death/combat cluster (`Die`, the post-death callback
  `fcn.00414fe0`, `fcn.0041cf50`). It is searched by id via the linked-list
  scan **`fcn.005438c0`** (walks entries matching field `+0x30`, returns the
  entry / its `+0x34`); `Die` queries id `0x5b` (91). **Correction:** the
  `0x35` (53) lookup goes through a *different* function **`fcn.00448df0`**,
  not `fcn.005438c0` as previously written. **Construction traced:**
  `fcn.00542340` builds it — ctor `fcn.0051f8f0`, then a registrar
  `fcn.005207f0` binds id sets (e.g. `0x393..0x396` = 915–918) into the
  list. So the registry is a boot-built id→entry table (`{+0x30 id, +0x34
  value}` nodes). Its **identity stays unnameable statically** — neither
  the ctor, the registrar, nor `fcn.005207f0`'s callees leak a `.cpp`/name
  string, and it `fopen`s nothing; naming it (notification / sound /
  event-handler registry) needs a string the binary doesn't expose. Bounded
  dead-end with cause.

## Citations

```text
div.exe:0x004188e0   fcn.004188e0   CNpc/CPartyMember::Die (vtable +0x34, agentfight.cpp).
div.exe:0x00414fe0   fcn.00414fe0   post-death callback ("Npc dead ahead of callback" guard).
div.exe:0x00414ef0   fcn.00414ef0   timer/schedule helper (defers the death callback).
div.exe:0x004273e0   fcn.004273e0   stop motion / clear active fight.
div.exe:0x00417550   fcn.00417550   HP-apply — early-out on dead flag +0x220 & 0x180/0x100/0x10.
div.exe:0x005438c0   fcn.005438c0   id->entry list scan on [0x7467fc] (match +0x30, return +0x34); Die id 0x5b.
div.exe:0x00542340   fcn.00542340   boot init of the [0x7467fc] descriptor registry.
flag: agent+0x220 bit 0x100 = dead · 0x80/0x100000 set by Die · 0x200 set by callback.
roll: Die rolls rand()%100 vs [[agent+0x228]+0x314] (percent); roll<threshold → set +0x100000.
str: "Npc dead ahead of callback" · "Corpse" · "Hardened Corpse" · "FeignDeath".
```
