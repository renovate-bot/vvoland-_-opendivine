# Wall elements

Divine Divinity builds rooms from **wall elements** — a construction
category distinct from catalogue objects ([`formats/objects.md`](formats/objects.md)).
A wall element is classified by a **category** and an **inside/outside
facing**, which together drive how the engine renders and culls walls
(and which walls a doorway cuts through). Recovered from the editor's
wall-element status readout **`FUN_005c4f50`**.

## Category enum

`FUN_005c4f50` calls the element's category accessor `FUN_005b4a30`
and switches on the returned value:

| Value | Category | |
|---:|---|---|
| 0 | `normal` | a plain wall segment |
| 1 | `window` | a wall with a see-through opening |
| 2 | `door` | a wall with a doorway (the interactable door object hangs here) |
| 3 | `special` | special-cased segment |
| ≥4 / none | *none selected* | no valid element |

For each category the readout then appends an **inside / outside**
suffix, taken from the facing accessor `FUN_005b49dc` (returns `al`;
non-zero ⇒ *inside*, zero ⇒ *outside*). So the full classification is
the 4 categories × 2 facings, e.g. `door inside`, `normal outside`.

The readout format is `Current wallelement = %d(%d) <category>
<facing>`, where the two numbers are the element index and a secondary
value read from the editor's selection state (`[esi+0x18]`).

## Storage — a 2D wall grid (`manager+4`)

The category/facing accessors are not single-object getters: **both take two
indices** (`x`, `y`) and walk a **2D grid**, so wall elements are stored
per-cell in a grid owned by a manager, not in a standalone file. The
indexing (identical in `FUN_005b4a30` / `FUN_005b49dc`) is:

```text
grid      = *(manager + 4)                  ; base of the row array
row       = *(grid + x*0x14 + 0x10)          ; row[x].+0x10 → column array
cell      =   row  + y*0x14                   ; the 20-byte wall-element record
            category = *(int*) (cell + 0x10)  ; 0/1/2/3 (FUN_005b4a30)
            facing   = *(byte*)(cell + 0x0c)  ; inside/outside (FUN_005b49dc)
```

Both levels use a **20-byte (`0x14`) stride**: the outer row descriptor is
20 bytes with its column-array pointer at `+0x10`, and each inner cell is a
20-byte wall-element record. The whole `0x5b49xx … 0x5b4axx` block is the
**`CWallElement` grid accessor family** — uniform `[x*0x14][y*0x14]`
get/set methods over this grid, e.g.:

| Fn | Op | Field |
|---|---|---|
| `FUN_005b49dc` | get byte | `cell+0x0c` = facing |
| `FUN_005b496e` | **set** byte (`mov [cell+0xc], cl`) | `cell+0x0c` = facing |
| `FUN_005b4a30` | get dword | `cell+0x10` = category |
| `FUN_005b4aa6` / `FUN_005b4ab8` | get word | row-descriptor `+0xa` / `+0xc` |

Callers pass the element's own cell coordinates as the two indices — e.g.
`FUN_005c5060` pushes `[esi+0x5c]` / `[esi+0x64]` (the element's grid x/y)
before calling the category accessor. So a wall element *is* a 20-byte cell
in this grid; the minor word fields at `+0xa`/`+0xc` of the row level and
the cell's `+0..+0x0b` bytes are not all individually named yet.

## Citations

```text
div.exe:0x005c4f50   FUN_005c4f50   wall-element status readout — category + facing switch;
                                    source of the enum in this doc.
div.exe:0x005b4a30   FUN_005b4a30   CWallElement category accessor (2D [x][y] grid); cell+0x10
                                    (0=normal,1=window,2=door,3=special).
div.exe:0x005b49dc   FUN_005b49dc   CWallElement facing accessor; cell+0x0c byte (inside/outside).
div.exe:0x005b496e   FUN_005b496e   CWallElement facing setter — mov byte [cell+0xc], cl.
div.exe:0x005b4905.. FUN_005b49xx   CWallElement grid accessor family — uniform [x*0x14][y*0x14]
                                    get/set over the grid at *(manager+4).
```

## Roof rendering (`.\WORLD\roof.cpp`)

The companion to wall facing is the **roof system** (`.\WORLD\roof.cpp`,
the `0x58e…0x591` function cluster) — the iso "fade the roof when the
player walks inside" feature. Its pieces:

- **Data**: the roof sprites are an imagelist (`static\imagelists\roofs.000`
  + `roofs.dat`, [`formats/cmp.md`](formats/cmp.md)/[`imagelists`](formats/imagelists.md)),
  with `dat\roofs.txt` the definition table; per-map roof placement is
  loaded by **`FUN_00590c30`** (`fread` of `0x41`/`0x46`-byte roof
  structures) and persisted via the *"Load roofs"* / *"Save roofs"* paths.
- **Roof-at-cell lookup** **`FUN_00590020`** maps a world cell `(x,y)` to
  its roof (asserts `"Invalid roof found %d,%d"` on a bad cell) — called
  from the roof render/update `FUN_00590570` / `FUN_00590aa0`.
- **Fade-on-enter**: the render path queries the roof covering the
  player's cell and drops its visibility so the interior shows; this is
  gated by the wall-element **inside/outside facing** (`FUN_005b49dc`,
  above) so only the roof over an interior the player has entered fades.
  The exact alpha-ramp / fade-timer math is the remaining detail 🟡.

## Status

- Category enum ✅ — 0/1/2/3 = normal/window/door/special, confirmed
  from the switch and the matching readout strings.
- Inside/outside facing ✅ — boolean from `FUN_005b49dc`; exact polarity
  (which value prints "inside") is from the `test al,al`/`je` branch.
- Wall-element struct & storage ✅ (recovered) — wall elements are
  **20-byte (`0x14`) per-cell records in a 2D grid** at `*(manager+4)`,
  not a standalone file: `grid[x].+0x10` → column array, `column[y]` =
  the cell, with **facing @ `cell+0x0c`** (byte) and **category @
  `cell+0x10`** (dword). The `0x5b49xx…0x5b4axx` accessor family is the
  uniform `[x*0x14][y*0x14]` get/set interface (facing setter
  `FUN_005b496e`). Residual: the minor byte/word fields (cell `+0..+0x0b`,
  row-descriptor `+0xa`/`+0xc`) are not individually named, and the
  geometry/imagelist binding is still open 🟡.
- Relationship to doors 🟡 (narrowed) — a `door` cell (category 2) is a
  20-byte grid record like any other; the binding to the interactable
  door object ([`object-interaction.md`](object-interaction.md)) is via
  the element's cell coordinates (`[esi+0x5c]`/`[esi+0x64]`), but the
  object→cell lookup is not yet traced.
- Relationship to roof rendering ✅ (subsystem located) — `.\WORLD\roof.cpp`
  (`0x58e…0x591`): roof imagelist `roofs.000`/`roofs.dat` + `roofs.txt`,
  per-map loader `FUN_00590c30`, roof-at-cell lookup `FUN_00590020`
  ("Invalid roof found"), render/update `FUN_00590570`/`FUN_00590aa0`; the
  fade-on-enter queries the roof over the player's cell and is gated by the
  wall inside/outside facing (`FUN_005b49dc`). Exact fade alpha-ramp 🟡.
