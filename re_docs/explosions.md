# Explosions (area effects)

The area-of-effect system (`.\WORLD\explosion.cpp`): timed blasts that
damage everything within a **radius** — fireball detonations, shockwave
arrows, the cart bomb. Structured like the other managed-instance
subsystems ([`traps.md`](traps.md), [`projectiles.md`](projectiles.md)):
a versioned list of live instances ticked each frame.

## Model

- **Explosion manager** — `"Loading explosion manager"` at load; the
  serialized list is versioned `ExplosionsV0.935`. It owns the live
  explosions and updates/expires them per frame.
- **Explosion instance (`CExplosion`)** — a **168-byte (`0xa8`)** object
  (ctor `FUN_00573ff0`, vtable `0x61e204`), fed by the silent trigger
  subtypes `FUN_00575460 / 575540 / 575810 / 575a10 / 575c90`. The fields
  the AoE update reads:

  ```text
  +0x18 / +0x1c   i32   center cell X / Y   (used for the distance test)
  +0x28           i32   radius              (in cells)
  +0x3c           i32   damage              (passed to each victim)
  +0x50           i32   lifetime/timer      (init 100)
  +0x54           i32   owner agent id      (-1 = none; resolved via CAgentManager)
  +0x58 / +0x5c   i32   world X / Y position
  ```

  Its 5-slot vtable: slot 0 dtor, slot 1 `FUN_00574140` detonate/spawn
  (logs `"Explosion"`, plays the animation), **slot 2 `FUN_00574440` the
  per-frame update / AoE apply**, slot 3 `FUN_00574810` serialize, slot 4
  `FUN_00574090`.

  **AoE damage (`FUN_00574440`).** Each active frame it walks the live
  agent list (`CAgentManager [0x658d50]`, the `[+0x0c]` pointer array,
  count `[+0x04]`, [`agent.md`](agent.md)) and for every agent computes
  the **octagonal distance** (`FUN_0040ecb0`, the same metric the movement
  stepper uses, [`pathfinding.md`](pathfinding.md)) from the explosion
  centre (`+0x18/+0x1c`) to the agent (`+0x1c/+0x20`):

  ```text
  for each agent A in CAgentManager:
      if octdist(center, A.pos) <= radius (+0x28):
          A.vtable[+0x28](owner, damage (+0x3c), elem=2, …)   // the combat resolver
  ```

  So the blast is a **flat-damage disc**: every agent whose octagonal
  distance is `≤ radius` takes the *same* `damage` (`+0x3c`) — **there is
  no distance falloff**. Damage is delivered by calling the victim's
  `CAgent vtable[+0x28]` — the same melee/combat resolver
  ([`combat.md`](combat.md), `FUN_00417b40`) — with the explosion's owner
  and damage and element code `2`, so explosions reuse the melee damage
  path rather than a bespoke one.
- **Triggers** (from the RTTI class names):
  - `CFireBallSpell` / `CFireBallExplosionSpell` — the fireball lands and
    detonates ([`skills-magic.md`](skills-magic.md): the Elemental
    `MeteorStrike` / fire effects).
  - `Shockwave arrow` / `Golden shockwave arrow` — Ranger arrow effects
    that explode on impact ([`projectiles.md`](projectiles.md)).
  - `CartExplosion` — the scripted cart bomb.

So a projectile or spell reaches its target, then spawns an explosion
instance; the explosion (not the projectile) does the AoE damage —
routed through the same combat event path as melee
([`combat.md`](combat.md)).

## Explosion subtypes & save-restore

`CExplosion` is the base; the recovered concrete subtypes (RTTI) are
**`CCartExplosion`**, **`CProjectileCartExplosion`** (the cart bomb +
its thrown variant), **`CThelyronExplosion`** (a scripted story blast),
**`CExplosion_TrailBomb`** (vtable `0x61e298`) and
**`CExplosion_DamageCloud`** (vtable `0x61e2c8`) — the lingering-cloud
forms — plus the visual twin `CAniEffect_CartExplosion`.

These are reconstructed on **save load** by the *Explosions* block reader
**`FUN_00575f30`** (`ExplosionsV0.935`, reached from the saveman loader
`fcn.00502bf0`, [`formats/savegame.md`](formats/savegame.md)): a **5-case
switch on the stored subtype** allocates the matching class (e.g.
`new(0x68)` + `CExplosion_TrailBomb`/`CExplosion_DamageCloud` vtable) and
restores its fields from the stream — centre `+0x18`/`+0x1c`, radius
`+0x28`, damage `+0x3c`. So an in-flight blast survives a save/reload.
(This is the **save-restore** path; the **live trigger** field-setup —
which spell/prop value seeds `+0x28`/`+0x3c` at cast time — remains the
open piece, distinct from this reader.)

## Citations

```text
div.exe:0x00573ff0   FUN_00573ff0   CExplosion ctor (explosion.cpp, 0xa8 bytes, vtable 0x61e204).
div.exe:0x00574140   FUN_00574140   slot 1 — detonate/spawn ("Explosion", animation).
div.exe:0x00574440   FUN_00574440   slot 2 — per-frame update / AoE damage apply.
div.exe:0x00574810   FUN_00574810   slot 3 — serialize.
div.exe:0x0040ecb0   FUN_0040ecb0   octagonal distance (shared with movement; pathfinding.md).
div.exe:0x00575f30   FUN_00575f30   Explosions save-block reader — 5-case subtype switch,
                                    restores centre/radius/damage; from saveman fcn.00502bf0.
RTTI: CExplosion, CCartExplosion, CProjectileCartExplosion, CThelyronExplosion,
      CExplosion_TrailBomb (vt 0x61e298), CExplosion_DamageCloud (vt 0x61e2c8),
      CAniEffect_CartExplosion; triggers CFireBallSpell/CFireBallExplosionSpell/CExplosiveArrow
```

## Status

- Subsystem ✅ — versioned (`ExplosionsV0.935`) managed list of AoE
  explosion instances; manager + ~10 functions located.
- Triggers ✅ — fireball spells, shockwave arrows, cart bomb (by RTTI
  name).
- AoE model ✅ (shape) — position + `radius`, timed, damages agents in
  range via the combat event path.
- Instance struct ✅ — `CExplosion`, 168 bytes; centre `+0x18/+0x1c`,
  radius `+0x28`, damage `+0x3c`, lifetime `+0x50`, owner id `+0x54`,
  world pos `+0x58/+0x5c`; extracted from the ctor + AoE update.
- Damage falloff ✅ (resolved: **none**) — `FUN_00574440` damages every
  agent with `octdist(centre, agent) ≤ radius` for the *flat* `+0x3c`
  amount, with no distance scaling; delivered via the victim's
  `CAgent vtable[+0x28]` (the melee combat resolver, element code 2).
- Subtypes ✅ — `CExplosion` base + `CCartExplosion` /
  `CProjectileCartExplosion` / `CThelyronExplosion` / `CExplosion_TrailBomb`
  / `CExplosion_DamageCloud` (RTTI), reconstructed on save load by the
  block reader `FUN_00575f30` (5-case subtype switch).
- Live-trigger field-setup 🟡 — `FUN_00575f30` is the **save-restore**
  path (not the trigger); which spell/prop value seeds radius (`+0x28`)
  and damage (`+0x3c`) at cast time is still the remaining piece.
