# Projectile / arrow types & trajectories

The taxonomy of **flying things** — arrows, bolts, magic missiles, the
special Ranger ammo, and the curved/homing paths spell projectiles follow.
[`projectiles.md`](projectiles.md) covers the basic arrow *flight object*
and impact; this enumerates the full RTTI roster and the **three
independent layers** a shot composes from:

1. **`CProjectile*`** — the **flight behaviour** (what the projectile *does*
   in the air and on impact: split, explode, poison, home). Base `CProjectile`
   vtable `0x61d6a8`.
2. **`CArrow*`** — the **visual** (which sprite/trail is drawn). Base
   `CArrow` vtable `0x61d3a8`.
3. **`CPath*`** — the **trajectory** (the curve the projectile travels).
   Base `CPath` vtable `0x612574`.

A fired projectile is a `CProjectile*` that owns a visual and a path, so
e.g. a homing fireball = a damage projectile + a flame `CArrow` + a
`CHomingNPCPath`, while a plain shot = `CProjectileBasicArrow` +
`CNormalArrow` + `CLinearPath`.

## `CProjectile*` — flight behaviour

| Class | Behaviour |
|---|---|
| `CProjectileBasicArrow` / `…AutoalignBasicArrow` / `…SlightlyDifferentBasicArrow` | ordinary shot (the autoalign one tracks slightly) |
| `CProjectileExplosiveArrow` | detonates on impact → [explosion](explosions.md) |
| `CProjectileFrostArrow` | frost/slow on hit |
| `CProjectilePoisonArrow` | leaves poison → [pain point](painpoints.md) |
| `CProjectileSplittingArrow` | forks into several arrows |
| `CProjectileFlareArrow` / `…LaserTrailArrow` / `…SplineLaserArrow` | trailing-FX shots |
| `CProjectileHelixArrow` | corkscrew flight |
| `CProjectileLizardArrow` / `CProjectileSmokeArrow` | creature / smoke ammo |

## `CArrow*` — visual sprite

`CNormalArrow` · `CDiscArrow` · `CExplosiveArrow` · `CFrostArrow` ·
`CPoisonArrow` · `CLineArrow` · `CHelixArrow` · `CShockWaveArrow` ·
`CGoldenShockWaveArrow` · `CImpShotArrow` · `CLizardArrow` · `CMummyArrow` ·
`CSpiderArrow` · `CSmokeArrow` · `CSplineArrow` · `CSplitArrow` ·
`CTornadoArrow` — the drawn form, chosen independently of the behaviour
(so a boss can fire a visually-distinct arrow with stock behaviour, e.g.
`CMummyArrow` / `CSpiderArrow`).

## `CPath*` — trajectory

| Path | Curve |
|---|---|
| `CLinearPath` | straight line (default) |
| `CHelixPath` / `CSineHelixPath` | corkscrew / sine-wobble |
| `CSpiralPath` | spiral |
| `CSplinePath` | spline through control points |
| `COrbitPath` | orbit around a point |
| `CAccelerateToTargetPath` | accelerating seek |
| `CHomingNPCPath` / `CAttachToNPCPath` | track / stick to an NPC |
| `CAttachToMousePath` | follow the cursor (targeting) |

So the path is a strategy object — magic missiles, the `MagicAttractor`,
and tracking spells reuse the same `CProjectile` flight with a homing or
spline `CPath` instead of the linear one.

## Mapping to the Ranger skills

The Warrior **Ranger** discipline ([skill-tree.md](skill-tree.md)) selects
the projectile behaviour:

| Ranger skill | Projectile |
|---|---|
| ExplosiveArrows | `CProjectileExplosiveArrow` |
| FrostArrows | `CProjectileFrostArrow` |
| PoisonedArrows | `CProjectilePoisonArrow` |
| SplittingArrows | `CProjectileSplittingArrow` |
| PiercingArrows | a basic arrow that doesn't stop on the first target |
| BlockArrows | defensive (intercept incoming arrows), not a fired type |

The on-hit consequences route to the systems already documented:
explosive → [explosions](explosions.md), poison → [pain points](painpoints.md),
the impact damage → [combat](combat.md).

## Special / non-arrow projectiles

Beside the arrow taxonomy, the same `CProjectile` family (vtables in the
`0x61d4xx` block, next to `CProjectile`/`CArrow`) holds the bespoke
thrown/cast attacks:

| Class | Behaviour |
|---|---|
| `CBoomerangAttack` → `CBoomerangComeback` | the **returning** thrown weapon — a two-phase flight: phase 1 (`Attack`) flies out to the target, phase 2 (`Comeback`) curves back to the thrower |
| `CKnife` | a thrown knife (the projectile form of a [thrown weapon](minor-mechanics.md), `.\WORLD\throwobject.cpp`) |
| `CDragonFire` / `CDragonFireTrackingTarget` | the Survivor **DragonBreath** skill ([skill-tree.md](skill-tree.md), `.\magic\dragonfire.cpp`) — a fire breath that tracks its target |
| `CHomingMissile` / `CHomingFlareTarget` | a tracking missile (rides a `CHomingNPCPath`) |
| `CShieldImpact` / `CShieldImpactMagic` | the **block** visual — a hit landing on a (magic) shield (`fcn.004cd410`) |

The two-phase boomerang is the clearest example of the `CPath`/behaviour
split: the `Comeback` phase swaps in a return trajectory while keeping the
projectile alive, so the weapon flies out and back without re-spawning.

## Status

- Three-layer model ✅ — `CProjectile*` (behaviour, base `0x61d6a8`) +
  `CArrow*` (visual, base `0x61d3a8`) + `CPath*` (trajectory, base
  `0x612574`), composed per shot.
- Full rosters ✅ — 13 `CProjectile*`, 17 `CArrow*`, 10 `CPath*` classes
  enumerated from RTTI.
- Ranger-skill mapping ✅ — Explosive/Frost/Poisoned/Splitting →
  the matching `CProjectile*Arrow`; Piercing = non-stopping basic; Block =
  defensive.
- Special projectiles ✅ — the two-phase returning `CBoomerangAttack`/
  `CBoomerangComeback`, `CKnife`, the DragonBreath `CDragonFire`,
  `CHomingMissile`, and the `CShieldImpact` block visual (same
  `0x61d4xx` family).
- Per-class behaviour internals / path math 🟡 — the exact split count,
  frost-slow amount, spline control points, and homing turn-rate are
  per-class fields not split here (the basic flight integration is in
  [projectiles.md](projectiles.md)).

## Citations

```text
vtables: CProjectile 0x61d6a8 · CArrow 0x61d3a8 · CPath 0x612574
         CProjectileBasicArrow 0x61d6c4 · …SplittingArrow 0x61d750 · …ExplosiveArrow 0x61d7c0
RTTI families: CProjectile*Arrow (13) · C*Arrow visuals (17) · C*Path trajectories (10)
skills: CWarriorRangerSkill_{Explosive,Frost,Poisoned,Splitting,Piercing,Block}Arrow
```
