# `Hufmann.cpp` — the world/save Huffman codec

`.\MISC\Hufmann.cpp` is the engine's general-purpose **Huffman
compression codec**. It is the compressor behind `.\WORLD\Compress.cpp`
([`osi-static.md`](osi-static.md)): the per-map world/save data files
(`world.x<n>`, `objects.x<n>`, `extfree.x<n>`, …) are Huffman-packed into
the `group.c0` / `group.c%d` ("**c**" = compressed) archives and into
`game.sav`. It is **not** the `.cmp` payload codec — Family-A `.cmp`
payloads are stored raw (`sound.cmp` = plain Ogg, [`cmp.md`](cmp.md)).

## Functions

```text
div.exe:0x004fa1e0   decode (decompress) entry point.
div.exe:0x004f9ea0   encode (compress) entry point.
div.exe:0x004f9700   codec setup — allocates the working/table buffer.
div.exe:0x004f9cf0   bit-level decode primitive (table lookup).
div.exe:0x004f9dd0   bit-level decode primitive (table lookup).
div.exe:0x004f9620   IL_malloc wrapper ("Program aborted: IL_malloc(%ld) failed …").
div.exe:0x004fa540   buffer free.
```

All four `.\MISC\Hufmann.cpp` linetrack tags sit on these functions
(`0x4f9635`, `0x4f9f71`, `0x4fa01b`, `0x4fa24c`).

## Algorithm shape (verified statically)

- **Block-based.** Both encode and decode allocate `0x2000`-byte (8 KiB)
  working buffers (`0x4fa25c push 0x2004`, `0x4fa281 push 0x2000`; the
  global byte counter at `0x6e0170` is bumped by `0x2000` per block), so
  the stream is processed in 8 KiB chunks rather than whole-file.
- **Table-driven (canonical), not adaptive.** The inner decode primitives
  `fcn.004f9cf0` / `fcn.004f9dd0` are pure bit-accumulator + table-index
  loops: they shift a bit window with `shl/shr …, cl`, mask the window to a
  ~12-bit index (`and …, 0xffe`) and read the next symbol/length from a
  precomputed byte/word table (`movzx ecx, byte [edx]`). There is **no
  per-symbol tree mutation** in the decode loop, so the tree is built once
  per block from a header and used as a fixed lookup table — a canonical
  Huffman decode, not an adaptive (FGK) one. `fcn.004f9700` allocates that
  table up front.

## Consumers (`.\WORLD\Compress.cpp`)

```text
div.exe:0x004f9ea0  encode ← fcn.00572d30, fcn.005738d0   (pack path)
div.exe:0x004fa1e0  decode ← fcn.00572d70, fcn.005732a0, fcn.00572d10  (unpack path)
div.exe:0x005732a0  .\WORLD\Compress.cpp save-state bundler (decode side)
div.exe:0x00572d70  .\WORLD\Compress.cpp unpacker (per-file Huffman decode)
div.exe:0x005738d0  .\WORLD\Compress.cpp packer (per-file Huffman encode)
```

So entering/leaving a map and saving (the `.x<n>` blob plumbing in
[`osi-static.md`](osi-static.md)) runs each data file through this codec
to produce/consume the `group.c*` compressed archives.

## Status

- Module & role ✅ — identified as `.\MISC\Hufmann.cpp`, the Huffman codec;
  encode/decode entry points and the bit-level decode primitives pinned;
  consumers traced to `.\WORLD\Compress.cpp` (the `group.c*` / `game.sav`
  compression). Disproves any assumption that `.cmp` payloads are
  Huffman-compressed (they are raw).
- Block model ✅ — 8 KiB (`0x2000`) block buffers; canonical (static,
  table-driven) Huffman per block, not adaptive.
- Exact bitstream/header format 🟡 — the per-block tree-table header
  layout, bit order, and any end-of-block marker are not yet itemized to
  the byte; recovering them requires walking `fcn.004f9700`'s table build
  and the `fcn.004f9cf0`/`fcn.004f9dd0` index math against a sample
  `group.c*` blob (no standalone `.huf` file ships to diff against).
