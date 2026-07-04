# `dat\props.000` — tunable property / progression tables

The game's **balance database**: named numeric properties, almost all
holding a 5-element array indexed by **skill level (1–5)**. This is the
data that drives the data-parameterised skill and magic effects (see
[`../skills-magic.md`](../skills-magic.md)) and combat tables
([`../combat.md`](../combat.md)) — the `CMagic*` effect classes look a
value up here by name rather than hard-coding it.

All integers little-endian, signed (i32).

## Format

```text
+0x00   u32   count                    — number of properties
        Property[count]:
            u32   name_len             — bytes of name incl. trailing NUL
            char  name[name_len]       — NUL-terminated ASCII
            u32   value_count          — entries in the table
            i32   values[value_count]  — the per-level values
```

Self-describing and length-prefixed; the records are variable-size.
The shipped file: `count = 192`, parsing consumes exactly all 9005
bytes. ✅

`value_count` is **5** for 187 of the 192 properties (the five skill
levels); a handful use 1, 2, 4 or 6 entries.

## What it holds

Each property is a small lookup curve. Examples from the shipped file:

```text
BackStabChanceTable      [10, 20, 30, 40, 50]      backstab % by level
HideInShadowsRate        [5, 4, 3, 2, 1]           lower = stealthier
PickPocketLevel          [6, 12, 18, 24, 30]
LockPickLevel            [1, 2, 3, 4, 5]
SkeletonAmount           [1, 2, 3, 4, 5]           summon count
PoisonCloudDamage        [10, 15, 20, 25, 30]
BlessBoostDuration       [15, 30, 60, 90, 120]
HealingBoostRate         [40, 60, 80, 100, 120]
ResurrectLevel           [20, 30, 40, 50, 75]
RepairQuality            [60, 70, 80, 90, 100]
HealthPotion             [20, 40, 100, 200, 400]
ElixerBoostDuration      [100, 200, 300, 400, 500]
```

So a level-3 backstab is 30%, a level-5 healing potion restores 400,
etc. The names map onto the skill/effect taxonomy: thief skills
(`PickPocket`/`LockPick`/`HideInShadows`/`BackStab`), wizard effects
(`PoisonCloud*`/`Bless*`/`Healing*`/`Skeleton*`/`Resurrect*`), warrior
(`PoisonedArrows*`/`WeaponCharming*`), and consumables
(`HealthPotion`/`ElixerPotion`).

Properties are resolved **by name** at load/use time (the engine warns
`Obsolete property '<name>' found, ignoring` for retired keys), the
same name-keyed lookup pattern as the item-stat vocabulary in
[`../inventory.md`](../inventory.md).

## Runtime lookup (`fcn.00500f10`)

The by-name resolver every consumer calls is **`fcn.00500f10`**. On first
use it **lazy-loads** the file into the props manager global **`[0x7402e0]`**
(`fcn.00500eb0`, `dat\props.000` / `dat\props.dat`). The loaded table is
`{ records* @+0, count @+4 }`, and each in-memory property record is
`{ char* name @+0, i32 value_count @+4, i32* values @+8 }`. The lookup
**`stricmp`-scans** the records for the requested name; on a miss it logs
(via `fcn.004f5310`) and returns null, on a hit it optionally writes the
`value_count` to the caller's out-param and **returns the pointer to the
`values[]` array** (`record+8`). The caller then indexes it by
**skill level − 1** itself (so the `CMagic*` / combat / trade code does
`props("Name")[level-1]`). So props is a single global name→curve table
resolved by linear name match — there is no per-call parse, and the same
lookup backs every system that reads a tunable (skills, magic, combat,
trade, boosts).

## Status

- Format ✅ — `u32 count` + length-prefixed `{name, value_count,
  values[]}` records; consumes the file exactly (192 props, 9005 bytes).
- Semantics ✅ — per-skill-level (1–5) progression / tuning curves,
  resolved by name; grounds the skill and magic per-level values.
- Index convention ✅ — array index = skill level − 1 (five entries =
  levels 1–5).
- Name → consumer wiring 🟡 — which effect/skill reads each property is
  inferable from the names but not yet traced to the `CMagic*` call
  sites.
