# Visual effects — `CAniEffect` / attached effects (`.\MISC\Anieffect.cpp`)

The engine's **world visual-effect** primitive: a short-lived animated
sprite spawned at a point in the world — the hit sparks on a sword blow,
the glow of a cast spell, an explosion flash, a death dissolve, a laser
beam, a flickering light. Distinct from [`screen-effects.md`](screen-effects.md)
(full-screen post-process tints) — these are ordinary world-space sprites
that live in the [cell grid](cell-grid.md), tick down a lifetime, and draw
themselves additively or alpha-blended.

There are two families:

- **`CAniEffect` / `CAniObject`** (`.\MISC\Anieffect.cpp`) — free-standing
  effects placed in the world, registered in the cell grid.
- **Attached effects** (`.\AGENTS\agenteffect.cpp`) — effects parented to a
  living [`CAgent`](agent.md)'s animation (gore, the equipment-visualizer
  overlay), ticked by the agent rather than the world.

## `CAniEffect` class hierarchy (MSVC RTTI)

Base **`CAniObject`** (vtable `0x6163e4`) → **`CAniEffect`** (`0x6163c0`),
a **0xa8 (168) byte** struct, base ctor `fcn.004edbf0`. The concrete kinds
(all share the same 5-slot vtable shape, overriding update/render):

| Class | vtable | Note |
|---|---|---|
| `CAniEffect_PlayAnimation` | `0x616414` | plain sprite playback (ctor `fcn.004edef0`) |
| `CAniEffect_PlayAnimationAdditive` | `0x609038` | **additive** blend — the universal hit/impact flash |
| `CAniEffect_PlayAnimationAlpha8` | `0x6143d4` | alpha (`0x800000`) blend |
| `CAniEffect_PlayAnimationAdditiveWithLight` | `0x61648c` | additive + a [light source](lighting.md) |
| `CAniEffect_PlayAnimationAdditiveAttachedToNpcCenter` | `0x6164a4` | additive, anchored to an NPC's centre |
| `CAniEffect_PlayLight` | `0x61642c` | a moving light, no sprite |
| `CAniEffect_Laser` | `0x616444` | beam |
| `CAniEffect_MummyAppear` | `0x6163fc` | scripted one-off |
| `CAniEffect_CartExplosion` | `0x61645c` | scripted one-off |
| `CAniEffect_Thelyron` | `0x616474` | scripted one-off (the Thelyron quest set-piece) |

## Struct layout (confirmed fields)

| Offset | Field |
|---|---|
| `+0x00` | vtable |
| `+0x04`/`+0x08`/`+0x0c` | cell-grid coords / **cell handle** (`-1` = not registered), set by `fcn.004eca40` |
| `+0x18`/`+0x1c` | animation references (image-list / frame index) |
| `+0x28`/`+0x2c`/`+0x30` | **world position** x / y / z (floats) |
| `+0x34`/`+0x38`/`+0x3c` | **per-frame velocity** vx / vy / vz (floats) |
| `+0x44` | the owned [`CAnimation`](animation.md) sprite object (alloc `fcn.00546ff0`) |
| `+0x50`/`+0x54` | render **blend mode** params (set by `virtual_16`) |
| `+0x60` | flags (bit 1 = alpha, bit 2 = …) |
| `+0x6c` | **lifetime — frames remaining** (`virtual_12` decrements; `0` → expire) |

## Virtual table (5 slots)

| Slot | Base fn | Role |
|---|---|---|
| `+0x00` `virtual_0` | `0x4eca20` | destructor |
| `+0x04` `virtual_4` | `fcn.004ec960` | **serialize-read** — pulls the effect's state from the savegame stream `[0x6e0124]` via the save read-prim `fcn.004f4c70` ([`formats/savegame.md`](formats/savegame.md)); rebuilds the `+0x44` sprite |
| `+0x08` `virtual_8` | `fcn.004160c0` | deleting destructor (dtor body `fcn.004edce0`, then free) |
| `+0x0c` `virtual_12` | `fcn.004ed5f0` | **update / tick** — `[+0x6c]--`; `position += velocity` (`+0x28 += +0x34`, …); advance frame + re-register in the cell grid (`fcn.004eca40`); at `0` the effect expires |
| `+0x10` `virtual_16` | `fcn.004ed6d0` | **set render blend** — additive writes `[+0x50]=0x200000`; the Alpha8 override (`fcn.004ed6f0`) sets `or [+0x60],2` / `[+0x50]=0x800000` |

So an effect's life is: construct at a point → each frame slide by its
velocity and advance its animation, decrementing the frame counter → when
the counter hits zero, expire and free. The tick (`virtual_12`) is invoked
polymorphically with **no direct xref** — an effect-list manager walks the
live effects each frame and calls it (consistent with the per-frame update
order in [`frame-loop.md`](frame-loop.md)).

## Spawning — `fcn.00416050`, the universal impact flash

`fcn.00416050` constructs a `CAniEffect_PlayAnimationAdditive` in a caller-
owned buffer (calls the `PlayAnimation` ctor `fcn.004edef0`, then stamps the
Additive vtable `0x609038`, `+0x24 = 4`, `+0x20 = 0x70`). It is the engine's
one **"drop an additive impact effect here"** helper, called from **18+**
sites:

- **[combat](combat.md)** — `fcn.00417b40` (`.\AGENTS\agentfight.cpp`) spawns
  it for each element of a landed hit (the spark/slash flash).
- **[explosions](explosions.md)** — `0x575b2f`/`0x575be0`/`0x575e01`/`0x575edd`.
- **[pain points](painpoints.md)** — `0x4fd1d7` (cloud/elemental field FX).
- **[magic](skills-magic.md)** — `0x4d6af6`/`0x4da3d6`/`0x4db89a`/… (spell death
  effects `CDeathEffect{1..4}Magic`).
- **[doors / objects](object-interaction.md)** — `0x442626`.

So every hit, spell, explosion, and cloud drops one of these effects at its
impact point; they are purely cosmetic (no damage — that is the separate
[combat](combat.md) / [pain-point](painpoints.md) path) but they **persist**:
because `virtual_4` serializes them, an effect mid-animation survives a
save/reload.

## Attached effects (`.\AGENTS\agenteffect.cpp`)

A parallel family parented to an agent's animation rather than the world,
ticked by the owning agent. Vtables in the `0x608f90..0x608ff4` block,
methods in the `0x414xxx` range (alloc/manager `fcn.00413ec0`):

| Class | vtable |
|---|---|
| `CAgentVisualizerAttachedEffect` | `0x608f90` |
| `CAnimationAttachedEffect` | `0x608fb4` |
| `CGoreAttachedEffect` | `0x608fd4` |
| `CAnimationReversedAttachedEffect` | `0x608fe4` |
| `CAnimationVerticalEffect` | `0x608ff4` |

`CGoreAttachedEffect` is the blood/gore overlay ([`minor-mechanics.md`](minor-mechanics.md));
`CAgentVisualizerAttachedEffect` ties into the equipment-driven hero sprite
([`render-hero.md`](render-hero.md) / [`clothing.md`](clothing.md)).

## Status

- `CAniEffect` hierarchy ✅ — base `CAniObject`/`CAniEffect` + 10 concrete
  subclasses (RTTI, vtables, `.\MISC\Anieffect.cpp`, 0xa8-byte struct).
- Lifecycle ✅ — ctor `fcn.004edbf0`/`fcn.004edef0`; `virtual_12` update
  (lifetime `+0x6c`, position `+0x28..0x30` += velocity `+0x34..0x3c`,
  cell-grid move `fcn.004eca40`); `virtual_16` blend mode; `virtual_4`
  save-serialize via `[0x6e0124]`.
- Spawn ✅ — `fcn.00416050` = universal additive-impact FX, 18+ callers
  across combat / explosions / pain points / magic / doors.
- Attached-effect family ✅ (enumerated) — `agenteffect.cpp`, vtables
  `0x608f90..0x608ff4`, manager `fcn.00413ec0`.
- Per-frame effect-list manager 🟡 — `virtual_12` is dispatched
  polymorphically (no direct xref); the global list that walks live effects
  each frame is not pinned to a single address.

## Fire engine — `CFireEng` (`.\effect\fireeng.cpp`)

A separate, dedicated **animated-flame** FX, distinct from the `CAniEffect`
sprite effects above. Two RTTI classes: **`CFireEng`** (the abstract base)
and **`CFireEngFlameEffect`** (one flame instance). The algorithm is now
**fully decoded** — a classic DOOM-style rising-fire cellular buffer:

- **Field**: a 128×192 8-bit intensity buffer (`fcn.00492fc0` base ctor:
  width/height/w·h at `+0xc/+0x10/+0x14/+0x18`, buffer at `+0x04`/`+0x08`).
- **Kernel** (`fcn.004932d0`, vtable slot 3 of `CFireEngFlameEffect`):
  for all rows but the bottom two (which are zeroed),
  `buf[x,y] = max(0, (buf[x−1,y+1] + buf[x,y+1] + buf[x+1,y+1] +
  buf[x,y+2])/4 − cooling)`, cooling = `[this+0x1c]`.
- **Seeding** is the *caller's* job (seeder helper `fcn.00493390`): the
  **sole client is the Strength buff** (`CStrengthSpell`, ctor xref
  `0x4e482c`, cooling set to 8 at `0x4e484b`), which stamps the buffed
  character's current animation frame as `rand()%256` noise into the
  field each tick — the burning-silhouette visual.
- **Emit** (slot 2 `fcn.00493020`): `dst16[i] = palette[buf[i]]` where
  the palette is the **256-entry fire gradient at `0x650340`** — a
  **768-byte RGB888 table** (`R,G,B`; every channel a multiple of 4, i.e.
  6-bit VGA precision `<<2`). Ramp (dumped): entries 0–7 a short dark
  **blue/violet ember** (only B rises, to `#000028`), 8–24 B falls while
  R rises to `#800000` (deep red), 25–~187 R saturates at `0xFC` then G
  rises (`#fc2400` orange → yellow), ~188–255 R,G at `0xFC` and B rises
  (`#fcfce0` → `#fcfcfc` white). The spell ctor (`0x4e48a0`) is an
  unrolled **RGB888→RGB565** packer (`(R>>3)<<11 | (G>>2)<<5 | (B>>3)`;
  the `[0x746a60]` guard selects the packer, but it is 565).

Three corrections to the earlier pass: `0x6dc3a8` is **not a palette
LUT** — it is bss scratch (u16 run pairs) private to `fcn.00493050`;
`fcn.00493050` (586 B, shared vtable slot 1) is not the flame generator
but an **RLE-sprite compiler** (8-bit field → the game's transparent
16-bit sprite format, palette passed as an argument); and vtable slot 0
`0x4937a0` is the **scalar deleting destructor**, not a shared drawing
method. Vtables stand as pinned: `CFireEng` = `0x610694` (abstract,
slot 3 `__purecall`), `CFireEngFlameEffect` = `0x6106c0` (derived ctor
`fcn.004932a0`). (Roster context: the fire *spells*
`CFireBallSpell`/`CMagicFireDamage`/`CDragonFire` are the
[skills-magic](skills-magic.md) effect classes; `CFireEng` is a
flame-rendering primitive — and in the shipped game it renders the
Strength buff, not torches.) No 🟡 left.

## Citations

```text
div.exe:0x004edbf0   fcn.004edbf0   CAniObject base ctor (.\MISC\Anieffect.cpp, 0xa8 bytes).
div.exe:0x004edef0   fcn.004edef0   CAniEffect_PlayAnimation ctor (vtable 0x616414).
div.exe:0x004ed5f0   fcn.004ed5f0   CAniEffect virtual_12 — update (lifetime--, pos+=vel, frame advance).
div.exe:0x004ed6d0   fcn.004ed6d0   CAniEffect virtual_16 — additive blend ([+0x50]=0x200000).
div.exe:0x004ec960   fcn.004ec960   CAniEffect virtual_4 — serialize-read from stream [0x6e0124].
div.exe:0x004eca40   fcn.004eca40   move effect to (x,y,z) + cell-grid re-register.
div.exe:0x00416050   fcn.00416050   universal additive-impact FX spawner (18+ callers).
div.exe:0x00413ec0   fcn.00413ec0   attached-effect alloc/manager (.\AGENTS\agenteffect.cpp).
div.exe:0x004932d0   CFireEngFlameEffect kernel (vtable slot 3) — DOOM-fire propagate: avg of 3-below + 2-below, /4, − cooling [this+0x1c].
div.exe:0x00493050   shared vtable slot 1 — RLE-sprite compiler (8-bit field → 16-bit sprite; scratch 0x6dc3a8).
div.exe:0x00493020   slot 2 — emit dst16[i] = palette[buf[i]] (fire gradient 0x650340, 256×RGB888).
div.exe:0x00493390   field seeder helper (caller stamps silhouette noise).
div.exe:0x00492fc0   CFireEng base ctor (vtable 0x610694; w/h/w·h fields + work buffer).
div.exe:0x004932a0   CFireEngFlameEffect ctor — calls 0x492fc0, swaps vtable to 0x6106c0.
div.exe:0x004e482c   CStrengthSpell — the sole CFireEng client (cooling 8 @0x4e484b; palette conv 0x4e48a0).
vtables: CAniEffect 0x6163c0 · CAniObject 0x6163e4 · PlayAnimation 0x616414 · PlayAnimationAdditive 0x609038 · CFireEng 0x610694 · CFireEngFlameEffect 0x6106c0.
```
