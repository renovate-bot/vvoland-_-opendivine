# `static\imagelists\*` — the imagelist family

The imagelist directory is the engine's master sprite & animation
bank. There are three families of paired blob/index files, all of
which use the same `TIndexedFile` ABI documented in
[`cpacked.md`](cpacked.md) (LZO1X-1 compressed blob, 56-byte index
record stride):

```text
static/imagelists/
    CPackedb.<n>c   CPackedi.<n>c    13 files (n=0..12)  cell sprites + tiles
    APackedb.<n>    APackedi.<n>      6 files (n=1..6)    animation metadata
    Collide.<n>     (no index?)       3 files (n=2,3,5)   collision masks
    cubelist.<n>                                          per-region cube lookups (referenced)
    roofs.000       roofs.dat                             roof overlays — see formats/roofs.md (TODO)
```

## CPacked family — 13 sprite/tile imagelists

| List | Entries | Flags seen | Dominant size | Likely role |
|---:|---:|---|---|---|
| 0 | 7,208 | `1` | 52×75, 57×77, 74×97 | Object sprites (trees, walls, NPCs, items) — referenced from `world.x*` `obj_id` and `static\objects.000`. **Reversed in [`cpacked.md`](cpacked.md)**. |
| 1 | 78,853 | `1` | 40×100, 44×100, 39×101 | **Animation frames** — character walk/idle/attack cycles, door open/close, etc. Indexed by APacked sets. |
| 2 | 3,363 | `0` (3161), `1` (202) | **64×64** | **Floor tiles** — referenced from `world.x*` cell header `floor_tile_id`. **Reversed**. |
| 3 | 1,336 | `1` | 48×48, 59×59 | Cube / wall sprites (likely 3D-effect block faces, indexed by `static\imagelists\cubelist.<n>`). |
| 4 | 383 | `1` | 128×122, 58×36 | UI plates and dialog backdrops (large fixed-position images). |
| 5 | 262 | `1` | 58×175, 34×34 | Inventory icons + character portraits. |
| 6 | 50 | `1` | 84×32 | Small isometric tiles — possibly dungeon-specific edge pieces. |
| 7 | 4,838 | `1`, `9` | varies | Mixed — many flag-`9` entries (= `0x01|0x08`, an unmapped sprite-mode bit). |
| 8 | 663 | `1` | 42×30, 47×47 | Misc small items — books, scrolls, wand-like objects. |
| 9 | 9 | `1` | 640×~100 | **Loading screen / cinematic backdrops** — only nine entries, all very wide. |
| 10 | 278 | `1` | 73×87 | Repeated 273× one fixed size — a single character variant set. |
| 11 | 131 | `1` | 881×535, 1374×575 | Large unique images — probably scrying / vision sequences. |
| 12 | 1,387 | `1` | 47×107, 159×248 | Mixed character-class sprites (warrior / mage / surfer poses). |

`flags = 0` entries (imagelist 2 and a few stragglers in 7) carry a
**raw 8192-byte RGB565 raster** (no header, no span table — the
`FUN_00559310` post-decode path). Everything else is the standard
sparse-RLE sprite format documented in [`cpacked.md`](cpacked.md).

## APacked family — animation metadata

Six small index files (sizes 80 / 80 / 4 / 1 / 0 / 26 entries —
APacked.5 is empty in the shipped install). The sprites these
imagelists describe **are not in APacked themselves**; APacked is the
animation table that maps `animation_index` (used by
`static\objects.000` `AnimationIndex` field) to a *range* of sprite
ids in **CPacked imagelist 1**.

APacked `.i` files use the `TIndexedFile` constructor with a
**16-byte stride** (= 0x10), distinct from heroes' 40-byte stride.
Each record holds `(frame_count, field_4, b_offset, reserved)`;
total of 679 anim classes across the six files. Full spec in
[`apacked.md`](apacked.md).

```text
APackedi.1  283  entries
APackedi.2  282
APackedi.3   15
APackedi.4    5
APackedi.5    2
APackedi.6   92
──────────
total       679 animation classes
```

Each class indexes into CPacked.1c's 78,853 frames; on average ~410
frames per animation class.

## Collide family — collision masks

```text
COLLIDE.2   28 KB
COLLIDE.3   24 KB
COLLIDE.5   45 KB
```

Loaded by `FUN_004717e0("static\\imagelists\\collide", ...)` with the
imagelist count = 16. The format is a custom collision-mask layout
that the engine uses for line-of-sight and click-target tests; per
the disassembly the entries pair with CPacked sprites by index, so
collide entry N is the collision mask for sprite N in some
catalogue. Not yet reversed in detail — only three files exist
(.2 / .3 / .5) so coverage is partial; matches `region_2..5` in some
mapping.

## roofs.000 / roofs.dat

`roofs.000` is 218 MB and **does** carry roof sprites + per-region
placement, despite the name. The loader is at `div.exe:0x005916f0`
(source `.\WORLD\roof.cpp`); each entry is a 68-byte header + variable
sprite data. Decoded enough to know the file is structured but not
enough to render — coverage TODO.

## TIndexedFile family ABI summary

All three families share:

```text
struct IndexEntry {            // 56 bytes
    u32 blob_offset;
    u32 width;                  // CPacked: pixels.    APacked: 0 or repurposed.
    u32 height;
    u32 flags;                  // family-specific bit field
    u32 anchor_x;               // CPacked: anchor.    APacked: probably first_frame_id.
    u32 anchor_y;               //                     APacked: probably frame_count.
    u32 width_inner;            // CPacked: tight bbox extent
    u32 height_inner;
    u32 packed_dims;
    u32 reserved[5];            // zero in shipped data
};
```

Blob entries always start with a u32 uncompressed-size header
followed by an LZO1X-1 stream — see [`cpacked.md`](cpacked.md). For
APacked the post-LZO payload is animation script data, not a sprite
raster.

## Loader citations

```text
div.exe:0x004e9b80   FUN_004e9b80   CPacked imagelist manager (loads up to 16 pairs)
div.exe:0x004e81a0   FUN_004e81a0   APacked animation manager
div.exe:0x004717e0   FUN_004717e0   Collide collision-list manager
div.exe:0x005916f0   FUN_005916f0   roofs.000 + roofs.dat reader
div.exe:0x004e9530   FUN_004e9530   per-entry LZO decompress (shared across all families)
```
