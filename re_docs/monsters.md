# Monster spawning (eggs)

Monsters in Divine Divinity are not placed directly — they hatch from
**eggs**, the engine's spawn-point abstraction (`.\AGENTS\eggman.cpp`,
`.\AGENTS\monstergen.cpp`). An egg names what monster to spawn and
where; a spawned monster's agent remembers its origin in the
`Source egg` field (`agent+0x274`, see [`agent.md`](agent.md)).

## Two kinds of egg

- **Manual eggs** — placed by designers in the world editor.
- **Generated eggs** — produced procedurally by a one-time
  *monster-generation* pass over each region.

The generation pass is a heavy precompute, gated behind a config
switch and a confirmation prompt:

```text
"I'm about to generate all monster regions. This will definitely take
 some time. Are you sure ?"
"All monster regions have been generated. Start up next time with
 'set monster generation 5' in your config.lcl file."
"Eggs have been saved to %s"
```

So `set monster generation <n>` in `config.lcl` drives the pass; the
resulting eggs are serialised to disk and reused on later runs.

## Balancing report (`FUN_0043d9e0`)

The monster-statistics aggregator walks every egg and emits a report —
the clearest window onto the data model. Its accumulator (`ebx`):

| Offset | Field |
|---|---|
| `+0x0c` | class-overview list (monster name → frequency) |
| `+0x10` | region-overview list |
| `+0x18` | amount of generated eggs |
| `+0x1c` | amount of manual eggs |
| `+0x20` | total experience (generated / story) |
| `+0x24` | total experience (manual eggs) |
| `+0x28` | total gold |

It prints overall totals, then a **per-monster-type** line:

```text
Monster %s - Frequency %d (%f %%) - Total xp =%d - Level=%d
           - Player clicks=%d - Hits on player=%d
```

— so each monster type is tracked by spawn **Frequency**, its
**Level** and XP **value**, and two **playtest-telemetry** counters,
`Player clicks` and `Hits on player`. The report also breaks down by
**class** (`Name,Frequency`) and by **object/region**
(`Name,Frequency,TotalValue`), i.e. the designers tuned encounter
density and reward economy from this dump.

## Runtime egg struct & hatch (`FUN_0043ccd0`)

The hatch pass `FUN_0043ccd0` iterates the egg array with a **92-byte
(`0x5c`) stride** — the same record size as on-disk
[`eggs.000`](formats/eggs.md) — so the runtime egg is the loaded record.
The fields it touches:

```text
egg (0x5c):
    +0x08   monster kind / template   (passed to the create call)
    +0x0c   flags   bit0 = ready-to-hatch (cleared after hatching),
                    bit1/bit2 = gating state, bit3 (0x08) = extra setup,
                    bit4 = a manager-gated variant
    +0x10   region / group id   (matched against the pass argument)
    +0x14/+0x18/+0x1c   spawn coordinates
    +0x20   egg id   → the spawned agent's Source egg (agent +0x274)
    +0x24   map id   → the spawned agent's CurrentMap  (agent +0x284)
```

Per egg the pass: (1) filters on the `+0x0c` flag bits and on
`+0x10 == regionArg`; (2) **validates the spawn cell is walkable** via
`FUN_0056e6b0(worldmap [0x74eca0], …, mask 0x13)` — the same `0x13`
movement-blocker mask the navigation walkability test uses
([`pathfinding.md`](pathfinding.md)) — skipping the egg if blocked; (3)
clears the egg's ready-to-hatch bit (`[+0x0c] &= ~1`); (4) **creates the
monster agent** via `FUN_005068f0` (the agent/monster create, manager
`[0x658bf0]`), which returns the new agent's handle; (5) resolves that
handle through `CAgentManager [0x658d50]` (`[[+0x0c]+handle*4]`,
[`agent.md`](agent.md)) and wires it up — `Source egg = egg+0x20`,
`CurrentMap = egg+0x24`, and sets the agent flag `+0x220 |= 0x20000`
(spawned-from-egg). So an egg is a one-shot spawner: it hatches its
monster onto a walkable cell, stamps the back-link, and disarms itself.

## Region population — `dat\monstergen.dat` (`fcn.0043ef40`)

Where eggs are *placed* spawn points, **`monstergen.dat`** is the
procedural **region-population script** that decides *what* populates a
region. It is a text, keyword-statement file (parsed by `fcn.0043ef40`,
`.\AGENTS\monstergen.cpp`, line-split via `fcn.004fdc40`/`fcn.004fdf40`,
numeric fields `atoi`), and its vocabulary (from the literals + diagnostics):

- **`new group "<name>"`** — define a named monster group.
- **`add group "<class>" probability <w>`** — add a monster **class** to
  the current group with a **probability weight**; a group is thus a
  weighted distribution over classes. The weights are validated —
  `"Incorrect distribution in group"` and `"Frequency list mismatch in
  monster generator"` fire if they don't sum/match.
- **`create group with behaviour <tactic>`** — bind the group's pack
  **`CGroupBehavior`** (War / Ambush / Surround / …,
  [`../ai-behaviour.md`](ai-behaviour.md)); `"Unknown group behavior %s"`
  on a bad name.
- A **region** then references named groups; `"Group %s not defined for
  region %s"` / `"Unknown region %s in monster generator"` /
  `"Unknown class %s"` are the resolve errors (region ids via the region
  system, [`formats/region.md`](formats/region.md)).

So generation is **region → its groups → each group's weighted class
distribution + a coordination behaviour**: the generator rolls the
distribution to pick creature classes and lays them down (as eggs) across
the region, with the group's `CGroupBehavior` steering them as a pack. The
per-type **Frequency** in the balancing report (`FUN_0043d9e0`) is the
aggregate result of these weights. This ties the **region** system, the
**egg** spawns ([`formats/eggs.md`](formats/eggs.md)), and the **group AI**
([`ai-behaviour.md`](ai-behaviour.md)) into one pipeline.

The parser chain is now pinned: the loader `fcn.005bdec8` opens
`dat\monstergen.dat` and hands it to `fcn.0043f2a0` → the section parser
**`fcn.0043f3d0`** (`.\AGENTS\monstergen.cpp`). It does *not* re-tokenise
per line — it walks a **pre-tokenised line vector** (element access via
`fcn.0058d700`/`fcn.0058d710`) and matches section/group keywords by
string scan (the `cmp si,ax` character-compare loops). Each group's
weighted class **distribution is consistency-checked**: a malformed one
trips `"Frequency list mistmatch in monster generator"` (`0x60cb80`) /
`"Incorrect distribution in group %s"` (`0x60ca0c`) — so the per-group
frequencies must form a valid list (they are the Frequency weights the
balancing report later aggregates). (The exact per-statement field grammar
is recovered at the vocabulary level and the parser/validation are now
pinned; the byte-precise token order within the section state-machine is
not split further 🟡.)

## Spawn-time stat init (deterministic — no stat roll)

When an egg hatches it creates the agent through **`fcn.005068f0`**, which
runs the agent init **`fcn.00424ea0`** (`.\AGENTS\agentmanager.cpp`): that
constructs the `CStats` sub-object (ctor `fcn.0055a190`, stored at
`agent+0x2c`, [`stats.md`](stats.md)), builds the `CAgent` (`fcn.0042c420`,
sets the `CAgent` vtable), and assigns the AI personality
(`fcn.0042cb80`, [`ai-behaviour.md`](ai-behaviour.md)).

**The stat values are deterministic** — there is **no random stat roll** on
spawn: the create→init→`CStats` path (`fcn.005068f0` / `fcn.00424ea0` /
`fcn.00506330` / `fcn.0042c420`) contains **zero `rand` calls**; a monster's
attributes come straight from its `CAgentClass` template, scaled by the
monstergen-assigned **Level**. The *only* randomness in the spawn chain is
**placement**: `fcn.00427a90` does a single `rand % range` over the
coordinate fields (`+0x14`/`+0x1c`/`+0x20`) to jitter the spawn position
within the area. So monster variety is *which* creature (the weighted class
distribution in `monstergen.dat`, above) and *where* it stands — **not**
per-instance rolled stats. This is consistent with the "genetic algorithms"
boot banner being a no-op ([`architecture.md`](architecture.md)): the
"genetic" part is the deterministic per-region class distribution, not stat
mutation.

## Citations

```text
div.exe:0x005068f0   fcn.005068f0   monster/agent create (from egg hatch).
div.exe:0x00424ea0   fcn.00424ea0   agent init (agentmanager.cpp) — builds CStats@+0x2c, CAgent, behaviour.
div.exe:0x00427a90   fcn.00427a90   spawn-position jitter — the only rand in the spawn chain.
div.exe:0x0043ef40   fcn.0043ef40   monstergen.dat parser — region/group/class/probability
                                    statements; "Group %s not defined for region %s".
div.exe:0x0043d9e0   FUN_0043d9e0   monster-statistics report — egg counts, XP/gold totals,
                                    per-monster Frequency/Level/value + click/hit telemetry.
div.exe:0x0043ccd0   FUN_0043ccd0   egg hatch pass — filter, walkability-check, create agent,
                                    wire Source egg / CurrentMap, disarm the egg.
div.exe:0x005068f0   FUN_005068f0   agent/monster create (manager [0x658bf0]).
div.exe:0x0056e6b0   FUN_0056e6b0   spawn-cell walkability test (worldmap, mask 0x13).
div.exe:0x0043e740   FUN_0043e740+  monstergen.cpp — region generation pass (eggs).
                                    .\AGENTS\eggman.cpp, .\AGENTS\monstergen.cpp
```

## Status

- Egg spawn model ✅ — monsters hatch from eggs; manual vs generated;
  agent records its `Source egg`.
- Generation pass ✅ — `set monster generation` precompute, eggs
  serialised to disk.
- Balancing data model ✅ — per-type Frequency/Level/XP/gold + click/hit
  telemetry, from the `FUN_0043d9e0` aggregator.
- Egg runtime struct ✅ — 92-byte record (matches `eggs.000`); the hatch
  `FUN_0043ccd0` reads, in order: **spawn x/y at `+0x00`/`+0x04`** (loaded
  first and fed to the walkability check `fcn.0043ca40` + placement — *not*
  `+0x14..+0x1c` as an earlier note said), kind `+0x08`, **flags `+0x0c`**
  (bit 0 = ready, *cleared* with `and …,0xfffffffe` after spawn; bits 2/3
  tested — so `+0x0c` is a flag word, *not* a spawn-count), region `+0x10`,
  three **agent-creation params at `+0x14/+0x18/+0x1c`** (passed to the
  agent constructor `fcn.005068f0` — the small-range link/behaviour slots,
  not world coords), egg id `+0x20` (→ `Source egg`), map/category `+0x24`.
- Hatch logic ✅ — `FUN_0043ccd0`: filter by flags/region → walkability
  check (mask `0x13`) → create agent (`FUN_005068f0`, mgr `[0x658bf0]`) →
  resolve via `CAgentManager` → set `Source egg`/`CurrentMap` + flag
  `+0x220|0x20000` → clear the egg's ready bit. One-shot spawn.
- Frequency-weight / level fields within the egg ✅ (resolved) — they are
  not separate egg-record fields: the balancing report derives Frequency by
  *counting* eggs per `monster_kind` (`+0x08`) and reads Level/XP/gold from
  the spawned **agent type**, not from the egg record. The egg's only
  per-spawn parameters beyond kind/position are the `+0x14/+0x18/+0x1c`
  agent-creation slots (above); there is no dedicated frequency or level
  word in the 92-byte egg.
- Egg on-disk format ✅ — `global\eggs.000` fully decoded: `u32 count`
  + 92-byte records (x, y, monster_kind, amount, egg_id, category, four
  link slots). See [`formats/eggs.md`](formats/eggs.md).
