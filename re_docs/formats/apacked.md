# `APacked` — animation-class → frame index (`static\imagelists\`)

The **APacked** family is the animation metadata that maps an
`AnimationIndex` (the field on `static\objects.000` records,
[`objects.md`](objects.md)) to a run of sprite frames. It is the index
[`imagelists.md`](imagelists.md) defers to; the sprite pixels themselves
live in the CPacked imagelists, not here.

Seven parallel file families, `n = 0..6`:

| File | Role |
|---|---|
| `APackedi.<n>` | **index** — one record per animation class |
| `APackedb.<n>` | **frame data** — the frames the index points into |

Both are **headerless arrays** (record count = file size / record size),
read via the `TIndexedFile` constructor ([`imagelists.md`](imagelists.md)).

## `APackedi.<n>` — animation-class index (16-byte records, verified)

```text
Record[count]:                 (16 bytes — 4 × u32 LE; count = filesize/16)
    u32 image_bank              CPacked imagelist containing the frames
    u32 frame_count            number of frames in this animation class
    u32 frame_offset           byte offset of the first frame in the .b data
    u32 reserved3              always 0
```

Verified against the shipped files: every `APackedi.<n>` is an exact
multiple of 16. The seven files contain 9,947, 283, 282, 15, 5, 2, and
92 records respectively, for 10,626 animation classes total.

The key relation, verified numerically: **consecutive `frame_offset`
values differ by `frame_count × 32`**. E.g. in `APackedi.1` the first
classes are `(_, 10, 0, _)`, `(_, 10, 320, _)`, `(_, 3, 640, _)`,
`(_, 3, 736, _)` — deltas `320, 320, 96` = `10·32, 10·32, 3·32`. So the
frame data is a packed run of **32-byte frame entries**, and a class
occupies `[frame_offset, frame_offset + frame_count·32)`; `frame_offset/32`
is the first frame's index. (`reserved` is zero in every shipped record;
`frame_count` ranges ~0..24.)

## `APackedb.<n>` — frame data (32-byte records)

The `.b` files are likewise an exact multiple of 32 and, given the
`×32`-strided offsets above, are an array of **32-byte frame records**
(`APackedb.6` = 56448 = 1764×32). Each record splits into:

```text
Record[count]:                 (32 bytes)
    +0x00 i32  image_index       image index inside `image_bank`
    +0x04 i32  width             image width
    +0x08 i32  height            image height
    +0x0c i32  offset_x          authored draw offset
    +0x10 i32  offset_y          authored draw offset
    +0x14 i32  image_handle      runtime handle; `-1` on disk
    +0x18 i16  mirror_offset_x   authored horizontal-mirror offset
    +0x1a i16  mirror_offset_y   authored vertical-mirror offset
    +0x1c i16  reserved          zero in the shipped data
    +0x1e i16  padding            uninitialised in some shipped data
```

The frame's image index is local to the image bank named by its owning
index record. For world objects, `AnimationIndex` is resolved in
`APackedi.1`; its records use CPacked imagelist 0, so the frame image can
be decoded by the same reader as the object's static sprite. For example,
the Candle catalogue entry (id 2981) names animation 160.

*(Non-result, recorded so it isn't re-tried: in `APackedb.1`'s first records
`+0x10` happened to equal `maxHeight − dim_b`, but that does not hold on
`APackedb.4`/`apackedb.0` — it was a coincidence, so `+0x10` is a free signed
param, not a height-derived offset.)*

## Status

- `APackedi.<n>` ✅ — 16-byte record layout verified byte-exact across the
  shipped files; `{image_bank, frame_count, frame_offset, reserved}` with
  the `frame_offset += frame_count·32` invariant confirmed numerically. An
  animation class resolves to `frame_count` 32-byte frames at `frame_offset`.
- `APackedb.<n>` ✅ — 32-byte frame records; image indices, dimensions, draw
  offsets, and mirror offsets are decoded above.
- `APackedi.1` ambient world-object playback 🟡 — animation 160 selects the
  Candle's frame sequence, and each frame is rendered from CPacked imagelist
  0. Interactive object animation remains state-driven work rather than an
  idle loop.
