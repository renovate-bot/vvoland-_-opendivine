# Roofs — building roof overlays (`.\WORLD\roof.cpp`)

The roof system draws building **roof overlays** on top of the world and
hides them when the player walks under one (the classic top-down "roof
fades so you can see inside" effect — the engine's **"Roof Mode"**). The
on-disk format is fully decoded: byte-exact framing and all field roles
(engine-named via the `roofs.txt` format string).

## Files

| File | Size (shipped) | Role |
|---|---|---|
| `static\imagelists\roofs.000` | ~218 MB | the roof **images** (a CPacked imagelist blob, [imagelists](imagelists.md)) |
| `static\imagelists\roofs.dat` | 7188 B | the roof **index / definitions** (137 roofs) |
| `dat\roofs.txt` | *(not shipped)* | the plaintext **source** the compiled `roofs.000`/`.dat` are built from |

Source unit `.\WORLD\roof.cpp` (asserts `"Invalid roof found %d,%d"`,
`"Attempt to deallocate roof image but that's not possible"`).

## Loader — `FUN_005916f0`

`fopen`s `roofs.000` then `roofs.dat` and drives the read (call site
`0x4a13f4`, `ecx = 0x751680` — the manager is the global object *at* that
address; gated on `[0x65b3d0]+0x104`, with the else-branch loading the
plaintext `dat\roofs.txt` via `fcn.005914f0` instead):
- `fcn.00590250` — free-and-reload guard: frees the existing roof list,
  then reads a `u32 count` + `count × 12-byte records` (overflow-checked
  `count·12` alloc, roof.cpp:498).
- reads the `u32` roof count, then per roof allocates a **0x494-byte**
  object (roof.cpp:68), constructs it via `fcn.0058f8b0`, and calls
  `fcn.00590d30`.
- `fcn.00590d30` — the **per-roof header reader**: 10 sequential `fread`s
  (one per scalar field), then calls `fcn.00590c30` on the tail sub-object
  at `+0x34`.
- `fcn.00590c30` — the **variable tail reader**: two counts + two allocated
  `u32` arrays (roof.cpp lines 65/70), i.e. the roof's tile/cell lists.
- `fcn.005913e0` — image binding (roofs.000). The manager class has RTTI:
  **`RoofModule`** (`.?AVRoofModule@@` @ `0x657bd8`).

## `roofs.dat` layout

```text
u32   count = 137
RoofRecord[count]                  — variable length
```

Each **RoofRecord** is a fixed **40-byte header** (10 × `u32`, read
field-by-field by `fcn.00590d30` — each `fread(size=4, n=1)`) followed by a
**variable tile/cell tail** (`fcn.00590c30`). The on-disk field order (= the
`fread` sequence) and meanings, with values from records 0/1 of the shipped
file:

All field roles are now **engine-named**: the plaintext `dat\roofs.txt`
reader `fcn.00590350` parses the format string at `0x6206a8` —
`"index # list # offsetx # offsety # worldx # worldy # worldheight #
map #"` — storing tokens at exactly the binary reader's offsets:

```text
disk  mem    field        rec0       rec1      meaning
+0    +0x10  index        0x0        0x1       roof index — NOT unique (placements)
+4    +0x14  list         0xb        0xb       imagelist selector (0xb in all 137)
+8    +0x1c  offsetx      0x415      0x24e     image-local depth-sort anchor X
+0xc  +0x20  offsety      0x228      0x37a     image-local depth-sort anchor Y
+0x10 +0x00  worldx       0x1d35     0x1203    world footprint origin X
+0x14 +0x04  worldy       0xd52      0x10fb    world footprint origin Y
+0x18 +0x08  worldheight  0x9f       0x87      roof plane's world Z
+0x1c +0x0c  map          0x0        0x0       world partition id (0..4)
+0x20 +0x2c  width        0x55e      0x4f1     footprint/image width  (rec0 = 1374)
+0x24 +0x30  height       0x23f      0x393     footprint/image height (rec0 = 575)
```

The runtime **footprint is `(worldx,worldy)-(+width,+height)`** — the
point-in-rect test `fcn.0058fd70` reads mem `+0/+4` as the rect origin
and *adds* mem `+0x2c/+0x30` as the extent. *(This corrects the earlier
"bbox `(x1,y1)-(x2,y2)` min/max corners" reading; rec0's `1374×575`
matches the roof imagelist dims — world footprint size = image size.)*
The mem `+0x18` slot is skipped by the `fread`s and zeroed — runtime
visibility state. Field notes (all verified in consumers):
- **`offsetx/offsety`** feed the draw path `fcn.005905a0`: when the A/B
  cut lists are empty, `offsetx+offsety` is added to the iso depth key
  (`0x59068e`, `0x5906a7–0x5906b2`) — usually the far corner `(w,h)`.
  *(Correction to an interim note: duplicate-index records do NOT always
  share the pair — index 113's two records differ in `offsety`,
  657 vs 652.)*
- **`worldheight`** is the third coordinate into the world→screen
  transform (`0x590611–0x590671`; the far corner uses
  `worldheight+height`). The dominant value 200 is simply the standard
  roof-plane height.
- **`map`** (the doc's earlier "flags" is wrong) — roofs register only
  on their own partition (`0x591619` vs `[0x750d38]+0x4c`);
  distribution 0×88, 1×19, 2×22, 4×8.
- **`index`**: 137 records carry 131 distinct values (0..130, gapless;
  ids 18 ×4, 23, 50, 113 duplicated — the same roof placed at several
  world positions).

After the 40-byte header, `fcn.00590c30` reads the **variable tail** — now
fully pinned (byte-exact, below):

```text
u32   countA           (0..5  across the file)
u32   countB           (0..13)
u32   A[countA]         diagonal cut lines: x+y thresholds (≤ w+h; max 3303)
u32   B[countB]         horizontal cut lines: y rows (≤ h; max 1485)
```

i.e. **both counts first, then both arrays** (`fcn.00590c30` reads
`[esi+8]`=countA, `[esi+0xc]`=countB, then `fread`s the two `u32` arrays;
`fcn.0058fe10`, called first, reads *nothing* from the file). So the full
record is `header(40) + countA + countB + countA·4 + countB·4` bytes.

**A/B semantics ✅ (resolved — they are not id lists):** they are
editor-placed, sorted **depth-band cut lines** slicing the roof's
`roofs.000` ChainedImage into `(countA+1)×(countB+1)` pieces in
`fcn.0058f1b0` — **A = diagonal `x+y` thresholds, B = horizontal `y`
rows**; an image piece at `(px,py)` belongs to band `(i,j)` iff
`A[i] ≤ px+py < A[i+1]` and `B[j] ≤ py < B[j+1]`, and each band is
queued with depth key `base + A[i+1]` (so different parts of one big
roof sort at different depths against the world). The appenders
`fcn.0058f030`/`fcn.0058f0b0` auto-prepend 0 and qsort — matching every
shipped array. Disambiguation: `roofs.000` holds only **131 images**, so
A (max 3303) cannot be an image index; the diagonal/row reading has zero
bound violations across all 137 records, and record 134 (A max 2945 >
width 2730) rules out a plain-x reading.

**Validation ✅** — parsing all 137 records this way consumes the file
**byte-exact (7188/7188)**. (The earlier fixed-48-byte guess fit record 0
only because its arrays are empty (`countA=countB=0` → a 48-byte record);
it desynced at record 4, the first roof with non-empty lists.) Note the
`+0x04` header field (`0xb`) is a roof **type**, *not* the tail count — the
tile-list counts are `countA`/`countB` in the tail; both the framing and
all field semantics are now complete.

## Runtime — "Roof Mode"

The roof manager is the global object **at `0x751680`** (RTTI
`RoofModule`; the loader `FUN_005916f0` is a `thiscall` on it — both call
sites do `mov ecx, 0x751680`, the address of the object itself, not a
pointer load; init.cpp `0x4a13dd`).

**Per-frame update — `fcn.0058faf0`** (called with `ecx=0x751680` from the
per-frame render fn `fcn.004a3fd0` at `0x4a4034`, gated on
`[0x65b3d0]+0x100`, [render-trace](../render-trace.md)): it reads the
**player agent** (`[0x658c04]`, [agent](../agent.md) — the player
*singleton*, distinct from the `[0x658c0c]` controller), adds half the
screen dims (`[0x64d9b8]/[0x64d9bc]`) to its position
(`+0x4a0/+0x4a4`), and consults the player's sector (via cell index
`+0x4c0` → `[0x658d50]+0xc`, with a `0x8000` comparison under a
sector-type==2 check). It then hit-tests a **10-entry `dh_house%d`
image-id table at `0x751650`** — lazily resolved on first run
(`sprintf("dh_house%d", 0..9)` → `fcn.0058dbf0` on the imagelist manager
`[0x751614]`, then the once-flag `[0x751678]` is set; that global is
*only* the init flag, not a data structure) — via `fcn.0058d710`, plus
the per-roof footprint rect test `fcn.0058fd70` (through helper
`fcn.0058fa30`). A hit flips the roof's **visibility state** (`[+0x18]`)
and arms a fade (`[+0x8]=0xfffffff1`, `[+0xc]=1`). So **"Roof Mode"** is a
per-frame *player-position → footprint/house-image hit-test →
hide-the-covering-roof (with fade)* pass using the `(x,y,w,h)` footprints
from `roofs.dat`. *(An earlier note that `fcn.005e5bda` "pages the roof
image" is retracted — `0x5e5bda` is MSVC `__security_check_cookie`, the
function's stack-cookie epilogue.)*

- The script verbs **`roofs #`** and **`force compiled roofs #`** drive
  loading / regeneration (the `force compiled` form mirrors the
  `set monster generation` precompute pattern, [monsters](../monsters.md)).
- **"Roof Mode"** / **"Roofs"** are the toggle labels.

## Status

- Files & loader ✅ — `roofs.000` (images) + `roofs.dat` (index, 137) +
  unshipped `roofs.txt` source; loader `FUN_005916f0` →
  `fcn.00590d30` (10-field header) + `fcn.00590c30` (variable tail).
- Record framing ✅ — `u32 count(137)` + variable-length records; count
  byte-exact.
- 40-byte header ✅ (complete — all roles engine-named via the
  `roofs.txt` format string `0x6206a8`): index (non-unique —
  placements), list selector, `offsetx/offsety` depth-sort anchor,
  `worldx/worldy` footprint origin (runtime rect test `fcn.0058fd70`),
  `worldheight` (world Z), `map` partition id, `width/height` (rec0
  `1374×575` cross-confirmed vs the roof imagelist).
- Variable tail ✅ — `{u32 countA; u32 countB; A[countA]; B[countB]}` (both
  counts first, then both `u32` arrays). **Byte-exact**: all 137 records
  parse and consume the file exactly (7188/7188).
- On-disk format ✅ **complete** — `roofs.dat` fully decoded byte-exact
  with every field role resolved, including the A/B tails (depth-band
  cut lines: A diagonal `x+y`, B horizontal `y`, banding in
  `fcn.0058f1b0`). No open semantics left.
- Roof-hide runtime ✅ (mechanism) — manager object at `0x751680`
  (`RoofModule`); per-frame update `fcn.0058faf0` (from render
  `fcn.004a3fd0` @ `0x4a4034`, gate `[0x65b3d0]+0x100`) reads the player
  `[0x658c04]`, hit-tests the lazily-built `dh_house%d` table `0x751650`
  (once-flag `[0x751678]`, resolver `fcn.0058dbf0` on `[0x751614]`, test
  `fcn.0058d710`) + the footprint rects (`fcn.0058fd70`/`fcn.0058fa30`),
  and arms the visibility/fade fields (`+0x18`, `+0x8`, `+0xc`).

## Citations

```text
div.exe:0x005916f0   FUN_005916f0   roofs.000 + roofs.dat loader (.\WORLD\roof.cpp; call site 0x4a13f4, else-branch dat\roofs.txt via fcn.005914f0).
div.exe:0x00590d30   per-roof header reader — 10 freads (mem offsets in the table) then fcn.00590c30 on [this+0x34].
div.exe:0x00590c30   roof variable tail reader — 2 counts + 2 u32 arrays (roof.cpp:65/70).
div.exe:0x00590250   free-and-reload guard — free list + u32 count + count×12-byte records (roof.cpp:498).
div.exe:0x0058faf0   per-frame roof update — player [0x658c04] vs footprints + dh_house table; visibility/fade writes.
div.exe:0x0058fd70   point-in-footprint test — origin [+0]/[+4], extent [+0x2c]/[+0x30].
div.exe:0x0058fa30   hide/show helper (re-reads player pos, screen-rect test).
data:0x751680        roof manager object (RTTI RoofModule) · 0x751650 dh_house%d id table (10) · [0x751678] its init flag · [0x751614] imagelist manager.
data:static\imagelists\roofs.dat   u32 count(137) + variable RoofRecords (header+tile lists).
```
