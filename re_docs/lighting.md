# Lighting & ambient

How the world is lit: a global **ambient** level that follows the
time of day, plus per-object **light sources**, composited as an
additive **glow** overlay. This is the subsystem that joins the clock,
objects and the renderer.

## Pieces

- **Ambient + twilight (`CTwilight`).** The day/night ambient colour is
  driven by a dedicated **`CTwilight`** object (global `[0x658c24]`,
  built in `.\GAME\compilestart.cpp` `FUN_00499990` and rebuilt at map
  load in `.\GAME\init.cpp` `FUN_004a0b10`; class confirmed by the
  `"CTwilight : %s file not found"` log). It loads the colour ramp from
  **`static\gradsmal.tga`** (`FUN_00597160`) into three parallel
  640-entry float tables — R/G/B — at object offsets `+0x34`, `+0xa34`,
  `+0x1434`. A live cursor `+0x1c` (init `320`, the ramp midpoint) is
  eased one step per frame toward a target index derived from the clock
  hour, and each frame the sampled `(R,G,B)` is pushed to the renderer.
  The transition **window bounds** are tick stamps set by the clock init
  `FUN_0050bea0`: `+0x08 = 14400` (05:00), `+0x0c = 17280` (06:00),
  `+0x00 = 66240` (23:00), `+0x04 = 69120` (24:00 = full day), plus
  per-edge float rate reciprocals at `+0x24/+0x28/+0x2c/+0x30`. So
  sunrise (05:00→06:00) and sunset (23:00→24:00) are the two crossfade
  edges; outside them the ramp sits at its day or night endpoint.

- **Per-frame driver `FUN_0050c000`.** Called from the per-frame update
  `FUN_004ab4a0`. It reads the current hour, picks a day (`9..21`) vs
  night/dusk target index, and calls the `CTwilight` worker
  `FUN_00597740`, which advances the cursor and emits the colour. The
  emit target is the active renderer backend's layer object
  `DllSlashedGetInfo(5)` (global `[0x746a58]`, one of six layer objects
  the loader caches at `[0x746a50..0x746a60]`,
  [`renderer-plugins.md`](renderer-plugins.md)): vtable `+0x7c` takes a
  mode flag plus the three colour floats, vtable `+0x94` takes a scalar;
  at the ramp endpoints the colour is set to `0` (`fldz`). The exact
  method names live in `slash*.dll`, not `div.exe`.

- **Light definitions.** Loaded from `static\light.lib` / `static\light.str`
  (`.\lightlib.cpp`, `FUN_00401490` / `FUN_00402bd0`). *(These files are
  not present in this install — editor/packed only — so the on-disk
  format is not decoded here.)*

- **Light sources.** Objects flagged `sb_light` (the object catalogue,
  [`object-interaction.md`](object-interaction.md)) emit light; a
  `light_value` sets the intensity/colour. Console commands manage them:
  `set agent light $`, `revive lights`, `kill all lights`.

- **Glow overlay.** The lit result is drawn through the renderer
  plugin's additive primitives `DllSlashedGlowDraw{Line,Rect,Square,Quad}`
  ([`renderer-plugins.md`](renderer-plugins.md)) — lighting is an
  **overlay pass**, separate from the sprite blit, so each backend
  (D3D6/Glide/DirectX/Software) renders glows natively.

## Flow

```text
clock hour ──▶ CTwilight cursor ──▶ gradsmal.tga R/G/B ──▶ backend layer 5 tint
sb_light objects ──▶ light sources (intensity = light_value)
        ▼
   composite ──▶ DllSlashedGlowDraw* additive overlay ──▶ frame
```

Related spells/effects: `Light of heaven`, `Daylight`, `CDummyLightSpell`
(magic that alters lighting, [`skills-magic.md`](skills-magic.md)).

## Status

- Architecture ✅ — ambient(time) + `sb_light` sources + additive glow
  overlay via the renderer plugin; the cross-subsystem wiring is
  established.
- Day/night ambient ✅ — `CTwilight` (`[0x658c24]`) samples the
  `static\gradsmal.tga` R/G/B ramp (three 640-entry float tables at
  `+0x34/+0xa34/+0x1434`) and eases a cursor (`+0x1c`) toward a
  hour-derived target, pushing colour to the renderer each frame via
  `FUN_0050c000` → `FUN_00597740`. Crossfade edges are sunrise
  `05:00→06:00` and sunset `23:00→24:00` (tick bounds `14400/17280` and
  `66240/69120`, set by `FUN_0050bea0`).
- Light data files ❓ — `static\light.lib` / `light.str` are not in this
  install; on-disk format not decoded. The twilight ramp
  (`gradsmal.tga`) **is** present, a standard TGA.
- Exact ramp math ✅ **RESOLVED (closed form; two corrections)** —
  `FUN_0050c000` (ecx = CClock) computes a tick-of-day `t` and
  `FUN_00597740(t, force)` maps it to the cursor:
  - **night** (`t < 14400` or `t > 69120`): cursor = 0, emit
    `vtbl+0x7c(mode=2, 0,0,0)` + `+0x94(0.0)`;
  - **day** (`17280 < t < 66240`): cursor = 320, emit pure white
    `(1,1,1)` + `+0x94(1.0)` — **the table is not even sampled**;
  - **sunrise** (`14400..17280`): `n = t−14400`;
    `k = trunc(121·n/2880)`; cursor = `trunc(320·k/121)` (0→317);
  - **sunset** (`66240..69120`): `n = t−66240`; cursor =
    `320 + trunc(320·k/121)` (320→637); scalar = `1 − n/2880`.
  Emit: `R/G/B = float[twi + cursor·4 + 0x34/0xa34/0x1434]` — a
  **single-entry sample; there is NO adjacent-pair lerp** (the earlier
  "(1−t)/t lerp" reading is retracted — the `1/121` rates are
  step→index rescaling and `1/2880` makes the `+0x94` scalar). 121
  quantised steps ≈ one cursor move per 23.8 ticks. **Overrides**:
  the player underground (`[player+0x4a4] ≥ 0x8000`) forces full night
  (`t=0`); active weather adds the **dim accumulator `[0x751824]`**
  (rain/storm ticks raise it, capped at `TicksPerHour/2` = 1440 =
  50 % into the sunset crossfade; the 9..21-hour gate belongs to this
  weather branch only, not the base day/night pick — correcting the
  earlier reading). So there is no separate per-hour target table:
  night=0, day=320, two one-hour index-linear ramps, plus the weather
  and underground overrides.
- Ambient-colour application ✅ — emitted to the active backend's layer
  object `DllSlashedGetInfo(5)` (vtable `+0x7c` colour, `+0x94` scalar).
- Per-source light math ❓ — how a `light_value` becomes glow geometry
  (radius, falloff, colour) is not yet traced.
