# Pathfinding (A*)

How agents route around obstacles. Divine Divinity uses **A\*** search
(`.\SPRANIM\path.cpp`) — `"Out of A* nodes"`, `"Not enough memory for
path"`, `"\nHas path"` / `"\nNo path"`. This is the engine-faithful
replacement for the straight-line click-to-walk in `internal/game/`
(whose `Update` notes *"Pathfinding TBD"*).

## Search grid

A* runs on a **half-cell node grid** — twice the resolution of the
64-px world cell grid (`512 × 1024` cells, see `internal/game/types.go`).
The core `FUN_005709e0` bounds node coordinates to:

```text
x ∈ [3, 0x3fe]   (3 .. 1022)     ≈ 2 × 511
y ∈ [3, 0x7fa]   (3 .. 2042)     ≈ 2 × 1021
```

So each 64-px cell is **2 × 2 nodes** (32-px node spacing), giving
finer paths than cell granularity while keeping the grid bounded.
Search is **16-directional** (correcting the earlier "8-dir"): the
neighbour-expansion loop runs `dir = 0..15` (`inc edi; cmp edi,0x10; jl`
@`0x57182d`) over the same **DIR16 iso step table `0x654e50`**
(`{du,dv}×16`) the movement stepper uses — 8 primary facings + 8 half-
step intermediates ([collide.md](formats/collide.md) has the table).

### Node record & passability

Each node is an **8-byte record** (`[base + index*8]`), and the grid is
addressed row-major with a **1024-node row stride** (`shl idx, 0xa`).
Per node:

```text
word flags    // [node-6]; a blocker/special bit (0x400) is tested and
              //   gates the neighbour out if set
byte height0  // [node-2]   the cell's two corner heights
byte height1  // [node-1]
```

Passability is **height-aware** — a neighbour is rejected when its
height delta from the current node exceeds **`0x50` (80) units**:

```text
delta = (height0 + height1) − current_height
if (delta > 0x50) reject neighbour          // too steep to step/climb
```

So this is slope-limited A*: agents cannot path up cliffs or off
ledges, only across steps ≤ 80 units. (The earlier "`cmp …,0x50`" in
the loop is this climb check, **not** a move cost.)

### Cell-flags word (the full bit map)

The flags `word` at **`cell+2`** is the one structure every navigation
consumer reads; they differ only in which bits they test:

| Bits | Meaning | Consumer |
|---|---|---|
| `0x01 \| 0x02 \| 0x10` (mask `0x13`) | **movement blocker** — cell is not walkable | walkability `FUN_0056f3c0` |
| `0x04 \| 0x08` (mask `0x0c`) | **sight blocker** — opaque to line-of-sight | LOS `FUN_0056fbc0` |
| `0x400` | **path blocker** — triggers the climb-height gate (Δ height > 80 ⇒ rejected) | A* core + walkability |

So a wall you can neither see nor walk through sets all of `0x1f`-ish +
the LOS bits; a low fence might block movement (`0x13`) but not sight
(`0x0c` clear); a ledge sets `0x400` so it is gated by the height delta
rather than hard-blocked.

### Direction-aware walkability (`FUN_0056f3c0`)

The local stepper (and combat/interaction reach tests, ~17 callers)
validate a move with `FUN_0056f3c0(worldmap, cellX, cellY, dir, mask)`.
Beyond the destination cell it **corner-checks** the cells the move
would clip past — a 16-case switch on `dir` (one per
[`[0x654e50]`](#local-movement-step-fun_0040f0a0) direction) reads the
relevant neighbour cells (offsets like `−0x2006` = one row up, the
`0x2000` = `1024*8` row stride) so diagonal steps cannot cut a corner
through a wall. Each examined cell is rejected on the `mask` blocker
bits, and on `0x400` it applies the same `> 0x50` (80) climb gate as A*.
Movement passes `mask = 0x13`; the bounds gate requires
`cellX,cellY ≥ 4` and the iso extents `(x−y)*32 < 0x7f00`, `y*32 <
0xff00`.

## Core (`FUN_005709e0`)

```text
- bounds-check the node (x/y in the navigable range above)
- expand 16 neighbours (DIR16 table 0x654e50), skipping blocked cells,
  running the same per-direction corner/pass-through walkability chain
  as fcn.0056f3c0 inline (switch 0x5719a8)
- g-cost: g(node) = g(parent) + STEP[dir], a fixed 16-entry i32 table at
  0x61ddb8 = {14,37,22,32, 10,14,10,22, 14,36,22,31, 10,14,10,22} ≈
  round(10·|DIR16[dir]|) (10 for the cardinals, 14 for √2, 22 for √5,
  31–32 for √10, 36–37 for √13; slight L/R rounding asymmetry). Relax
  on g < existing (0x5716bb).
- h-cost: h(node) = 10·octdist(node, goal) (octdist = fcn.0040ecb0,
  1.0390625·max + 0.3984375·min — the weighted octagonal metric; ×10 via
  lea+add). g and h share the ×10 scale so f = g + h is commensurate.
- climb: the >0x50 (80) height delta is a HARD neighbour rejection, not a
  cost term.
- draw nodes from a fixed pool — exhausting it logs "Out of A* nodes"
  and fails ("Not enough memory for path")
- on success, emit a waypoint list; the agent records "Has path"
  (its path state is printed by FUN_0042d790, the agent dumper)
```

The search is **budget-bounded**: the script/console command
`"pathfinding power #"` sets how hard A* may search before giving up —
matching the fixed node pool. A path that can't be found within budget
yields `"No path"` and the agent falls back (or stops).

Collision uses the same blocker data the renderer/movement already use:
the per-object collision cubes ([`formats/collide.md`](formats/collide.md))
rasterised into the cell grid (the `colliders` / `colliderGrid` in the
Go port), plus closed doors ([`object-interaction.md`](object-interaction.md)).

## Per-frame move interpolation (`FUN_00427d30`, `CNpc::virtual_16`)

The stepper picks a destination cell; **the agent then slides there over
several frames** rather than teleporting. The per-frame mover
`FUN_00427d30` (the `CNpc` move/animate virtual) does linear
interpolation each tick:

```text
if Walkcount (+0x278) > 0:
    Fx (+0x0c) += CellDx (+0x22c)      // accumulate sub-pixel position
    Fy (+0x10) += CellDy (+0x230)      //   by the per-tick velocity
    X (+0x04) = ftol(Fx);  Y (+0x08) = ftol(Fy)   // integer screen pos (FUN_005E5D40)
    height (+0x14) += (+0x18)
    Walkcount-- ;  arrived when it reaches 0
```

So **`Walkcount` (`agent+0x278`) is the count of remaining sub-steps to
the destination** (it counts *down*, reset on each new move /
`CurrentAction` change by the action-selector `FUN_00427180`), and
**`CellDx`/`CellDy` (`+0x22c`/`+0x230`) are the per-tick position
velocity** — confirming the `+0x22c..` "Cell*" block ([`agent.md`](agent.md))
is the move vector, not just a destination. `Fx`/`Fy` (`+0x0c`/`+0x10`)
are the accumulating sub-pixel position whose integer truncation is the
rendered `X`/`Y` (`+0x04`/`+0x08`). The animation frame advances off the
same `Walkcount`/`Walkspeed` pacing ([`animation.md`](animation.md)), so
one counter drives both the slide and the walk-cycle frame. Movement is
gated by a freeze flag (`+0x2e0`) and a scene-sync check (`+0x2e4`).

**On arrival (`Walkcount == 0`).** The other half of `FUN_00427d30`
projects the agent's iso position back to a **cell** — `cell = (X + Y +
midpoint offsets) >> 5` (the same `/32` cell scale) — and compares it to
the **destination cell `CellDestinationX`/`Y` (`+0x234`/`+0x238`)**,
confirming that the second half of the `+0x22c` "Cell*" block is the
target cell. If it has **not** reached the destination it re-arms the next
sub-step toward it via `FUN_004273e0` (which recomputes the velocity +
`Walkcount` for the next leg); once at the destination it advances the
`agentscript` program counter (`+0x2ba += 2`, the per-line stride,
[`npc-ai.md`](npc-ai.md)) to the next behaviour statement and clears the
pending-move flag (`+0x220 & 0x1000`). So one tick of `FUN_00427d30` is
**either** one interpolation step **or**, at a cell boundary, the
re-path / next-script-line decision — the bridge from a finished move
back into A*/the stepper or the behaviour script.

## Line of sight (`FUN_0056fbc0`)

The same cell grid backs the **line-of-sight raycast** used by NPC
perception ([`npc-ai.md`](npc-ai.md)) and many interaction checks (~14
callers). It is a textbook **Bresenham** walk between two world points:

```text
cell = world >> 5                       // 32 world-units per cell (signed-correct ÷32)
grid = [worldmap+8]   (worldmap = [0x74eca0])
index = y*1024 + x    (shl y, 0xa)      // same 1024 row stride as A*
for each cell stepped from start→end (major axis = max(|dx|,|dy|)):
    if [grid + index*8 + 2] & 0x0c: return 0   // a sight-blocker → no LOS
return 1                                        // clear
```

So LOS reads the **low byte of the per-cell flags word at `cell+2`** and
treats bits `0x4 | 0x8` as opaque (walls / sight blockers). This is a
different bit from the A* **path-blocker `0x400`** (the high byte of the
same flags word) — geometry can block movement without blocking sight,
or vice versa. The walk shares the grid base, the `>>5` world→cell
scale, and the `*1024` row stride with the A* core above; it does **not**
consult the height bytes (LOS is purely 2-D over the block bits).

## Local movement step (`FUN_0040f0a0`)

A* gives a waypoint list, but the actual per-frame stepping toward the
next waypoint (or a nearby target) is a **greedy 16-direction descent**,
`FUN_0040f0a0` (the move helper `agentbehaviour.cpp` calls,
[`npc-ai.md`](npc-ai.md)). It works in cell space (`world >> 5`, the same
32-unit cells) from the agent position `X = [agent+0x1c]+[agent+4]`,
`Y = [agent+0x20]+[agent+8]`:

```text
cur   = agent cell;  best = octdist(cur, target)
for d in 0..15:                                  // 16 iso directions, [0x654e50]
    n = cur + DIR[d]                             // DIR[d] = {dx,dy}, 8-byte entries
    s = octdist(n, target)
    if s <= best and walkable(worldmap[0x74eca0], n.x, n.y, d, 0x13):
        best = s;  chosen = d
if chosen valid: face/walk DIR[chosen] (CurrentAction[+0x216]=1), via [0x659218]
```

**Direction table `[0x654e50]`** — 16 `{i32 dx, i32 dy}` **isometric**
step vectors (not square 8-way): the ratios `1:0, 3:1, 2:1, 3:2, 1:1`
and their reflections approximate the 16 screen facings of the iso view.
This is the same table the perception look-ray uses ([`npc-ai.md`](npc-ai.md)).

**Octagonal distance `FUN_0040ecb0`** — a fixed-point Euclidean
approximation over `max = max(|dx|,|dy|)`, `min = min(|dx|,|dy|)`:

```text
dist = max + max>>5 + max>>7 + min>>2 + min>>3 + min>>6 + min>>7
     ≈ 1.04·max + 0.40·min        // tuned so a pure diagonal ≈ √2·max
```

Walkability per candidate cell is `FUN_0056f3c0(worldmap, x, y, dir,
0x13)` — the same `[0x74eca0]` grid the A* core and LOS use, with a
`0x13` flags mask (so movement, sight, and pathing each read their own
bit subset of the shared cell flags). So the **global** planner (A*) and
the **local** stepper share the grid, the `>>5` scale, and the distance
metric; they differ only in scope.

## Relation to the Go port

The current engine walks a straight line to the click target and
cancels on a blocker (`playerBlocked`). To match the original it should:

1. build the half-cell node grid from the collision grid,
2. run 16-directional A* (`FUN_005709e0`) from the agent to the target,
3. follow the resulting waypoint list,
4. respect a node budget (the `pathfinding power` analogue).

An open door (collider disabled, [`object-interaction.md`](object-interaction.md))
becomes passable to the search automatically, since it shares the
collision grid.

## Citations

```text
div.exe:0x005709e0   FUN_005709e0   A* core — half-cell grid, 16-dir (DIR16 0x654e50), g=STEP[0x61ddb8], h=10·octdist, node pool.
div.exe:0x0042d790   FUN_0042d790   agent path state ("Has path" / "No path").
div.exe:0x005706d0   FUN_005706d0   A* helper (open/closed set or node alloc).
div.exe:0x0056fe20   FUN_0056fe20   A* helper (grid / neighbour access).
div.exe:0x0056fd50   FUN_0056fd50   A* helper.
div.exe:0x00427d30   FUN_00427d30   CNpc per-frame move interpolation (Fx/Fy += CellDx/CellDy, Walkcount--).
div.exe:0x00427180   FUN_00427180   action-state selector (sets CurrentAction, resets Walkcount).
div.exe:0x004273e0   FUN_004273e0   per-leg velocity/Walkcount setup (CellDx/CellDy vs step count).
div.exe:0x0056fbc0   FUN_0056fbc0   Bresenham line-of-sight raycast over the cell grid.
div.exe:0x0074eca0   DAT_0074eca0   worldmap object; [+8] = the cell grid base.
div.exe:0x0040f0a0   FUN_0040f0a0   greedy 16-direction local movement step.
div.exe:0x0040ecb0   FUN_0040ecb0   octagonal distance (≈1.04·max + 0.40·min).
div.exe:0x0056f3c0   FUN_0056f3c0   per-cell walkability test (worldmap, flags mask 0x13).
div.exe:0x00654e50   DAT_00654e50   16 × {i32 dx, i32 dy} isometric direction table.
```

## Status

- Algorithm ✅ — A* on a half-cell (32-px) node grid, 16-directional,
  fixed node pool; bounds `[3,1022] × [3,2042]`.
- Failure modes ✅ — `Out of A* nodes` / `Not enough memory for path` →
  `No path`; budget set by `pathfinding power`.
- Result ✅ — waypoint list; agent path state `Has path` / `No path`.
- Node record & passability ✅ — 8-byte nodes, 1024 row stride; a flags
  word at `cell+2` and two height bytes; A* rejects neighbours with the
  path-blocker bit `0x400` (flags high byte) or a height delta exceeding
  `0x50` (80) — slope-limited search.
- Line of sight ✅ — `FUN_0056fbc0` is a Bresenham raycast over the same
  grid (`world>>5` cells, `*1024` stride) testing the **flags low byte
  `0x4|0x8`** as opaque — a *distinct* bit from the `0x400` path-blocker,
  so sight and movement occlusion are independent. Feeds NPC perception
  ([`npc-ai.md`](npc-ai.md)); ignores heights (purely 2-D).
- Local movement step ✅ — `FUN_0040f0a0` is a greedy 16-direction
  descent: from the agent cell it picks the passable neighbour
  (`FUN_0056f3c0`, mask `0x13`) that most reduces the octagonal distance
  (`FUN_0040ecb0`, `≈1.04·max + 0.40·min`) to the target, using the
  isometric `{dx,dy}` table `[0x654e50]`. Shares the grid/`>>5`/distance
  with A*; it is the per-frame stepper, A* the global planner.
- Cell-flags word ✅ — fully mapped at `cell+2`: movement blockers
  `0x01|0x02|0x10` (walkability mask `0x13`), sight blockers `0x04|0x08`
  (LOS), and `0x400` path-blocker that gates on the `>80` climb delta.
  Walkability `FUN_0056f3c0` is direction-aware (16-case corner-check so
  diagonals can't clip walls). The exact role of each individual bit
  within `0x13` (vs the data source that sets them) is not split out 🟡.
- Cost model ✅ (characterised) — costs are **not** the classic fixed
  `10`/`14` integer constants: the score updates are register-derived
  (`add edx,eax` / `add edx,esi`, with a doubling `add eax,eax`), i.e.
  the g-cost accumulates from the **actual node displacement plus the
  height delta**, and the heuristic is computed from the real
  coordinate difference to the goal — a displacement-weighted cost, not
  a tabled per-step constant. The hard per-step gate is the `0x50` (80)
  climb limit, checked once per direction. (The exact weighting
  coefficients are buried in the register math; reimplementations can
  approximate with Euclidean/Chebyshev distance + a height penalty
  capped at 80 — behaviourally faithful.)
- Helper roles ✅ — `FUN_005706d0` is the node-validity / grid-index
  helper (it re-checks the `[3,0x3fe]×[3,0x7fa]` bounds); `FUN_0056fe20`
  is the larger neighbour-expansion (it repeats the `0x50` height
  gate). **Node pool / open-set identified:** it is **not** a heap
  open-list but a **flat per-cell node-state grid** at the globals
  `[0x654e50]` / `[0x654e54]` — `FUN_0056fd50` indexes it with a
  `shl …,4` (×16 stride, so 16-byte node records per cell) and compares the
  `+0/+4` fields (the per-cell A* state: cost / parent / set-membership).
  `FUN_0056d1d0` is a tiny coordinate clamp (`cmp eax,0x7fff` → ≤32767). So
  the search keeps per-cell node state in a fixed grid keyed by cell index,
  not a dynamically-allocated open-list.
- Dynamic blockers ✅ — **the A\* pathfinder does *not* treat moving agents
  as obstacles**: the walkability test `FUN_0056f3c0` reads only the static
  cell-grid blocker bits (the `0x13` movement mask, the `0x400` path-blocker
  bit, the `0x50`/80 climb-height gate) and never consults the agent
  manager `[0x658bf0]` or any agent list. So paths are planned over static
  world collision alone. Agent↔agent avoidance is instead **reactive, at the
  movement/stepping layer**: as the agent slides toward its next waypoint,
  the per-step collision (`FUN_00440320` → proximity `FUN_00415120` → cube
  cell-query `FUN_00571df0`, [`formats/collide.md`](formats/collide.md))
  gathers nearby agents' collision cubes and stops/slides the mover on
  contact. Two agents therefore do not path around each other — the mover
  halts or re-steps when it bumps one. (This is why crowded NPCs jostle
  rather than detour.)
