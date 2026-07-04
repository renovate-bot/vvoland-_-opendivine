# `world.x<n>` — paged world cell-grid

The world map is split into **5 partitions** (`world.x0`..`world.x4`)
and within each partition into **1024 sectors** of 512 cells each —
one sector per Y-row of a 512×1024-cell partition (the working model
validated by the OpenDivine port; the earlier "32 × 32 grid of 96 × 96
fixed-size sectors" intro was a geometric guess, retracted below).
Chunk size on disk is variable.

All integers little-endian.

## Outer layout

```text
struct WorldFile {
    u32 chunk_offsets[1024];   // one per sector (Y-row); offsets are absolute byte offsets in this file
    u8  data[];                // sector chunks concatenated, addressed by chunk_offsets[i]
};
```

`chunk_offsets[0]` always equals `0x1000` (= `4 * 1024`, i.e. the size
of the offset table itself). Sector `i`'s payload occupies the file
range `[chunk_offsets[i], next)` where `next` is the smallest offset
greater than `chunk_offsets[i]` (typically `chunk_offsets[i+1]`, but
sectors are not strictly required to appear in order, and the final
sector ends at EOF).

## Sector chunk

Each sector covers **512 logical cells** (not 96×96 — that geometric
guess was wrong). The sector layout is a u16 pointer table followed
by per-cell records of variable length:

```text
struct Sector {
    u16   cell_offsets[512];   // 1024 bytes; each u16 is the byte offset (relative to the
                               //   start of the records area, i.e. just after this table)
                               //   of that cell's record.
    Record records[];          // variable count and total size
};

struct Record {
    i16   floor_tile_id;       // index into CPacked imagelist 2 (3363 64×64 floor tiles).
                               //   Most-common values across world.x0:
                               //   274 (134k cells, "void"), 0 (63k), 17 (grass), 100 (cobblestone).
                               //   88% of cells carry a non-zero floor id.
    i16   overlay_tile_id;     // secondary tile id; -1 means "no overlay"
                               //   (519k of 524k cells observed in world.x0).
    i16   _pad_off4;           // always 0 in shipped data
    u8    object_count;        // 0 = no per-cell objects (16-byte header still present)
    u8    flag_off7;           // bit 5 (0x20) = "skip this object" in the upgrade path
    i16   _pad_off8;           // always 0
    i16   header_h5;           // small enum, 15 distinct values; pairs with h6
    i16   header_h6;           // mirrors h5's distribution (likely a precomputed
                               //   adjacency / neighbour-mask used by the engine renderer)
    i16   _pad_off14;          // always 0
    Object objects[object_count];
};

struct Object {                // 8 bytes per object instance
    u32   xy_kind;             // bits  0..5 : sub_x  (0..63, added to cell_x)
                               // bits  6..11: sub_y  (0..63, added to cell_y)
                               // bits 12..15: per-object flags index — passed through
                               //              FUN_00581fa0 to derive runtime flags
                               // bits 16..31: unused / reserved
    u32   ord_kind;            // bits  0..9 : a 0..1023 value passed to the placer (role 🟡 — see note)
                               // bits 10..23: object catalogue id (→ FUN_00572100)
};
```

The 9216-byte minimum is `1024 + 512 * 16` — 1024-byte pointer table
plus 512 empty records (16-byte header, zero objects). Sectors with
trailers carry one or more 8-byte object records appended to the
matching cells.

This account matches the parser at `div.exe:0x0059ce90`: the outer
loop walks the 1024 chunk-offsets table (`world.x*`), reads each
sector into a buffer, then iterates 512 cells per sector consuming
2 bytes of pointer-table per step and dispatching object placements
via `FUN_00572100`.

**Bit-field extraction confirmed** (parser at `0x0059d0c0`): `ord_kind &
0x3ff` → bits 0..9 (the `0..1023` value), `(ord_kind >> 0xa) & 0x3fff` →
bits 10..23 catalogue id; `xy_kind & 0x3f` sub_x, the sub_y shift, and
`xy_kind >> 0xc` → the flags index fed to `FUN_00581fa0` (derived runtime
flags). So the *layout* is verified against the binary. The **bits-0..9
role** stays 🟡: the value is pushed as `param_3` of the placer
`FUN_00572100` (a multi-caller function — also used by `CObject::Use`), and
its use is not traceable to a Z/elevation add without deep dataflow, so the
earlier "probably stack height" guess is left as an *unconfirmed inference*,
not asserted.

**Dynamic-state refinement** ([osi-static](osi-static.md)): once a map
is live, the cell object entry's first u32 repurposes the high bits —
bits 0..5 `sub_x`, 6..11 `sub_y`, and **bits 12..31 = the
`objects.x<n>` instance handle** (`shr 0xc` at `0x585e2d`/`0x580600`,
handle = record index into the 28-byte instance heap), not the static
4-bit flags index. The static-file reading above applies to the shipped
`world.x<n>` as such.

### Floor / overlay tile rendering

Each `floor_tile_id` indexes directly into **CPacked imagelist 2**
(`static\imagelists\CPackedb.2c` + `CPackedi.2c`, 3363 entries —
mostly 64×64 RGB565 raw tiles with `flags=0`, but **202 of the 3363
carry `flags=1`** and are span-table sprites with transparent edges,
handled like any standard CPacked cell). Tile id `274` is the single
"void" sentinel (pure black, not drawn); **tile `0` is a real floor
texture (dirt) and is drawn** — an earlier note calling both `0` and
`274` sentinels was wrong (both corrections validated by the
OpenDivine port against the shipped imagelist and map).

`overlay_tile_id`, when not `-1`, is a second tile drawn on top of
the floor (decals, road segments, stains). In shipped `world.x0`
about 4.8k of the 524k cells carry an overlay (most use values in
the 1978..1996 range — a small set of overlay-tile ids).

Together with the per-cell objects, these three layers
(floor → overlay → objects sorted by Layer/Y) give a complete
native-resolution render. The **render** path has no dependency on
`dat\tiles.dat`.

`dat\tiles.dat` itself is **not** unshipped or editor-only (correcting an
earlier note): it ships as plaintext in [`global.cmp`](cmp.md) and drives
the **footstep-sound** path — it maps each **tile-index range** to a
terrain **material** whose walk/run sounds correlate with `sound.dat`:

```text
{ 14 material → sound-handle ids }
var "grass","1000"  var "dirt","1001"  var "stone","1002"  var "wood","1003"
var "water","1004"  var "sand","1005"  var "grittystone","1006"
… + run variants  grassrun/dirtrun/…  "1007".."1013"

{ 645 tile-range assignments }
set range 0,11    set sound dirt   set runsound dirtrun
set range 12,27   set sound grass  set runsound grassrun
set range 48,49   set sound stone  set runsound stonerun
…
```

So a cell's `floor_tile_id` selects the material via its containing
`set range A,B`, and walking vs running picks `sound` vs `runsound` — the
terrain-footstep audio table ([`../sound-runtime.md`](../sound-runtime.md)),
distinct from the render layers above.

## Hex dump — `main/startup/world.x0` first offsets

```text
00000000  00 10 00 00                                      chunk_offsets[0]    = 0x00001000
00000004  00 34 00 00                                      chunk_offsets[1]    = 0x00003400
00000008  00 58 00 00                                      chunk_offsets[2]    = 0x00005800
0000000c  00 7c 00 00                                      chunk_offsets[3]    = 0x00007c00
00000010  00 a0 00 00                                      chunk_offsets[4]    = 0x0000a000
00000014  00 c4 00 00                                      chunk_offsets[5]    = 0x0000c400
00000018  08 e8 00 00                                      chunk_offsets[6]    = 0x0000e808  (← +0x2408 = 9224, irregular)
0000001c  20 0f 01 00                                      chunk_offsets[7]    = 0x00010f20
…
00001000  …                                                sector[0].cells[0..]
```

Sector 0..5 are the minimum 9216 bytes (offsets step by exactly
`0x2400`); sector 6 has an 8-byte trailer (step `0x2408`); etc.

## Loader citations

```text
div.exe:0x005a0300   FUN_005a0300   ctor: open file, read 0x400 u32 offset table into +0x7, append EOF as offsets[0x400]
div.exe:0x0059ce90   FUN_0059ce90   per-sector parser: walks 1024 sectors × 512 cells; pulls (sub_x, sub_y, obj_id, flags) from each record's object array and dispatches via FUN_00572100
div.exe:0x00572100   FUN_00572100   per-object placement: looks up sprite catalogue entry, applies adjustments, calls draw helpers
div.exe:0x0056d260   FUN_0056d260   per-cell height/flags stream loader (DAT_00750d64 path; usually `<save>\height.x<n>`)
div.exe:0x0057ca10   fcn.0057ca10   .\WORLD\mapman.cpp:191 — cwd-guarded blob loader (231 B): ftell/fseek to size the file remainder, fread it, parse as a count-prefixed array of 264-byte records (fcn.004fa920, .\MISC\Misc.cpp:63), optional _chdir around the load.
div.exe:0x00592950   fcn.00592950   .\WORLD\tileman.cpp:52 — tile name→id registry insert (89 B): lookup via fcn.005928b0 (atoi fast-path, else linear scan of the {char* name, int id} array at +0x10/+0x14), on miss grow the array and _strdup the name.
```

Source path leak: `.\WORLD\World.cpp:0xce`. (`mapman.cpp` and `tileman.cpp`
are small helper units within this same WORLD map system — the map
loading/orchestration itself is the boot step 17 `fcn.005a0300` above and
[architecture](../architecture.md), not a separate manager. Their unit
attribution comes from the debug-allocator `file,line` tags pushed inside
each function, and neither unit has an RTTI manager class.)

Object layout used by the in-memory class (selected fields):

| Offset | Type | Meaning |
|---|---|---|
| `+0x04` | `char*` | strdup'd path |
| `+0x0c` | `FILE*` | file handle |
| `+0x1c` | `u32*` | offset table buffer (0x1004 bytes: 1024 offsets + appended file size) |
| `+0x20` | `u32` | sector grid stride along X (set to 128 by ctor) |
| `+0x24` | `u32` | sector grid stride along Y (= `param_4 * 2 + 0x40`; usually 128) |
| `+0x34` | `u32` | sector pixel size hint (16) |

## Companion partitions

Each `world.x<n>` is paired with `objects.x<n>`, `extfree.x<n>`,
`shroud.x<n>`, and a `mapv.<n>` version stamp. Their formats are not
yet reversed in detail.
