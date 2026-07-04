# Party, ownership & minions (`.\AGENTS\agentparty.cpp`)

How the engine tracks **who controls an agent** — the runtime behind
summoned creatures (skeletons, demons, rats), charmed monsters, and
script-owned NPCs that fight alongside (or follow) a master. This is the
ownership layer on top of plain [`CAgent`](agent.md)s; the spells that
*create* these minions live in [`skills-magic.md`](skills-magic.md), and
this doc covers the link they establish and how the AI/combat code honour
it.

## The two key fields

| `CAgent` field | Meaning |
|---|---|
| `+0x1b0` | **owner / "Summoned by"** — the agent index of the master (`-1` = nobody owns it). Printed by the agent dumper `fcn.0042d230` as `"Summoned by =%d"` (alongside `Current weight = [+0x1a4]`). |
| `+0x224 & 0x8000` | the **slave flag** — the agent is under its owner's / a script's control rather than acting freely. |

So a minion is just a `CAgent` whose `+0x1b0` points at its master and
whose slave bit is set. Both are saved, so summoned/charmed companions
survive a reload ([`formats/savegame.md`](formats/savegame.md)).

### Combat honours the owner

The combat resolver checks `cmp [agent+0x1b0], -1` ([`combat.md`](combat.md)):
when a summoned creature acts, the engine walks to its **owner** (via the
agent manager `[0x658d50]`, `owner = agents[[+0x1b0]]`) so the master gets
the credit/blame — kills by your skeleton count as yours, and a charmed
monster's victims are attributed to you. A standalone (un-owned) agent has
`+0x1b0 == -1` and skips that redirection.

### The slave flag gates autonomy

A slaved minion normally obeys its owner / its [agentscript](npc-ai.md),
not its own perception. When combat is forced on it, the script runner
`fcn.004329a0` **clears the slave bit** so the creature can defend itself —
the log lines `"Slave flag cleared to allow for fight"` (when an enemy is
in range) and `"Script frame finished for npc %s … Slave flag cleared"`
(when its scripted action ends). So the flag is the "follow orders vs.
fight freely" switch, toggled around combat and script frames.

## Where minions come from

- **Summon spells** — `CMagicSummonSkeleton` / `CMagicSummonDemon` /
  `CMagicSummonRats` (script keywords `Summon lesser/greater monster`,
  `Summon demon`, `Summonrats`): spawn a [monster](monsters.md) from an
  [egg](formats/eggs.md), set its `+0x1b0` to the caster, and raise the
  slave bit. Power scales through props like `SummonedShieldsLevelBoost`
  ([`formats/props.md`](formats/props.md)).
- **Charm** — `CMagicCharmMonster` (magic-script `charm` keyword;
  `"Levels in a charm are obsolete"` → charm is **single-tier**, not
  leveled). An existing hostile monster is re-owned to the caster for
  `CharmDuration`, resisted by `CharmResistance` and modulated by
  `charmquality` / `charmlevel`. When the duration lapses the ownership
  link clears and it reverts to its [alignment](alignment.md).
- **Script** — the `set owner $` command assigns ownership directly, so
  the [story](osiris.md) / [agentscript](script-commands.md) can make any
  NPC a follower without a spell.

## The party (`agentparty.cpp`, `CPartyMember`)

`CPartyMember` is the player/companion subclass of `CAgent` (it and `CNpc`
share the combat virtuals — `Die`, the damage-apply, etc.). Party-level
state is handled in `agentparty.cpp` (`fcn.0042a990` / `fcn.0042bd10` /
`fcn.0042c210`, the latter a `CPartyMember` virtual), and the debug
`"Party parameter %d"` dumps its membership slots. The party is the set of
owned/controlled agents that move and fight as a group with the player.

## Status

- Owner link ✅ — `CAgent+0x1b0` = summoner/owner index (`-1` = none),
  dumped as `"Summoned by"` (`fcn.0042d230`); combat redirects to the
  owner via `[0x658d50]`.
- Slave flag ✅ — `+0x224 & 0x8000`; cleared by the script runner
  `fcn.004329a0` to allow fighting / at script-frame end.
- Minion sources ✅ — summon (`CMagicSummon{Skeleton,Demon,Rats}`) + charm
  (`CMagicCharmMonster`, single-tier, `CharmDuration`/`CharmResistance`) +
  the `set owner $` script command.
- Party container 🟡 — `CPartyMember` + `agentparty.cpp` functions located
  (`fcn.0042a990`/`fcn.0042bd10`/`fcn.0042c210`); the exact party-slot
  struct (max members, membership list layout) is not split field-by-field.
- Owner-field writer 🟡 — `+0x1b0` is read/checked everywhere; the precise
  write site (in the summon/charm handlers + `CAgent::Read`) is not pinned
  to a single instruction (no plain `mov [reg+0x1b0]` — set via a computed
  pointer).

## Citations

```text
div.exe:0x0042d230   fcn.0042d230   agent dumper — "Summoned by =[+0x1b0]", "Current weight =[+0x1a4]".
div.exe:0x004329a0   fcn.004329a0   agentscript runner — clears slave flag (+0x224 & 0x8000) for combat.
div.exe:0x0042c210   fcn.0042c210   CPartyMember virtual (agentparty.cpp).
field: CAgent+0x1b0 = owner/Summoned-by (-1=none) · +0x224 & 0x8000 = slave flag.
str: .\AGENTS\agentparty.cpp · "set owner $" · "Slave flag cleared to allow for fight" · "Levels in a charm are obsolete"
classes: CMagicSummon{Skeleton,Demon,Rats} · CMagicCharmMonster.
```
