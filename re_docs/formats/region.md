# `global\region.00N` — named region tables (per subsystem)

The world's **named regions**: labelled polygon areas the engine fires
"enter region" triggers against. Region names thread through the whole
scripting layer — e.g. trap binding rejects an unknown region with
`"Unknown region %s in region trigger trap"` ([`../traps.md`](../traps.md)),
and Osiris quest logic keys off the same names.

The eight `region.00N` files are **not** world partitions — each is the
region table for a **different subsystem**, identified by its version
string. Verified across all eight (the `"Divinity Regions"` magic +
version + count + named-record framing holds for every one):

| File | Version | Count | Subsystem |
|---|---|---:|---|
| region.000 | `StoryV1.0` | 435 | Osiris story / quest trigger areas |
| region.001 | `MusicV1.0` | 155 | music zones |
| region.002 | `MonsterV1.0` | 111 | monster-spawn regions ([`../monsters.md`](../monsters.md)) |
| region.003 | `TreasureV1.0` | 1 | treasure region |
| region.004 | `ReverbV1.0` | 89 | audio reverb zones ([`../sound-runtime.md`](../sound-runtime.md)) |
| region.005 | `DungeonV1.0` | 90 | dungeon areas |
| region.006 | `TrapsV1.0` | 81 | trap trigger regions ([`../traps.md`](../traps.md)) |
| region.007 | `ShroudV1.0` | 90 | shroud / fog-of-war regions |

So the named regions a trap or quest references live in the matching
subsystem file (a trap region in `region.006`, a monster region in
`region.002`, …).

The loader `FUN_0058e690` confirms this from the binary side: it
validates the file's version against a requested **region type**,
erroring with `"Regiontype desired does not match <story|music|…>"`.
The full type enum (from that validation chain):

```text
0 story   1 music   2 reverb   3 treasure   4 monster
5 teleport   6 dungeon   7 traps   8 shroud
```

Note: the file *number* does not equal the type enum (e.g. `region.002`
is the **monster** table = type 4), and **teleport** (type 5) has no
shipped `region.00N` file in this install.

All integers little-endian.

## Header

```text
"Divinity RegionsV1.0\0"      magic (raw NUL-terminated string)
u32  len + "<Subsys>V1.0\0"   subsystem/version (length-prefixed:
                              Story / Music / Monster / Treasure / Reverb /
                              Dungeon / Traps / Shroud)
u32  count                    number of regions in this table
u32  ptr[2]                    two baked heap pointers — this is a serialized
                              in-memory structure (the values differ per file,
                              ~0x0146xxxx in Story vs ~0x01edxxxx in Treasure),
                              like treasure.cmp, not file-relative offsets
```

## Geometry encoding

The single-region `TreasureV1.0` file (175 bytes, one unnamed region)
pins the geometry value types: the per-region block after the name mixes

- **IEEE `float` coordinates** — e.g. `0xC1400000 = -12.0` appears as a
  vertex component;
- the **cell constant `64`** (`cellPx`) as a recurring marker, so
  coordinates are cell-relative;
- more **baked pointers** (large `0x01edxxxx`-range values), confirming
  the geometry is part of the serialized heap structure.

The block repeats in vertex-like groups (a flag, a float, `64`, two
more floats, …) — a polygon/cell list — but the exact per-vertex stride
isn't yet locked because the float data is interleaved with the baked
pointers. Names may be empty (the Treasure region is unnamed).

**Read model** (from the loader `FUN_0058e690`): the file is parsed
**sequentially**, top to bottom, via a chain of `fread(buf, size, 1,
f)` calls (observed chunk sizes `4, 8, 12, 16, 24, 28`) over a
count-bounded loop — it does *not* seek within the file using the baked
pointers. The pointers are reconstructed/ignored at load, confirming
they are not file offsets. So the geometry is sequentially walkable
once the per-region field sizes are fully enumerated.

## Region records

`count` variable-length records follow, each:

```text
u32   name_len                bytes of name incl. trailing NUL
char  name[name_len]          e.g. "front_house_Mardaneus", "Aleroth_orb_pentagram_West"
…     geometry block          per-region bounds + polygon/cell vertices
                              (variable size, ~140–240 bytes in the shipped file)
```

The shipped `region.000` declares **435** regions; walking the record
stream recovers 431 names heuristically (the remainder sit behind the
not-yet-pinned geometry stride). Sample names show the role clearly:

```text
front_house_Mardaneus  george_attack  Aleroth_orb_pentagram_{West,north,East}
house_lanilor  herb_garden_Lanilor  pool_aleroth  dh_house4 … dh_house9
```

— building interiors, quest set-pieces (the Aleroth pentagram orbs),
and town-house clusters: exactly the areas story scripts and traps need
to name.

## Runtime containment (point-in-region)

The baked file layout doesn't matter for behaviour because the **runtime
test is a plain point-in-polygon** over each region's vertex loop. The
entry point `FUN_004ff040(x, y)` (two `double`s) wraps the core
**`fcn.004fef20`**, a double-precision **winding-number** test: it walks
the region's edge list and, per edge, computes the edge's x-crossing at the
test point's `y` (`(edi.x-edx.x)/(edi.y-edx.y) * (edx.y - y)` reversed to an
x) and compares it to the point's `x`, accumulating a signed crossing count
in `ebp`; the point is **inside when the winding magnitude reaches `±4`**
(`cmp bp,4` / `cmp bp,0xfffc` — one full turn). The per-edge `cmp eax,6;
ja` selects the crossing case by edge orientation.

The **point→region lookup `FUN_0058d620`** walks the region list (count at
`mgr+0x14`) and returns the first region whose polygon contains the point
(via the per-region tester `fcn.004f5a00` → `fcn.004fef20`).

So a single winding point-in-polygon over the float vertex loops is the
whole runtime: it backs the **shroud region-flood reveal**
(`FUN_0053f780` → this test, [`shroud.md`](shroud.md)), the **Osiris
region enter/leave trigger areas**, the per-region **music/reverb**
(`region.004`, [`../sound-runtime.md`](../sound-runtime.md)), and
**no-magic zones** — all "which named region is this point in?" queries.
This confirms the reimplementation note: the baked pointers can be ignored;
reading the vertex loops + this winding test reproduces region behaviour.

## Status

- Subsystem indexing ✅ — per-subsystem region tables, not world
  partitions; verified across all eight files **and** confirmed by the
  loader `FUN_0058e690`, which gives the 9-value region-type enum
  (0 story … 8 shroud, incl. type-5 teleport with no shipped file).
- Header ✅ — magic + length-prefixed `<Subsys>V1.0` version + `u32
  count` + two file-level coordinate values.
- Record framing ✅ — variable-length records, each a length-prefixed
  name followed by a geometry block; 431/435 names recovered in the
  Story table, 111/111 in Monster, etc.
- Role ✅ — named trigger areas referenced by traps, monsters, audio,
  Osiris and quests, each in its matching subsystem file.
- Serialized heap structure ✅ — the file-level and in-geometry large
  values are baked heap pointers (differ per file), so region.00N is a
  serialized in-memory tree like treasure.cmp, not a flat record array.
- Coordinate type ✅ — geometry uses IEEE `float` coordinates
  (e.g. `-12.0`) cell-relative to `64`-px cells, not fixed-point.
- Per-region geometry stride 🟡 — float vertex groups interleaved with
  baked pointers; the exact stride can't yet be walked deterministically
  without the loader's pointer fixup.
- Loader shape ✅ — `FUN_0058e690` first **constructs several empty list
  members** per region (the `esi+0xc..0x30` slots zeroed via
  `FUN_004fa540`) — so a region owns *multiple* lists (vertex list, and
  likely sub-area/cell lists), not one flat array. It verifies the
  header tag with a byte-by-byte compare after a `fread(…, 0x15)` and
  loops regions against a count (`cmp [esi+0x24], …`). This matches the
  heap-dumped, pointer-baked layout: the lists are rebuilt at load.
- **Reimplementation note** ✅ — region.00N is a heap dump with baked
  pointers, so a port should *not* parse it byte-for-byte; instead
  derive regions from a clean source (or replay the loader's pointer
  reconstruction). The faithful runtime behaviour is simply a
  **point-in-polygon test** of an agent's cell-relative position against
  each region's float vertex loop (the Osiris trigger areas,
  [`../osiris.md`](../osiris.md)) — that is what a reimplementation needs,
  not the exact on-disk pointer soup.
