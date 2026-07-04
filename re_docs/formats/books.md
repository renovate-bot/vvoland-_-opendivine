# `books.000` — in-game book / note text

The readable books, notes and signs the player finds in the world.
293,290 bytes; **250 books**. **Fully decoded, byte-exact** (this
corrects the earlier "heap-dump, needs deserialize-replay" reading —
see the history note at the bottom).

## Structure ✅ (byte-exact)

```text
u32   count = 250
Rec[count] (12 bytes each):
    u32  text_ptr     baked heap pointer — GARBAGE on disk (the writer
                      fwrites the in-memory record array verbatim); ignore
    u32  image_tag    0xffffffff = text-only book; else an IMAGELIST index
                      (shipped: 47 books, tags 542..601) for the book's
                      illustration plate
    u32  len          text length INCLUDING the trailing NUL
u8    texts[]         the 250 texts concatenated IN RECORD ORDER,
                      len bytes each (plain ASCII, \n formatting)
```

Validation: `4 + 250×12 + Σlen = 293,290` — exact file size; every text
decodes cleanly. Picture-only books have text `"\n\0"` (e.g. book 36,
tag 581 = "LIBER AESOPI DE NATURA ANIMALIUM"). The earlier "12-byte walk
mis-aligns" finding was simply the wrong field order: it assumed
`{ptr,len,extra}`; the real order is `{ptr,tag,len}` (the "586" it read
as a length is book 0's *tag*).

## Code side (`.\WORLD\BookContent.cpp`)

Singleton `CBookContent` at `0x74cb90` = `{count@+0, rec_array@+4}`.

- **Writer `fcn.0056c560`** — writes `dat\books.txt`
  (`"[bookentry] %d %d\n"` + `"%s"`) *and* the binary:
  `fwrite(&count,4,1)`, `fwrite(recs,12,count)` (live `char*` pointers
  included → the baked pointers), then per record `fwrite(text,1,len)`.
  Sole caller: the map-export dumper `fcn.00598600` (writes the whole
  `dynamic\*.001` set, Huffman-packed via `fcn.005738d0`). The shipped
  `books.000` is this exporter's output.
- **The binary is never read back.** All four load paths compile from
  **text**: `fcn.0056c830` (`fopen "rt"`), `fcn.0056c9b0` (packed-VFS
  `[0x6ddd24]`), `fcn.0056cb50`/`fcn.0056cd50` (UTF-16 variants) — all
  `sscanf "[bookentry] %ld %ld"` (2nd number = the image tag),
  accumulating text lines until the next `[bookentry]`; `AddBook` =
  `fcn.0056c740` (stores `len = strlen+1`, BookContent.cpp:176/179).
  Orchestrator `fcn.0056cf60` (builds `dat\[<locdir>\]books.txt`) is
  called from boot `0x49a5d3` and map-reload `0x4a1a2b`. The registry
  slot `reg+0x10` (`0x750d48`, `dynamic\books.000`) has **zero**
  readers. All other `books.000` references are copy/scratch plumbing.
- **Image tag semantics** (consumer `fcn.004f10c0` → `GetText`
  `fcn.0056c670` / `GetTag` `fcn.0056c690`): tag ≠ −1 opens the
  `"BookImagePlate %lx"` window (`fcn.00517d80`); `fcn.00518070` binds
  imagelist `[0x658bd8]→+0x14[tag]`, sub-image 8 — the illustration.
  −1 = plain text book.
- **World-object → book-index link ✅ (verified against shipped
  instances)**: a placed book/note/scroll object carries its book index
  (0..249) in the **`s_function_parameter` value slot** (`flags_a` **bit
  0**, a **big-endian `u16`** in the value pool, unpacked by
  `fcn.005918b0`), gated by **`s_function` (bit 17) == 10** (the
  "read book / show text" action). Confirmed on `objects.x0..x4`: of the
  482 instances of `s_function==10` kinds (`Book`/`Manuscript`/`scroll`/…),
  476 (99 %) carry an `s_function_parameter` in `0..249` (the exact
  `books.000` range; 3 outliers > 249 are special tomes). The other
  candidate `s_value` (bit 16) is **not** the index — it is the item's
  gold value (0 for all these instances; `Sign` kinds carry only
  `s_value`). So the reader UI resolves the book by
  `books[s_function_parameter]`.

**Reimplementation note:** parse the binary with the table above
(skipping `text_ptr`), or compile from `dat\books.txt` like the engine
does. Both are now byte-understood.

## History — how the earlier "heap-dump virtual-stream" reading fell

An earlier pass concluded the index stride was "recoverable only by
replaying the interned-handle generic deserialize" shared with
`objects.x<n>`. Both pillars of that theory were wrong:
`fcn.005e5eec` — read as "the handle interner" — is just the
**`MSVCR90._strdup` import thunk**, and the WORLD registry (`0x750d38`,
ctor `fcn.0057c650`) merely holds strdup'd path strings; there is no
generic virtual-stream deserializer behind it. See
[`osi-static.md`](osi-static.md) for the registry field map and the
(also now decoded) `objects.x<n>` record format.
