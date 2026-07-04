# Weather (`.\WORLD\weather.cpp`)

Ambient precipitation and storms — rain, storm, and the lightning
flashes that punctuate a storm. A small polymorphic system: a current
`CWeather` instance ticked each frame, plus discrete `CWeatherAction`
events it spawns.

## Init

Boot step *"Loading rain"* (`fcn.00500f90`, [`architecture.md`](architecture.md))
sets up the precipitation: it runs the animation helpers (`fcn.004e84e0`
Animan, `fcn.004eb3b0`) and then a **`rand()` loop that pre-seeds a pool of
raindrop positions** (random x/y across the screen), with the fall
speed/extent from embedded float constants near the vtable (`7.0`,
`32768.0`). The active weather object itself is built by the manager
`fcn.00597fa0` (driven from `fcn.005983b0`), which allocates a `CWeather`
and promotes it to the concrete subclass.

## Class hierarchy (MSVC RTTI)

```text
CWeather              base                       vtable 0x620b6c
├─ CRainWeather       falling rain               vtable 0x620b88
└─ CStormWeather      rain + wind + lightning    vtable 0x620bbc

CWeatherAction        abstract event (purecall)  vtable 0x620b98
├─ CWeatherActionFlash        lightning flash    vtable 0x620ba4
└─ CWeatherActionTerrorFlash  intense flash      vtable 0x620bb0
```

### `CWeather` virtuals

Diffing the three weather vtables (slot 0 is the per-class scalar-deleting
dtor):

| Slot | Off | Role | Weather / Rain / Storm |
|---:|---|---|---|
| 0 | `+0x00` | dtor | `0x5982f0` / `0x598310` / `0x598410` |
| 1 | `+0x04` | **per-frame tick** (advance + draw precipitation) | `0x597960` / `fcn.00597a90` / `0x597d60` |
| 2 | `+0x08` | base query (shared) / **storm extra** | `0x51b980` / `0x51b980` / `fcn.00597f00` |

So **slot 1 is the weather tick**: `CRainWeather`'s (`fcn.00597a90`) calls
the base advance (`fcn.00597960`), rounds the updated raindrop coordinates
(`fcn.005e5d40`), and blits them (`fcn.00501290`) — i.e. *step each drop
down/across, wrap, draw*. `CStormWeather` overrides both the tick
(`fcn.00597d60`) and slot 2 (`fcn.00597f00`). Slot 2 is **not** the
lightning scheduler (an earlier reading misattributed it): it is a small
timer-gated **state query** — it samples the game clock (`fcn.0040ecb0`),
compares it against a threshold at `+0x20`, and checks the counters at
`+0x24`/`+0x30` (`ret 8`). The lightning scheduling lives entirely in the
**storm tick** (slot 1, `fcn.00597d60`) — see below.

### Storm → lightning scheduling (`fcn.00597d60`, exact cadence)

Each storm tick, after advancing the rain (the rain-step rate is scaled by
the storm via `(100 − (n·100/d))·[esi+0x14]` into the blit `fcn.00501290`),
the storm rolls for a lightning event — but only when **no flash is
currently active** (gated on `[esi+0x30] == 0`, which holds the live
action pointer). The roll is:

```text
r = rand() % 200
  r  < 2   →  spawn CWeatherActionTerrorFlash  (12B, ctor fcn.00597c40)   ~1.0% / tick
  r == 2   →  spawn CWeatherActionFlash        (16B, ctor fcn.00597b10)   ~0.5% / tick
  r  >= 3  →  no flash this tick                                          ~98.5%
```

The new action is stored at `[esi+0x30]` (so only one flash runs at a
time), constructed with the storm's `[esi+0x2c]` parameter. Note the
naming is counter-intuitive: the *more frequent* event (≈1%) is the
**TerrorFlash**, the rarer (≈0.5%) the plain **Flash**.

### `CWeatherAction` events

`CWeatherAction` is abstract (its first two slots are the `_purecall`
stub `0x5e5e44`). The two concrete storm actions both fire the **same
combined lightning effect** — `fcn.00548ad0` — directly from their
constructors (so the flash plays the instant it is spawned):

- `fcn.00548ad0` is **lightning illumination *plus* a thunderclap**: it
  drives the screen light flash (the lighting path,
  [`lighting.md`](lighting.md)) **and** plays a thunder sample through
  FMOD (`FSOUND_Stream_Play` + `FSOUND_SetVolume`,
  [`sound-runtime.md`](sound-runtime.md)) — not light alone.
- **`CWeatherActionTerrorFlash`** (`virtual_0` = `fcn.00597cc0`, ctor
  `fcn.00597c40`) is the common storm flash.
- **`CWeatherActionFlash`** (`virtual_0` = `fcn.00597ba0`, ctor
  `fcn.00597b10`) is the rarer variant.

So a storm's tick periodically (per the roll above) queues a flash action
that briefly lights the scene and cracks thunder, on top of the rain draw.

## Status

- Init ✅ — boot `fcn.00500f90` seeds the raindrop pool (`rand` loop +
  Animan setup); active weather built by manager `fcn.00597fa0`
  (`fcn.005983b0`).
- Class hierarchy ✅ — `CWeather` + `CRainWeather` / `CStormWeather`, and
  `CWeatherAction` + `Flash` / `TerrorFlash` (RTTI, vtables listed).
- Tick ✅ — slot 1 is the per-weather advance+draw (`CRainWeather`
  `fcn.00597a90` → coord round `fcn.005e5d40` → blit `fcn.00501290`);
  storm overrides slot 1 (tick) + slot 2 (`fcn.00597f00`, a timer-gated
  **state query** — clock `fcn.0040ecb0` vs `+0x20`, *not* the lightning
  scheduler, correcting the earlier note).
- Lightning ✅ — the combined effect `fcn.00548ad0` is **flash + thunder**
  (screen light via the lighting path *and* an FMOD thunder stream,
  [`sound-runtime.md`](sound-runtime.md)), fired straight from both action
  ctors.
- Storm→action scheduling ✅ (closed) — scheduled in the storm **tick**
  `fcn.00597d60`, not slot 2: gated on `[esi+0x30]==0` (one flash at a
  time), then `r = rand()%200` → `r<2` spawns `CWeatherActionTerrorFlash`
  (12B, `fcn.00597c40`, ≈1%/tick), `r==2` spawns `CWeatherActionFlash`
  (16B, `fcn.00597b10`, ≈0.5%/tick), else none.

## Citations

```text
div.exe:0x00500f90   "Loading rain" init — seeds raindrop pool (rand loop) + Animan.
div.exe:0x00597fa0   weather manager/factory — allocates CWeather→concrete subclass.
div.exe:0x005983b0   caller that installs the active weather.
div.exe:0x00597a90   CRainWeather tick (slot 1) — advance + draw raindrops (blit fcn.00501290).
div.exe:0x00597d60   CStormWeather tick (slot 1) — rain-step scale + rand()%200 lightning roll.
div.exe:0x00597f00   CStormWeather slot 2 — timer-gated state query (clock fcn.0040ecb0 vs +0x20), ret 8.
div.exe:0x00597c40   CWeatherActionTerrorFlash ctor (12B) — fires fcn.00548ad0.
div.exe:0x00597b10   CWeatherActionFlash ctor (16B) — fires fcn.00548ad0.
div.exe:0x00548ad0   lightning effect — screen flash + FMOD thunder (FSOUND_Stream_Play/SetVolume).
div.exe:0x00597ba0   CWeatherActionFlash::virtual_0.
div.exe:0x00620b6c   vtable.CWeather (0x620b88 Rain, 0x620bbc Storm, 0x620ba4 Flash).
```
