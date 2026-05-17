# `.cmp` — compiled data containers

`.cmp` is a generic extension Larian uses for "compiled" tables produced from text `.dat`/`.txt` sources at build time. The shipped game contains only the `.cmp` outputs. Two distinct schemas share the extension; identify which by xref'ing the file path string in `div.exe`.

All offsets little-endian (PE32 x86).

---

## Family A — string-indexed archive (self-contained `[index][data]`)

Used by `dat\flat.cmp`, `dat\global.cmp`, `dat\sound.cmp`, `localizations\<lang>\text.cmp`, `dat\mono.cmp`, `localizations\<lang>\mono.cmp`. The index entries' offsets point into the same file, so a `.cmp` of this family is a self-contained archive (index + concatenated payloads).

### Header / record layout

```text
struct CmpArchive {
    u32 count;
    Entry entries[count];
    u8   data[];                // referenced by entries[i].offset (offset is from start of file)
};

struct Entry {
    u32  name_len;              // bytes, NOT counting NUL
    u8   name[name_len];        // OBFUSCATED — see decode below
    u8   nul;                   // 0x00, present iff name_len > 0
    u32  offset;                // absolute offset of payload from start of .cmp
    u32  size;                  // payload size in bytes
    u32  flags;                 // 0 in all observed entries
};
```

### Name obfuscation

Each name byte is decoded by NOT'ing it and XOR'ing with a 32-byte rolling key:

```python
KEY = bytes.fromhex(
    "0c40550c2d41622d0306481e0548140530323334636346331809280f06223917"
)

def decode_name(buf: bytes) -> bytes:
    return bytes((~b & 0xff) ^ KEY[i % 32] for i, b in enumerate(buf))
```

Encoding is `byte = ~plaintext ^ KEY[i % 32]`.

The 32-byte key lives at `div.exe:0x00650c8c`. The deobfuscation loop is in the per-entry reader at `div.exe:0x004f5b50`. The archive open / count-loop is at `div.exe:0x004f60f0`.

### Hex dump — `dat/sound.cmp` first entry

File header + first entry (offsets in file):

```text
00000000  b1 05 00 00                                count = 0x000005b1 = 1457
00000004  1f 00 00 00                                name_len = 0x1f = 31
00000008  84 de dc af 93 d3 ff bb 99 97 d4 84 a6 f5  obfuscated name (31 bytes) ─┐
00000016  8e 92 a6 a3 a8 8f fd de cb a5 83 91 b2 de                              │
00000024  96 ba a1                                                              │
00000027  00                                          NUL                       │
00000028  22 57 01 00                                offset = 0x00015722 = 87842
0000002c  9e 18 08 00                                size   = 0x0008189e = 530590
00000030  00 00 00 00                                flags  = 0
```

Decoding the name with the key:

```text
i  enc  ~enc  key  out  chr
0  84   7B    0C   77   'w'
1  DE   21    40   61   'a'
2  DC   23    55   76   'v'
3  AF   50    0C   5C   '\\'
4  93   6C    2D   41   'A'
5  D3   2C    41   6D   'm'
... → "wav\\Ambience\\BehindDaBridge.ogg"
```

Verified end-to-end: seeking to offset `0x00015722` in `sound.cmp` and reading `0x0008189e` bytes yields a file whose first four bytes are `4f 67 67 53` (`OggS`, the Ogg Vorbis magic).

### Loader citation

```text
div.exe:0x004f60f0   FUN_004f60f0   open + read count + per-entry alloc
div.exe:0x004f5b50   FUN_004f5b50   read one entry: u32 nl, nl+1 bytes, deobf, 3×u32
div.exe:0x00650c8c   <key>          32-byte XOR key
```

---

## Family B — fixed-record table (no strings, no obfuscation)

Used by `dat\magic.cmp`, `dat\statuspl.cmp`. Each is just a count followed by `count` packed structs. There is no name table and no obfuscation; the per-record schema is defined by whichever compiler in `div.exe` writes the file.

**Note:** `dat\itemgen.cmp` / `dat\<lang>\itemgen.cmp` is **not** Family B — it has a magic sentinel header and type-polymorphic variable-length records. See `re_docs/formats/itemgen.md`.

**Note:** `dat\treasure.cmp` is **not** Family B — it is a serialized three-level tree with a version-string header and runtime heap pointers used as occupancy flags. See `re_docs/formats/treasure.md`.

### Skeleton

```text
struct CmpTable {
    u32 count;
    Record records[count];      // record size depends on file
};
```

### Known stride

| File | Record size | Compiler in div.exe | Source `.dat` |
|---|---|---|---|
| `dat\magic.cmp` | 28 bytes (7×u32) | `0x004c93d0` (FUN_004c93d0) | `dat\magic.dat` |
| `dat\statuspl.cmp` | 12 bytes (3×u32) | `FUN_0052e7b0` (inline writer, `.\PLATE\statuspl.cpp`) | `bmg\statuspl.dat` |

### Hex dump — `dat/magic.cmp` first 16 bytes

```text
00000000  60 00 00 00                                count = 96
00000004  00 00 00 00                                record[0] field 0
00000008  00 00 00 00                                record[0] field 1
0000000c  0a 00 00 00                                record[0] field 2
```

### How to decode another Family-B file

1. In Ghidra, search strings for the `.cmp` filename and look at `get_xrefs_to` on its address.
2. The xref source is the writer (it `fopen`s, writes the count, then `fwrite`s a fixed stride).
3. Count `FUN_00489150` (iostream `operator>>` reading 4 bytes) calls in the matching reader; that is the number of `u32` fields per record.
4. Field semantics come from the compiler — the source `.dat` is plaintext with `#`-prefixed comments and tokens parsed in the same order as the `FUN_00489150` calls.

### `magic.cmp` compiler walk-through

The source-text → binary path for `magic.cmp` is at
`div.exe:0x004c93d0` (`FUN_004c93d0`, `.\magic\magic.cpp:0xfe`):

1. Open `dat\magic.dat` (text source — **not shipped** in the
   binary install).
2. First pass — count records:
   - For each line, skip any line beginning with `#` (comment).
   - Parse the line to bump the count by 1.
3. Allocate `count * 0x1c` (= 28 bytes per record) bytes.
4. Re-open `dat\magic.dat`, second pass:
   - Skip `#` comments.
   - For each non-comment line, call `FUN_00489150` exactly **seven
     times** at offsets +0, +4, +8, +0xc, +0x10, +0x14, +0x18 of the
     current record. Each `FUN_00489150` is the iostream `operator>>`
     pulling a single `u32` token from the stream.
5. Write `dat\magic.cmp` as `[u32 count][record × count]`.

So `magic.cmp` is exactly **`[u32 count] [7×u32 records]`** with no
strings or sub-records. The 7 columns map 1-to-1 to the 7 tokens per
line in the source `magic.dat`. Without the source file shipped, the
column meaning has to come from the engine's *consumer* code — the
spell system's accessors of these fields. That mapping is not yet
done.

The same writer pattern (text → 7-or-N×u32 records) applies to
`treasure.cmp`, `statuspl.cmp`, and friends — different writers in
the binary, identical shape.

Field-level semantics (which u32 is damage, element, mana cost, etc.)
are not yet mapped and must not be guessed.

### `statuspl.cmp` — UI element layout table

**Not** a count+records format: the entire file is exactly **0x168 = 360 bytes**,
a flat array of **30 records × 12 bytes** each. There is no count field.

Confirmed from `div.exe:0x0052e7b0` (`FUN_0052e7b0`, `.\PLATE\statuspl.cpp:0x107`):

```c
// Writer (inline in constructor):
fread(param_1[0xd], 0x168, 1, iVar4);  // reads 360 bytes as a single blob

// Compiler (from bmg\statuspl.dat text source):
while (iVar4 < 0x168) {                // 0x168 / 0x0c = 30 records
    switch(alignment_char) {           // 'L'/'C'/'R' → 0/1/2
        *(u32 *)(iVar4 + 8) = val;     // +0x08: alignment
    }
    FUN_00489150(iVar4 + 0);           // +0x00: x (screen pixel)
    FUN_00489150(iVar4 + 4);           // +0x04: y (screen pixel)
    iVar4 += 0x0c;
}
```

```text
// statuspl.cmp is 360 bytes — no count header
struct StatusPlRecord {      // 12 bytes (0x0C)
    s32 x;                   // +0x00 — screen pixel X of UI element
    s32 y;                   // +0x04 — screen pixel Y of UI element
    u32 alignment;           // +0x08 — 0=left 1=center 2=right
};
StatusPlRecord records[30]; // fixed 30 entries
```

Source file `bmg\statuspl.dat` uses format per line: `[L|C|R] <x> <y>`.
The 30 entries correspond to the 30 labelled stat/attribute UI elements on the
character status plate screen (labels and value fields for Strength, Dexterity,
Intelligence, Constitution, Speed, XP, Level, etc.).

---

## Status

- Family A: schema fully reversed end-to-end and verified against `sound.cmp` (Ogg payload). Safe to implement a reader and an extractor. Writer is symmetric.
- Family B (`magic.cmp`): skeleton + stride confirmed; per-record field meanings are TBD and require reading each compiler's consumer.
- Family B (`statuspl.cmp`): fully reversed ✅ — 30 × 12-byte UI element positions. No Go parser needed (UI-only, not game data).
