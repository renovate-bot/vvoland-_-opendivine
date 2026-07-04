# Pain points — lingering damage fields (`.\MISC\painpoint.cpp`)

A **pain point** is a persistent, area damage source dropped into the
world — the runtime behind the **lingering cloud / elemental** spells
(PoisonCloud, WallOfSmoke, Freeze, Sparks, …): a zone that keeps damaging
whatever stands in it for its lifetime, distinct from a one-shot hit or an
[explosion](explosions.md)'s single blast.

## Class hierarchy (MSVC RTTI)

`.\MISC\painpoint.cpp`, ctor `fcn.004fd480`. Four concrete kinds by
damage model:

| Class | vtable | Model |
|---|---|---|
| `CSingularDamagePainpoint` | `0x616e0c` | one-shot point damage |
| `CSustainedDamagePainpoint` | `0x616df4` | **damage over time** for the field's lifetime |
| `CPoisonDamagePainpoint` | `0x616e24` | poison DoT (element-typed) |
| `CFreezeDamagePainpoint` | `~0x616e3c` | freeze DoT (element-typed) |

(The two on-disk record sizes `0x4c`/`0x54` in the save block correspond to
the singular vs. the sustained/element-typed variants.)

## Creation, ticking, persistence

- **Created by the magic effects.** The ctor `fcn.004fd480` is called from
  ~8 sites in the `0x4c1…0x4c3` `CMagic*` effect cluster — the
  elemental/cloud spells ([`skills-magic.md`](skills-magic.md)). So when a
  cloud spell lands it *drops a pain point* rather than applying damage
  directly (the elemental-damage verbs' "drop a damaging cloud" path).
- **Manager** — the painpoint list lives at the global **`[0x6e02bc]`**;
  each frame the active pain points damage the agents inside their area
  (the area query is the spatial [`cell grid`](cell-grid.md), the damage is
  the shared combat HP-apply `fcn.00417550`, [`combat.md`](combat.md)),
  decrementing the field's lifetime until it expires.
- **Persistence** — saved as the **`Painpoints`** block (reader
  `FUN_004fd9c0`, manager `[0x6e02bc]`, type-tagged `0x4c`/`0x54` records,
  [`formats/savegame.md`](formats/savegame.md)), so a burning/poison field
  survives a save/reload.

So pain points are the engine's **area damage-over-time** primitive,
shared by the cloud spells (and trap/effect sources): spell → drop a
pain point → per-frame area damage via the cell grid + combat HP-apply →
expire. This connects the elemental spells, the cell grid, the combat
damage path, and the savegame block.

## Status

- Class set ✅ — `CSingular`/`CSustained`/`CPoison`/`CFreezeDamagePainpoint`
  (RTTI, `.\MISC\painpoint.cpp`, ctor `fcn.004fd480`).
- Creation ✅ — dropped by the `0x4c1…0x4c3` `CMagic*` elemental/cloud
  effects (the "drop a damaging cloud" path).
- Manager + persistence ✅ — global `[0x6e02bc]`; `Painpoints` save block
  `FUN_004fd9c0` (type-tagged `0x4c`/`0x54` records).
- Per-frame damage tick 🟡 — area-damage model is clear (cell-grid victims
  × combat HP-apply over a lifetime); the exact tick fn + per-record field
  layout (radius / damage / lifetime offsets) not split field-by-field.

## Citations

```text
div.exe:0x004fd480   fcn.004fd480   CSustainedDamagePainpoint ctor (.\MISC\painpoint.cpp).
div.exe:0x004fd9c0   FUN_004fd9c0   Painpoints save block reader (mgr [0x6e02bc]).
div.exe:0x006e02bc   DAT_006e02bc   pain-point manager (active damage fields).
vtables: 0x616e0c Singular · 0x616df4 Sustained · 0x616e24 Poison.
```
