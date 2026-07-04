# Main loop & per-frame update

The top of the engine: the loop that drives one simulation+render frame
and ties every subsystem together.

## The loop (`main`, `FUN_0040d850`)

`main` runs a classic frame loop: call the per-frame update, check the
quit flag, repeat.

```text
0x0040df30:   loop top
   …          (per-frame work)
0x0040e39b    call FUN_004ab4a0          ; the per-frame update
0x0040e3a0    cmp  [0x658cdc], 0         ; quit flag set?
0x0040e3a6    je   0x0040df30            ; no → loop
```

The loop rate is capped by `config.lcl`'s `game rate 40 fps`
([`formats/config.md`](formats/config.md)) — so the simulation steps 40
times per real second. (This is what fixes the day/night wall-clock
rate; see [`world-clock.md`](world-clock.md).)

## The per-frame update (`FUN_004ab4a0`)

A ~6 KB function that calls ~40 subsystem updates in order, followed by
the debug overlay `FUN_004ab750` (which prints FPS / agent counts /
`Hour=…`). It is the single place where each documented subsystem ticks
once per frame. The confirmed anchors, in execution order:

| Order | Call | Subsystem |
|---|---|---|
| early | `FUN_0050a7f0` | command/message processing (near the dispatch `FUN_0050a290`, [`messages.md`](messages.md)) |
| ~mid | **`FUN_00505bc0`** | `CClock::Advance` — world clock tick ([`world-clock.md`](world-clock.md)) |
| mid | `FUN_004a….` cluster | agent updates — movement / AI / combat (the `.\AGENTS\` range) |
| mid | **`FUN_00561b30`** | projectile update ([`projectiles.md`](projectiles.md)) |
| late | `FUN_0050c000` | day/night-dependent update (reads the clock hour) |
| late | **`FUN_00547000`** | `CSpriteSorter::Render` — depth-sort + blit the world ([`render-trace.md`](render-trace.md)) |
| end | `FUN_004ab750` | debug overlay (FPS, clock, agent counts) |

So one frame is: **process input/commands → advance the clock → update
agents (move/AI/fight) and projectiles → depth-sort and render → draw
overlay.** Simulation and render are not separated — the world updates
and the frame is drawn in the same pass, 40× a second.

## Citations

```text
div.exe:0x0040d850   FUN_0040d850   main — the frame loop (back-edge 0x40e3a6 → 0x40df30).
div.exe:0x004ab4a0   FUN_004ab4a0   per-frame update — ~40 subsystem calls.
div.exe:0x00505bc0   FUN_00505bc0   → CClock::Advance.
div.exe:0x00547000   FUN_00547000   CSpriteSorter::Render (world depth-sort + blit).
div.exe:0x00561b30   FUN_00561b30   projectile update.
div.exe:0x004ab750   FUN_004ab750   debug overlay.
```

## Status

- Main loop ✅ — `main` (`FUN_0040d850`) frame loop, capped by `game
  rate 40 fps`; back-edge confirmed.
- Per-frame entry ✅ — `FUN_004ab4a0`, ~40 ordered subsystem calls.
- Phase anchors ✅ — clock advance, projectile update, sprite-sort
  render, debug overlay confirmed; command processing located.
- Full call labelling 🟡 — most of the ~40 calls (the `.\AGENTS\`
  update cluster especially) are silent dispatchers, identified by
  range/order but not each named.
- Window pump ✅ — `main`'s loop also calls `FUN_0040d1d0` (alongside
  the update `FUN_004ab4a0`): it runs the **Windows message pump**
  (`TranslateMessage` / `DispatchMessageA`) plus shroud/resolution
  handling and the day/night hour check. (Its two `Sleep` calls are
  *fixed* `Sleep(1000)` / `Sleep(2000)` error/retry waits — e.g. a
  failed resolution change — **not** the frame throttle.)
- Timestep ✅ (resolved, approximable) — real time is measured via
  `timeGetTime` and the engine exposes `game rate 40 fps`, so the loop
  is rate-aware. The cap is **not** a `Sleep` in div.exe's per-frame
  routines (`FUN_0040d1d0`'s sleeps are fixed error waits;
  `FUN_00505bc0` has no timer gate) **and not in the renderer present
  either**: `slash4.dll` (Software/DirectDraw)'s `DllSlashedEndFrame`
  (`0x10023f10`) just does `GetClientRect` + DirectDraw surface **Blt/
  Flip via COM vtable** (`call edx`) — no `Sleep`. The slash DLL's only
  `QueryPerformanceCounter` use (`fcn.10037c36`) is the MSVC
  `__security_init_cookie` (GetSystemTimeAsFileTime + ProcessId +
  ThreadId + TickCount + QPC, called from `entry0`), **not** a frame
  gate. So the 40 fps cap is the **DirectDraw primary-surface flip
  waiting for vblank** in the present path, with `game rate 40 fps` as
  the software ceiling. A reimplementation can reproduce it as a **fixed
  25 ms (40 Hz) frame step** — behaviourally faithful without emulating
  vblank. ([`renderer-plugins.md`](renderer-plugins.md).)
