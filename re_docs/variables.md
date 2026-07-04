# Script variables & state (`.\AGENTS\agentvars.cpp`)

The **named-variable** layer the engine's scripting runs on — the store
behind quest flags, per-NPC script state, and the computed values scripts
read. All four script systems ([Osiris](osiris.md),
[agentscript](npc-ai.md), the [mgcscrpt VM](script-language.md), and
[dialogue](dialogue.md)) read and write through it, and two of its scopes
are saved, so the world's story state survives a reload.

## The manager — `CVariableManager`

`CVariableManager` (vtable `0x617b48`) is the base store: a set of named
variables (name → value). A [`CVariableExpression`](script-language.md)
reads a variable through it, and a `CSAssignment` statement writes one — so
in the script VM `x = y + 1` is a manager read of `y`, an evaluate, and a
manager write of `x`.

## Scopes

| Scope | Class | Lifetime / use |
|---|---|---|
| **Global** | `CGlobalVariables` (`0x612ebc`) | story-wide named variables — the **quest/story flags** [Osiris](osiris.md) rules test and set, plus any script globals. The largest persisted block. |
| **Per-agent** | `CAgentVariableManager` (`0x60bf98`, `agentvars.cpp`) | variables scoped to one [`CAgent`](agent.md) — the NPC's own script state (its `agentscript` / dialogue progress). |
| **Automatic** | `CAutomaticVariables` / `CGlobalAutomaticVariables` (`0x612fdc`) | **computed / read-only built-ins** — values the script reads but doesn't store (derived from engine state, refreshed on access rather than persisted). |

The split mirrors the script systems: global variables are the story
database's flags, agent variables are an NPC's private memory, and the
automatic variables are the read-only "sensors" (engine state exposed to
scripts).

## Persistence

Two of the scopes are savegame blocks ([`formats/savegame.md`](formats/savegame.md)):

- **`GlobalVars`** — `CGlobalVariables`, read by `FUN_004adcb0` (the largest
  single block, ~90 KB in a fresh start — it carries the whole story flag
  set).
- **`AgentVariables`** — the per-agent stores, restored through the agent
  manager `[0x658d50]`; each agent's variable sub-object (`mgr+0x24`) is
  also read inline in `CAgent::Read` (`fcn.0042fdc0`, which logs
  `"Failed to retrieve agent variable id %d"` and runs
  `ConvertScriptPointersFromReading`).

The **automatic** variables are *not* saved — they are recomputed, which is
exactly why they form their own scope.

## How it connects

- **Osiris** — story rules' facts/flags are global variables; setting one
  is what advances a quest ([`osiris.md`](osiris.md), [`quest-log.md`](quest-log.md)).
- **Script VM** — `CVariableExpression` / `CSAssignment` resolve through
  these managers ([`script-language.md`](script-language.md)); the same
  read/write path serves spell scripts and item-affix expressions.
- **agentscript / dialogue** — an NPC's conversation and behaviour branch
  on its agent variables and the global flags.

## Status

- Manager ✅ — `CVariableManager` (`0x617b48`) named store; read via
  `CVariableExpression`, written via `CSAssignment`.
- Scopes ✅ — Global (`CGlobalVariables` `0x612ebc`), per-agent
  (`CAgentVariableManager` `0x60bf98`, `agentvars.cpp`), and automatic /
  computed (`CAutomaticVariables` / `CGlobalAutomaticVariables` `0x612fdc`).
- Persistence ✅ — `GlobalVars` (`FUN_004adcb0`) + `AgentVariables` blocks;
  automatic vars are recomputed, not saved.
- Variable record format / value typing ✅ (re-traced; **corrects an
  earlier misread**) — the store is a **two-level vector** (`.\MISC\vars.cpp`):
  `fcn.005058a0` reads an outer list of **16-byte vector-headers** (alloc
  `count × 0x10`, `vars.cpp:216`) and runs **`fcn.005053d0`** on each. Each
  16-byte header is a **`std::vector`-like descriptor**: `[+0]` = element-data
  pointer, `[+4]` = element count (read first), `[+8]` = a capacity/count
  copy (`mov [esi+8], [esi+4]`), `[+0xc]` = an aux field. `fcn.005053d0` then
  allocates **`count` × 12-byte elements** (`vars.cpp:171`) into the `[+0]`
  array — and **each 12-byte element is a variable entry `{+0 string, +4 int,
  +8 int}`**, read via the length-prefixed string reader `fcn.004f4d10`
  (`element+0`) then two ints `fcn.004f4c70` (`element+4`/`+8`). So a variable
  carries **both a text and a numeric (2-int) form**. *(Correction: a
  previous pass mis-read `esi` as the per-variable record and labelled the
  header's `+8`/`+0xc` as "name"/"value-object pointer" — those are actually
  the vector's capacity/aux fields; `esi` is the container throughout
  `fcn.005053d0`, and the variable data lives in the 12-byte elements at
  `element+0/+4/+8`.)*
  **There is
  no explicit type-tag enum.** `fcn.005053d0` does *not* switch on a stored
  type byte — it reads the fixed element shape (string + 2 ints) and branches
  only on **data presence** (`test eax; je` on an empty string / zero count),
  not on a type code. So the int-vs-string distinction is
  *structural* (the element simply holds both forms, the 12-byte alloc at
  `vars.cpp:171`), not an enum field — the "type-tag enum" was a mis-framing.
  With the container (`std::vector` header) and the element layout
  (`{string,int,int}`) both decoded, the variable store is reimplementable
  as a list of `{text, num, num}` entries; the only nuance left is whether
  `element+0`'s string is the variable name or a string-valued payload (it is
  read as the keyed string either way).

## Citations

```text
vtables: CVariableManager 0x617b48 · CGlobalVariables 0x612ebc · CAgentVariableManager 0x60bf98
         CGlobalAutomaticVariables 0x612fdc
div.exe:0x004adcb0   FUN_004adcb0   GlobalVars save block reader (5 header dwords → fcn.005058a0).
div.exe:0x005058a0   fcn.005058a0   variable-list reader — count + count×0x10 records (.\MISC\vars.cpp:216).
div.exe:0x005053d0   fcn.005053d0   per-variable record reader (typed: int via fcn.004f4c70, string via fcn.004f4d10).
div.exe:0x004f4d10   fcn.004f4d10   length-prefixed string fread (.\MISC\divsave.cpp:75).
div.exe:0x0042fdc0   fcn.0042fdc0   per-agent variable read in CAgent::Read ("Failed to retrieve agent variable id %d").
str: .\AGENTS\agentvars.cpp
```
