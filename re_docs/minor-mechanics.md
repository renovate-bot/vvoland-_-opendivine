# Minor world mechanics

Small, self-contained gameplay subsystems — each too thin for its own
page, documented here for completeness.

## Throwing (`throwobject.cpp`)

Throwing an item (a spear, a potion) sends it flying and resolves on
landing. The thrown object travels as a **projectile**
([`projectiles.md`](projectiles.md)) using the item's own sprite, then
on impact applies its effect (weapon damage / potion shatter). The
`Throw` command and the `SpearThrowDamageModifier` property
([`formats/props.md`](formats/props.md)) tune thrown-weapon damage — a
thrown spear does modified weapon damage at the landing cell. Likely
one of the unnamed agent command verbs in
[`messages.md`](messages.md). (4 functions in `.\WORLD\throwobject.cpp`.)

**Thrown-damage scaling (`@0x004ddca0`).** The landing-damage routine
follows the **same props-driven `/20` shape as spell/trap elemental
damage** ([`skills-magic.md`](skills-magic.md)): it resolves the
`SpearThrowDamageModifier` curve via the props lookup `FUN_00500f10`
(lazily cached into the global array `[0x6dff7c]`), indexes it by a level
tier `idx = power / 20` (the `imul 0x66666667; sar edx,3` divide-by-20
idiom), adjusts a base value by the **target's `CStats`** (`FUN_0055c960`,
gated on the victim flag `[+0x220] & 0x40`), and multiplies:

```text
throwDamage ≈ adjust(base, target.CStats) × SpearThrowDamageModifier[power/20]
```

so a thrown weapon's damage is its base value scaled by the per-level
modifier curve. (The exact `base`/`power` source within the throw object
and the flight→impact hand-off are not fully pinned — the same open seam
as the projectile impact step, [`projectiles.md`](projectiles.md).)

## Blood & gore (`bloodspot.cpp`)

Combat leaves **blood decals** on the floor: `CGoreAttachedEffect`
spatters and persistent blood spots. These are flat ground sprites laid
after a hit/death animation. The render pipeline has a dedicated
**back-emit pass at z=30000** specifically for blood decals lying flat
after their animation ends (see [`render-trace.md`](render-trace.md),
`FUN_00547000`) — so gore is depth-sorted to the floor, under
everything. (3 functions in `.\WORLD\bloodspot.cpp`.)

**Object model.** `CGoreAttachedEffect` (vtable `0x608fd4`) and the
sibling `CDeathBlood` (vtable `0x608fc4`) are **`.\AGENTS\agenteffect.cpp`
effect objects, 168 bytes (`0xa8`)** — the *same size* as `CExplosion`
([`explosions.md`](explosions.md)) and built with the *same* sprite-handle
acquire helper `FUN_00546ff0` used by explosions and projectiles, so the
floor-decals are members of the shared agent-effect family (one effect
base, multiple subclasses). The common ctor `FUN_00413EC0` sets the
position/state words (`+0x14`/`+0x18`/`+0x1c`), the sprite handle
(`+0x24`), and a sub-object (`+0x28`); small per-type **factory** thunks
wrap it and stamp a **gore-subtype id at `+0x20`** (the observed values
`3`, `4`, `7` distinguish death-blood vs hit-spatter vs the variant). So
a hit/death spawns a typed 168-byte effect that plays its spatter
animation, then the renderer's z=30000 pass keeps the final frame as the
flat floor decal.

## Quick objects (`quickobject.cpp`)

A lightweight object spawn path (`.\WORLD\quickobject.cpp`, 2 functions)
for transient/effect objects that don't need the full catalogue-backed
`CObject` lifecycle ([`object-interaction.md`](object-interaction.md)).
Used for short-lived visual props.

## Camera definitions (`.\MISC\camera.cpp`)

Camera behaviours are defined in the text file **`dat\camera.dat`** (a
dev-source text file — *not shipped* in this install, though the exe
references it and can rewrite it; so the record *tail* beyond the header
fields below — runtime camera/transform state — can't be sampled), loaded
once at boot as the *"Loading camera"* step of `init.cpp`
([`architecture.md`](architecture.md)): driver `fcn.004f27b0` →
`fcn.004f26e0` opens it `"rt"` and `fgets`-loops each line into the
per-line parser **`fcn.004f23c0`** (`.\MISC\camera.cpp`).

Each line is whitespace-tokenised (`fcn.004fdc40`/`fcn.004fdf40` split on
`"\n"`); a line beginning with `/` (`0x2f`) is a comment, and a line must
have **8 or 9 columns** or it is rejected with `Line %d - Syntax error in
cameras: %s`. The two column counts allocate two record sizes — **8 cols →
`0x1b1` (433 B)**, **9 cols → `0x1bc` (444 B)** (`fcn.004fa4f0`) — so a
camera entry is a large record (it holds runtime camera/transform state;
the line only supplies the header fields). Verified header fields:

```text
col 0   name        interned through the string registry [0x750d2c]
                    (fcn.0057bf90); rejected if unknown
int     -> +0x10    atoi column
int     -> +0x14    atoi column
str     -> (strdup) a string column kept verbatim
int     -> +0x0c    atoi column
"CameraEvents"      a column resolved through the config/props lookup
                    fcn.00505650 — binds the entry to a named CameraEvents
                    section (the event → camera-move trigger table)
int, int            two trailing atoi columns; the 9th exists only for the
                    9-column form
```

So `camera.dat` is a table of **named camera presets**, each carrying a few
integer parameters, a string, and a binding to a **`CameraEvents`** section
that drives scripted/cutscene camera moves on game events. The exact
column→field order beyond the offsets above, and the 433/444-byte records'
remaining layout, are not fully pinned 🟡.

## Ambient birds (`CBirdManager`)

The flying-bird ambient decoration. Loaded at boot as the *"Loading birds"*
step of `init.cpp` ([`architecture.md`](architecture.md)). There are **two
distinct files** — the earlier doc conflated them:

- **`dat\birds.cfg`** — the *manager configuration*, loaded by the ctor.
  `fcn.004f0800` allocates a 36-byte (`0x24`) heap `CBirdManager` (stored to
  the global `0x6e00e8`), stores the cfg name, and calls the cfg loader
  `fcn.004efde0`. That loader `fopen`s `birds.cfg`, and if it can't load one
  logs `"Can't create birdmanager configuration"` / `"Can't get %s"`. Its
  real job is to resolve the **4 bird animation-index slots** (`0x650ad4..
  0x650ae4`) by looking the `"walk"` animation name up in the animation-index
  map (`fcn.00439300` / `fcn.00439380` / `fcn.00439470`,
  [`animation.md`](animation.md)). So `birds.cfg` only binds the shared
  animation set; it is not the bird *placement* data.

- **`dat\birds.000`** — the actual **bird-placement data**, loaded by
  `fcn.004f0fe0` (ecx = the manager from `0x6e00e8`). It is **binary** with a
  4-byte magic `'brd0'` (`0x30647262`), read either from the packed VFS
  (`[0x6ddd24]` open → `fcn.004f5e30`/`fcn.004f61c0`) or via raw `fopen "rb"`
  + `fread`; both validate the magic then call the same record reader
  (`fcn.004f0c90`). After loading, the manager is published into the
  world-data registry (`0x750d38`, [`osi-static.md`](osi-static.md)).

### `birds.000` record format (`fcn.004f0c90`, `.\MISC\birds.cpp`)

A two-level table — **groups** of **entries**:

```text
i32   magic = 'brd0'                       (consumed by fcn.004f0fe0)
i32   groupCount                           → [mgr+0x1c]; group array = groupCount × 8 bytes
      Group[groupCount]:
          i32 entryCount                   → group[g].count
          (then entryCount × Entry, each instantiating a bird object)
      Entry:
          i32 param0                        spawn parameter (x / area)
          i32 param1                        spawn parameter (y / density)
          i32 type                          0..3 — which ambient class (below)
```

Each `Entry`'s `type` selects one of **four ambient-creature classes**
(RTTI-named, constructed then stored in the group's pointer array), with a
distinct object size and a `+0x3c` type tag matching the file `type`:

| `type` | class | object size | ctor | `+0x10` range field |
|---:|---|---:|---|---:|
| 0 | **`CEagle`** | 156 (`0x9c`) | `fcn.004f0360` | `0x400` (1024 — flies high/wide) |
| 1 | **`CButterfly`** | 160 (`0xa0`) | `fcn.004f0480` | `0x80` (128) |
| 2 | **`CButterfly2`** | 160 (`0xa0`) | `fcn.004f05b0` | `0x80` |
| 3 | **`CBirdRegular`** | 168 (`0xa8`) | `fcn.004f06e0` | `0x80` |

The two `param` ints are normalized against `0x10000` (65536 fixed-point)
by `fcn.004f7b40` and stored as floats on the object (`+0x14`/`+0x18`); the
entry's index within its group is kept at `+0x44`. `CEagle`'s larger range
field (`0x400` vs the others' `0x80`) is consistent with eagles ranging
higher/farther than the butterflies and regular birds. The `"regular bird
selected"` / `"unknown bird - please add in %s:%d"` strings and the
`editor\birds.txt` name-list (`fcn.005d5aaa`) belong to the **map editor's**
bird-placement UI, not the runtime loader. Birds remain pure ambience —
sprites cycling the `"walk"` animation set, no gameplay interaction.

## Status

- Ambient birds ✅ (load path + data format) — boot step `fcn.004f0800`
  (36-byte `CBirdManager` → `0x6e00e8`). **Two files, now separated:**
  `dat\birds.cfg` (`fcn.004efde0`) only binds the 4 `"walk"`
  animation-index slots (`0x650ad4`); `dat\birds.000` (`fcn.004f0fe0`,
  magic `'brd0'`, record reader `fcn.004f0c90`) is the **placement table** —
  `groupCount` groups of `{i32 param0, i32 param1, i32 type}` entries, each
  `type` (0..3) instantiating one of four RTTI classes **`CEagle` /
  `CButterfly` / `CButterfly2` / `CBirdRegular`**. **Per-frame flight tick ✅**
  — each bird's `virtual_4` (`CBirdRegular` = `fcn.004efa60`, `CEagle` =
  `fcn.004ef440`) is a **straight-line drift**: position `+0x14`/`+0x1c` += velocity
  `+0x2c`/`+0x34` each frame (FP, via the round helper `fcn.005e5d40` and the
  fixed-point normalizer `fcn.004f7b70`), with the **altitude `+0x18` clamped**
  against the constant `[0x616538]` = **float `384.0`** (`00 00 C0 43`, the
  altitude ceiling). So birds fly in a constant
  direction at constant speed, bounded in altitude — pure ambience, no steering.
  The two `param` ints (read as `0x10000` fixed-point) seed the bird's
  start position/velocity at these float fields. **Both butterflies share a
  distinct tick** (`CButterfly`/`CButterfly2` `virtual_4` = `fcn.004ef640`):
  the same `pos += velocity` drift, but it `fcom`-tests the *three* velocity
  components (`+0x2c`/`+0x30`/`+0x34`) against 0 — a 3-axis **bounded
  bounce/flutter** (velocity flips at bounds) rather than the bird's straight
  altitude-clamped drift. So all four ambient classes are characterised:
  Eagle/BirdRegular = straight drift + altitude clamp, Butterfly×2 = 3-axis
  flutter.
- Camera definitions ✅ (format) — `dat\camera.dat`, text, `.\MISC\
  camera.cpp` (`fcn.004f23c0`): comment `/`, 8/9 columns → 433/444-byte
  records, interned name + int fields (`+0xc`/`+0x10`/`+0x14`) + a string +
  a `CameraEvents` config binding. Full column order / record tail 🟡.
- Throwing ✅ (model) — thrown item → projectile → impact damage via
  `SpearThrowDamageModifier`. Damage scaling ✅ — `base ×
  SpearThrowDamageModifier[power/20]` (props array `[0x6dff7c]`, the
  shared `/20` props-damage pattern), target-CStats adjusted; `base`/
  `power` source + flight→impact hand-off 🟡.
- Blood/gore ✅ — `CGoreAttachedEffect`/`CDeathBlood` are 168-byte
  `agenteffect.cpp` effect objects (the `CExplosion` family; shared
  sprite helper `FUN_00546ff0`), ctor `FUN_00413EC0`, gore-subtype id at
  `+0x20` (`3`/`4`/`7`) set by per-type factory thunks; depth-sorted to
  the floor via the z=30000 pass. Exact per-subtype meaning + decal
  lifetime/fade 🟡.
- Quick objects 🟡 — a transient-object spawn path; exact use not fully
  traced.
