# Full-motion video — Bink playback (`.\GAME\binkplay.cpp`)

The intro logos, the opening cutscenes and the exit movie are **RAD Bink
Video** (`.bik`) files, played through `binkw32.dll`. (Previously only
named in passing in [architecture](architecture.md); this is the playback
mechanism.)

## The `.bik` files (`static\`)

| File | Role |
|---|---|
| `larian.bik` | **Larian** developer logo (played at startup from `main`, `0x40dc6b`) |
| `cdv.bik` | **CDV** publisher logo (played first, `0x40dc63`) |
| `scene1.bik`, `scene2.bik` | opening **intro** cutscenes (scene-index dispatch `0x40dfa0`) |
| `kz.bik` | an additional logo / cutscene (startup `0x40dc86` + in-game `0x40dfdf`) |
| `exitenglish.bik`, `exitgerman.bik` | the **exit** movie (localized by language) |

All 7 names are referenced by code; **only 4 ship in this install**
(`larian`, `kz`, `scene1`, `scene2` — `cdv.bik` and the two `exit*.bik`
are absent from `static\`, so those paths no-op here). The exit movie is
driven by the wrapper `fcn.0040d680` (bik + 3 slide TGAs), selected by
language at `fcn.0047b620` (`[0x65b3d0]+0x124`: 1→english+`slide1e/2e/3`,
2→german+`slide1g/2g`), followed by `PostMessageA(WM_CLOSE)`.

## Bink API surface (imported from `binkw32.dll`)

Nine entry points are used:

```text
_BinkOpen            open a .bik file → a Bink handle
_BinkSetSoundSystem  bind the audio backend (DirectSound)
_BinkOpenDirectSound open the DirectSound output for the movie
_BinkWait            pace to the next frame's presentation time
_BinkDoFrame         decode the current frame
_BinkCopyToBuffer    blit the decoded frame to a target surface
_BinkNextFrame       advance to the next frame
_BinkClose           release the handle
_BinkLogoAddress     the RAD logo data (required for the logo-build licence)
```

## Playback (`binkplay.cpp` methods + the `PlayMovie` driver)

The `binkplay.cpp` unit is a set of small **methods on a movie object**;
the pump *loop* lives in their caller **`fcn.0040d1d0` = `PlayMovie`**
(all 7 movie call sites funnel into it):

- **`fcn.00499200`** — **open a movie**: `_BinkSetSoundSystem(
  _BinkOpenDirectSound, 0)` then `_BinkOpen(name, 0)`; on success
  allocates a `width×height×2` frame buffer (`binkplay.cpp:78`).
- **`fcn.00499290`** — **open the logo**: same, but
  `_BinkOpen(_BinkLogoAddress(), 0x4000000 /*from-memory flag*/)` — the
  RAD-required logo build (`binkplay.cpp:97`).
- **`fcn.00499380`** — **do one frame**: `_BinkDoFrame` →
  `_BinkCopyToBuffer` (two paths: direct-to-surface via the renderer
  vtable `[0x746a5c]`, or into the frame buffer then row-blit via
  `fcn.00499110`) → conditional `_BinkNextFrame` (skipped on last frame).
- **`fcn.00499330`** — thin `_BinkWait` wrapper; **`fcn.00499320`** —
  finished? (`frame# >= framecount`); **`fcn.00499340`** — `_BinkClose` +
  free.
- **`fcn.00499110`** — a **row-blit helper** (stride math + per-row
  `memcpy` into the target surface, `binkplay.cpp:38`). *(An earlier
  reading called this "the frame pump" — retracted: it contains zero Bink
  calls; the pump loop is in `fcn.0040d1d0`: Wait `0x40d404` → DoFrame
  `0x40d47a` → finished? `0x40d4e6` → loop `0x40d526` → Close `0x40d61b`.)*

So the flow is `PlayMovie(name)` → open (+ sound; logo variant for the
RAD logo) → `{Wait, DoFrame+CopyToBuffer, NextFrame}` loop → close, with
audio through **DirectSound**. The startup path (`main`) plays
`cdv.bik` → `larian.bik` (→ `kz.bik`) before the menu; the intro
`scene*.bik` and the localized `exit*.bik` (via `fcn.0040d680`) play at
game start and quit.

## Status

- File set ✅ — 7 `.bik` names referenced (2 logos, 2 intro, `kz`, 2
  localized exit); 4 shipped in this install (no `cdv`/`exit*`).
- API surface ✅ — the 9 `binkw32` imports above (stdcall-decorated:
  `_BinkOpen@8`, `_BinkCopyToBuffer@28`, …), IAT `0x6064a8..0x6064c8`.
- Playback ✅ — driver `fcn.0040d1d0` (loop) + the `fcn.0049932x/3xx`
  method set; open `fcn.00499200` / logo `fcn.00499290`; startup logos
  triggered from `main`.
- Frame-pump order ✅ — Wait → DoFrame → CopyToBuffer → NextFrame → until
  finished, enumerated instruction-level above (former 🟡 resolved).

## Citations

```text
div.exe:0x0040d1d0   PlayMovie driver — owns the Wait/DoFrame/finished?/Close loop (callers: 0x40d6d9/0x40d6e9 exit, 0x40dc63 cdv, 0x40dc70 larian, 0x40dc86 kz, 0x40dfb1 scene2, 0x40dfc8 scene1, 0x40dfdf kz).
div.exe:0x00499200   open movie — BinkSetSoundSystem(BinkOpenDirectSound) + BinkOpen(file) + framebuffer alloc (.\GAME\binkplay.cpp).
div.exe:0x00499290   open logo — BinkOpen(BinkLogoAddress(), 0x4000000).
div.exe:0x00499380   frame step — BinkDoFrame + BinkCopyToBuffer (direct or via row-blit 0x499110) + BinkNextFrame.
div.exe:0x00499330   BinkWait wrapper · 0x00499320 finished? · 0x00499340 BinkClose+free.
div.exe:0x00499110   row-blit helper (per-row memcpy; NOT the pump).
div.exe:0x0040dc6b   main — plays static\larian.bik at startup (cdv at 0x40dc63, kz at 0x40dc86).
div.exe:0x0040d680   exit-movie wrapper — bik + slide TGAs; language select fcn.0047b620 ([0x65b3d0]+0x124).
imports: binkw32.dll  _BinkOpen@8/_BinkClose@4/_BinkDoFrame@4/_BinkNextFrame@4/_BinkWait@4/
                      _BinkCopyToBuffer@28/_BinkOpenDirectSound@4/_BinkSetSoundSystem@8/_BinkLogoAddress@0.
```
