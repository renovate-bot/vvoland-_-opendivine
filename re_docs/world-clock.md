# World clock & day/night

The in-game clock that advances time and drives the day/night ambient
lighting — the "Day/night cycle" STATUS item.

## Two clocks: in-game `CClock` vs real-time `Gameclock`

There are **two distinct clocks**, easily confused:

- **`CClock`** (`[0x658c1c]`, this doc) is the **in-game** clock —
  `TickCounter` / `Hour` / `Day`, advanced by the simulation tick. It
  drives day/night, perception, combat-time, shop hours, etc. (the
  time-of-day consumer table below). Its only calendar fields are a flat
  **`Day` counter and `Hour`** — there is **no month / week / season**.
- **`Gameclock`** (save block `GameclockV0.935`, reader `FUN_00505b90`,
  [`formats/savegame.md`](formats/savegame.md)) is the **real-time / wall**
  clock: a raw 36-byte (`0x24`) state read straight back, after which the
  loader **re-baselines the live fields from `timeGetTime`** (WINMM) —
  i.e. it tracks elapsed real time (play-time / timing), not in-world date.
  So saving/reloading re-anchors the wall clock to the current real time
  while the in-game `CClock` day/hour are restored separately.

So "what day/hour is it in the world" is `CClock`; "how much real time has
elapsed" is `Gameclock`. A port needs the `CClock` Day/Hour model and can
treat `Gameclock` as a real-time stopwatch.

## Clock struct (`CClock`, global `[0x658c1c]`)

Layout taken from the **accessor and advance logic** (authoritative),
not the debug overlay — the overlay's printf labels are crossed (see
the note below).

| Offset | Field | |
|---|---|---|
| `+0x00` | `TickCounter` | monotonic; `+1` on every `Advance` |
| `+0x04` | `HourCountdown` | ticks left in the current hour; when it expires the hour rolls |
| `+0x08` | `Day` | day counter |
| `+0x0c` | `Hour` | 0–23 |
| `+0x10` | `TicksPerHour` | reload value for `HourCountdown` |
| `+0x14` | `Freeze` | nonzero pauses hour/day advancement |

`GetHour` (`FUN_0050bfe0`) returns `(Hour + [0x650f88]) % 24`, with the
global offset `[0x650f88] = 0` in the shipped build — i.e. just the
`Hour` field mod 24. `Day = TickCounter`-derived via the wrap in
`Advance`.

> Correction: the debug overlay `FUN_004ab750` prints
> `Hour=%d Day=%d Remaining=%d` from `[clock+4] / +8 / +0xc`, which
> labels `+0x04` as "Hour" and `+0x0c` as "Remaining". That is
> backwards: `GetHour` and the wrap logic in `Advance` prove `+0x0c` is
> the **Hour** (it is what wraps 23→0 and rolls the day) and `+0x04` is
> the in-hour countdown. Trust the code, not the overlay label.

## Advancement (`FUN_0050bf40`)

Called once per game tick (from the main update path —
`FUN_00499990`, `FUN_004a1c80`, `FUN_00505bc0`):

```c
void CClock::Advance() {
    this->TickCounter++;                 // [+0x00]
    if (this->Freeze) return;            // [+0x14] nonzero → time frozen
    if (this->HourCountdown > 0) {       // [+0x04] still inside the hour
        this->HourCountdown--;
        return;
    }
    this->HourCountdown = this->TicksPerHour;   // reload [+0x04] from [+0x10]
    this->Hour++;                               // [+0x0c]
    if (this->Hour > 23) {                      // wrap
        this->Day++;                            // [+0x08]
        this->Hour = 0;
    }
}
```

So one game hour = `TicksPerHour` advance-ticks; the real-seconds-per-
game-hour rate is `TicksPerHour ÷ (advance calls per second)`.

## Construction (`FUN_0050be80` + `FUN_0050bea0`)

The clock is created in `FUN_00499990` (`.\GAME\compilestart.cpp`):

```text
FUN_0050be80   ctor — zeroes +0x00,+0x04,+0x08,+0x0c,+0x14
[0x658c1c] = clock
FUN_0050bea0(2880, 120, 120)   init:
    clock->TicksPerHour  = 2880   ([+0x10])
    clock->HourCountdown = 2880   ([+0x04])
```

So **`TicksPerHour = 2880`** (a game hour is 2880 advance-ticks; a game
day is `2880 × 24 = 69120`).

The same init (`FUN_0050bea0`) is **shared with the `CTwilight` object**
at `[0x658c24]` (the day/night ambient-colour driver, see
[`lighting.md`](lighting.md)): it writes that object's transition-window
bounds as tick stamps (`hour × 2880`):

```text
+0x08 = 2880 ×  5 = 14400   (05:00, sunrise start)
+0x0c = 2880 ×  6 = 17280   (06:00, sunrise end)
+0x00 = 2880 × 23 = 66240   (23:00, sunset start)
+0x04 = 2880 × 24 = 69120   (24:00, full day)
```

— i.e. the dawn (`05:00→06:00`) and dusk (`23:00→24:00`) crossfade
edges, plus per-edge float rate reciprocals at `+0x24/+0x28/+0x2c/+0x30`.
*(Earlier notes called `[0x658c24]` a standalone "hour-boundary
timestamp table"; it is actually the `CTwilight` instance, and these are
its gradient-window bounds.)*

## Day / night threshold

The per-frame window routine `FUN_0040d1d0` (the Windows message pump +
shroud/resolution handling, called from `main`'s loop — see
[`frame-loop.md`](frame-loop.md)) **also** classifies the current hour
for lighting:

```c
h = GetHour();                 // FUN_0050bfe0
if (h <= 5 || h >= 22)
    night_branch();            // 22:00–05:59
else
    day_branch();              // 06:00–21:59  (reads the hour for the gradient)
```

So **night** is hours `{22, 23, 0, 1, 2, 3, 4, 5}` and **day** is
`{6 … 21}`. The day branch reads the exact hour to drive the ambient
gradient between dawn/dusk; the night branch takes a fixed darker
ambient. A second consumer, `FUN_0050c000` (called from the per-frame
update `FUN_004ab4a0`), uses a `9..21` day window to pick the target
gradient index and drives the **`CTwilight`** ambient-colour object
`[0x658c24]` — i.e. it **is** lighting, not spawn behaviour (corrects an
earlier guess); see [`lighting.md`](lighting.md).

## Time-of-day consumers

`GetHour` (`FUN_0050bfe0`) is read across the engine — time-of-day is a
pervasive input, not just a lighting concern. The callers, by subsystem:

| Consumer | What the hour gates |
|---|---|
| Lighting / `CTwilight` (`FUN_0050c000`, documented above) | day/night ambient gradient (night `≤5 \|\| ≥22`) |
| **Perception** (`FUN_004356f0`, [`npc-ai.md`](npc-ai.md)) | sight/hearing range — reduced at night |
| **Combat** (`FUN_005512c0` / `FUN_00552900`, [`combat.md`](combat.md)) | the time-of-day modifier on the opposed to-hit chance |
| **Trade** (`FUN_00435410` / `FUN_00435570`, agenttrade.cpp, [`items.md`](items.md)) | merchant availability / greeting by hour |
| **Daytime-window helper** (`FUN_004f0060`) | an explicit **08:00–17:00** check (`cmp hour,8` / `cmp hour,0x11`) — "business hours" for NPC activity |
| Spawn / monster-gen (`FUN_00441770`, the `0x441` area-spawn path that also runs the cell query) | time-gated (re)spawns |
| Magic (`0x4d0ebe`) | a time-dependent spell/effect |
| Diary / UI (`FUN_00481770` / `FUN_004819c0`) | the on-screen clock readout |
| Main update (`FUN_0040d1d0`) | top-level per-tick hour checks |

So the world clock drives **perception, combat hit-chance, merchant hours,
NPC daytime activity (the 8–17 window), spawns, and lighting** off the
single `Hour` field — the day/night cycle is wired into gameplay, not
cosmetic. (A few callers are thin `GetHour` wrappers like `FUN_004f0060`
that expose an is-daytime predicate the gameplay code branches on.)

## Citations

```text
div.exe:0x00658c1c   DAT_00658c1c   ptr to the world clock object (CClock).
div.exe:0x0050be80   FUN_0050be80   CClock ctor — zeroes the fields.
div.exe:0x0050bea0   FUN_0050bea0   CClock init(2880,…) — sets TicksPerHour and the
                                    CTwilight ([0x658c24]) crossfade-window bounds.
div.exe:0x00658c24   DAT_00658c24   ptr to the CTwilight day/night ambient object.
div.exe:0x0050c000   FUN_0050c000   per-frame CTwilight driver (from FUN_004ab4a0).
div.exe:0x00597740   FUN_00597740   CTwilight worker — eases the gradient cursor + emits colour.
div.exe:0x00597160   FUN_00597160   CTwilight loader — reads static\gradsmal.tga into the ramp.
div.exe:0x00499990   FUN_00499990   creates the clock (.\GAME\compilestart.cpp).
div.exe:0x0050bf40   FUN_0050bf40   CClock::Advance — per-tick; rolls hour/day.
div.exe:0x0050bfe0   FUN_0050bfe0   CClock::GetHour — (Hour + [0x650f88]) % 24.
div.exe:0x0040d1d0   FUN_0040d1d0   per-frame window pump/throttle routine; among other
                                    things classifies night (hour<=5 || hour>=22) for lighting.
div.exe:0x004ab750   FUN_004ab750   debug overlay (labels crossed — see note).
```

## Status

- Clock struct ✅ — all six fields decoded from `GetHour` + `Advance`
  (corrects the overlay's crossed Hour/Remaining labels).
- Advancement algorithm ✅ — tick → countdown → hour → day, with a
  freeze gate; `FUN_0050bf40`.
- Day/night threshold ✅ — night `hour<=5 || hour>=22`, day `6..21`.
- `TicksPerHour` value ✅ — `2880`, set by `FUN_0050bea0`; a game day is
  `69120` ticks.
- Wall-clock rate ✅ — the advance runs **once per main-loop frame**:
  the frame loop in `main` (`FUN_0040d850`) calls `FUN_004ab4a0` →
  `FUN_00505bc0` → `CClock::Advance` each iteration, then loops back
  (`0x0040e3a6 je 0x0040df30`). Capped by `config.lcl`'s `game rate 40
  fps` ([`formats/config.md`](formats/config.md)), so a game **hour =
  2880/40 = 72 real seconds** (day ≈ 28.8 min).
- `CTwilight` object `[0x658c24]` ✅ — the day/night ambient driver, not
  a standalone timestamp table; `FUN_0050bea0` writes its crossfade
  bounds (sunrise `14400/17280`, sunset `66240/69120`). See
  [`lighting.md`](lighting.md).
- The `120`/`120` init params ✅ (easing reduced) — `FUN_0050bea0(2880,
  120, 120)` is a thiscall on `CTwilight` (`[0x658c24]`) that first derives
  the four boundary ticks as **integer multiples of `TicksPerHour=2880`**:
  `5·2880 = 14400` / `6·2880 = 17280` (sunrise start/end, `+0x08`/`+0x0c`)
  and `23·2880 = 66240` / `24·2880 = 69120` (sunset start/end, `+0x00`/
  `+0x04`) — i.e. sunrise spans hours 5→6 and sunset hours 23→24, each a
  one-hour (2880-tick) window. The two `120`s are the **crossfade step
  counts**: each is incremented (`inc ecx`, a `+1` fencepost so the ramp
  reaches its endpoint) and used as the divisor for the per-step rate
  reciprocals at `+0x24`/`+0x28`/`+0x2c`, with `+0x30 = 1.0/2880` the base
  per-tick fraction. So the day↔night colour transition is a **linear
  ramp over 120 steps** across each one-hour window — not a non-linear
  easing curve.
- Ambient-colour application ✅ — traced: `FUN_0050c000` →
  `FUN_00597740` samples `gradsmal.tga` and pushes R/G/B to the renderer
  backend's layer object (`DllSlashedGetInfo(5)`); see
  [`lighting.md`](lighting.md).
