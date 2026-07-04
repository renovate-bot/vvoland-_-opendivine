# `dat\magic.cmp` — compiled magic parameter table

A compiled table of **96 magic/skill parameter records**. The count
matches the spell/skill taxonomy size in
[`../skills-magic.md`](../skills-magic.md), so this is the numeric
parameter table behind the magic effects — the compiled output of a
text `magic.dat` source (the `.cmp` = "compiled" convention, see
[`cmp.md`](cmp.md)).

All integers little-endian, signed (i32).

## Format

```text
+0x00   u32   count                  — 96
        Record[count]:               — 7 × i32, 28 bytes each
            i32 f[7]
```

`4 + 96 × 28 = 2692` = the exact file size. ✅

The loader `FUN_004c7060` confirms this: it reads the 4-byte count, then
slurps `count × 0x1c`-byte records straight into the magic manager at
`[manager+0x0c]`. The loader assigns no field names — it is a thin
copy — which is why the column semantics live in the (scattered)
consumers, not here.

**Manager identity & record access (verified):** the `manager` is the
global **magic-effect manager `div.exe:0x658bf4`** (the same singleton the
`CMagic*` `Cast` path registers with, [`../skills-magic.md`](../skills-magic.md));
the loader is invoked at boot as `fcn.004c7060` with `ecx = [0x658bf4]`
(`fcn.00499530 @ 0x4995e1`). The manager keeps a parallel **id → dense
index** lookup at `+0x04`/`+0x08` resolved by **`fcn.004c7170`** (a small
find-by-id with ~20 callers across the `CMagic`/effect cluster), so a
consumer reaches a parameter record as
`rec = [0x658bf4]+0x0c + fcn.004c7170([0x658bf4], id)·0x1c`. (The access
*path* is pinned; the column *roles* are recovered from the grouped data
structure below — a per-spell banded curve — leaving only the group→spell
naming 🟡.)

*(Consumer-trace outcome, so it is not re-tread: the effect functions that
call `fcn.004c7170` — the shared `Cast` virtual `fcn.00539820`, the
`0x4bef…` damage-effect cluster — use its **return as a slot/index** (e.g.
`Cast` makes the `+0x118` expiry `idx·1e7+2e6`), not as a base for direct
`[rec+foff]` column reads. So the per-column reads do not appear at these
call sites; pinning the roles needs either the unshipped `magic.dat` text
source or dynamic tracing, not another div.exe sweep of the effect
cluster. The one `[0x658bf4]` consumer in the GUI/plate range
(`fcn.00505bc0`) was also checked — it is **effect screen-positioning**
(coordinate-range + projection math on a large effect object, `+0x4a0`/
`+0x4a4` screen offsets), **not** a record-field display, so the spell-info
GUI path does not surface the columns either. Effect cluster, `Cast`, and
GUI consumer classes are now all exhausted.)*

## Field characterisation

Parsing all 96 records and **grouping by `f0`** (re-done this pass against
the shipped file) reveals the real shape: `magic.cmp` is a set of
**per-spell, level/threshold-*banded* parameter curves**, not a flat
classification table. Each `f0` value names one group (a spell/effect), and
within a group the records partition a **0–100 axis** into consecutive
`[f1, f2)` bands, each band carrying a payload:

| Field | Role |
|---|---|
| `f0` | **group / effect id** (0..66, with gaps) — one block of records per effect |
| `f1` | band **low** bound on the 0–100 axis |
| `f2` | band **high** bound (so `[f1,f2)` is one band; a single-band group is `[0,100]`) |
| `f3` | **wind-up** duration (effect ticks) — pre-action phase |
| `f4` | **onset** duration — the real action fires at `elapsed == f3+f4` |
| `f5` | **sustain** duration — how long the effect holds *(corrects the earlier "magnitude" reading — see below)* |
| `f6` | **linger/fade tail** duration (consumed only via the lifetime sum, so the "fade" semantics are the plausible reading) |

**All four payload columns are phase durations in effect ticks.** The
factory stores them at effect slots `+0x14/+0x18/+0x1c/+0x20` and every
case computes `f3+f4+f5+f6 → slot+0x4c` — the active-effect **lifetime**,
decremented per tick. Proof per phase: summon onset countdown
(`0x4e04b0`, fires at `0x4d74d7`); Freeze releases the cast at
`elapsed == f3` (`0x4d5f1a`) and sets the frozen flag
(`agent+0x224 |= 0x200`) at `elapsed == f3+f4` (`0x4d5f4f–0x4d5fb5`);
Lightning idles f3+f4 then strikes (`0x4ce7bb`); Invisibility applies at
f3+f4 (`0x4d6c0e`) and overwrites its f5 with `0xfffffff` (sustained by
mana drain instead). Damage magnitudes come from `props.000` curves
(e.g. Lightning `0x4ce81b–0x4ce850`), **not** from `f5` — the earlier
"magnitude" reading conflated the ramping sustain durations with a
value curve. Two groups are **dead data**: effect ids 0 (Bless) and 43
(Poisoned) never call the lookup on the factory path.

The banding is explicit in the data. Multi-record groups split 0–100 into
**5 bands** `[0,19)[19,39)[39,59)[59,79)[79,100]` (the five skill levels,
matching the 5-entry `props.000` curves) or **10 bands** of width 10, and
`f5` ramps with the band:

```text
f0=13:  f5 = 300, 600, 900, 1200, 1800     (5 level bands — longer sustain per level)
f0=43:  f5 = 40, 80, 120, 160, 200
f0=49:  f5 = 125, 250, 375, 500, 625
f0=0 :  f5 = 128,256,…,65536 (2^7..2^16)    (10-band *template* ramp)
f0=5 :  [0,90]→128, [90,100]→32767          (Freeze: the top band's huge sustain
                                             pairs with the skip-resistance path
                                             at 0x4d5f7d)
```

**Correction:** the earlier "`f5` = power-of-two **bitflag**, records =
per-spell **classification / element-flag**" reading was an artifact of the
two **template-ramp** groups (`f0=0` and `f0=10`, 20 of the 96 records),
whose `f5` happens to be `2^7..2^16`. Across the *other* groups `f5` is a
plain ramping value (`300`, `40`, `125`, `30`, `12800`, …), and `f1/f2`
are a band range, not a `±10` pair. So the column **roles are now fully
recovered** — id / band-lo / band-hi / wind-up / onset / sustain / tail —
and the table is a **banded phase-duration curve per effect**, parallel to
`props.000` but with explicit band boundaries rather than a fixed 5-entry
array. (The spell **tunable
parameters** — cost, mana, range, duration, chance, damage — remain the
`props.000` curves bound via `skills.txt` `\v`,
[`../skills-magic.md`](../skills-magic.md); `magic.cmp` is a separate
banded-curve table, not those parameters.)

**`f0` is the effect-type-id — proven by the lookup.** The record lookup
`fcn.004c70c0` (called from the effect factory `fcn.004df5d0`, each case
pushing its literal case number as the id and the `[desc+0x14]` 0–100
magnitude as the band value) compares `cmp [rec], <id>` (record `f0`
== the searched id) **and** `cmp [rec+4], <val>` / `cmp [rec+8], <val>`
(`f1 ≤ val < f2`, the band). So the access is a **`(effect-id, value)` →
band query**. **What fills the band `value` (`[desc+0x14]`) — NOT the skill
rank.** Tracing the callers: the combat on-hit resolver `fcn.004c6500`
passes a **combat power/magnitude** argument, and the OSIRIS spell functions
(`.\OSIRIS\osinpc.cpp`) build the descriptor with `[+0x14]` = the caster's
**agent Index/handle** (`caster+0x214`, clamped `[0,99]` by the factory).
The per-skill **rank** runs on a *separate* track — the `props.000` curves
indexed by `SP_<skill>` via `skills.txt` — and does **not** flow into this
band value on any static path (the shared `CMagic*` `Cast fcn.00539820`
resolves its slot via `fcn.004c7170` and runs the `.mgc` visual; it never
calls the factory). So the band axis is a caster/combat attribute, not the
cast rank. Signature correction (from the full disassembly): the lookup
does **not** return a row pointer — it is
`void __thiscall(id, val, int* f3, int* f4, int* f5, int* f6)` (`ret 0x18`),
copying `rec+0x0c/+0x10/+0x14/+0x18` through the out-pointers
(`0x4c711b–0x4c715f`); on miss it writes the defaults **16/16/64/16**
(`0x4c70f9–0x4c7111`). All 63 call sites are the effect factory's cases. Two consequences:

- **`f0` is the same id space as the `fcn.004e27b0` apply-body jump table**
  (`0..82`; `f0`'s `0..66` fits). The very `[desc+0x14]` id that selects a
  spell's apply/update/remove callbacks *also* selects its `magic.cmp`
  group. So `magic.cmp` group `f0=N` belongs to the effect whose apply body
  is jump-table case `N` ([`../skills-magic.md`](../skills-magic.md)).
- The group→spell naming is therefore reachable through the **effect-id →
  apply-body** map (the named `SMagic.cpp` bodies, e.g. Invisibility),
  **not** blocked on the unshipped `magic.dat` — the `CMagic*` registration
  ids (`LockPick`=38, etc.) were only one of several keys into the same id.

So `magic.cmp` is fully structurally recovered: a `(effect-id, band)`-keyed
parameter-curve table sharing the effect-type-id space with the effect
pipeline. **The naming is now done** — the effect-id → name table is
resolved in [`../skills-magic.md`](../skills-magic.md) (key source: the
charm parser's `stricmp` name→id chain at `0x4b2610..0x4b364e` in
`itemstatistic.cpp`, corroborated by the `.mgc`-ctor visuals and the
`skills.txt` props bindings). Group occupancy: f0 ∈ {0–11, 13–17,
19–56, 60, 66}; the absent ids are the props-curve-parameterised
effects; f0=38 has a record but no body.

**Bound on that path (verified):** only **three** spells register through
the directly-traceable `fcn.004c7510` path with recoverable ids —
`LockPick`=38, `PickPocket`=51, `Repair`=275 (all category 2). The other
~93 effects' id assignment is *not* via those three ctors (no further
`fcn.004c7510` callers exist), so the **full 96-spell id→record mapping is
not cleanly recoverable *via the `CMagic*` ctors*** — only 3 of 96 register
through the directly-traceable `fcn.004c7510` path. **But that is no longer
the only key:** since `f0` is the effect-type-id (proven below), each group
maps to the apply body installed by `fcn.004e27b0` case `f0`, so the
group→spell identity is reachable through the effect-id → apply-body table
(the named `SMagic.cpp` bodies), independent of the unshipped `magic.dat`.
The only piece still missing is a human label for each effect-id whose apply
body is not individually named yet — the column **roles** and the **access
mechanism** are fully recovered.

## Citations

```text
div.exe:0x004c7060   FUN_004c7060   loader — read count, slurp count×0x1c into [manager+0x0c].
                                    Source: dat\magic.cmp (compiled from dat\magic.dat).
div.exe:0x004c70c0   record lookup — cmp [rec]==id (f0), cmp [rec+4]/[rec+8] vs val (band f1..f2).
div.exe:0x004df5d0   effect factory — passes [desc+0x14] effect-type-id to both fcn.004e27b0 and this lookup.
div.exe:0x004e27b0   effect-id switch (jump table 0x4e302c, ≤82) installing the apply/update/remove callbacks.
```

## Status

- Format ✅ — `u32 count(96)` + 96 × 7-`i32` records; consumes the
  2692-byte file exactly; loader `FUN_004c7060` confirms the stride.
- Field shapes ✅ — all 96 records parsed and grouped by `f0`.
- Column roles ✅ *(recovered this pass from the grouped structure)* — the
  table is a **per-spell level/threshold-banded curve**: `f0` group/spell
  id, `f1`/`f2` the `[lo,hi]` band on a 0–100 axis, `f5` the per-band
  magnitude (scales across bands), `f3`/`f4`/`f6` secondary params. The
  earlier "power-of-two element bitflag / classification record" reading is
  retracted — that pattern is only the two template-ramp groups (`f0=0`,
  `f0=10`).
- **Access mechanism ✅** — `fcn.004c70c0` is a `(effect-id, value)` →
  band-row query: matches `f0 == id` and `f1 ≤ value < f2`, returns that
  band's row (`f5` = magnitude). So `f0` **is** the effect-type-id (proven,
  not just a candidate), shared with the `fcn.004e27b0` apply-body switch.
- Per-spell naming ✅ **RESOLVED** — the human name for each `f0`
  (=effect-id) group is recovered via the charm-parser name→id chain +
  `.mgc` ctor visuals + props bindings; full table in
  [`../skills-magic.md`](../skills-magic.md).
- **`f0` is the effect-type-id (proven)** (integrates the spell/effect
  residual) — the effect pipeline ([`../skills-magic.md`](../skills-magic.md))
  keys on a single data value, the **effect-type-id**, which the factory
  `fcn.004df5d0` reads from the effect-source record's **`+0x04`**
  *(record-layout correction: the earlier "+0x14" was the 0–100 band
  magnitude, clamped ≤ 99 and passed as the lookup* value*)* and uses as
  the `case` index into the registration switch `fcn.004e27b0` (cases
  `0..82`). The dataflow is fully traced: the magic.cmp record lookup
  **`fcn.004c70c0`** matches a record by **`f0 == id`** *and* the
  `[f1,f2)` band — and every factory case N pushes the literal N as the
  group id, re-proving the identity per-case. The former "needs
  `magic.dat`" naming caveat is obsolete — the names came from the
  charm-parser chain (above).
- Row → effect mapping ✅ (mechanism) — a record maps to its effect by
  **`f0` = effect-type-id** (proven above): `f0 → registration-switch case
  (`fcn.004e27b0`) → apply body`, [`../skills-magic.md`](../skills-magic.md).
  So record *i*'s owning effect is its `f0`; the per-level tuning values
  live in [`props.md`](props.md). Only the effect-id → human spell *name*
  stays open (the data-bound `magic.dat` mapping).
