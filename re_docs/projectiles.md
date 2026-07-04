# Projectiles

Arrows, bolts, thrown objects and bolt-like spell effects fly as
**projectiles** (`.\WEAPON\projectile.cpp`, `.\WEAPON\projectilelist.cpp`,
`.\WEAPON\arrows.cpp`). They are tracked in a versioned list
(`ProjectilesV0.935`, debug `"Amount of projectiles %d"`).

## Projectile object

A projectile is a ~`0x60`-byte object built by `FUN_00562e90` (and an
arc variant `FUN_00563560`). It stores a **precomputed trajectory** as
source/destination points, each kept in **two coordinate spaces** —
world and iso-projected screen — so the flight path can be drawn
without re-projecting every frame:

```text
Projectile (partial, from the constructor):
    +0x0c / +0x14 / +0x18   owner / type / source ids
    +0x20,+0x24             point A   (computed +0x20 ← transform(+0x24))
    +0x28,+0x2c             point B
    +0x30,+0x34             point C
    +0x38,+0x3c             point D
    +0x40,+0x44             point E
    +0x40,+0x44             point E / destination end point
    +0x48 (byte)            state flag
    +0x50                   sprite / image handle
    +0x54                   flight progress / timing counter
    +0x5c, +0x60            per-frame velocity (X / Y step), added each tick
```

The class is **`CPVAppAnimation`** (vtable `0x61d690`), with a 4-slot
vtable: slot 0 dtor, **slot 1 `FUN_005630F0` the per-frame advance**,
slot 2 `FUN_005634A0` **save/serialize** (it `fwrite`s `+0x14`,`+0x18`,
`+0x1c`,`+0x24`,`+0x2c`,`+0x34`,`+0x44`,`+0x48`,`+0x4c` — the live-state
field set), and slot 3 `FUN_00563560` its **load/deserialize** twin (reads
the same fields back via `FUN_005e5cc4` and re-projects the world coords
through `FUN_004e84e0`/`FUN_004e78d0` into `+0x24`/`+0x28`). So the four
virtuals are **dtor / per-frame flight / Save / Load** — there is **no
impact or damage virtual**.

**This class is a visual-only object** (`CPVAppAnimation` = the
projectile *appearance*-animation): it animates a sprite along a
trajectory and serializes, but never resolves a hit or applies damage.
**Scope correction:** that conclusion holds for `CPVAppAnimation`
only — the **`CProjectile*` family DOES carry and apply damage at
impact** (the earlier generalization "there is no arrival→damage
hand-off" was wrong). The real chain ([combat.md](combat.md) has the
detail): fire-time roll in `fcn.00418e70` (dice `fcn.0055a530` +
Accuracy) → clock-scheduled shot (`0x415740`, param+0 = dmg) →
arrow-type dispatcher `0x55f580` (table `0x748b00[28]`) → spawner
`0x564d50` with **proj+0x10 = damage**, +0x14 = mode, on-hit callback
proj+0x3c = `fcn.00564220`; flight list `[0x748b94]`, Update
`fcn.00563ec0` (vt+0x10), collision `fcn.00561660` (vt+0x14); on hit
the callback calls the target's **vt+0x28 = `FUN_00417b40`** (mode≠0)
or **vt+0x24 = `fcn.00417550`** (mode 0) with the carried damage
(`0x5642f4–0x564315`). So the damage is still *rolled* at fire time —
the projectile carries it — but it is *applied* on arrival through the
CAgent combat virtuals. Splitting arrows use the arrival callback pair
proj+0x48 (`0x569ac0`); poisoned/elemental procs go via `fcn.004c6500`.

## Per-frame flight (`FUN_005630F0`)

Each tick the projectile **steps its screen position by a constant
velocity** — `x += [+0x5c]`, `y += [+0x60]` — i.e. straight-line motion at
a fixed per-frame delta (no gravity term on the basic path; the arc
variant precomputes its curved point list up front instead). It
interpolates the cached trajectory points to a pixel position (the
`__ftol` helper `FUN_005E5D40`), ticks the sprite-playback object at
`+0x50` (a ~0xa8-byte object constructed by `FUN_005465D0` — the *ctor*,
not the advance), and bumps the `+0x54` progress counter. It then
**bounds-checks against the 640×480 screen** (`[0x64d9b8]` = 640,
`[0x64d9bc]` = 480): a projectile that leaves the screen expires. The
flight ends at the destination `+0x40` and simply returns — **no damage is
applied on arrival** (the projectile carries none; see above).

Each coordinate pair is produced by the engine's world↔iso transform
helpers `FUN_004e84e0` and `FUN_004e78d0` (the same projection used by
sprite placement): the constructor feeds a world value through them and
caches the projected result in the adjacent slot. The sprite is
acquired via `FUN_004fa4b0` / `FUN_00546ff0`.

## Spawners

Projectiles are created by **type-specific spawner** functions in
`projectilelist.cpp`, each of which calls the constructor `FUN_00562e90`:

```text
FUN_005681e0, FUN_00568750, FUN_00568a30, FUN_00568d30  → FUN_00562e90
```

These correspond to the different launch kinds (straight arrow vs arc
vs thrown, etc.). The Warrior Ranger skills
([`skills-magic.md`](skills-magic.md): Explosive/Frost/Piercing/
Poisoned/Splitting arrows) modify which spawner/parameters are used.

## Citations

```text
div.exe:0x00562e90   FUN_00562e90   projectile constructor — builds the dual-coordinate trajectory.
div.exe:0x00563560   FUN_00563560   slot 3 — load/deserialize (twin of slot-2 Save).
div.exe:0x004e84e0   FUN_004e84e0   world↔iso transform (one axis).
div.exe:0x004e78d0   FUN_004e78d0   world↔iso transform (other axis).
div.exe:0x005681e0   FUN_005681e0   a projectile spawner (projectilelist.cpp; + 0x568750/0x568a30/0x568d30).
div.exe:0x0061d690   vtable.CPVAppAnimation   projectile object vtable (4 slots).
div.exe:0x005630f0   FUN_005630f0   slot 1 — per-frame flight advance (x+=[+0x5c], y+=[+0x60]).
div.exe:0x005634a0   FUN_005634a0   slot 2 — save/serialize.
div.exe:0x005465d0   FUN_005465d0   sprite-playback object ctor (~0xa8 B, zero/sentinel init) — NOT a per-frame advance.
```

## Status

- Projectile object ✅ — **`0x184` (388) bytes** (size confirmed from the
  savegame `Projectiles` reader's `new(0x184)`, [`formats/savegame.md`](formats/savegame.md));
  the documented `+0x14..+0x60` fields (trajectory source/dest in world &
  iso space, sprite handle, velocity, progress, state) are the live-flight
  subset of the full object.
- Dual-coordinate trajectory ✅ — constructor caches `transform(world)`
  via `FUN_004e84e0`/`FUN_004e78d0` next to each world point.
- Type-specific spawners ✅ — four+ spawners in `projectilelist.cpp`
  feed the shared constructor.
- Per-frame flight step ✅ — `CPVAppAnimation` slot 1 (`FUN_005630F0`)
  advances the screen position by a constant per-frame velocity
  (`x+=[+0x5c]`, `y+=[+0x60]`), interpolates the cached trajectory
  (`__ftol`), advances the sprite (`+0x50`), bumps the `+0x54` progress
  counter, and expires when it leaves the 640×480 screen. Straight-line
  motion, no gravity (the arc variant precomputes its curve).
- Velocity / progress fields ✅ — `+0x5c`/`+0x60` = per-frame X/Y step,
  `+0x54` = flight progress, `+0x40` = destination.
- Hit detection & damage ✅ (resolved — no projectile-side damage) — the
  `CPVAppAnimation` vtable is `dtor / flight / Save / Load` with **no
  impact or damage virtual**, and the flight's arrival branch is a bare
  epilogue. The projectile is **visual-only**; ranged/thrown damage is
  resolved by the firing combat code at attack time
  ([`combat.md`](combat.md)), decoupled from the projectile's flight.
  (The exact fire-time ranged-damage site is not separately pinned, but it
  is confirmed *not* on the projectile.)
- Save/Load ✅ — slot 2 (`0x5634a0`) `fwrite`s the live fields; slot 3
  (`0x563560`) reads them back and re-projects the world coords (corrects
  the earlier "arc re-init" label — it is the deserialize twin).
