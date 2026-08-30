What does the main game loop look like? Does it use a fixed timestep? How does it handle performance drops?

---

The game has a **nominal fixed simulation period**, but it is not a
conventional fixed-timestep accumulator.

The simulation is driven from the rendered-frame loop:

- The shipped rate is **40 Hz**, with a 25 ms simulation period.
- `smooth 0` changes it to **20 Hz**, with a 50 ms period.
- `game rate N fps` is parsed but ignored.
- The world-tick body runs **at most once per rendered game frame**.
- If a frame is late, the engine does **not** run multiple catch-up ticks.

Therefore a performance drop makes per-tick simulation run less often in real
time, while the scheduler's target and counter can jump forward.

## Overall loop

The top-level loop is in `main` at `0x0040d850`. Each iteration is roughly:

```text
while not quit:
    latch previous input state

    if PeekMessage reports a message:       // PM_NOREMOVE probe
        drain the entire Windows message queue
            GetMessage
            TranslateMessage
            DispatchMessage
        until PeekMessage reports no message

    if quit:
        break

    if application is inactive:
        continue

    play a queued Bink movie, if any
    choose one screen handler
    if quit:
        break

    gameplay frame:
        process deferred file and camera work

        DllSlashedStartFrame(renderClock)
        advance the render clock

        process input playback, sound, and other per-frame work

        pace the simulation
        run at most one world tick

        update per-frame gameplay and render state
        draw the world, effects, UI, text, and cursor

        flush the renderer
        DllSlashedEndFrame()       // presents the frame

        update mouse-repeat timers
        clear per-frame input events
        process save/load/screenshot requests
```

The in-game handler is `0x004ab4a0`. Other screen handlers follow the same
renderer `StartFrame`/`EndFrame` bracket.

Presentation is implemented by the renderer plug-in. In fullscreen, the
inspected DirectDraw backends use `Flip(..., DDFLIP_WAIT)`, so they can wait for
vertical blank. The inspected windowed paths use `Blt`/`BltFast` without a
guarantee of vsync pacing.

## Simulation timing

The timing code has a scheduler target and current counter. Its relevant shape
is:

```text
elapsed = now - lastTimestamp
if elapsed < tickPeriod:
    if the observed no-wait camera/selected-agent guards match:
        return                         // target does not advance
    busy-wait until tickPeriod elapses
    now = timeGetTime()
    elapsed = now - lastTimestamp

lastTimestamp = now
target += elapsed / tickPeriod

if current == target:
    return

current = target - 1

do:
    run one world tick
    current++
while current < target
```

The no-wait guard details are intentionally abbreviated here; the precise
observed conditions are described below rather than inferred from this rough
pseudocode. The scheduler's `current` counter and `target` are not the
world-clock `absTicks`. When a world tick does execute, that tick increments
`absTicks` once; a jump in the scheduler target does not execute the discarded
ticks.

The assignment `current = target - 1` means that the loop body executes once,
even if `target` advanced by several periods. This is different from a normal
accumulator such as:

```text
while accumulator >= tickPeriod:
    simulate_one_tick()
    accumulator -= tickPeriod
```

The original also resets its timestamp to `now`, rather than preserving the
fractional remainder, so elapsed sub-period time is discarded.

The nominal rate comes from `smooth`, not from `game rate N fps`:

```text
rate = 40 if smooth != 0 else 20
period = 1000 / rate
```

The shipped configuration has `smooth 1`, giving 40 Hz. The period is latched
before the later `config.div` parse, so a `smooth` value in `config.div` cannot
change it.

## Performance drops

Suppose the normal period is 25 ms and one frame takes 200 ms:

```text
elapsed / period = 200 / 25 = 8
```

The scheduler advances its target by 8, but the world-tick body runs only once:

```text
target += 8
current = target - 1
run one world tick
current becomes target
world-clock absTicks increments once
```

The other seven simulation steps are discarded.

Consequences include:

- Monster AI and other per-tick logic execute only once.
- Projectile and movement updates lose steps.
- Tick-based animations and effects advance fewer times than wall-clock time
  suggests.
- Systems reading the scheduler's current counter or target can observe a large
  jump and skip intermediate values.
- The world-clock `absTicks` is separate and increments only for the one world
  tick that actually executes.
- There is no catch-up burst and therefore no traditional accumulator
  "spiral of death", but behavior diverges from a real-time simulation.

With sustained low performance, active/paced gameplay executes per-tick logic
at approximately the rendered-frame rate. These examples are approximate:

```text
40 FPS rendering  -> up to 40 world ticks/sec
30 FPS rendering  -> about 30 world ticks/sec
10 FPS rendering  -> about 10 world ticks/sec
```

The scheduler's target may still advance as if 40 ticks per second had elapsed,
but tick-dependent code is called only once per frame.

## Frames with no world tick

The precise behavior is **zero or one tick per frame**, not exactly one.

If the pacer does not advance the target, the world-tick function sees:

```text
current == target
```

and returns without running a tick. One identified no-wait case is the
standing-still/camera-follow path, and it applies only while `elapsed < period`.
It also has guards for the camera object's raw field at `+8 == 0`, the camera
being bound to the local/selected agent, and the selected agent's two movement
floats being exactly `0.0f`. The meaning of the camera field at `+8` is
unproven; these are observed guards, not semantic labels. The busy-wait is
skipped in that case, allowing extra render frames with no simulation tick.

## Pacing and the two clocks

The gameplay loop has no ordinary `Sleep`-based frame limiter. When pacing is
needed, it busy-waits on `timeGetTime`. The wait is conditional:

- Normally it waits until the tick period has elapsed.
- While `elapsed < period`, it can skip the wait when the observed
  standing-still/camera-follow guards hold.
- Fullscreen presentation can independently block in
  `Flip(DDFLIP_WAIT)`.

The render clock passed to `DllSlashedStartFrame` is separate from the
scheduler's simulation counter. It advances at most once per rendered frame and
does not catch up after a hitch. It must not be replaced with the simulation
counter when calling `DllSlashedStartFrame`; the evidence establishes the
argument and the separation, not that every renderer plug-in interpolates with
it.

## Sources

- `re_docs/frame-loop.md` — reconstructed frame order and timing behavior.
- `re_docs/world-clock.md` — simulation tick and clock semantics.
- `re/findings/w22-frame-loop.md` — instruction-level evidence, including
  `0x00505a20` (pacing) and `0x00505bc0` (world-tick dispatch).

`re/findings/w22-frame-loop.md` contains an older "exactly once" statement in
its fact section. Its later analysis, together with `re_docs/frame-loop.md` and
`re_docs/world-clock.md`, corrects this to **at most once**, with zero ticks when
the pacer does not advance the target.
