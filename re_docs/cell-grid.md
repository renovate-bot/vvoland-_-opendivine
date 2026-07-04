# Spatial cell grid (`.\WORLD\Cell.cpp`)

The engine's **broad-phase spatial index**: a uniform grid that buckets
every object/agent by world position so "what is near here / in this
area / in this direction" can be answered without scanning all objects.
It feeds perception, area-of-effect magic, collision/range checks, and
spawn placement. Distinct from the per-map `world.x<n>` cell *grid file*
([`formats/world.md`](formats/world.md)) — this is the runtime query
structure built over the live object set.

## Construction (`fcn.00571fa0`, "Loading cell manager")

The cell-manager ctor (`.\WORLD\Cell.cpp`) allocates:

- a **`1024 × 1024` grid of 16-byte cells** — `0x1004000` bytes
  (`1024·1024·16` + a small header), `memset` to `0`. Each cell holds the
  head of a per-cell **bucket list** of the objects currently in it (plus
  per-cell bookkeeping in the 16 bytes).
- a **node/record pool** (`0xdffe4` bytes) for the bucket-list links.
- a 256-byte direction/quadrant **lookup table at `0x74cba0`**, `memset`
  to `0xff` (`-1`), used to step the grid by facing.

Cell addressing is `cell = base + (row·1024 + col)·16` — the `shl …, 0xa`
(`×1024`) row term recurs throughout the unit; `fcn.005706d0` is the
coordinate→cell helper (clamps to grid bounds and walks the addressed
cells).

## Query API

| Fn | Role |
|---|---|
| `fcn.005706d0` | **cell addressing** — map a world coord / rect to grid cells (`row·1024+col`, stride 16) |
| `fcn.005709e0` | **area query** — gather every object in a region |
| `fcn.005719f0` | **directional query** — a facing-cone / ray cell walk (uses `sin`/`cos`) |

`fcn.005709e0` (≈4 KB, the workhorse) walks the cells covering a
rectangular/radial region (stepping rows by `×1024`), reads each cell's
bucket, and **accumulates the hits into a scratch result list at
`0x74cca0`** (built with the list helpers `fcn.0056fd50` add / `fcn.0056fe20`
clear). Its callers are exactly the broad-phase consumers: agent
perception / targeting (`fcn.004274a0`, `fcn.00427920`), area spawn and
group placement (`fcn.00441250`, `fcn.00442c20`, `fcn.004453a0`), and
area-effect magic (`0x4dc…`). `fcn.005719f0` adds a **directional** variant
(the `0x74cba0` quadrant table + `sin`/`cos`) for facing-cone scans (e.g.
"who is in front of me").

So the cell grid is the shared "objects near X" service: an object is
bucketed into its cell and re-bucketed as it moves, and the area/direction
queries replace any O(n) scan over the global object set.

## Status

- Grid ✅ — `1024×1024`, 16-byte cells (`0x1004000` alloc, zeroed), node
  pool `0xdffe4`, direction table `0x74cba0` (`0xff`-init); ctor
  `fcn.00571fa0` (`.\WORLD\Cell.cpp`). Cell index `row·1024 + col`.
- Query API ✅ — addressing `fcn.005706d0`, **area query** `fcn.005709e0`
  (results → scratch list `0x74cca0` via `fcn.0056fd50`/`fcn.0056fe20`),
  **directional query** `fcn.005719f0` (`sin`/`cos` + `0x74cba0`).
- Consumers ✅ — agent perception/targeting (`0x427xxx`), area spawn/group
  placement (`0x441…`/`0x442…`/`0x445…`), AoE magic (`0x4dc…`).
- Cell-struct fields 🟡 — the 16 bytes per cell (list head + count/bounds/
  flags) and the exact grid-base global are not split field-by-field.

## Citations

```text
div.exe:0x00571fa0   cell-manager ctor — alloc 1024x1024x16 grid + node pool; ".\WORLD\Cell.cpp".
div.exe:0x005706d0   cell addressing — world coord/rect → grid cells (row*1024+col).
div.exe:0x005709e0   area query — gather objects in a region into scratch list 0x74cca0.
div.exe:0x005719f0   directional query — facing-cone / ray cell walk (sin/cos, table 0x74cba0).
div.exe:0x0056fd50   scratch-list add · 0x0056fe20 scratch-list clear.
div.exe:0x0074cba0   direction/quadrant lookup (0xff-init) · 0x0074cca0 query scratch list.
```
