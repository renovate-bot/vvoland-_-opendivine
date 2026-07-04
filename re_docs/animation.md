# Animation playback

How object/NPC sprites animate. (The hero's per-equipment sprite
*composition* is separate — see [`clothing.md`](clothing.md); this doc
is the general animation **state machine**: which frames play, and when
they advance.)

## Data model — animations are referenced by index

Every object/agent **class** record (the `.000` class table) carries
animation columns, verbatim from the class-table CSV header:

```text
… AnimationIndex … BreakAnimationIndex … WeaponAnimation … sb_dont_loop_animation …
```

- **`AnimationIndex`** — the class's primary animation (walk / still /
  action sequence).
- **`BreakAnimationIndex`** — the secondary animation played on
  break/destruction (e.g. a barrel smashing).
- **`WeaponAnimation`** — the weapon-specific attack animation overlay.
- **`sb_dont_loop_animation`** — a class flag: when set the sequence
  plays once and holds the last frame instead of looping.

So a class doesn't embed frames; it **indexes** into the animation
table the manager loads.

## The animation manager (`.\MANAGERS\Animan.cpp`)

`Animan` loads and owns the animation definitions. Its loader cluster is
`fcn.004e7b30`, `fcn.004e7d10`, `fcn.004e8070`, `fcn.004e8120`,
`fcn.004e81a0`, `fcn.004e8620` (all linetrack-tagged `.\MANAGERS\Animan.cpp`,
`0x6160d0`). A class with neither a walk nor a still animation errors
`Class %s does not have a walk or a still animation !!!`.

For each frame sequence Animan calls down into the **image manager**
`.\MANAGERS\Imageman.cpp` to build a **56-byte image descriptor**
(`fcn.004e9050` → `fcn.004e8fa0`, `new(0x38)`; both tagged
`.\MANAGERS\Imageman.cpp`, `0x6161c0`, *not* Animan — corrected here). So
the 56-byte record is an **Imageman image/frame descriptor**, owned by the
image manager, that Animan references; the animation *definition* itself
lives in the Animan cluster above. Verified descriptor fields (from
`fcn.004e9050`'s stores): `+0x0c = 1`, `+0x10`/`+0x14` frame/image counts,
`+0x18`/`+0x1c`, a four-field group `+0x20..+0x2c` (zeroed then filled),
`+0x30 = source[+0x10]`, **`+0x34` = the source image-list/manager
pointer**. `fcn.004e8fa0` pre-scans the source as a run of **6-byte image
records** (`add eax, 6`), counting frame/gap transitions to size the
descriptor. The frames themselves are **`.cmp` sprite sequences** (the same
archive format as the hero clothing parts).

There is also a small family of effect-overlay animations
(`CAnimationAttachedEffect`, `CAnimationReversedAttachedEffect`,
`CAnimationVerticalEffect`) for attached visual effects.

## Per-agent playback

The agent struct ([`agent.md`](agent.md)) drives playback:

- **`CurrentAction` (`+0x216`)** selects which animation of the agent's
  class is playing (idle / walk / attack / the scripted action).
- **`Walkspeed` (`+0x217`)** / **`Walkcount` (`+0x278`)** pace movement
  animation against locomotion.
- `agentscript` ([`npc-ai.md`](npc-ai.md)) drives it with commands like
  **`set animation`**, **`reset animation`**, **`start animation for #
  frames`** and **`set weapon animation`** — so animation changes are
  scripted alongside behaviour.

Frames advance on the **40 fps simulation tick** (the main loop,
[`frame-loop.md`](frame-loop.md)); a sequence either loops or, with
`sb_dont_loop_animation`, stops on its last frame. The depth-sorted
blit of the current frame happens in `CSpriteSorter::Render`
([`render-trace.md`](render-trace.md)).

## Status

- Data model ✅ — classes reference animations by **index**
  (`AnimationIndex` / `BreakAnimationIndex` / `WeaponAnimation`), with
  the `sb_dont_loop_animation` loop flag; frames are `.cmp` sprite
  sequences.
- Manager ✅ — `Animan.cpp` loads the animation definitions (loader cluster
  `fcn.004e7b30`..`fcn.004e8620`) and calls **`Imageman.cpp`**
  (`fcn.004e9050`/`fcn.004e8fa0`) to build the **56-byte image/frame
  descriptors** (`+0x34` = source image-list ptr; `fcn.004e8fa0` pre-scans
  6-byte image records) — corrected from the earlier Animan attribution;
  effect-overlay anim classes enumerated.
- Per-agent selection ✅ — `CurrentAction` `+0x216` picks the animation;
  `agentscript` `set/reset/start animation` commands drive it; advance
  on the 40 fps tick.
- Sprite-playback object ✅ (partial) — playback state is a **~0xa8-byte
  object** (constructor `fcn.005465d0` zero/sentinel-inits `+4`..`+0xa4`;
  `u16` at `+0x38` init `0xffff` / `+0x3a` init `0` are the frame / sub-
  frame cursors, `+0x18` and `+0x40` init `-1`, `+0x2c` init `0xff`). The
  agent/projectile holds it (a projectile carries it at `+0x50`,
  [`projectiles.md`](projectiles.md)) and ticks it each frame.
  `fcn.005465d0` is only the **constructor**, *not* the per-frame advance.
- Playback object embedding ✅ — the playback object (ctor `fcn.005465d0`)
  is the **universal embedded sprite-animation state**: its ctor is called
  from ~12 sites — agents (`0x414xxx`), objects, projectiles
  (`0x561xxx`–`0x563xxx`), and the shared sprite helper `fcn.00546ff0`. So
  every drawable carries one.
- Per-frame advance ✅ (resolved — it's the behaviour tick) — the advancer
  is the **agentbehaviour tick `fcn.00411380`** (`agentbehaviour.cpp`), on
  the animation-state object (`this`=`esi`, `edi=[esi+4]`=the agent). Each
  call it does **`inc [esi+0x38]`** — `+0x38` is a **per-tick accumulator**,
  not a raw frame index — then, gated on `agent+0x218 == 1` (active/visible),
  compares it against `[0x65b3d0]+0x40`. **Correction (the "divisor"
  reading was wrong):** `[0x65b3d0]` is the **config/settings singleton**
  (alloc `0x40c183–0x40c1c3`, ctor `fcn.0048d7d0`) and `+0x40` is the
  **game rate in fps** — default **40** (`mov [esi+0x40], 0x28` @
  `0x48d907`; the `smooth` config key maps 1→40 / 0→20 @ `0x48d2f8`;
  the `game rate # fps` key in config.lcl is parsed but its handler is
  a **no-op** — the value is ignored). The `0x4113ca` compare is against
  **2×GameRate (80 ticks ≈ 2 s)** — a staleness/recency gate, *not* a
  ticks-per-frame divisor: while the gate is open, animations can step
  **every 40 fps tick**. The debug HUD prints the field as
  `"Game rate %d"` (`0x4ab7f0`).
- Per-frame duration ✅ (on the playback object) — the frame-hold field
  is **`playback+0x34`**: at `0x411403` the branch does
  `mov eax,[esi+0x34]; test; jle advance; else dec [esi+0x34]; return`
  — i.e. hold the current frame while the counter is > 0, advance when it
  hits 0. So `+0x34` is the per-frame duration/hold counter (distinct from
  the `+0x38` staleness accumulator above). The advance path then computes
  a target grid cell from **agent** position fields (`[edi+0x1c]+[edi+4]`,
  `[edi+0x20]+[edi+8]`, `sar …,5`) and copies frame position bookkeeping
  (`+0x30→agent+0x260`, `+0x2c/+0x1c/+0x20`).
- Descriptor / directional sub-ranges 🟡 — the branch above reads the
  **playback-state object + agent fields**, *not* the separate 56-byte
  Imageman descriptor (`new(0x38)`, frame/image counts at `+0x10`/`+0x14`,
  source ptr `+0x34`, built in `fcn.004e9050`). So the descriptor's
  directional sub-range fields are consulted elsewhere; how the 8 facings
  map to sub-ranges is still unpinned (the frame set is stored as
  concatenated direction blocks — [render-hero](render-hero.md) — so the
  mapping is likely `blockLen·dir` but the field driving it isn't located).
