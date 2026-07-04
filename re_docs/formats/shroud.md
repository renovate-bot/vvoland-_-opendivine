# `shroud.tmp` / `shroud.x<n>` — fog of war

The exploration **shroud** (fog of war): a per-cell map of how much of
the world the player has uncovered. `shroud.tmp` is the active buffer;
`dynamic\shroud.x<n>` is the per-map persisted copy. The runtime
updater is logged as `"DBG Update Shroud"`, and a debug toggle
(`"no fog"`, *"disables fog rendering"*) turns it off.

Not to be confused with **object visibility** (`sb_invisible`,
`"make object invisible for agent"`, `"set object $ visibility #"`),
which hides individual objects/NPCs — a separate system from the
terrain shroud.

## Layout

```text
+0x00   u8    version (= 1)
+0x01   u8    level[width × height]      — one byte per grid point
```

The shipped `shroud.tmp` is **525826** bytes = `1 + 513 × 1025`. The
grid is the **cell-corner grid**: `(512+1) × (1024+1)` = one point per
corner of the `512 × 1024` world cell grid (see
`internal/game/types.go`). So `1 + 513×1025 = 525826` checks out
exactly. (A `512 × 1027` reading with a 2-byte header also factors;
the single `version = 1` byte favours `513 × 1025`.)

## Shroud levels

Each byte is a shroud/fog level. Observed values and frequencies across
the shipped buffer:

| Value | Count | Meaning |
|---|---:|---|
| `0xbf` | 383 944 | unexplored (the default fill — most of the map) |
| `0x2f` | 123 639 | fully revealed |
| `0xff` | 9 170 | fog gradient |
| `0x3f` | 8 816 | fog gradient |
| `0x7f` | 256 | fog gradient |

So the high nibble carries the reveal level (`0xb` unexplored →
`0x2` clear, with `0x3/0x7/0xf` as the dawn/dusk-of-vision gradient at
the boundary of explored areas), and the low nibble is a constant `0xf`.
The renderer darkens each cell by its level; the boundary gradient
gives the soft fog edge.

## Reveal logic — region flood (`FUN_0053f810` / `FUN_0053f780`)

The gameplay reveal is **region-based, not line-of-sight**. Each tick
`FUN_0053f810` walks the live agents (`CAgentManager [0x658d50]`) and, per
agent, `FUN_0053f780` resolves which **region** the agent currently stands
in via the point→region lookup `FUN_0058d620` (region table `[0x751628]`,
16-byte records) and caches it in the agent's shroud-state `+0x04`. **Only
when that region changes** does it reveal: `FUN_0053f2b0` walks the new
region's bounding box and, for each cell inside the region polygon
(`FUN_004ff040` point-in-region test), sets that cell's shroud byte to a
revealed level (`or byte[cell], 0x9f` / `and …, 0xef`).

So walking into a room/area **floods the whole region clear at once** —
there is no per-cell sight raycast (`FUN_0056fbc0`, the LOS used by
[combat perception](../npc-ai.md), has no caller here). The
`"line of sight on/off"` console toggle is a *rendering/debug* switch, not
the shroud updater. The `ShroudV1.0` region table (`region.007`,
[`region.md`](region.md)) overlays always-/never-revealed zones on top.

## Related

- `region.007` is the `ShroudV1.0` region table
  ([`region.md`](region.md)) — areas with special shroud behaviour
  (e.g. always-revealed or never-revealed zones).

## Status

- Layout ✅ — `u8 version` + a `513 × 1025` (cell-corner) byte grid;
  size `1 + 513×1025 = 525826` verified.
- Level encoding ✅ — high nibble = reveal level (`0xb` unexplored →
  `0x2` clear, `0x3/0x7/0xf` gradient); low nibble constant `0xf`.
- Grid orientation 🟡 — `513 × 1025` vs `1025 × 513` (row/col order) not
  yet confirmed against the loader.
- Update logic ✅ (corrected) — the reveal is **region flood**, not
  line-of-sight: `FUN_0053f810` per agent resolves its region
  (`FUN_0058d620`, table `[0x751628]`), caches it at agent shroud-state
  `+0x04`, and on a region change floods the new region
  (`FUN_0053f2b0`: bbox × point-in-region `FUN_004ff040` → `or 0x9f`).
  The earlier "line-of-sight based" note was wrong (the LOS raycast
  `FUN_0056fbc0` is not called from the shroud code). `"DBG Update
  Shroud"` (`FUN_00489cf0`) is still just a debug buffer dump.
- Reveal byte op 🟡 (mechanism located) — `FUN_0053f2b0` computes the
  region bbox (`fcn.004ff060`), converts its corners to **64-cell block
  coords** (the four `cdq; and edx,0x3f; add eax,edx; sar eax,6` blocks at
  `0x53f2f6…0x53f334` are the standard **signed ÷64**, *not* a dispatch —
  correcting an earlier misread), then walks the bbox cells, **point-in-
  region tests** each (`fcn.004ff040`, the `call edx` at `0x53f346`), and on
  a cell inside applies the byte write: `and byte [cell], 0xef` (`0x53f3c8`,
  clears bit 4) on one branch and `or byte [cell], 0x9f` (`0x53f3cd`, sets
  bits `7,4,3,2,1,0` → low nibble forced to `0xf`) at the merge. The exact
  two-level→byte mapping (these two ops do not by themselves reproduce the
  observed `0xbf → 0x2f`, which clears `0x90`) is the remaining detail; the
  flood structure (bbox → ÷64 block coords → point-in-region → byte op) is
  pinned.
