# Agent instance struct (`CAgent`)

Every active world entity in Divine Divinity — the player, NPCs,
monsters, summons, and placed *object* agents — is a `CAgent`
(`.\AGENTS\agents.cpp`). This is the runtime memory layout of one
agent, recovered from the engine's verbose debug-dumper
**`FUN_0042d230`**, which `fprintf`s nearly every field with a literal
label. The dumper is to `CAgent` what `FUN_00581520` is to the object
catalogue: the most direct field-offset reference, because each label
sits next to the `[esi + offset]` read that feeds it (`esi` = the agent
`this`). **`esi` is set once (`mov esi,ecx` at `0x42d25a`) and only
restored at the epilogue (`pop esi`) — never reloaded mid-function
(verified)** — so every `[esi+off]` in the dumper is reliably
agent-relative, and the offset labels below are sound rather than
register-aliased. (This is the identity check that distinguishes a clean
`this`-relative dumper from a container/element reader where the base
register changes; it was confirmed here after that exact pitfall was hit
in an unrelated reader.)

The object-interaction work ([`object-interaction.md`](object-interaction.md))
touches the object-agent subset of this struct; this doc maps the
whole thing.

All integers little-endian; offsets are from the agent base pointer.

## Layout

Confidence: ✅ singular field with an unambiguous label; 🟡 grouped or
order-entangled within a multi-arg `printf`.

### Spatial / movement

| Offset | Type | Field | |
|---|---|---|---|
| `+0x04` | i32 | `X` | world cell/pixel X |
| `+0x08` | i32 | `Y` | world cell/pixel Y |
| `+0x0c` | f32 | `Fx` | fractional X (sub-pixel) | ✅ (dumper `fld [esi+0xc]` for the `Fx=%6.2f`) |
| `+0x10` | f32 | `Fy` | fractional Y | ✅ (dumper `fld [esi+0x10]` for the `Fy=%6.2f`) |
| `+0x18` | i32 | `Deltaheight` | ✅ (dumper `[esi+0x18]` = 1st arg of `Deltaheight=%d MidpointValid=%d`; **corrects** prior `MidpointValid`) |
| `+0x1c` | i32 | `Height` | |
| `+0x20` | i32 | `Mx` | midpoint X | ✅ (dumper's 4th int arg `mov edx,[esi+0x20]`; **`My` is not a real field** — the `Mx=%d My=%d` line pushes only 4 ints for 5 `%d`, so `My` prints misaligned data, a dumper off-by-one) |
| `+0x24` | i32 | `MidpointValid` | ✅ (dumper `[esi+0x24]` = 2nd arg; **corrects** prior `Deltaheight`/`My` — resolves the entangled-labels note) |
| `+0x216` | u8 | `CurrentAction` | ✅ current action id |
| `+0x217` | u8 | `Walkspeed` | ✅ |
| `+0x218` | u8 | `PriorityRing` | ✅ |
| `+0x278` | i32 | `Walkcount` — **remaining move sub-steps** (counts down; arrived at 0), reset on each new move; drives the slide + walk-cycle frame (`FUN_00427d30`, [pathfinding.md](pathfinding.md)) | ✅ |
| `+0x22c..0x238` | f32×4 | `CellDx, CellDy, …` — `+0x22c`/`+0x230` = the **per-tick move velocity** added to `Fx`/`Fy` each frame (`FUN_00427d30`); the latter two are the destination | ✅ (velocity) |
| `+0x27c` / `+0x280` | i32 | `OldCellX` / `OldCellY` | ✅ |
| `+0x284` | i32 | `CurrentMap` | ✅ |

### Identity / type

| Offset | Type | Field | |
|---|---|---|---|
| `+0x114` | char[] | `Name` | ✅ (used as `parent->Name` at the parent read) |
| `+0x214` | u16 | `Index` | ✅ agent index/handle |
| `+0x21c` | char* | header `Name` ptr | ✅ |
| `+0x28` | i32 | `Npc` | ✅ NPC flag/id |
| `+0x2c` | ptr | **`CStats`** — the polymorphic stats object (`CAgentStatistics` / `CMonsterStatistics` / `CPlayerStatistics`, vtables `0x61cde8` / `0x61ce5c` / `0x61ce98`), built by `fcn.0055a1f0` in `CAgent::Read`. *Not* an "agent script" field (see note). | ✅ |
| `+0x30` | ptr | `Alignment` → a **CAlignment entity** (`.\AGENTS\alignment.cpp`), carrying a name + numeric id; *not* an inline i32. Drives faction relations (see [`npc-ai.md`](npc-ai.md)). | ✅ |
| `+0x34` | u8 | `Ai class` | ✅ |
| `+0x220` | i32 | `Parameter 1` | ✅ |
| `+0x224` | i32 | `Parameter 2` / **agent flags word** (see below) | ✅ |
| `+0x228` | ptr | `Parent` | ✅ |

> **`+0x224` flag bits.** For NPC/object agents `+0x224` is a 32-bit
> flags word (the dumper labels it `Parameter 2`), tested/toggled across
> the engine. The bits anchored so far:
>
> | Bit | Meaning |
> |---|---|
> | `0x20000000` | **`promoted`** — the agent is fully active/loaded (vs a distant lightweight placeholder). The single most-tested bit (~70 sites): a *non-promoted* NPC skips look/sleep/say and even script-frame execution, logged `"Agent (non promoted) …"` / `"… because not promoted"`. The CPU-saving active/dormant distinction. |
> | `0x00400000` | **`sees all`** — perception bypass (set by the `sees all` command, [npc-ai.md](npc-ai.md)). |
> | `0x00000200` | **`asleep`** — the NPC sleep state ([npc-ai.md](npc-ai.md)); also set by a magic effect's apply (effect-id 5, `fcn.004d5f00` does `or [+0x224],0x200`), i.e. a sleep/incapacitate spell drives the same bit. |
> | `0x00000100` | **`hidden`** — toggled by the `visibility set to %d` agentscript command (visibility 0 ⇒ set, hidden; ≥1 ⇒ clear), **and by the magic Invisibility effect** (effect-id 6, apply `fcn.004ce090` does `or [+0x224],0x100` at `0x4ce0e5`; its remove clears it, **and** effect-id 13 `fcn.004d6b40` sets the same bit — a second invisibility-class effect) — code-anchored setters that confirm this bit. A hidden agent is one half of the `0x10000100` perception-skip mask. |
> | `0x00000800` | **`continue-this-frame`** — agentscript flow flag: when set the interpreter parses the *next* script line in the **same** frame instead of yielding (logged `"Parsing next line in same frame for %s"`). |
> | `0x00010000` | **`cannot die`** — essential/invulnerable NPC; set by `cannot die`, cleared by `can die` (logged `"Npc %s can/cannot die"`). |
> | `0x40000000` | **`repulsive`** — set by the `repulsive flag` command (`"Npc %s repulsive flag: %d"`); tested in the combat/behaviour code to exclude the agent from some interaction (targeting/collision). |
> | `0x10000100` | **perception-skip** mask — an object with either bit (`hidden 0x100` or `0x10000000`) is skipped by the NPC detection scan (invisible / non-targetable). **Magic drives both halves:** `0x100` via the Invisibility-class effects (eids 6/13), and `0x10000000` via a magic-effect body's `or [+0x224],0x10000000` at `0x4d0f6d` — so a spell can make a target fully non-targetable, not just hidden. |
>
> The remaining bits (`0x4`/`0x20`/`0x40`/`0x8000`/`0x100000`/`0x1000000`/
> `0x80000000`) are toggled **not** by agentscript commands but by the
> egg-hatch / trap / combat code (e.g. the `0x4`/`0x40` sets sit in
> `agentmanager.cpp` and the treasure path), so the "adjacent log string"
> anchor does not name them — they need **consumer-side** analysis (what
> each test gates) and are left 🟡 rather than guessed.

> **`+0x220` flag bits (a *second*, distinct flags word — "Parameter 1").**
> Where `+0x224` holds script/state flags, `+0x220` is the
> object/perception flags word the combat and detection code read. Bits
> seen so far (the perception ones are inferred from their role in the
> scan, [npc-ai.md](npc-ai.md), not yet string-anchored):
>
> | Bit | Meaning |
> |---|---|
> | `0x00020000` | **spawned-from-egg** — set by the egg hatch (`FUN_0043ccd0`, [monsters.md](monsters.md)); ✅ code-anchored. |
> | `0x10` | **protected / cannot be attacked** — the attack-engage `fcn.00417050` bails if the *target* has this bit, unless the attacker has `0x40`; ✅ code-anchored ([npc-ai.md](npc-ai.md), `attack $`). |
> | `0x20` | **concealed / in-shadow** — in the detection scan `fcn.004356f0` a target with this bit gets its detection radius **halved by day, quartered at night** (`CClock::GetHour` `fcn.0050bfe0` on clock `[0x658c1c]`; day = hour 5..22); ✅ code-anchored. |
> | `0x40` | **ignores relation/attack gating** — on the *attacker* it lets `fcn.00417050` attack a `0x10`-protected target, and in the detection scan `fcn.004356f0` it bypasses the faction/relation filter (`fcn.004380a0`); ✅ code-anchored. |
> | `0x100` | **un-perceivable / dead** — hard-skip in the detection scan `fcn.004356f0`: a target with this bit is dropped from the perceivable list at collection and skipped at scan time; ✅ code-anchored. |

> **Correction (`+0x2c`).** The dumper prints `Agent script= %s` from
> `[esi+0x2c]`, which earlier read as an "Agent script" field. In fact
> `[esi+0x2c]` is the **`CStats` pointer**: `CAgent::Read` (`fcn.0042eac0`,
> `0x42ebc8`) stores the result of the stats factory `fcn.0055a1f0` (which
> sets a `CPlayerStatistics`/`CMonsterStatistics`/`CAgentStatistics`
> vtable) there, and combat/perception read it as such — `Hp` at
> `[+0x2c]+0x04`, `Level` at `[+0x2c]+0x1c`, `sight` at `[+0x2c]+0x20`
> ([stats.md](stats.md), [combat.md](combat.md), [npc-ai.md](npc-ai.md)).
> The dumper's "Agent script" string is just the output of a **`CStats`
> virtual** (`vtable[+0x34]` = `fcn.0055b630`/`0055be30`) it calls on that
> object, not a distinct field. So `+0x2c` = `CStats`, full stop.

### Combat / AI

| Offset | Type | Field | |
|---|---|---|---|
| `+0x35` | u8 | `FightWalkSpeed` | ✅ (dumper `fcn.0042d230` push order; **corrects** prior `StepsCounter`) |
| `+0x36`/`+0x38`/`+0x3a` | i16 | `FightWalkCounter, FightWalkSteps, FightWalkStepsCounter` | ✅ (dumper order; **corrects** prior `Speed/Counter/Steps`) |
| `+0x58`/`+0x5c`/`+0x60` | i32 | `SpellKnowledge[3]` | ✅ |
| `+0x64`/`+0x68`/`+0x6c` | i32 | `SpellLearned[3]` | ✅ |
| `+0x70` | i16 | `Runaway` | ✅ |
| `+0x74` | i32 | `AiParameter` | ✅ |
| `+0x78`/`+0x7a`/`+0x7c`/`+0x7e`/`+0x80` | i16×5 | `Image indeces` (5 values) | ✅ (dumper reads `movsx word [esi+0x78..0x80]`) |

### Stats / trade

| Offset | Type | Field | |
|---|---|---|---|
| `+0x1a4` | i32 | `Summoned by` | ✅ (dumper `fcn.0042d230` reads `[esi+0x1a4]` directly) |
| `+0x1b0` | i32 | `Current weight` | ✅ (dumper `fcn.0042d230` reads `[esi+0x1b0]` directly) |
| `+0x1b4`/`+0x1b8`/`+0x1bc` | i32 | `Treasure type, IdentifyCost, HealCost` | ✅ |
| `+0x1c0` | f32 | `RepairCost` | ✅ |
| `+0x1c4` | i32 | `Attitude` | ✅ |
| `+0x208` / `+0x20c` | f32 | `RelativeSellPrice` / `RelativeBuyPrice` | ✅ |

### Inventory / social

| Offset | Type | Field | |
|---|---|---|---|
| `+0x23c` | ptr | inventory (nonzero ⇒ "Has inventory") | ✅ |
| `+0x240` / `+0x244` | i32 | `Inventory` (2 values) | ✅ (dumper `mov eax,[esi+0x240]` / `mov edx,[esi+0x244]` → `Inventory=%d,%d`; `+0x248`=`InventoryType`, `+0x24c`=`InventoryLevel` follow) |
| `+0x248` / `+0x24c` | i32 | `InventoryType` / `InventoryLevel` | ✅ |
| `+0x250` | i32 | `Synchparameter` | ✅ |
| `+0x264` / `+0x268` | i32 | `CurrentDialog` / `TalkingTo` | ✅ |
| `+0x26c` / `+0x270` | i32 | `Group` / `GroupIndex` | ✅ |
| `+0x274` | i32 | `Source egg` | ✅ |

### Region list & behaviour program

| Offset | Type | Field | |
|---|---|---|---|
| `+0x2a8` | i32 | `Thissize` | ✅ |
| `+0x2ac` | i32 | `Locked` | ✅ |
| `+0x2c0` | i32 | `Program counter` | ✅ |
| `+0x2c4` | i32 | `Amount of regions in list` | ✅ |
| `+0x2c8` | i32 | `Real region list size` | ✅ |
| `+0x2cc`/`+0x2d0`/`+0x2d4` | i32 | `Behavior[3]` | ✅ |
| `+0x2d8` | i32 | `Loop points` | ✅ |

The struct extends past `+0x2dc` (region-list and behaviour-program
content printed via loops); those array bodies are not yet sized here.

## Global registry: `CAgentManager` (`[0x658d50]`)

Agents are owned and resolved through one static singleton,
**`CAgentManager`** (`.\AGENTS\agentmanager.cpp`, ctor `fcn.00422370`),
at `[0x658d50]` — created in `.\GAME\compilestart.cpp` (`fcn.00499990`)
and rebuilt at map load (`fcn.004a0b10`), the same lifecycle as the
clock / `CTwilight` / `SMagic` singletons. It is referenced from ~1170
sites — the lookup layer behind "find the agent for this handle".

```text
[0x658d50] = CAgentManager:
    +0x00   ptr    iteration list base   (perception et al. walk this)
    +0x04   i32    iteration count       (end = [+0x00] + [+0x04]*4)
    +0x0c   ptr    handle-indexed agent pointer array   (init 0 in ctor)
    +0x10   i32    array count / capacity               (init 0 in ctor)

    resolve:  agent = [ [mgr+0x0c] + handle*4 ]   (NULL slot ⇒ absent)
```

The **handle** is the agent's `Index` (`+0x214`). The handle-indexed
form is how combat and the `agentscript` interpreter dereference a
target by id (e.g. the melee resolver `FUN_00417b40` does
`obj = [[mgr+0x0c] + idx*4]` then reads the object's flags/vtable), while
the per-frame NPC perception scan ([`npc-ai.md`](npc-ai.md)) iterates the
`[+0x00]`/`[+0x04]` list. Savegame restores agent state through this same
manager (`AgentVariables`, [`formats/savegame.md`](formats/savegame.md)).
The global **object/tile** array used by the renderer and line-of-sight
([`pathfinding.md`](pathfinding.md)) is a *separate* structure (the
worldmap `[0x74eca0]`), not this manager.

## Citations

```text
div.exe:0x0042d230   FUN_0042d230   CAgent debug-dumper — fprintf of every labelled field;
                                    this doc is built from its format strings + [esi+off] reads.
                                    Source path: ".\\AGENTS\\agents.cpp".
div.exe:0x00658d50   DAT_00658d50   CAgentManager singleton (the global agent registry).
div.exe:0x00422370   FUN_00422370   CAgentManager ctor (".\\AGENTS\\agentmanager.cpp").
```

## Status

- Field labels & offsets ✅ — ~50 fields recovered directly from the
  dumper's `printf` labels. **Caveat:** a dumper label names the *output*
  of the read, which can be a virtual call rather than a raw field —
  `+0x2c` ("Agent script") is really the `CStats` pointer (corrected
  above), so labels on polymorphic pointers should be cross-checked
  against the field's consumers.
- Global registry ✅ — `CAgentManager` `[0x658d50]`
  (`.\AGENTS\agentmanager.cpp`) resolves agents by handle (`Index`
  `+0x214`) via `[[mgr+0x0c]+handle*4]`, with an iteration list at
  `[+0x00]`/`[+0x04]`; ctor `fcn.00422370`. **Non-array fields ✅ (from the
  ctor):** `+0x1c` = current/last-index sentinel (init `-1`), **`+0x34` =
  `0x800` (2048) = the agent capacity** (max live agents), `+0x38` = `0x400`
  (1024, a sub-capacity), `+0x3c`/`+0x4c` = allocated buffers (the `+0xc`
  handle table is one), and `+0x14`/`+0x18`/`+0x20`/`+0x28`/`+0x2c`/`+0x30`/
  `+0x40`/`+0x44`/`+0x48` zeroed counters/heads. So the registry holds up to
  2048 agents in a handle-indexed table.
- Spatial block (`+0x04..+0x24`) ✅ (mostly resolved this pass via the
  dumper `fcn.0042d230` direct reads) — `X`=`+0x04`, `Y`=`+0x08`,
  `Fx`=`+0xc` (f32), `Fy`=`+0x10` (f32), `Height`=`+0x1c`,
  **`Deltaheight`=`+0x18`**, **`MidpointValid`=`+0x24`** (the last two were
  *swapped* in the table and are now corrected from the dumper's
  `Deltaheight=%d MidpointValid=%d` read order). **`Mx`=`+0x20`** ✅ (the
  dumper's 4th int read `mov edx,[esi+0x20]`); **`My` is not a real field** —
  the `X/Y/Height/Mx/My` line has *five* `%d` specifiers but the dumper
  pushes only *four* ints (`+4`,`+8`,`+0x1c`,`+0x20`), so `My=%d` prints
  misaligned stack (a dumper off-by-one), which is why it "entangled" with
  no stable offset. The spatial block is now fully resolved.
- Field *types* ✅ (for dumper-read fields) — the dumper `fcn.0042d230`
  reads each field with an **exact-width** instruction, so the type is
  observed, not guessed: `movzx word`→u16, `movsx byte`→u8, `mov dword`→i32,
  `fld dword`→f32 (e.g. image indices `movzx word`, `CurrentAction` `movzx
  byte`, `Fx`/`Fy` `fld dword`, weight/inventory `mov dword`). Only fields
  *not* surfaced by the dumper retain provisional i32-vs-ptr typing.
- Behaviour-program block ✅ (dumper-pinned) — `Program size`=`+0x2b8`
  (i16), `Program counter`=`+0x2ba` (i16), **`Behavior[3]`**=`+0x2cc`/
  `+0x2d0`/`+0x2d4` (i32×3, the dumper's `mov [esi+0x2cc/+0x2d0/+0x2d4]`
  for `Behavior=%d,%d,%d`), `Loop points`=`+0x2d8` (i32). (Confirms the
  `PC +0x2b8` / `Behavior[3] +0x2cc` notes from [npc-ai.md](npc-ai.md).)
- Trailing **region-list** array ❓ — the `Amount of regions in list` /
  `Real region list size` / `Region list content` counts are loaded
  indirectly (not a direct `mov [esi+off]`) and the content is printed by a
  loop, so it is a **variable-length array** reached via a pointer/sub-list,
  not a fixed field run; element layout not decoded.
- Consumers ❓ — this maps the storage; the movement/AI/trade functions
  that read and write these fields are documented elsewhere as they are
  reversed.
