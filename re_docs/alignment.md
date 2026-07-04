# Alignment / factions (`.\AGENTS\alignment.cpp`)

The **faction system** — how the engine decides who is hostile to whom.
Every NPC belongs to a named **alignment** (a faction: a monster family,
a town guard, a wildlife group, the player's party). A pairwise
**relation** between alignments determines friend/foe, and the AI's
[perception/behaviour](ai-behaviour.md) tick consults it to pick targets.
This is the data behind "the orcs attack me but not each other", "turning
a town hostile", and the Osiris story changing a faction's stance.

## The manager — `[0x658de8]`

The **AlignmentManager** is a *static instance* embedded at **`0x658de8`**
(callers do `mov ecx, 0x658de8`, not a pointer load). Versioned
**`AlignmentmanagerV0.935`** in the save stream. Its sub-objects:

| Manager field | Holds |
|---|---|
| `+0x41c` | relation sub-manager (named-relation bookkeeping) |
| `+0x420` | the **alignment-entity vector** (all defined factions) |
| `+0x424` | the **relation bit-matrix** `{ +0 = bit array, +4 = row stride }` |

### `CAlignmentEntity` — a faction

Each faction is a **`CAlignmentEntity`** (0x18 bytes, vtable `0x60c018`),
created by `fcn.004381d0` and appended to the manager's vector (grows by
`0x20`):

| Offset | Field |
|---|---|
| `+0x00` | vtable (`0x60c018`) |
| `+0x10` | **id** (the matrix index; also the parent link for hierarchy) |
| `+0x14` | name |

An NPC references its faction by id; an NPC without one trips the warnings
`"Npc %s doesn't have an alignment at all !!!!"` /
`"… doesn't have an alignment entity !!!!"` (and the developer easter-egg
assert `"Evil attempt by story 'person' to destroy my magnificent code …"`).

## `dat\alignment.dat` — the faction definition file

Loaded at boot (`init.cpp`, `fcn.004a0b10`) and parsed by **`fcn.00438430`**.
A **keyword-command text script**: lines are split on `"\n"`
(`fcn.004fdc40`) and tokenised (`fcn.004fdf40`), and the leading keyword
indexes a **command dispatch table at `0x64a940`** (the same line-parser
idiom as the other `dat\*.txt`/`.dat` loaders). Commands include:

- **`new alignment $`** — define a faction (creates a `CAlignmentEntity`
  via `fcn.004381d0` into `manager+0x420`); duplicate names are rejected
  (`"%s is not a unique alignment name !"`).
- **`set relation`** — set the relation between two named alignments (two
  `atoi` numeric params); an unknown name logs `"Set relation does not
  know %s!"`.
- malformed input → `"Syntax error in alignment.dat : %s"`.

So the shipped `alignment.dat` is the authoritative faction roster and the
default relation matrix.

## The relation query — `fcn.004380a0` (72 callers)

The hot path. `relation(idA, idB)` on the manager `[0x658de8]`:

1. **Same faction** (`idA == idB`) → `0` — a faction is never hostile to
   itself.
2. Otherwise a **bit-matrix lookup**: index `bit = stride·idA + idB` into
   `manager+0x424`; word `bits[bit >> 5]`, bit `1 << (bit & 0x1f)` →
   boolean. The matrix is the precomputed pairwise table.
3. If no matrix is present, fall back to **`fcn.00437fd0`**, a
   **hierarchy-distance** walk: alignments nest via a parent link
   (`CAlignmentEntity+0x10`, walked by `fcn.00437710`), and two factions
   count as related when their tree distance is **< 25**. So factions form
   a tree and relatedness can be inherited from a parent group.

The matrix bit is *set* by **`fcn.00438790`** (same `stride·a + b >> 5 /
& 0x1f` index, OR in `1 << bit`) — the mutator behind `set relation` and
the runtime `change group alignment` verb.

The **72 callers** sit in the `0x40f7xx..0x410axx`
[`CAgentBehavior`/perception](ai-behaviour.md) cluster: each behaviour
fetches the two agents' alignment ids (via the agent manager `[0x658d50]`)
and calls `relation()` to decide whether the other agent is an enemy — the
friend/foe gate that drives [combat](combat.md) target selection and
aggro.

## Script & Osiris control

The faction graph is mutable at runtime:

- **agentscript / script-commands** — `set alignment $` / `new alignment $`
  (assign/define), `set alignment relation`, `change group alignment`,
  `set egg group` ([`script-commands.md`](script-commands.md),
  handler cluster `fcn.00511140`); unknown ids log
  `"Unknown alignment in set alignment relation (%s)"` etc. So a script
  can flip a town hostile or re-faction a spawn group.
- **Osiris** — `CDIVINITYOsirisNpcFunction.virtual_4` calls the hierarchy
  relation `fcn.00437fd0`, letting [story rules](osiris.md) query an NPC's
  stance.

## Persistence

The manager serializes (entities + relation matrix) into the savegame
(`"Alignment saving"` / `"Alignment load"`, integrity-checked with
`"Mismatch in alignment load"`), so script-driven faction changes survive
a save/reload ([`formats/savegame.md`](formats/savegame.md)).

## Status

- Manager ✅ — static instance `[0x658de8]` (`AlignmentmanagerV0.935`);
  entity vector `+0x420`, relation matrix `+0x424`, relation mgr `+0x41c`.
- `CAlignmentEntity` ✅ — 0x18 bytes, vtable `0x60c018`, `+0x10` id /
  parent, `+0x14` name; ctor `fcn.004381d0`.
- `alignment.dat` ✅ — keyword-command text file (parser `fcn.00438430`,
  dispatch `0x64a940`); `new alignment` / `set relation`.
- Relation query ✅ — `fcn.004380a0`: same→0, else bit-matrix, else
  hierarchy distance `< 25` (`fcn.00437fd0` via parent walk `fcn.00437710`);
  setter `fcn.00438790`; consumed by the AI behaviour cluster (72 callers).
- Script/Osiris control ✅ — `set/new alignment`, `set alignment relation`,
  `change group alignment` (cluster `fcn.00511140`); Osiris query
  `fcn.00437fd0`.
- Persistence ✅ (noted) — serialized in the savegame.
- Agent's own alignment-id field offset ✅ — **`CAgent+0x30`**. The AI
  relation-query cluster (`fcn.0040f740`/`fcn.00410090`/`fcn.00410310`/
  `fcn.00410660`, ~70 sites) uniformly fetches the id as
  `[[controller+4] + 0x30]` and passes it to `fcn.004380a0` with the
  manager `[0x658de8]`. `[controller+4]` is the `CAgent` itself — the same
  pointer is read at `+0x228` (the inventory-count field) and `+0x258` in
  those functions ([`agent.md`](agent.md) / [`formats/savegame.md`](formats/savegame.md)),
  so the alignment id sits at `+0x30`, just past the `CStats` pointer
  (`+0x2c`). The field is **persisted by name** in `CAgent::Read` (string
  read → resolve to the alignment entity, whose id `+0x10` is what lands in
  `+0x30`).

## Citations

```text
div.exe:0x00658de8   AlignmentManager (static instance; +0x420 entities, +0x424 relation matrix).
div.exe:0x004380a0   fcn.004380a0   relation(idA,idB) query — bit-matrix + hierarchy fallback (72 callers).
                                    Callers fetch the agent alignment id as [[controller+4]+0x30] = CAgent+0x30
                                    (e.g. fcn.0040f740 @0x40f7a5, fcn.00410660 @0x410676 reads same agent +0x228).
div.exe:0x00437fd0   fcn.00437fd0   hierarchy-distance relation (parent walk fcn.00437710; <25 = related).
div.exe:0x00438790   fcn.00438790   relation-matrix bit setter.
div.exe:0x00438430   fcn.00438430   alignment.dat parser (dispatch table 0x64a940).
div.exe:0x004381d0   fcn.004381d0   CAlignmentEntity ctor (vtable 0x60c018).
str: dat\alignment.dat · "new alignment $" · "set relation" · "set alignment relation" · "change group alignment"
```
