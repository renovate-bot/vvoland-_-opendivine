# `global\eggs.000` — monster spawn points

The world's **egg** table: every monster spawn point in the game. An
egg names a monster kind and a world position; at runtime it hatches
into a monster agent that records its origin in the agent `Source egg`
field (`agent+0x274`, see [`../agent.md`](../agent.md)). Background on
the spawn model is in [`../monsters.md`](../monsters.md).

Loaded by the map-load orchestrator `FUN_0057c650` (which reads
`global\eggs.000`).

All integers little-endian, signed (i32).

## File layout

```text
+0x00   u32   count                 — number of eggs
+0x04   Egg[count]                  — fixed 92-byte (0x5c) records
```

The shipped file: `count = 8861`, and `4 + 8861 × 92 = 815216` =
the exact file size. ✅

## Egg record (92 bytes)

Field roles below are derived from the value distributions across all
8861 shipped eggs (offsets within one record):

```text
struct Egg {                 // 0x5c
    i32  x;            // +0x00 — world X pixel   (observed 1090..31704, fits world 32768)
    i32  y;            // +0x04 — world Y pixel   (observed 863..60788, fits world 65536)
    i32  monster_kind; // +0x08 — monster type id (188 distinct values; the spawn's creature)
    i32  flags;        // +0x0c — flag word (NOT a count): bit 0 = "ready", cleared by the hatch
                       //          (FUN_0043ccd0: `and …,0xfffffffe`) after the one-shot spawn;
                       //          bits 2/3 tested during hatch. (The 1..11 values seen in the
                       //          file are flag-bit combinations, not a pack size.)
    i32  region;       // +0x10 — region id, compared by the hatch for the region gate
    i32  spawn[3];     // +0x14/+0x18/+0x1c — passed to the agent constructor (fcn.005068f0) as
                       //          per-spawn params (behaviour/group/variant; small signed
                       //          ranges +0x14: -1..5, +0x18: -1..16, +0x1c: -14..26, mostly -1
                       //          across the 8861 shipped eggs) — NOT world coords, and NOT a
                       //          catalogue id (the creature is monster_kind at +0x08, 4..373)
    i32  egg_id;       // +0x20 — unique sequential id 0..count-1; the value an agent's
                       //          `Source egg` points back to
    i32  category;     // +0x24 — egg category 0..4 (e.g. manual vs generated class)
    u8   runtime[0x34];// +0x28 — zero on disk; runtime-only state filled on hatch
                       //          (spawned-agent handle, respawn timer, …)
};
```

The trailing 52 bytes are all-zero in the file — the loader uses them
for live state once the egg hatches.

## Verification

- File size is exactly `4 + count·92`.
- `x`/`y` ranges sit inside the engine world bounds (`32768 × 65536`).
- `egg_id` is a dense `0..8860` permutation — a unique key, matching its
  use as the agent `Source egg` back-reference.
- `monster_kind`'s 188 distinct values with frequency-like counts
  (top types appear ~200–320×) match the per-type **Frequency** in the
  monster balancing report (`FUN_0043d9e0`, see `monsters.md`).

## Egg model vocabulary (`.\AGENTS\eggman.cpp`)

The egg subsystem is `.\AGENTS\eggman.cpp` (boot banner *"Loading egg
manager"*). Two consumer sites confirm the conceptual fields behind the
record's link/enum slots:

- **Editor readout** `fcn.0049c7d0` prints `Egg editor - Class %s - Group
  %d (%d eggs in it) - Behaviour %d`, so eggs are organised by a named
  **Class** (resolved through the class array at `[0x658d50]+0x28`), a
  numeric **Group**, and a **Behaviour** id. These are the meanings behind
  the record's `ref[4]` link slots (`+0x10..+0x1c`) — class / group /
  behaviour references — and the `set egg group` script command and
  `Unknown alignment in set egg group (%s)` error operate on the same Group
  axis.
- **Manual vs generated** is a real, tracked split (the egg-stats dump
  `fcn.0043d530` / the balancing report `fcn.0043d9e0` print *Amount/Total
  experience of manual eggs* vs *generated eggs*). At runtime the
  distinction is carried in an egg **flag word** (`test byte [egg], 8` —
  bit 3) that feeds the **shared bitmask→category resolver `fcn.00591920`**
  (the same `0x655a98`-table engine used for treasure drop-kind and item
  category, cf. [`treasure.md`](treasure.md) / [`../items.md`](../items.md)).
  So the on-disk `category` (`+0x24`, 0..4) is this manual/generated-class
  enum, consumed through the common category resolver rather than a
  bespoke egg switch.

(The precise file-offset → Class/Group/Behaviour binding is read by the
per-map egg loader, not the boot manager init; eggs load at map time via
`FUN_0057c650`, which dispatches `global\eggs.000` to the egg manager.)

## Citations

```text
div.exe:0x0057c650   FUN_0057c650   map/save load orchestrator; opens global\eggs.000.
div.exe:0x0049c7d0   FUN_0049c7d0   egg editor readout — "Class %s - Group %d - Behaviour %d".
div.exe:0x0043d530   FUN_0043d530   egg-stats dump — manual vs generated egg totals.
div.exe:0x0043d9e0   FUN_0043d9e0   monster/egg balancing report (manual vs generated).
div.exe:0x00591920   FUN_00591920   shared bitmask→category resolver (table 0x655a98); egg flag bit 3.
```

## Status

- File layout ✅ — `u32 count` + `count × 92-byte` records; size checks
  out exactly.
- Core fields ✅ (reconciled against the hatch `FUN_0043ccd0`) — `x`
  (`+0`), `y` (`+0x04`), `monster_kind` (`+0x08`), `egg_id` (`+0x20`),
  `category`/map (`+0x24`). **Correction:** `+0x0c` is a **flag word**, not
  an `amount`/pack-count — the hatch clears its bit 0 ("ready") after the
  one-shot spawn (`and …,0xfffffffe`) and tests bits 2/3; the 1..11 file
  values are flag combinations. The spawn position the hatch uses is
  `x`/`y` at `+0`/`+0x04` (fed to the walkability check), confirming those
  are the world coords.
- Link / spawn slots ✅ (reconciled) — `+0x10` = **region** (the hatch's
  region gate), and `+0x14/+0x18/+0x1c` are the three **agent-creation
  params** passed to `fcn.005068f0` (the Class/Group/Behaviour references
  from the egg-editor readout `fcn.0049c7d0`; small signed `-1..26`). These
  are *not* world coords (an earlier `monsters.md` note mislabelled them).
  **Dataflow pinned:** the hatch `FUN_0043ccd0` passes them (with
  `monster_kind` `+0x08` and `region` `+0x10`) as **args 4/5/6 of the
  agent-create `fcn.005068f0`** (`mgr [0x658bf0]`), in egg order
  `+0x14`→arg4, `+0x18`→arg5, `+0x1c`→arg6. `fcn.005068f0` is a thin wrapper
  that **forwards** them (it re-reads them at `arg_34h`/`arg_38h`/`arg_3ch`
  and passes them to the deeper agent constructor **`fcn.00424ea0`** at
  `0x50694a`), so the exact param→{class,group,behaviour} field assignment
  needs tracing `fcn.00424ea0`'s parameter stores (the named next-hop) 🟡 —
  but the three slots are confirmed as the creation-time class/group/
  behaviour inputs, not coords. **Negative result narrowing it:**
  `fcn.00424ea0` sets the agent's **attribute block** (`+0x80`/`+0x84`/`+0x88`
  = Str/Dex/Int) and **Name** (`+0x21c`) from *other* args (the
  template/"monsters authored" path, [`../stats.md`](../stats.md)), so the
  egg's `+0x14/+0x18/+0x1c` are **not** the attributes — confirming they are
  the group/behaviour/class refs (the small `-1..26` values), the exact
  field offsets being the only open part.
- `category` (`+0x24`, 0..4) ✅ (meaning confirmed) — the **manual vs
  generated** class, a real tracked split (egg-stats `fcn.0043d530`,
  balancing report `fcn.0043d9e0`); the runtime egg flag word's bit 3
  (`test byte [egg], 8`) feeds the shared category resolver `fcn.00591920`
  (table `0x655a98`), the same engine as treasure drop-kind / item
  category. Exact 0..4 sub-value labels still 🟡.
- Runtime tail (`+0x28..`) ❓ — zero on disk; the hatch fills it (agent
  handle, timers) — see the hatch logic open item in `monsters.md`.
