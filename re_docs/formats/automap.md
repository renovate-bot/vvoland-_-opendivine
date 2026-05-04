# Automap tile pyramid — `.lzb` + `.lzi`

The world automap is a 3-level mip pyramid stored as paired files per region:

```text
dat\automap_<region>_<size>.lzb     // tiles, LZO1X-compressed
dat\automap_<region>_<size>.lzi     // u32 offset table into the .lzb
```

`<region>` is `0..4` (clamped, hard-coded in the loader). `<size>` is one of `1024`, `2048`, `4096`, selected by a detail-level argument (1→1024, 2→2048, anything else→4096). The three levels are powers-of-two zooms of the same map; tile counts grow by 4× per step.

Tile size is fixed at **64 × 64 pixels, 16-bit (RGB565)** — `0x2000` (8192) bytes per decompressed tile. Total uncompressed size of a level is `width_tiles * height_tiles * 0x2000`.

All integers little-endian (PE32 x86).

## `.lzb` layout

```text
struct Lzb {
    u32 width_px;                  // total pixel width  (e.g. 1024)
    u32 height_px;                 // total pixel height (e.g. 2048)
    u32 width_px_alt;              // duplicate of width_px in observed files
    u32 height_px_alt;             // duplicate of height_px
    u8  compressed_tiles[];        // tiles concatenated, in row-major order
};
```

Tile count is `(width_px / 64) * (height_px / 64)`. There is no per-tile size or magic in the `.lzb` itself — boundaries come from the `.lzi`.

## `.lzi` layout

```text
struct Lzi {
    u32 offsets[N];                // file size must be a multiple of 4
};
```

`N == tile_count + 1`. `offsets[0] == 16` (the `.lzb` header size). `offsets[i]` is the byte offset of tile *i*'s compressed payload from the start of the `.lzb`. The compressed size of tile *i* is `offsets[i+1] - offsets[i]`. The final entry points one past the last tile (i.e. it is the `.lzb` size, useful as the upper bound).

Loader rejects any `.lzi` whose size is not a multiple of 4.

### Hex dump — `dat/automap_0_1024.lzi` first entries

```text
00000000  10 00 00 00       offsets[0] = 0x10  (== sizeof Lzb header)
00000004  54 0f 00 00       offsets[1] = 0x0f54  → tile 0 compressed size = 0xf44
00000008  99 1f 00 00       offsets[2] = 0x1f99  → tile 1 compressed size = 0x1045
…
```

`automap_0_1024.lzi` has 2052 bytes → 513 entries → 512 tiles (level 1024 has a 16×32 grid of 64-px tiles for region 0).

## Compression — LZO1X-1

The engine's decompressor is at `div.exe:0x005a4ce0`, called from `div.exe:0x0044ede0` (tile-load method on the `TiledBitmap` class — source path leak `.\automap\TiledBitmap.cpp`). Cross-referencing it against the miniLZO reference confirms it is **stock LZO1X-1**:

- Literal-run prefix with the `byte > 0x11` "first-instruction" path and the `byte - 17` literal length.
- Run-length extension when a length nibble is 0: read bytes and add 255 until non-zero.
- Match-distance encoding split at `0x10`, `0x20`, `0x40`, with the `* -4 - (b >> 2) - 1` and `* -8 - ((b >> 2) & 7) - 1` shapes.
- Backref windows of `0x800` (short) and `0x4000` (long) bytes.

There are no engine-specific tweaks to the bitstream, so any LZO1X-1 implementation can decompress these tiles. Use `pkg/lzo` (or an embedding of `github.com/rasky/go-lzo` once we settle on a dependency).

## Post-decompression fixup

After decompression, the engine ORs every output `u32` with `0x08210821` (one-pixel-wide bit on the LSB of each RGB565 channel — the `0x0821` mask is `R=00001 G=000001 B=00001`). This is presumably a "make-this-pixel-non-black" mask for the explored-area overlay; reproduce it for visual parity but it can be skipped if you only want raw tile data.

## Object layout (informational, for cross-RE)

`TiledBitmap` instance fields used by the loader (`div.exe:0x0044f900`):

| Offset | Type | Meaning |
|---|---|---|
| `+0x00` | `u32` | header word 0 (`width_px`) |
| `+0x04` | `u32` | header word 1 (`height_px`) |
| `+0x08` | `u32` | header word 2 (duplicate width) |
| `+0x0c` | `u32` | header word 3 (duplicate height) |
| `+0x10` | `u32*` | primary tile-handle grid (`width_tiles * height_tiles * 4` bytes) |
| `+0x14` | `u32*` | secondary grid (used when flag `+0x3c` set) |
| `+0x20` | `i32` | width in tiles |
| `+0x24` | `i32` | height in tiles |
| `+0x38` | `FILE*` | `.lzb` handle |
| `+0x3c` | `u8` | "double-buffered" flag |
| `+0x40` | `u32*` | `.lzi` offset table |
| `+0x44` | `u8*` | scratch buffer for compressed tile reads |
| `+0x48` | `i32` | scratch buffer capacity |

## Loader citations

```text
div.exe:0x0044f900   FUN_0044f900   open .lzb, read 16-byte header, then call .lzi loader
div.exe:0x0044f730   FUN_0044f730   load .lzi (size-mod-4 check, slurp into +0x40)
div.exe:0x0044ede0   FUN_0044ede0   per-tile: seek+read compressed bytes, call decompressor, OR fixup
div.exe:0x005a4ce0   FUN_005a4ce0   LZO1X-1 decompressor
```
