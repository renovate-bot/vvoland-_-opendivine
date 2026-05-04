# `static\heroes\` — character sprite archives

Per-class character sprite banks shared across the engine and APacked
imagelists. Each character class ships as a triplet plus a per-bank
data/index split:

```text
static\heroes\
    <class><sex>.key                       # text directory of named ranges
    <class><sex>{A,B,C,D,E}.idc            # 5 sprite indexes (40-byte records)
    <class><sex>{A,B,C,D,E}.bic            # 5 sprite payloads (matching pairs)
```

Six character classes ship: `surf`, `surm`, `warf`, `warm`, `wizf`,
`wizm` (`sur`/`war`/`wiz` × female/male). Sizes vary widely — the
largest is `wizmB.bic` at 88 MB.

The `.idc` + `.bic` pair is the shared `TIndexedFile` archive format;
the same shape is used by `static\imagelists\APacked{i,b}.<n>` (see
[`imagelists.md`](imagelists.md)).

## `.key` — text directory

Plaintext (CRLF-terminated, ASCII) line-based. Three line types:

| Line                          | Meaning                                                |
|-------------------------------|--------------------------------------------------------|
| `Max size W,H`                | Max bounding-box of any sprite (e.g. `Max size 232,253`)|
| `Center X,Y`                  | Pivot point (optional; not present in surf.key)         |
| `<name>,<start>,<end>`        | Named sprite-id range — frames `[start..end)` in the .idc family |

Names follow the engine's animation classification (e.g. `MAA0`,
`MAA1` are three poses of the same animation; `M1B0` etc. are a
second sub-class). Each class has 102 unique 3-character prefixes ×
up to 6 variants for `surf`.

## Loader citations

```text
div.exe:0x004e9fe0   FUN_004e9fe0   TIndexedFile constructor — strdup(.bic), strdup(.idc), record stride = 0x28
div.exe:0x004ea060   FUN_004ea060   TIndexedFile::Open (vtable[1]) — opens .bic + .idc; fread entire .idc into buffer
div.exe:0x004ea300   FUN_004ea300   TIndexedFile::Read (vtable[5]) — seeks to idc.offset in .bic, freads idc.size bytes
div.exe:0x004ea4b0   FUN_004ea4b0   vtable[11] GetIndexRecord — reads the 40-byte idc entry only
div.exe:0x0050bb10   FUN_0050bb10   "%s%c.bic" / "%s%c.idc" loader — opens A..E banks; calls FUN_004e9fe0
div.exe:0x0050a840   FUN_0050a840   .key parser — recognises "$,#,#" / "Max size #,#" / "Center #,#"
div.exe:0x0050aa50   FUN_0050aa50   animation group decompressor — fseek to block, read u32 compressed_size,
                                    fread compressed data, LZO decompress into per-group cache buffer
div.exe:0x0050ac30   FUN_0050ac30   MakeAnimationDirectionsFromKeys — calls FUN_0050aa50 per group, builds
                                    baked per-frame metadata; source: .\NPC\Combine.cpp
div.exe:0x005a4ce0   <lzo>          LZO1X-1 decompressor (called by FUN_0050aa50)
div.exe:0x0042c8e0   FUN_0042c8e0   "%s.key" + .bic/.idc bundle entry from .\AGENTS\agents.cpp
```

`FUN_0050bb10` also pins per-class sprite dimensions (passed to
downstream sprite math but not in any file):

| Class | W   | H   |
|-------|-----|-----|
| warm  | 0x5A | 0x9A |
| wizm  | 0x5E | 0xC0 |
| surm  | 0x5A | 0x9E |
| warf  | 0x54 | 0x96 |
| wizf  | 0x52 | 0xB8 |
| surf  | 0x58 | 0x92 |

## `.idc` — fixed 40-byte sprite directory

```text
struct IDCRecord {              // 40 bytes
    u32  offset;                 // decompressed byte offset within the animation group's decompressed buffer
    u32  size;                   // decompressed frame byte size
    u16  width;                  // sprite bounding-box width in pixels
    u16  height;                 // sprite bounding-box height in pixels (NOT the stored scanline count)
    u16  hotspot_x;              // (field_8) sprite horizontal anchor / center
    u16  hotspot_y;              // (field_a) sprite vertical anchor AND number of stored scanlines
    u8   reserved[24];           // 0xFF-filled in shipped data (= 6 placeholder u32s)
};
```

`surfA.idc` (657,600 bytes = 16,440 records) is fully validated:

- All `reserved[24]` bytes are 0xFF in every record.
- Every `width`/`height` is a plausible sprite dimension (1..1024).
- `offset[0] == 0`.
- `offset` is a decompressed byte offset **within the animation group's decompressed buffer**,
  NOT an absolute offset into the `.bic` file. Offsets reset to 0 at each group boundary
  (defined by the `.key` file). The ~50 apparent back-steps in `surfA.idc` are exactly the
  group boundaries.
- `hotspot_y` doubles as the scanline count: for 479/479 MAA0 frames verified, the decompressed
  frame contains exactly `hotspot_y` scanlines in its span table (not `height` lines).

## `.bic` — sprite payload

The `.bic` file is a flat sequence of **per-animation-group compressed blocks**, one block per
`.key` group in the order the groups are listed. Each block has:

```text
struct AnimGroupBlock {
    u32  compressed_size;          // byte count of the LZO1X-1 data that follows (NOT the uncompressed size)
    u8   lzo_data[compressed_size]; // LZO1X-1 stream; decompresses to a flat buffer holding all frames
};
```

Block k begins at absolute `.bic` offset `Σ(4 + compressed_size_i)` for `i = 0..k-1`.
`FUN_0050aa50` locates the block by fseek-ing to this offset, reading the 4-byte
`compressed_size`, then reading and decompressing the LZO stream into a per-group cache buffer.

Example from `surfA.bic` (group 0 = MAA0, 480 frames):
- `compressed_size` = 637,199 (0x9B90F) at bic offset 0
- Decompresses to 1,178,896 bytes (0x11FD10)
- Group 1 (MAA1) starts at bic offset 637,203

### Decompressed frame format

Within a group's decompressed buffer, each frame starts at `idc.offset` bytes from the
buffer beginning. The frame uses the same span-table codec as CPacked / CPackedb:

```text
struct HeroBicFrame {
    u32  total_size;               // = idc.size (redundant; decompressed frame byte count)
    u32  pixel_data_offset;        // byte offset from frame start to the pixel data block
    u16  hotspot_x;                // = idc.hotspot_x; sprite horizontal anchor
    u16  hotspot_y;                // = idc.hotspot_y; also = number of stored scanlines
    Line lines[hotspot_y];         // span table; same encoding as CPacked
    u8   pixel_data[];             // RGB565 pixel runs, 2 bytes per pixel
};
```

`idc.height` is the full **bounding-box height**; `idc.hotspot_y` is the number of
scanlines actually stored. The two differ: a sprite can have blank rows at the top or
bottom of its bounding box that are absent from the span table.

### Line and Span format (identical to CPacked)

```text
// Empty line (num_spans == 0):
struct EmptyLine {                 // 8 bytes
    u16  num_spans;                // 0
    u8   _pad[6];
};

// Non-empty line (N > 0 spans):
struct NonEmptyLine {              // (N + 2) × 4 bytes
    u16  num_spans;                // N
    u32  pixel_offset;             // MISALIGNED — starts at byte 2 of this struct
                                   // relative to pixel_data_offset within the frame
    Span spans[N];
    u16  _pad;
};

struct Span {
    u16  start_x;                  // first pixel column (0-based)
    u16  length;                   // number of opaque RGB565 pixels
};
```

The `pixel_offset` field being misaligned at byte 2 of the line header is a confirmed
quirk shared with the CPacked codec (same engine fixup at `div.exe:FUN_00558290`).

### Verification

MAA0 frame-0 (`surfA.bic`, group 0):
```text
total_size=2366, pixel_data_offset=792, hotspot_x=18, hotspot_y=65
Line[0]: num_spans=1, pixel_offset=0,   span=[x=6,  len=5]
Line[1]: num_spans=1, pixel_offset=10,  span=[x=5,  len=8]
...  (65 lines total)
```

Bulk check: 479/479 MAA0 frames satisfy `hotspot_y == n_stored_lines` (100%).

## Status

- `.key` ✅ — fully decoded; validated against shipped `surf.key`.
- `.idc` ✅ — 40-byte stride confirmed; all fields named; `hotspot_y == n_stored_lines`
  verified for 479/479 MAA0 frames.
- `.bic` ✅ — per-animation-group LZO1X-1 blocks confirmed; decompressed frame format
  (CPacked-compatible span table + RGB565 pixels) fully reversed and verified.

## Companion spec

`static\heroes\*` shares the **`TIndexedFile` constructor** with
APacked imagelists (`static\imagelists\APacked{b,i}.<n>`) and party
inventory (`<save>\inv.{b,i}<N>`), but each pair uses a **different
record stride**:

| Pair                              | Stride  | Records / file (max)        |
|-----------------------------------|--------:|-----------------------------|
| `heroes\<class><sex>{A..E}.idc`   |   40 B  | 16,440 (`surfA.idc`)        |
| `imagelists\APacked{i,b}.<n>`     |   16 B  |    283 (`APackedi.1`)       |
| `inv.{b,i}<N>` (savegame)         |   28 B  |  5,043 (`inv.i1`)           |

See [`apacked.md`](apacked.md) and [`inventory.md`](inventory.md).
The earlier claim that APacked uses the same 40-byte record as
heroes was **wrong** — APacked's stride is 0x10, pinned by the
`FUN_004e9fe0(.bpath, .ipath, 0x10, 8, 1)` call.
