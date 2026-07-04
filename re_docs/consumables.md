# Consumables — potions, elixirs (`fcn.00587a20`)

Using a **potion / consumable** from the inventory: the handler that
restores health/mana/stamina, grants a timed buff, cures status, or turns
the drinker invisible. Distinct from equipping an item or casting a spell —
consumables are not [`CMagic*`](skills-magic.md) effects; they run through
one dedicated handler that reads the effect size from the
[`props.000`](formats/props.md) curves.

## The handler — `fcn.00587a20`

A single function applies every consumable. It is already known as "the
potion handler" ([`inventory.md`](inventory.md)); this documents what it
actually does. It dispatches on the item's consumable subtype through a
**37-case jump table** (`0x588a68`, indexed by a byte class id at
`0x588ac8`), and for each kind looks up the magnitude in `props.000` via
the props lookup `fcn.00500f10`, indexed by the potion's tier (the props
arrays are 5-entry ladders, e.g. `HealthPotion = [20,40,100,200,400]`).
`esi = [agent+0x2c]` is the drinker's [`CStats`](stats.md).

## What each consumable does

| Consumable (prop) | Effect |
|---|---|
| `HealthPotion` | restore **Hp**, clamped to MaxHp (`CStats+0x0c`) |
| `MagicPotion` | restore **Mana**, clamped to its max (`CStats+0x10`) |
| `StaminaPotion` | restore **Stamina** |
| `RestorationPotion` | cure / restore (status removal) |
| `InvisibilityPotion` | grant **invisibility** |
| `ElixerPotion` (+`ElixerBoostDuration`) | a **timed attribute boost** |
| `StrengthPotion` (+`StrengthBoostDuration`) | a timed **Strength** boost |

The instant potions (Health/Magic/Stamina) **add the props amount to the
pool and clamp at the maximum** — the handler reads the pool's cap
(`CStats+0x0c` for Hp, `+0x10` for Mana) and compares (`cmp eax,
[esi+0x0c]`) so a quaff never overfills. The two `*BoostDuration`
consumables (Elixer, Strength) instead push a **timed boost** onto the
[`CStats` boost list](stats.md) (the same `Duration`-bounded mechanism the
buff spells use), so their effect wears off; the `…BoostDuration` prop is
how long.

## How it connects

- **Tiers** — a potion's strength is its tier (1–5) into the
  [`props.000`](formats/props.md) ladder, the same per-level indexing every
  skill and spell uses.
- **Stats** — restores write the [`CStats`](stats.md) pools; the timed
  ones use the boost list (`+0x74`/`+0x78` from stats.md), so Elixir/
  Strength potions are bounded buffs, not permanent.
- **Items** — the consumable is an inventory [item](items.md); using it
  decrements the stack and removes it when depleted ([inventory.md](inventory.md)).
- **Alchemy** — these same potions are what the Survivor **Alchemy** skill
  ([skill-tree.md](skill-tree.md)) brews via the magic-script
  `create object # alchemy level #` command.

## Status

- Handler ✅ — `fcn.00587a20`, a 37-case dispatch (`0x588a68`) applying
  every consumable; magnitude from `props.000` via `fcn.00500f10`.
- Effects ✅ — Health/Magic/Stamina restore (clamped to the `CStats`
  max at `+0x0c`/`+0x10`), Restoration cure, Invisibility, and the
  `Elixer`/`Strength` **timed boosts** (`*BoostDuration` props → boost
  list).
- Per-case exact stat-offset map 🟡 — the prop set and the Hp/Mana max
  offsets (`+0x0c`/`+0x10`) are pinned; the full 37-case → stat-field
  table (e.g. the Stamina/secondary-pool offsets `+0x48`/`+0x8c`/`+0x90`)
  is not split case-by-case.

## Citations

```text
div.exe:0x00587a20   fcn.00587a20   consumable-use handler (37-case switch 0x588a68).
div.exe:0x00500f10   fcn.00500f10   props.000 lookup (effect magnitude by tier).
fields: CStats+0x0c = MaxHp (HealthPotion clamp) · +0x10 = MaxMana (MagicPotion clamp).
props: HealthPotion · MagicPotion · StaminaPotion · RestorationPotion · InvisibilityPotion
       ElixerPotion/ElixerBoostDuration · StrengthPotion/StrengthBoostDuration
```
