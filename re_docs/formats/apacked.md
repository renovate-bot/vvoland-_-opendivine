# `APacked` — animation-class → frame index (`static\imagelists\`)

The **APacked** family is the animation metadata that maps an
`AnimationIndex` (the field on `static\objects.000` records,
[`objects.md`](objects.md)) to a run of sprite frames. It is the index
[`imagelists.md`](imagelists.md) defers to; the sprite pixels themselves
live in the CPacked imagelists, not here.

Two parallel file families, `n = 1..6`:

| File | Role |
|---|---|
| `APackedi.<n>` | **index** — one record per animation class |
| `APackedb.<n>` | **frame data** — the frames the index points into |

Both are **headerless arrays** (record count = file size / record size),
read via the `TIndexedFile` constructor ([`imagelists.md`](imagelists.md)).

## `APackedi.<n>` — animation-class index (16-byte records, verified)

```text
Record[count]:                 (16 bytes — 4 × u32 LE; count = filesize/16)
    u32 reserved0              always 0 in the shipped files
    u32 frame_count            number of frames in this animation class
    u32 frame_offset           byte offset of the first frame in the .b data
    u32 reserved3              always 0
```

Verified against the shipped files: every `APackedi.<n>` is an exact
multiple of 16 (`.1` = 4528 = 283×16, `.2` = 4512 = 282×16, `.3` = 240 =
15×16, `.4` = 80 = 5×16), matching the per-file class counts in
imagelists.md (283/282/15/5/2/92 = **679** animation classes total).

The key relation, verified numerically: **consecutive `frame_offset`
values differ by `frame_count × 32`**. E.g. in `APackedi.1` the first
classes are `(_, 10, 0, _)`, `(_, 10, 320, _)`, `(_, 3, 640, _)`,
`(_, 3, 736, _)` — deltas `320, 320, 96` = `10·32, 10·32, 3·32`. So the
frame data is a packed run of **32-byte frame entries**, and a class
occupies `[frame_offset, frame_offset + frame_count·32)`; `frame_offset/32`
is the first frame's index. (`reserved0`/`reserved3` are zero in every
shipped record; `frame_count` ranges ~0..24.)

## `APackedb.<n>` — frame data (32-byte records)

The `.b` files are likewise an exact multiple of 16 and, given the
`×32`-strided offsets above, are an array of **32-byte frame records**
(`APackedb.6` = 56448 = 1764×32). Each record splits into:

```text
Record[count]:                 (32 bytes, 8 × u32 LE)
    +0x00 u32  frame_id         sequential id — VERIFIED: increments by 1
                                across the whole file (78852/78852 in
                                apackedb.0, 35/38 in APackedb.4)
    +0x04 u32  dim_a            small positive (e.g. 32) — a width/height
    +0x08 u32  dim_b            small positive (e.g. ~62) — the other dim
    +0x0c i32  placement/param  small (often 0; a signed anchor/offset)
    +0x10 i32  placement/param  small (signed)
    +0x14 u32  = 0xffffffff      VERIFIED constant across every file — the
                                runtime-pointer sentinel (baked -1)
    +0x18 u32  data_offset/ptr  large (e.g. 0x1a00260) — byte offset / baked
                                pointer into the packed pixel data
    +0x1c u32  flags            few distinct values, high-word bits
```

So the first ~16 bytes are the persisted frame descriptor (id + two
dimensions + placement) and the trailing 16 are the loader's runtime
pointer/sentinel mirror (`+0x14 = -1`, `+0x18` = baked data pointer) — the
same "baked pointer + `0xffffffff` sentinel" shape seen in the other
`TIndexedFile`/heap-dump structures.

*(Non-result, recorded so it isn't re-tried: in `APackedb.1`'s first records
`+0x10` happened to equal `maxHeight − dim_b`, but that does not hold on
`APackedb.4`/`apackedb.0` — it was a coincidence, so `+0x10` is a free signed
param, not a height-derived offset.)*

## Status

- `APackedi.<n>` ✅ — 16-byte record layout verified byte-exact across the
  shipped files; `{reserved0, frame_count, frame_offset, reserved3}` with
  the `frame_offset += frame_count·32` invariant confirmed numerically. An
  animation class resolves to `frame_count` 32-byte frames at `frame_offset`.
- `APackedb.<n>` ✅ (structure + key fields) — 32-byte frame records; field
  split tabled above. **`+0x00` = sequential `frame_id`** and **`+0x14` =
  `0xffffffff` sentinel** are verified byte-exact across the shipped files;
  `+0x04`/`+0x08` are the two dimensions, `+0x18` the baked data pointer.
- 🟡 remaining (narrowed) — only the width-vs-height labelling of
  `+0x04`/`+0x08` and the exact semantic of the `+0x0c`/`+0x10` placement
  params and the `+0x1c` flag bits (needs the imagelist consumer); and the
  precise `APackedi.<n>` ↔
  `APackedb.<n>` ↔ CPacked-imagelist pairing (the shipped `.b` set is
  `.4/.5/.6` only, so some `.i` indices point at frame banks not present in
  this install). The index→frame *mechanism* is reimplementable.
