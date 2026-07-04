# Osiris (story / quest engine)

Osiris is Divine Divinity's **story engine** — a fact **database + rules
engine** (Prolog-flavoured) that drives quests, scripted events and NPC
story behaviour. It lives in **`OsirisDLL.dll`** (`COsiris`); the
compiled story + runtime state is saved as **`main\startup\story.000`**.
This is the Tier-1 critical-path gap in [`../RE_STATUS.md`](../RE_STATUS.md)
— *in progress*.

## Engine API (`OsirisDLL.dll` exports)

```text
Compile(src, mode)       compile storyeditor\story.div → the database
Load / Save(path)        (de)serialize the database (story.000)
Merge / PrepareMerge     merge story databases (content-editor workflow)
Event(...)               assert an event → fires matching rules
RegisterDIVFunctions     register the engine callbacks the story may call/query
_ReadGoalData            story is organised into Goals
Minilog_Create/…         a trace/log ring buffer
GetStoryVersion / NoStoryLoaded
```

So the model is: the game **registers DIV functions** (the calls,
queries, and event hooks that bridge story ↔ engine), the story is a set
of **Goals** containing **rules** (`IF <events/conditions> THEN
<actions>`), and `Event(...)` asserts facts that fire rules, whose
actions call back through the DIV functions.

## `story.000` — saved database

Verified byte-exact against the shipped file (offsets are absolute):

```text
+0      u8     0
+1      char   "Osiris save file dd. 08/19/02 23:48:01. Version 1.4."  (save header)
+53     u8     0                                       NUL terminator
+54     u8 u8  01 04                                   save version 1.4
+56     u8 u8  00 00                                   (debug-flag bytes; version-gated)
+58     char[128]  "2.7.68\0…"  zero-padded            story/engine version (ends +186)
+186    u32    128                                     (leading count, meaning TBD)
+190    u32    14322                                   DIVObject (symbol) count
+194    …      14322 DIVObject records                 (see below; ends +606571)
+606571 u32    1175                                    Function count
+606575 …      1175 Function records                   (see below; ends +682095)
+682095 u32    10759                                   RETE-node count (_ReadReteNodes)
+682099 …      10759 RETE-node records                 (per-node header parsed — see below; only the variable edge list is unwalked)
        …      RETE adaptors / dbases / goals / global actions
```

After the RETE nodes, the remaining sections follow the **same
`{u32 count; record[count]}` shape**, each with its own exported reader
(verified from the OsirisDLL symbols): **`COsiris::_ReadReteAdaptors`**
(`0x1000d850`) reads the adaptor count via the stream API (`fcn.10014b80`;
error `"SaveLoadError: cannot read number of rete adaptors"`) then loops
`count` records through the per-adaptor reader **`0x10025d80`** (manager
`0x100a0e30`; error `"SaveLoadError: error reading adaptor %d"`); the goals
are **`COsiris::_ReadGoalData`** (`0x1000d9b0`). So the whole `story.000`
body is a sequence of count-prefixed record arrays read by named
`_ReadRete*`/`_ReadGoalData` methods — the only un-itemised detail is the
per-adaptor / per-goal field list inside those record readers, regenerable
from `story.div`.

The per-adaptor reader `0x10025d80` confirms the **uniform record-read
mechanism** is used story-wide: it does the same `fcn.10014b80 (flag byte);
fcn.100021a0; fcn.1002ba8b (allocate); ctor` shape as the node readers,
allocating **`0x60` = 96 bytes** for the adaptor record. So every
`story.000` record — RETE node (`0x34`/`0x18`/`0x90`/`0xbc`/`0x68`) and
adaptor (`0x60`) — is `{flag byte → alloc(kind size) → ctor populates from
the shared stream reads}`. `COsiris::_ReadGoalData` (`0x1000d9b0`) likewise
confirms the documented **80-byte (`0x50`) goal record** from its alloc
site (`push 0x50`; count error `"cannot read number of osiris story
goals"`). So all three `story.000` section record sizes are now confirmed
straight from the binary — node (`0x34`/`0x18`/`0x90`/`0xbc`/`0x68`),
adaptor (`0x60`), goal (`0x50`); the on-disk body is one mechanism
throughout, fully reimplementable.

**Adaptor/dbase field lists ✅ (resolved — the last un-itemised
readers).** Both were walked and verified **byte-symmetric against
their writers** (`COsiris::_WriteReteAdaptors` `0x1000e930`/`0x10020a40`,
`_WriteReteDBases` `0x1000e9d0`/`0x10020b80`):

- **Adaptor** (`fcn.10025d80`, `new(0x60)`): `u32 adaptorId`
  *(correction: a u32 id via `fcn.10014b80`, not a flag byte)* +
  `COsiColumnIdxValuePairList` + two previously undocumented byte
  lists — a **projection vector** (`u8 n; n×u8`) and a **column remap
  map** (`u8 n; n×{u8 from, u8 to}`).
- **DBase** (`fcn.1002b4d0`, `new(0x58)`): `u32 dbaseId` + column type
  list + `u32 rowCount` + rows of `{u8 arity (ignored on load),
  arity×COsiTypedValue}`, re-inserted through the normal engine insert
  `fcn.10028c80`.
- **`COsiTypedValue` wire format** pinned: tag `'0'`/`'1'`; strings
  XOR-`0xAD`; object values persisted **by name** and re-resolved
  against the registry at `0x10063fc8`
  (`"LOAD ERROR: object '%s' is not defined."`).

Everything not on disk (factory indexes, dedup/index structures,
resolved pointers) is rebuilt at load — the former "adaptor/dbase
per-record fields" residual is **closed** (regenerable runtime state +
the field lists above).

The shipped file is **3.98 MB**. **Correction (was wrong):** the body
**is** obfuscated — every *string* is **XOR-encrypted with the byte
`0xAD`** (so a `0x00` terminator is stored as `0xAD`). Integers (counts,
ids, type codes) are stored **raw little-endian, not XORed**. `story.001`
(360 KB) is a second segment/delta — **opened by `fcn.00516c20`** (builds
the path via `"%s.001"`, `fopen`, then calls
`COsiris::GetStoryVersion`), i.e. it is a **version-checked second story
segment** (a content patch/expansion delta validated against the main
`story.000` story version), not a standalone story. (Its body did not
XOR-`0xAD`-decode to readable strings the way `story.000`'s does, so it is
a distinct binary/dbase segment, consistent with a delta.) `binary.div` is
**not** bytecode — it is a 151-byte build log (`### Compile(...) /
Save(...)`).

### Host-side story load + savegame version-patch (`fcn.00516ff0`)

The div.exe side of `COsiris::Load` is **`fcn.00516ff0`**, which drives the
load through the `_Osiris` singleton (`OsirisDLL.dll`, import slot
`0x60640c`). Flow:

1. If a developer **`storyeditor\story.div`** exists (`_access(..., "r")`),
   it is used as the story source — the editor override; otherwise the
   shipped `main\startup\story.000` is loaded. Failures log
   `"Story load error: cannot access file '%s'."` /
   `"Story load error: loading file '%s'."`.
2. **Savegame story-version compatibility.** When loading a save, the
   save's embedded story version is compared to the installed version; if
   the save is **older**, the engine prompts
   `"Story of savegame (v. %d.%d.%d.%d) is older than version installed
   (v. %d.%d.%d.%d). Continue will patch the savegame."` and, on continue,
   **patches the save's story up to the installed version**. This is how
   old saves survive a game patch — the persisted RETE database is migrated
   rather than rejected. The `_Osiris` flags at `+0xfc` (load state) and
   `+0x118` (story-loaded) gate the path.
3. **Merge work-files.** The patch produces the intermediate
   **`story_merge_g.dat`** and **`story_merge_f.dat`**, which are
   **`_unlink`ed** once the merge completes (hence they do not ship). A
   failed merge logs `"There was an error patching the story"`.

So story loading is version-aware: a reimplementation must (a) accept an
editor-`story.div` override, and (b) version-check the persisted story in a
save and migrate older ones — both keyed on `COsiris::GetStoryVersion`
(cf. the `story.001` delta check by `fcn.00516c20` above).

### The cipher, from the deserializers (authoritative)

`COsiSmartBuf` holds the cipher byte at **`+0`**. `_ReadHeader` stamps it
**version-derived**: the 1.4 branch does `mov byte [esi], 0xad`
(`0x1000d53a`). The two read paths differ:

- **Integer / raw reads** — `fcn.10014b80` (u32) → `fcn.10014a10` (ensure
  N available, refill via `fcn.10014870`) → `fcn.10014aa0` (copy N
  bytes). **No XOR** anywhere. This is why the byte reader `fcn.10014a70`
  looked transform-free.
- **String reads** — `fcn.10014bf0`: read one raw byte (`fcn.10014a70`),
  `XOR` it with `byte[smartbuf+0]` (the cipher), repeat until the
  *decoded* byte is `0`. So strings are **NUL-terminated XOR-`0xAD`**,
  not length-prefixed.

So the rule is simply: **strings are XOR-`0xAD`, everything else raw.**

### Function record — `COsiFunctionData`, from `fcn.100078d0`

`_ReadFunctions` reads `u32 count` (1175) then loops `new(0x34)` +
`fcn.100078d0` (the record reader) + insert into the function map at
`0x1007bfb8`. The record begins **immediately after the count** (the
first record's `+0x04` field is the `45795` once misread as a section
"aux word"). The record reader fills the 52-byte object:

```text
read order            field      notes
u32  fcn.10014b80  →  +0x04      (~45795 in rec0)
u32  fcn.10014b80  →  +0x08
u32  fcn.10014b80  →  +0x0c
u32  fcn.10014b80  →  +0x14      numeric id (e.g. 6679)
u8   fcn.10014a70  →  +0x1c      function type (1..7 — the EFunctionType)
4×u32 fcn.10007240 →  +0x20..0x2c
string + params  (fcn.10007570 → COsiFunctionDef):
   fcn.10014bf0   →  name        NUL-terminated XOR-0xAD string
   value-type-list              the parameter signature (variable) ← read last
```

So the fixed prefix is **33 bytes** (eight u32 + one type byte), then the
XOR name, then the parameter signature. (Note: the **type is the byte at
`+0x1c`**, not the u32 at `+0x14` — an earlier draft of this file had
them swapped; the `EFunctionType` table above was always computed from
the `+0x1c` byte and is correct.)

#### Parameter signature (value-type-list)

Trails each name. From the two readers in `fcn.10007570`
(`fcn.10004560` then the value-type-list virtual `fcn.10018610`), and
verified by chaining **all 1175 records**:

```text
u32   L               out-param mask length in bytes (1 for ≤7 params, else 2)
u8[L] outParamMask    raw, NOT XOR'd; which params are OUT (query results)
u8    nParams
nParams × u32  paramTypeCode
```

The `outParamMask` is the clean tell that a function is a query — e.g.
`RealDivide` (Query, 4 params, mask `0x10`), `PlayerExp` (Query, mask
`0x70`); it is `0` for Event/Call/Proc/Database. The **paramTypeCode
reuses the DIVObject type taxonomy**: low codes are primitives
(`1`=INTEGER, dominant at 825; `13`=REAL; `12` frequent at 128 — likely
STRING/GUIDSTRING) and `4..11` are the entity types (NPC=4, OBJECT=5,
REGION=7, LOCATION=8, NPC_CLASS=9, OBJECT_CLASS=10, DIALOG_EVENT=11) —
so e.g. `Attitude(NPC, …)` carries a `4`, `FDObstacleRegion(REGION, …)` a
`7`, and `TradeStats` (8 params, the one `L=2` case) carries
`[4,2,2,2,2,2,2,1]`. `nParams` ranges 0..8.

With this, the whole Functions section walks: it spans **`+606571` ..
`+682095`** (75524 bytes), all 1175 records accounted for.

### RETE node section — `+682095`, 10759 nodes (`_ReadReteNodes`)

`_ReadReteNodes` reads `u32 count` (**10759**) then, per node, a **1-byte
kind tag (1..8)** dispatched through a jump table (`0x1000d82c`) to eight
polymorphic readers. Each reader reads a `u32` (the node id), allocates
the kind's node struct, runs a kind-specific deserializer, and links the
node (`fcn.10024a20`). On an unknown tag it errors `invalid rete node
type trying to read node %d`.

The eight readers are **inline branches inside `_ReadReteNodes`** at
`0x1000d6dc / d6f1 / d703 / d727 / d739 / d74b / d75d / d715` (the jump
table `0x1000d82c` entries, in tag order). The per-kind functions named in
the node table below (`fcn.10023880` etc.) are the node **constructors**
(verified: each does `call <shared base ctor>; mov [esi], <vtable>` — e.g.
Fact/Event share base `fcn.10023770`), *not* the field readers; the field
deserialize is the inline body at each branch. So a reimplementation walks
the node array by: read tag → (branch) read `u32` id → allocate via the
kind ctor → run that branch's inline field reads → link.

**Per-kind deserializer chain (pinned).** Each inline branch is a thin
`push esi; mov ecx, 0x100a0df8 (deserialize manager); call <reader>` to the
real per-kind **reader** — tag-ordered: `0x10025770` (Fact) / `0x10025810`
(Event) / `0x100258b0` (DIVQuery) / `0x100259f0` (And) / … / `0x10025950`
(InternalQuery). The reader reads the node's fields **then** calls the kind
ctor: e.g. `CReteEvent`'s reader `0x10025810` reads the event **name** via
the XOR-`0xAD` string reader `fcn.10014b80`, then constructs via
`fcn.10023880` — matching the file decode of node 0 (`NewHour`). So the
chain per node is **branch → reader (`0x102558xx`) → field reads (XOR-`0xAD`
string + scalars) → kind ctor (`fcn.10023…`) → link**; the kind ctors (in
the table below) only allocate + set the vtable.

The readers do their field reads through a **shared nested stream-read
API** — `fcn.10014b80 → fcn.10014a70 → fcn.10014a10` (the generic
read-primitive chain, the same helpers the `RuleActionPart` reader uses);
the chain bottoms out at **`fcn.10014a10`**, which reads from the in-memory
**stream buffer at `[stream+8]`** (the deserialize cursor) — the concrete
read primitive every node field flows through —
so each reader is just an *ordered sequence of generic reads* (a string,
some `u32`s) into the node before the ctor runs. This makes the node
deserialize **fully reimplementable**: walk the array as `tag → reader's
ordered generic reads → construct → link`.

**The eight readers are uniform** (refines the above): each `0x102558xx`
reader is the *same* call shape — `fcn.10014b80 (read a byte);
fcn.100021a0; fcn.1002ba8b (ALLOCATE the node); <kind ctor>;
fcn.10024a20 (link)`. *(Correction: `fcn.1002ba8b` is **not** a read helper —
it is the node **allocator** (`operator new`-style), called with the kind's
**node size as an immediate**: the Event reader does `push 0x34;
call fcn.1002ba8b` = allocate the 52-byte `CReteEvent`. The other readers
pass `0x18`/`0x90`/`0xbc`/`0x68` — so the alloc sites **confirm the node
sizes** in the table below, read straight off the binary.)* So per node:
`fcn.10014b80` reads a **byte** (the node flag) from the stream, the node
is **allocated** at its kind size, and the **kind ctor** runs. For the
alpha entries (tags 1–2) the kind ctor's shared base `fcn.10023770` calls
**`fcn.10020f50` — the shared `CReteNode` header reader documented below**
(validated on node 0: `u8 tag · u32 id · u32 · XOR-`0xAD` name · u8 flag ·
variable edge list`). **So the dispatch chain connects end-to-end to the
already-documented field reader**: `_ReadReteNodes` → branch → per-kind
reader (`0x102558xx`) → flag byte + alloc(size) + kind ctor → base ctor
`fcn.10023770` → header reader `fcn.10020f50` → id/name/scalars/edges. **Resolved — the on-disk node format is *uniform*, no per-tag residual.**
Checking the non-alpha ctors (e.g. `CReteAnd`'s base `fcn.1002ba60`) shows
they only **set up containers** (vtable + `fcn.1002fcb2`/`fcn.1002ce25`),
they do **not** read extra fields; and the join readers read only the flag
byte + the shared `fcn.10020f50` header. So a join's left/right inputs are
**the shared edge/connection list itself** (read by `fcn.10020f50`), not
separate serialized fields. The per-kind *larger* node sizes
(`0x90`/`0xbc`/`0x68` vs the alpha `0x34`) are **runtime-only tuple/join
memory** — allocated by `fcn.1002ba8b` at the kind size but **not read
from the file**; they are rebuilt at runtime by the match/fire engine. So
**every node serializes as the same shared header (tag/id/name/flag/edge
list); there is no per-tag on-disk field order to decode** — the
deserialize is *fully* reimplementable, and the earlier "tuple/join-memory
internals" gap is **runtime state, not a file-format residual**.

All eight node classes are recovered by name (from the vtables each
deserializer installs):

| tag | size | class | deserializer | role |
|---|---|---|---|---|
| 1 | `0x34` | **CReteFact**          | `fcn.10023860` | alpha entry — an asserted fact |
| 2 | `0x34` | **CReteEvent**         | `fcn.10023880` | alpha entry — an event (tag-2 node binds `NewHour`) |
| 3 | `0x18` | **CReteDIVQuery**      | `fcn.10021450` | test node — calls a DIV query |
| 4 | `0x90` | **CReteAnd**           | `fcn.10021490` | beta **join** (left ∧ right) |
| 5 | `0x90` | **CReteNAnd**          | `fcn.100214b0` | beta **negative join** (left ∧ ¬right; "AND NOT") |
| 6 | `0xbc` | **CReteRelCondition**  | `fcn.10022b70` | relational-condition test (largest single-input) |
| 7 | `0x68` | **CReteRuleActionPart**| `fcn.10024460` | terminal/production node — the rule's RHS action |
| 8 | `0x18` | **CReteInternalQuery** | `fcn.10021470` | test node — internal (engine) query |

Class hierarchy (shared base readers):

```text
CReteNode (fcn.10020f50)         common header: u32 id (+0x10/+0x14), flag byte (+0x0c)
 ├─ CReteFact, CReteEvent        (base fcn.10023770)
 ├─ CReteQuery (fcn.10021430)
 │    ├─ CReteDIVQuery   (tag 3)
 │    └─ CReteInternalQuery (tag 8)
 ├─ CReteBinaryNode (fcn.10021230)  embeds two CReteConnection sub-objects:
 │                                  LEFT input @ node+0x58, RIGHT input @ ~node+0x6c
 │                                  (each ~0x14 B: vtable + link fields), then join u32s
 │    ├─ CReteAnd        (tag 4)
 │    └─ CReteNAnd       (tag 5)
 ├─ CReteRelCondition    (tag 6)    (base fcn.10021100)
 └─ CReteRuleActionPart  (tag 7)    (base fcn.10021100)
```

This is a textbook RETE: `CReteEvent`/`CReteFact` are the alpha entry
points, `CReteDIVQuery`/`CReteInternalQuery`/`CReteRelCondition` are the
condition tests, `CReteAnd`/`CReteNAnd` are the beta joins (each holding
a **left** and **right** `CReteConnection`), and `CReteRuleActionPart`
is the production (rule-action) terminal. The engine source paths
(`osiris\source\rete.cpp` / `Rete.h` / `Rete_BasicTypes.h`) are still
embedded in the binary.

**Shared `CReteNode` header** (`fcn.10020f50`), validated on node 0:

```text
u8    kind tag (1..8)        read by _ReadReteNodes' loop
u32   id                     read by the per-kind case reader (node 0: 1)
u32   ?                      (node 0: 0)
char[] name + 0x00           XOR-0xAD string (node 0: "NewHour")  via fcn.10014bb0
u8    flag
…     connection list        variable — fcn.10005a70 loops reading the node's edges
```

So node 0 is a `CReteEvent` named `NewHour`, id 1. The trailing
**connection list is variable-length** (a node's outgoing links), which
is what makes every node variable on disk — there is no size prefix.

**Re-verified against the shipped `main\startup\story.000`** (independent
data read): the `u32` at `+682095` is exactly **10759**, and the first
record at `+682099` is `02 01000000 00000000 00` then the XOR-`0xAD`
bytes `e3 c8 da e5 c2 d8 df ad` → **`"NewHour"`** (`0xAD` = the encoded
NUL terminator), matching the header decode above; the next decoded name
in the section is the predicate **`"Time"`**. So node 0 (tag 2 =
`CReteEvent`, id 1, name `NewHour`) is the **per-game-hour clock event**
([`world-clock.md`](world-clock.md)) wired as a RETE alpha entry — the
root of every time-scheduled story rule.

**Node-0 record boundary (from the shipped file):** node 0 spans
`+682099 … +682133` = **34 bytes** — the 18-byte header (`u8 tag` + `u32
id` + `u32` + `"NewHour\0"` (8 B) + `u8 flag`) then a **16-byte edge
region** (four `u32`s `3, 3, 1, 1`) — after which **node 1 begins at
`+682133` with `tag = 7`** (a valid kind), so the per-node record *is*
walkable. But node 1's tag-7 record does **not** share node 0's field
layout (parsing it with the tag-2 shape yields garbage), confirming the
records are **per-class typed** — a complete byte-walk of all 10759 needs
each of the **8 per-kind case readers**, which is the remaining work; the
tag-2 (`CReteEvent`) record is now fully pinned.

**Scope of the residual (confirmed):** node 1 (`tag 7` =
`CReteRuleActionPart`, the rule RHS-action node, reader `fcn.10024460`)
does **not** decode by inspection — its on-disk bytes carry interleaved
`u32`s and a trailing XOR-`0xAD` string that bleeds into the next record,
so the only reliable way to walk the non-alpha node classes (tags 3–8) is
to follow their named per-kind readers (`fcn.10021450`/`…1490`/`…14b0`/
`10022b70`/`10024460`/`10021470`). I.e. the alpha-entry records (tags 1–2,
`CReteFact`/`CReteEvent`) are eyeball-decodable and **pinned**; the join /
condition / action / query records (tags 3–8) are reader-bound — a
mechanical replay of the eight readers, **not** an unknown structure (each
reader and its in-memory record size is already tabled above), and the
whole node set is regenerable from `story.div`. This is the precise scope
of the Osiris on-disk residual.

**Reader internals (tag 7, `CReteRuleActionPart`, `fcn.10024460`):** it
first calls the **shared base-node reader `fcn.10021100`** (tag/id/header),
then reads the **action part** — `fcn.10020e20` populates the node's
`+0x64` field and a sub-reader chain (`fcn.1000d210` / `fcn.10014a70` →
`fcn.10014a10`) reads the rule's **DIV-call list** (the production node's
RHS — the calls the rule fires, matching the match/fire model above). So
each tag-3…8 reader is `base-header reader` + `class-specific body
reader`; documenting the exact body bytes is a mechanical descent through
those named sub-readers (regenerable, low priority) rather than an
unknown — the *structure* (base + typed body, and for the action node a
DIV-call list) is established.

🟡 Walking all 10759 nodes to reach `_ReadReteAdaptors` therefore needs
each kind's full read sequence tallied, including the connection-list
loop and the per-kind sub-readers (`fcn.1001f440`/`fc00`/`fcn.10021a70`
for Fact/Event; the `CReteConnection` pair for And/NAnd; `fcn.10021100`
for RelCondition/RuleActionPart). This is several layers deep and is
ongoing. (`_WriteReteNodes` is a symmetric cross-check.)

**Priority note:** the compiled RETE in `story.000` is *regenerated* by
`Compile()` from `storyeditor\story.div`. A reimplementation that
recompiles the story needs the **engine semantics** (how these node
kinds match and propagate events) and the `story.div` source grammar far
more than a byte-exact parse of the shipped compiled network. So after
the node taxonomy (done), the higher-value Osiris targets are the
**match/fire engine** (`COsiris::Event` → node propagation) and the DIV
function semantics — see [`../RE_STATUS.md`](../RE_STATUS.md).

### DIVObject (symbol) table — `+194`, 14322 records ✅

A `u32` count (14322) then that many variable-length records, each a
name→id binding for every game entity the story refers to:

```text
char[]  name + 0x00    (XOR-0xAD; e.g. "NPC_CLASS_Mosquito", "REGION_Tingalf")
u8      type
u32     type            (repeats the u8 — verified equal for all 14322)
u32     id              (global symbol id; observed range 0..20015)
u8[8]   0               (reserved/flags — zero in the shipped story)
```

The `type` taxonomy (count → prefix), all 14322 verified:

| type | meaning | n | example |
|---|---|---|---|
| 4  | NPC (instance)   | 850  | `NPC_Garth` |
| 5  | OBJECT (instance)| 1808 | `OBJECT_magic_orb1` |
| 6  | DIALOG           | 490  | `DIALOG_Tutamun` |
| 7  | REGION           | 435  | `REGION_Tingalf` |
| 8  | LOCATION         | 1602 | `LOCATION_stps_Swen_Larian` |
| 9  | NPC_CLASS        | 382  | `NPC_CLASS_Mosquito` |
| 10 | OBJECT_CLASS     | 7208 | `OBJECT_CLASS_Object_2196` |
| 11 | DIALOG_EVENT     | 1455 | `DIALOG_EVENT_dream_cat_dead` |
| 12 | ENGINE           | 1    | — |
| 13 | FUNCTION         | 1    | — |
| 15 | SREGION          | 90   | `SREGION_…` |

**Cross-confirmation:** the 1455 `DIALOG_EVENT` symbols are exactly the
1455 named dialogue flags in `dialogids.dat` ([`dialogue.md`](dialogue.md))
— the story's `EventChanged` booleans are Osiris DIVObjects.

### Function table — `+606571`, 1175 records ✅

The relations the story uses (calls, queries, events, fact databases).
A `u32` count (1175) + a `u32` aux word (45795), then 1175 records; each
carries a `u32` id, a `u8` **function type**, a signature/param block,
and the XOR-`0xAD` name. The type byte sits 17 bytes before the name
start. All 1175 parsed; the type distribution **matches Osiris's
`EFunctionType` enum** (the same engine lineage later shipped in
*Divinity: Original Sin*):

| type | EFunctionType | n | examples |
|---|---|---|---|
| 1 | Event    | 77  | `ObjectUsed`, `NpcResurrected`, `NpcUsesTeleporter` |
| 2 | Query    | 58  | `AmountInInventory`, `InFightMode`, `RealDivide` |
| 3 | Call     | 187 | `ExecuteDeal`, `HatchMonsterEggs`, `SetNpcFleeingFlag` |
| 4 | Database | 446 | `Attitude`, `Diary_LoanData`, `FDObstacleRegion` |
| 5 | Proc     | 392 | `CreateDefaultAttitude`, `MorphFakeDwarf`, `KillGeorge` |
| 6 | SysQuery | 10  | `Count`, `Status`, `IsCompleted`, `WasActive` |
| 7 | SysCall  | 5   | `ActivateGoal`, `CompleteGoal`, `SetGoalSleeping`, `Clear` |

So `Event` facts (type 1) are asserted by gameplay (`ObjectUsed` ties
straight back to `CObject::Use`, [`object-interaction.md`](object-interaction.md));
`Database` relations (type 4, the largest group) are the stored fact
tuples the RETE network joins over; `Proc`s (type 5) are the story
subroutines; and the `SysCall`s (type 7) are the built-in goal-control
verbs (`ActivateGoal`/`CompleteGoal`/`SetGoalSleeping`) that drive the
quest state machine.

> **Lucky break:** `OsirisDLL.dll` ships with **full symbol names**, so
> every Osiris function is named — Osiris is far more tractable to
> reverse than `div.exe`.

## Architecture — a RETE rules network

`COsiris::Load` (`0x10012720`) reveals Osiris is a **RETE network** — the
classic forward-chaining production-rule engine. Asserted facts/events
flow through a network of nodes that incrementally match rule
conditions; matches fire the rules' actions. The `story.000` database is
read in these sections, in order:

```text
section                in-mem rec       on-disk / notes
_ReadHeader            —                version-branched (1.2/1.3/1.4); save string +
                                         version bytes + 128-byte story-version buffer
_ReadDIVObjects        32 bytes (0x20)  ✅ DECODED: +194, count 14322, variable records
                                         (name+type+id) — the named game entities
_ReadFunctions         52 bytes (0x34)  ✅ DECODED: +606571, count 1175, variable records
                                         (id+type+sig+name) — the DIV relations/calls
_ReadReteNodes         variable/typed   the RETE network nodes (rule-condition matchers)
_ReadReteAdaptors      variable         node adaptors (join/filter wiring)
_ReadReteDBases        variable         the fact databases (beta memories / stored tuples)
_ReadGoalData          80 bytes (0x50)  the Goals (story organised into goals of rules)
_ReadGlobalActionList  —                global actions
```

The fixed sizes (DIVObject 32 B, Function 52 B, Goal 80 B) are the
**in-memory** struct sizes; **on disk** each record is variable-length
(a count-prefixed array of name+fields, the name being a XOR-`0xAD`
NUL-terminated string — see the decoded layouts above). The RETE
node/adaptor/database records are polymorphic (typed, variable-size).
Reads use the `COsiSmartBuf` primitives (read-byte `fcn.10014a70`,
read-N `fcn.10014aa0`, plus u32/string wrappers; the string wrapper
applies the XOR-`0xAD` cipher).

then post-load fixup: `_GetDIVHandles`, `GenerateFunctionList`,
`GenerateNodeList`. So a reimplementation needs a RETE engine plus
readers for these eight sections.

## Gameplay → story: the event queue (`.\OSIRIS\Events.cpp`)

Gameplay does not call `COsiris::Event` directly at each event site — it
**enqueues** events and flushes them in a batch. The queue is a struct at
**`[0x7447d8]`** (`+0` capacity, `+4` count, `+8` buffer pointer; the
`[0x7447dc]` "event manager" [`messages.md`](messages.md) names is this
struct's *count* field). Each entry is **8 bytes**: `+0` the DIV **event
handle** (a type-1 Event function handle — `ObjectUsed`, `NpcResurrected`,
…) and `+4` the **argument descriptor** (`COsiArgumentDesc*`, or a small
type marker for argless events). The **enqueue** path appends
`{handle, argDesc}` at `buffer[count++]`, growing the buffer by realloc
(`fcn.004fa4f0`, `Events.cpp:1057`, 8 bytes/entry) when `count == cap`.

The **flush** (`fcn.0050fff0`) drains the queue: while `count > 0` it
loops the entries and, per entry, calls the imported
**`COsiris::Event(handle, argDesc)`** (`0x006063e0`) on the Osiris
singleton (`[0x60640c]`). So gameplay→story is **deferred and batched**:
events accumulate during the frame and are asserted into the RETE network
together — the mirror image of the story→gameplay [DIV Router](#the-div-router--story--engine-dispatch-osirisroutercpp).

### The raise wrappers (one per event)

`Events.cpp` (`0x50c000`–`0x510200`) is a flat bank of **70 raise
wrappers** — one per gameplay→story event, matching the ~77 type-1 Event
functions in the function table. Each wrapper builds the event's argument
list and enqueues it; the **70 `inc [0x7447dc]`** sites are exactly these
wrappers. A wrapper builds a **linked list of `COsiArgumentDesc` nodes**,
one per event parameter, each `new`-allocated (`fcn.004fa4b0`, 12 bytes):

```text
COsiArgumentDesc  (12 bytes)
+0  u8   type      the parameter's DIV type (4=NPC, 5=OBJECT, 7=REGION, …)
+4  u32  value     the typed osiris value = (arg << 4) | type
+8  ptr  next      next descriptor (NULL-terminated list)
```

The `value` field uses the **same `(value << 4) | type` encoding** as a DIV
handle and a Query out-param — one unified osiris value representation
across the whole engine. The sequence of `+0` type bytes a wrapper writes
is therefore the **event's parameter signature** (e.g. a wrapper laying
down `[4, 7]` raises an `event(NPC, REGION)`).

**Dead-end (recorded):** the wrappers carry **no event-name strings** —
`div.exe` is symbol-stripped, so unlike the `OsirisDLL.dll` side these are
anonymous `RaiseXxx` bodies. Mapping each of the 70 wrappers to its named
event (`ObjectUsed`, `NpcResurrected`, …) needs the **handle → function-
table** correlation (the registration order assigns each event its
handle), which is not recoverable from static strings here. What *is*
recoverable per wrapper: its event handle constant and its parameter-type
signature (the `+0` bytes).

The single in-engine assert path is then `COsiris::Event`:

## Runtime: the match/fire cycle

`COsiris::Event(uint handle, COsiArgumentDesc* args)` (`0x10010ba0`) is
the engine's front door — each flushed event is asserted through it:

1. **Look up** the event by `handle` in the function table. Undefined →
   logs `Function handle 0x%08x undefined.`; defined but unused by any
   rule → logs `event %s/%d ignored (no rules defined using it)` and
   returns (the cheap common case).
2. **Activate the event's node**: fetch its `CReteEvent` node and call
   the node's **propagate** virtual at **vtable +0x28** (`node->vt[10]`,
   `0x10010d7e: call edx`).
3. **Assert the fact**: the `CReteEvent` propagate (`0x1002a680`) builds
   a tuple and calls the insert core `fcn.10029fc0` — adding the fact to
   the node's database (failure logs `## insert (add fact) failed:
   tuple:`). The node's fact database is a **linked list/set of tuples**:
   the DB insert `fcn.10025ff0` walks the chain (via `+0x1c` next-links),
   runs a **tuple-equality test** (`fcn.100219d0`) to **deduplicate**,
   and only `new`-allocates + links a fresh tuple if absent (a duplicate
   is the "insert failed" case). So a node's memory is a deduplicated
   tuple set — the RETE alpha/beta memory.
4. **Propagate to children** (`fcn.10028c80`): iterate the node's child
   connections and call each child's insert virtual, so the fact flows
   into the `CReteAnd`/`CReteNAnd` **beta joins**.
   - **The join itself** is `CReteAnd::Insert` (vtable+0x28,
     `0x100274c0`): when a token arrives on one input it **scans the
     *opposite* input's token-memory array** (`[mem+0x10]` base …
     `[mem+0x14]` end, 4-byte token slots) and, per stored token, runs
     the **match test** (`fcn.10026300`, shared by `CReteFact` /
     `CReteAnd` / `CReteNAnd` / `CReteRelCondition`). On a successful
     match `fcn.10026300` builds the **combined token**, inserts it into
     *this* node's deduplicated memory (the chain-walk `fcn.10025ff0`),
     and propagates it onward (`fcn.1002b5df`). `CReteNAnd` inverts the
     test — it emits the left token only when **no** right token matches
     (the `"## (node %d) valid test fails: NOT … for token:"` log marks
     that negative path). So the beta memory grows incrementally: each
     new token is joined against the already-stored opposite side, never
     a full re-scan.
5. **Fire rules**: a fully-matched condition reaches a
   `CReteRuleActionPart` terminal, which **fires the rule** — logged
   `<name> fires. Rule variables: …` — and runs its action as a DIV
   **call** list (`[call]`; failure → `*** call fails ***`), calling back
   out through the DIV function table.

So events → facts → incremental joins → rule actions → DIV calls; this
is the standard RETE forward-chaining loop, here hand-written
(`osiris\source\rete.cpp`).

## Goals & rules (the `story.div` source structure)

The story is organised into **Goals**; the compiler's error strings spell
out the grammar exactly:

```text
Goal(<integer>) {
    INIT { <DIV call list> }    // run when the goal is activated
    KB   { <rule list>     }    // the IF…THEN production rules (the RETE)
    EXIT { <DIV call list> }    // run when the goal completes
}
```

Each rule is `<condition> → <rule action part>` (`rule_first_part`,
`rule_action 1..N`). Goals carry runtime state the engine gates on:
`Goal(%d).RulesDisabled`, completed/enabled flags, and parent-goal
`CheckEnableRules` — i.e. activating/completing a goal enables or
disables whole blocks of rules. The `SysCall` functions
(`ActivateGoal`/`CompleteGoal`/`SetGoalSleeping`, type 7 in the function
table) are the verbs that drive this goal state machine.

## Goal record on disk (`_ReadGoalData`, 80 B) & the tail sections

`_ReadGoalData` reads `u32 count` then loops `new(0x50)` +
`fcn.100092f0` (the goal reader) + register. The goal reader's read
sequence (matching the `Goal(n){INIT/KB/EXIT}` grammar):

```text
u32     id
char[]  name + 0x00     (XOR-0xAD)
u8      state           → +0x10 (RulesDisabled / completed / sleeping flags)
u32     n1; n1×{ u32 + element }     a ref list (rule/sub-goal references)
u32     n2; n2×{ u32 + element }     a second ref list
u8      flag
list    INIT DIV-call list           (fcn.1000d210)
list    EXIT DIV-call list           (fcn.1000d210)
```

The two trailing lists are read by **`fcn.1000d210`** — the **DIV-call
list** reader, which is *also* what `_ReadGlobalActionList` uses. So a
goal's `INIT`/`EXIT` blocks and the story's global actions share one
on-disk list format (a sequence of DIV calls with their arguments). The
`KB` rules are not in the goal record — they live in the RETE network
(the `CReteRuleActionPart` terminals), referenced from the goal.

The remaining tail sections (each `u32 count` + per-record reader):

```text
_ReadReteAdaptors      fcn.1000d850 → per-record fcn.10025d80   (join/filter wiring)
_ReadReteDBases        fcn.1000d900 → per-record fcn.1002b4d0   (fact databases)
_ReadGoalData          fcn.1000d9b0 → per-record fcn.100092f0   (80 B goals, above)
_ReadGlobalActionList  fcn.1000dad0 → fcn.1000d210              (one DIV-call list)
```

**Adaptor & DBase record contents** (from the per-record deserializers'
vtables):

- **ReteAdaptor** — a **96-byte** object (`new(0x60)`, reader
  `fcn.10024da0`) built around a **`COsiColumnIdxValuePairList`** plus a
  few flag bytes: a list of `(columnIndex, value)` pairs — the
  **column-remap / constant-bind wiring** an adaptor applies to tuples
  flowing between nodes (RETE's column projection + constant tests).
- **ReteDBase** — an **88-byte** object (`new(0x58)`, reader
  `fcn.10029dc0`): a **`COsiValueTypeList`** (the relation's **column
  schema**, read via `fcn.10018610` — the same value-type-list format as
  function params) followed by a **`COsiVector` of `CTuple`** (the
  stored **rows**). So a RETE database is a typed fact relation =
  schema + tuple vector (the persisted beta memory).

## The DIV Router — story → engine dispatch (`.\OSIRIS\Router.cpp`)

When a rule fires, its action runs the rule's DIV **calls**; conditions
run DIV **queries**. Both cross from `OsirisDLL.dll` back into `div.exe`
through callbacks the game hands the engine at startup via
**`COsiris::RegisterDIVFunctions(TOsirisInitFunction*)`**. The binding
site is `fcn.00516b90` (run at compile-start `fcn.00499990` and map-load
`fcn.004a0b10`): it fills a 4-pointer `TOsirisInitFunction` struct and
passes it in, then `Minilog_Create` / `Compile` / `InitGame`.

The four callbacks are:

```text
+0  MakeHandle  fcn.00516aa0   pack (idx, word, type) → a DIV handle
+4  Call        fcn.00516ac0   engine→game action  (invokes handler vtable slot 0)
+8  Query       fcn.00516b30   engine→game query   (invokes handler vtable slot 1)
+c  Error       fcn.00516b70   printf("Osiris error is %s\n", msg)
```

**Handle encoding** (`MakeHandle`, `fcn.00516aa0`):
`handle = ((idx & 0x3ff) << 16 | word) << 4 | (type & 0xf)`, giving three
packed fields (verified by decoding it back inside the OBJECT handler,
below):

```text
bits 0..3    type   target type code  (NPC=4, OBJECT=5, DIALOG=6, REGION=7,
                                        LOCATION=8, NPC_CLASS=9, OBJECT_CLASS=10,
                                        DIALOG_EVENT=11, ENGINE=12, FUNCTION=13)
bits 4..19   word   the entity's runtime id — the key for the world lookup
bits 20..29  idx    the verb id — which action/query the handler performs
```

So the **low nibble of every DIV handle is its target type code** (the
DIVObject taxonomy), the middle 16 bits select the entity, and the top 10
bits select the verb. `type` routes to the handler; `word` and `idx` are
the handler's two inputs.

**Dispatch** (`Call`/`Query`, `fcn.00516ac0`/`fcn.00516b30`) is identical
bar the vtable slot: `type = handle & 0xf` (clamped to 0..13), then
`handler = (*CRouter)[type]` — a per-type handler object — then the arg
binder `fcn.00516a60` walks the incoming `COsiArgumentDesc` list and
caches each value pointer into the handler (pointer array at `handler+4`,
count at `handler+0x400`), and finally `handler->vtbl[slot](handle)` is
called (slot 0 for Call, slot 1 for Query). `Call` additionally
short-circuits type 12 (ENGINE) when the handle's top bits are `0x5600000`
and the gate `[0x744804]` is set.

`CRouter` is the singleton at **`[0x744808]`**; `[CRouter+0]` is the
14-slot handler array, and the ctor `fcn.00516d70` `new`s one handler per
type code 0..13, each a **0x404-byte** object (the 256-entry arg-pointer
cache at `+4` plus the `+0x400` count exactly fill `0x404`) carrying a
two-method vtable. The slots are **RTTI-verified** as the
`CDIVINITYOsiris*Function` family, keyed by target type:

| type | DIVObject | class (`CDIVINITYOsiris…Function`) | vtable | slot0 Call | slot1 Query |
|---:|---|---|---|---|---|
| 0–3 | (primitives) | `…Function` (base)        | `0x618814` | `0x5132d0` | `0x5132d0` |
| 4   | NPC          | `…NpcFunction`           | `0x618820` | `0x513850` | `0x513530` |
| 5   | OBJECT       | `…ObjectFunction`        | `0x61882c` | `0x515cf0` | `0x5163a0` |
| 6   | DIALOG       | `…DialogFunction`        | `0x618838` | `0x5104d0` | `0x5105e0` |
| 7   | REGION       | `…RegionFunction`        | `0x618844` | `0x516600` | `0x5166e0` |
| 8   | LOCATION     | `…LocationFunction`      | `0x618850` | `0x5132d0` | `0x5132d0` |
| 9   | NPC_CLASS    | `…NpcClassFunction`      | `0x61885c` | `0x5105e0` | `0x5132d0` |
| 10  | OBJECT_CLASS | `…ObjectClassFunction`   | `0x618868` | `0x5132d0` | `0x5165b0` |
| 11  | DIALOG_EVENT | `…DialogEventFunction`   | `0x618874` | `0x50c0a0` | `0x5105e0` |
| 12  | ENGINE       | `…EngineFunction`        | `0x618880` | `0x511b30` | `0x511260` |
| 13  | FUNCTION     | `…FunctionFunction`      | `0x61888c` | `0x5132d0` | `0x5132e0` |

The base method `0x5132d0` is a default/no-op; a slot pointing at it means
that type has no specialised behaviour on that axis (so LOCATION and
FUNCTION are inert both ways; NPC_CLASS has only a Call, OBJECT_CLASS only
a Query). The handler is keyed only by the **target type**; *which* verb
to run is the handle's `idx` field, and each handler **switches on it
internally**.

Worked example — the **NPC Call** handler (`0x513850`, slot 0 of type 4):
it resolves the live agent through the shared `CAgentManager [0x658d50]`
(logging `"Can't get npc in story (%d,%d)"` on failure) and **starts a
script frame on it** (`"Starting script frame %s for agent %s from
Osiris"`, the `NpcScriptFrames` table) — i.e. a fired story rule drives
the NPC's agentscript ([`npc-ai.md`](npc-ai.md)). This confirms slot 0 is
the action/Call side.

The **NPC Call** handler is the largest: a **93-case switch** (jump table
`0x515b7c`, `idx` 1..93) over the live agent (resolved via `CAgentManager
[0x658d50]`), of which **78 are real verbs and 15 are no-ops** (shared stub
`0x514fc2`). It is the story↔NPC action API. The verbs cluster into
families (handler fcn / identifying log string in parentheses):

- **Inventory / items** — give, transfer, drop, replace, and transfer-
  unique-item between NPC inventories; all route through the inventory
  resolver `fcn.004afb70`. Verbs 27, 28 (`"Transfer between inventories
  from %s to %s"`), 62/63/85 (`"Trying to move an osiris object to a non
  inventory object"`), 65 (`"Npc drop object doesn't find osiris object
  %d"`), 66 (`"Replace osiris object in npc…"`), 86 (`"…unique item…"`),
  92.
- **Magic / transform** — apply or remove a magic effect on the NPC via
  `SMagic [0x658c38]` (`fcn.004df5d0` spawn / `fcn.004d3dd0` remove): verbs
  8, 50, 76, and **82 = polymorph** (`"Unknown npc class in osi npc
  function polymorph"`, via `fcn.004f5230`).
- **Faction** — verb 60 = **set alignment** (`fcn.00437ed0`, the alignment
  matrix; `"Trying to set alignment for npc %d, which does NOT exist!"`).
- **Movement** — position / pathfind the NPC: verbs 30 (`fcn.004273e0`),
  42, 57 (`fcn.005719f0`, the pathfinder).
- **Spawn / treasure / trade** — create NPC (39, 40, `.\OSIRIS\osinpc.cpp`),
  give a treasure table (46, `"Warning - treasure table %s does not
  exist"`, `fcn.0041c5c0`), set the trade/portrait image (74, `"Trade image
  index %d out of range"`).
- **Flags / state** — direct writes to the agent flag words, e.g. verb 54
  sets `agent+0x224 |= 0x80000000` (a paired verb clears it) — a new
  control bit beyond the [`npc-ai.md`](npc-ai.md) set.
- **Combat** — verb 13 calls the agent's `vtable[+0x50]` (the to-hit /
  skill channel).

The 15 no-op slots (idx 17, 20–22, 26, 34–35, 37, 47, 51–53, 58, 77, 80)
return success without acting. (Not every one of the 78 real verbs is
individually named here — the families and their handler fcns above are
firm; the remaining unnamed verbs are direct agent-field mutations whose
per-field meaning is not yet pinned.)

Worked example — the **OBJECT Call** handler (`0x515cf0`, slot 0 of type
5): it decodes the same handle layout — `word = (h >> 4) & 0xffff` is the
object id, `idx = (h >> 20) & 0x3ff` is the verb — looks the object up in
the object manager `[0x658bdc]` (`fcn.00585d10`; miss →
`"CDIVINITYOsirisObjectFunction: Don't find %d X=%d Y=%d Ti=%d as osiris
object"`), then runs **`switch(idx - 1)`** over a **22-entry jump table**
(`0x516344`, `idx` 1..22).

**The object property API.** Each object carries a runtime flags word at
`obj+0x30` and a parallel value pool at `obj+0x34` (the `sb_*` bits,
[`object-interaction.md`](object-interaction.md)). The verbs read/write it
through three shared helpers: **`fcn.00591940(valuePool, flags, bitIdx)`**
= *get* a property, **`fcn.005919e0(valuePool, flags, bitIdx, 0, 0)`** =
*set/clear* a property bit, and **`fcn.00585780([0x658bdc], obj, …)`** =
the object-manager **refresh** run after a state change (re-evaluates
sprite/blocking). The `bitIdx` is the `sb_*` bit number (e.g. `0x15`=21
`sb_locked`, `0x19`=25 `sb_closed`, `0xd`=13 `sb_wt_use`, `0x12`=18
`sb_move`, `0x1d`=29 `sb_invisible`, `0xc`=12 `sb_transforms`, `0x13`=19
`sb_item_class`).

The complete verb table (decoded from each case body; `idx` is the
handle's verb field):

| idx | case | effect |
|---:|---|---|
| 1 | `0x515dc1` | if `sb_locked`: read it (`get 0x15`) and **refresh** — re-assert lock state |
| 2 | `0x515e07` | **unlock & open** — clear `sb_locked`(21); if `sb_closed`(25) clear it; refresh |
| 3 | `0x515e52` | clear `sb_wt_use`(13) if set; refresh |
| 4 | `0x515eb3` | hand to manager `[0x75182c]` (`fcn.005a3690`) — separate subsystem |
| 5 | `0x515ed2` | resolve the object, **spawn a magic effect** on it via `SMagic [0x658c38]` (`fcn.004df5d0`, descriptor `{0x54,5,0x63,…}`) |
| 6 | `0x515ff0` | **create/spawn** via `[0x658bf0]` (the agent/monster create mgr, `fcn.00506fb0`) |
| 7 | `0x51601b` | read `sb_move`(18) + `sb_wt_use`(13) (`get`) |
| 8 | `0x516062` | toggle `sb_invisible`(29) on the linked object (`obj+8`); refresh |
| 9 | `0x516323` | *no-op* (default) |
| 10 | `0x5160be` | read `sb_locked`(21) + `sb_closed`(25) (`get`) |
| 11 | `0x51612f` | **open/close** — set `sb_closed`(25); refresh |
| 12 | `0x516323` | *no-op* |
| 13 | `0x516323` | *no-op* |
| 14 | `0x51619e` | resolve the agent (`fcn.005169a0`) and act on it (`fcn.00589fe0`, `agent+0x214`) |
| 15 | `0x5161cc` | `sb_transforms`(12) + `sb_wt_use`(13) — object **transform**; refresh |
| 16 | `0x516323` | *no-op* |
| 17 | `0x516323` | *no-op* |
| 18 | `0x516243` | **item-link spawn** (from `itemlink.dat`; `"Item %s is undefined in itemlink.dat"`); `sb_item_class`(19) |
| 19 | `0x515f91` | spawn a magic effect on the **agent** via `SMagic [0x658c38]` (`fcn.004df5d0`, descriptor `{0x44,9,…}`) |
| 20 | `0x516282` | property set (shares the verb-8 set/refresh tail) |
| 21 | `0x516297` | property get/set (shares the verb-8 tail; `get 0x591940`) |
| 22 | `0x5162c1` | hand to `fcn.004f7b40` (descriptor not fully decoded) |

So the type-5 handler is a state-mutation API over the object's `sb_*`
flags (lock/close/transform/visibility, verbs 1–3, 8, 11, 15, 20–21),
plus a few that reach into other subsystems — magic effects (5, 19),
spawning (6, 18), and per-agent actions (14). Five slots (9, 12, 13, 16,
17) are unused no-ops. The property helpers and `sb_*` bit numbers are
shared with `CObject::Use` ([`object-interaction.md`](object-interaction.md)),
so a story `Call` and a player `Use` mutate object state through the same
machinery.

Worked example — the **ENGINE Call** handler (`0x511b30`, slot 0 of type
12): the engine-global verbs that act on no single entity. It decodes the
verb the same way (`idx = (h >> 20) & 0x3ff`, `dec`, bound `idx-1 ≤ 113`)
and runs a **114-entry jump table `0x513104`** — far larger than the
object set — dispatching real engine calls (it is an executor, not a
lister). Of the 114, **85 are real verbs and 29 are no-ops** (shared stub
`0x5130e3`). The verb families, from the error strings each case emits and
the handler each calls (`fcn.005169a0` is the common arg-resolve prologue):

- **Trap control** (verbs 97/98/104, handler `fcn.00443250`) — `"Unknown
  trap %d in osiris break_trap"` / `repair_trap` / `execute_trap`: spring,
  repair, or disarm placed traps ([`traps.md`](traps.md)).
- **Faction** (verb 34, `fcn.00437e40`; verb 38, `fcn.00437450`) —
  `"Unknown alignment in change group alignment %s"`: retarget a group's
  alignment in the relation matrix.
- **Camera / cutscene** (verbs 18, 94) — `"Can't find camera %s in context
  %s"` (named `Cameras`), `"dynamic\shot.xxx"` (the `dynamic_shot`
  cinematic, `fcn.0047a5b0`); plus the dialogue-staging plate
  `DualDialogPlate` (verb 88, `fcn.005270c0`).
- **Ambience / sound** (verbs 111/112, `fcn.00547c20`) — `"STORY: Unknown
  ambience entry %s"`: set the zone ambience track.
- **Engine objects** (verbs 77/78/79/91, source `.\OSIRIS\osiengine.cpp`,
  alloc `fcn.004fa4b0`) — create/destroy the story's engine-side objects.
- **Region / shroud** (verb 110, `fcn.0053f190` — the shroud reveal range,
  [`formats/shroud.md`](formats/shroud.md)).

The remaining real verbs each call a dedicated handler fcn (recorded in
the jump table `0x513104`) but are not all individually named here — most
take the `fcn.005169a0`-resolve-then-`fcn.00xxxxxx` shape. So the type-12
handler is the story's **engine command set** (85 verbs), the counterpart
to the per-entity NPC/Object handlers — same `switch(idx-1)` shape, a much
wider table.

### Query handlers (how the story reads state)

Slot **1** of each handler is the **Query** path — what the story reads,
the read-only counterpart to the slot-0 Call. The **OBJECT Query**
(`0x5163a0`, type 5) shows the pattern: it resolves the object through the
same object manager `[0x658bdc]` (`fcn.00585d10`), then runs a smaller
`switch` (a **9-verb** jump table `0x516580`, verbs `idx` 9..17) of
read-only probes — property *get* `fcn.00591920` (e.g. property `0x1a`),
flag-bit tests (`[obj+0x30] & (1<<n)`), and lookups into the object/region
tables (`[0x750d38]`, `[0x751614]`). Two things distinguish Query from
Call:

- **No runtime-event logs.** Call cases log (`"Attack %s"`, `"…casting
  spell…"`); Query cases are silent — they only read.
- **A returned value, not a mutation.** Each case writes its result back
  through a **bound out-param pointer** (`[handler+8]` / `[handler+0xc]` —
  one of the arg slots `fcn.00516a60` cached from the `COsiArgumentDesc`
  list) as a **typed osiris value**: `*out = (value << 4) | typeTag`, the
  same low-nibble-is-type encoding as a DIV handle. The handler still
  returns `al` = found/not-found.

This is the runtime realisation of the **out-param mask** in the function
record ([above](#parameter-signature-value-type-list)): the mask marks
which parameters are query results, and the Query handler is what fills
them. So Call (slot 0) = mutate + log + bool; Query (slot 1) = read +
write typed out-param + bool.

The other two Query handlers follow the same convention but use a
**two-level dispatch** — `idx − base` → byte index-map → jump table — and
their verb ids sit in handler-specific bands (so the global DIV id space is
partitioned per handler/slot):

- **NPC Query** (`0x513530`, type 4) — `idx − 17`, byte-map `0x513804` →
  64-slot table `0x5137c4`: **16 distinct cases, 49 no-ops**, ids 17..80.
  Most cases read an agent field directly into the out-param (stat/state
  getters); the named ones delegate — id 26 **get alignment** (`"Querying
  alignment from npc %s but he hasn't…"`), ids 34/35 **inventory queries**
  (the inventory resolver `fcn.004afb70`), id 58 alignment (`fcn.00437fd0`).
- **ENGINE Query** (`0x511260`, type 12) — `idx − 35`, byte-map `0x511b04`
  → 40-slot table `0x511aac`: **22 distinct cases, 19 no-ops**, ids 35..74.
  A long run (ids 57..73) reads engine-global state into out-params (game
  counters/flags), plus id 74 **skill query** (`"Unknown skill %s"`,
  `fcn.00541c00`) and ids 35/36 a registry read (`fcn.00505770`).

With this the **entire DIV Router is enumerated** — Call handlers NPC (93)
/ OBJECT (22) / ENGINE (114) and Query handlers NPC (64-slot) / OBJECT (9)
/ ENGINE (40-slot), plus Dialog (1) — every story↔engine verb accounted
for, even where individual field-getter verbs are not separately named.

### Thin handlers: Dialog (type 6)

Not every handler is a wide verb table. The **Dialog Call** handler
(`0x5104d0`, type 6, source `.\OSIRIS\osidialog.cpp`) decodes the handle
the same way (`word = dialog id`, `idx = verb`) but implements a **single
real verb, `idx 1` = StartDialog**; every other verb id falls through to
`return 1` (no-op). StartDialog looks the dialog up in the dialog manager
`[0x658c40]` (`fcn.004724c0`), allocates a dialog context
(`osidialog.cpp:92`) and starts the session (`fcn.0051d700` /
`fcn.0051bc70`). So Osiris's only dialogue action is "begin conversation
*id*"; the branching itself then runs inside `CDivDialogSystem.dll`
([`dialogue.md`](dialogue.md)), and the branch flags it toggles are
exactly the Osiris **`DIALOG_EVENT`** symbols (`EventChanged`).

## How the rest of the engine plugs in

- **Event manager** `[0x7447dc]` — the hub gameplay events route to
  ([`messages.md`](messages.md)); using a scripted object, combat, etc.
  raise Osiris events.
- **`CDIVINITYOsiris*Function`** — the per-type DIV handler family
  ([above](#the-div-router--story--engine-dispatch-osirisroutercpp)); e.g.
  `CDIVINITYOsirisObjectFunction` (type 5) binds scripted-object use into
  `CObject::Use` ([`object-interaction.md`](object-interaction.md)); the
  `sb_osiris` object flag + `"Object now belongs to Osiris (Key=%d)"`.
- **Quest regions** — `region.000` (`StoryV1.0`,
  [`formats/region.md`](formats/region.md)) are the Osiris trigger areas.
- **Dialogue flags** — `dialogids.dat`'s 1455 named flags are the story
  booleans (`EventChanged`) Osiris gates branches on
  ([`dialogue.md`](dialogue.md)).

## Status

- Role & API ✅ — story rules/DB engine; `OsirisDLL.dll` exports
  recovered (Compile/Event/Load/Merge/RegisterDIVFunctions/Goals/Minilog).
- `story.000` header ✅ — save-format header + version `1.4` + story
  version `2.7.68`; `binary.div` confirmed a build log, not bytecode.
- Architecture ✅ — RETE forward-chaining rules network; the 8
  `story.000` DB sections enumerated from `COsiris::Load` (symbol-named).
- Encoding ✅ — **strings are XOR-`0xAD` obfuscated** (cipher byte stamped
  by `_ReadHeader` per story version: `0xad` for 1.4); integers are raw
  little-endian. Corrects the earlier "not obfuscated" claim — the raw
  byte reader showed no transform because the XOR lives in the *string*
  reader. Once decoded, sections are plain.
- Header layout ✅ — byte-exact: save string (+1), version `01 04` (+54),
  debug bytes (+56), 128-byte story-version buffer (+58), section data
  from +186.
- DIVObject (symbol) table ✅ — `+194`, count 14322, `name+type+id`
  records fully decoded; 11-value type taxonomy verified; the 1455
  `DIALOG_EVENT` symbols == the 1455 `dialogids.dat` flags.
- DIV Router (story↔engine dispatch) ✅ — `RegisterDIVFunctions`
  (`fcn.00516b90`) hands the engine a 4-callback `TOsirisInitFunction`
  {MakeHandle, Call, Query, Error}; a DIV handle's low nibble is its
  target type. `CRouter [0x744808]` (ctor `fcn.00516d70`, `.\OSIRIS\
  Router.cpp`) holds a 14-slot handler array keyed by type code, each a
  `0x404`-byte `CDIVINITYOsiris*Function` (RTTI-verified: Npc=4/Object=5/
  Dialog=6/Region=7/Location=8/NpcClass=9/ObjectClass=10/DialogEvent=11/
  Engine=12/Function=13) with a 2-method vtable (slot 0 = Call, slot 1 =
  Query; arg pointers cached by `fcn.00516a60`). Handle bit-layout
  verified both ways (encoder + decoder): `type` = bits 0..3, `word`
  (entity id) = bits 4..19, `idx` (verb id) = bits 20..29. NPC Call
  (`0x513850`) resolves the agent via `CAgentManager [0x658d50]` and runs a
  **93-case switch** (`0x515b7c`, 78 real verbs + 15 no-ops) — families:
  inventory/items (resolver `fcn.004afb70`), magic/polymorph (`SMagic`),
  set-alignment (60, `fcn.00437ed0`), movement, spawn/treasure/trade,
  flag-writes (verb 54 sets `agent+0x224 0x80000000`), combat (verb 13 →
  `CAgent vtable+0x50`); ENGINE Call (`0x511b30`, type 12) is a
  `switch(idx-1)` over a **114-slot** jump table (`0x513104`, 85 real verbs
  + 29 no-ops) of engine-global actions — families: trap control
  (97/98/104, `fcn.00443250`), faction/alignment (34/38), camera & cutscene
  (18/94/88: `Cameras`/`dynamic_shot`/`DualDialogPlate`), ambience/sound
  (111/112), engine-object create (`osiengine.cpp` 77/78/79/91), shroud
  (110); OBJECT Call (`0x515cf0`) is a `switch(idx-1)`
  over **all 22 object verbs** (jump table `0x516344`) — fully enumerated:
  `sb_*` state mutations via the property API (get `fcn.00591940` / set
  `fcn.005919e0` / refresh `fcn.00585780`) for lock/close/transform/
  visibility (verbs 1–3, 8, 11, 15, 20–21), magic-effect spawns (5, 19 via
  `SMagic`), create/item-link (6, 18), per-agent actions (14), and 5
  no-ops (9, 12, 13, 16, 17); OBJECT **Query**
  (`0x5163a0`, slot 1) is the read path — 9 verbs (jump table `0x516580`),
  property *get* / flag tests, **no logs**, writing the result back through
  a bound out-param as a typed value `(value<<4)|type` (the out-param mask
  realised). Call = mutate; Query = read. NPC Query (`0x513530`, ids 17..80
  via byte-map `0x513804`→`0x5137c4`, 16 cases) and ENGINE Query
  (`0x511260`, ids 35..74 via `0x511b04`→`0x511aac`, 22 cases) likewise
  read agent/engine state into typed out-params (NPC get-alignment/
  inventory; ENGINE game-state + skill query). **Whole DIV Router
  enumerated** (Calls NPC 93/OBJECT 22/ENGINE 114; Queries NPC/OBJECT/
  ENGINE; Dialog 1). Dialog Call (`0x5104d0`, type 6,
  `osidialog.cpp`) is a thin handler — one real verb, `idx 1` = StartDialog
  (dialog manager `[0x658c40]`); the conversation then runs in
  `CDivDialogSystem.dll` ([`dialogue.md`](dialogue.md)).
- DIV function table ✅ — `+606571`, count 1175, fully enumerated; the
  `EFunctionType` taxonomy (Event/Query/Call/Database/Proc/SysQuery/
  SysCall) verified across all records. (This is the table
  `RegisterDIVFunctions` binds into.)
- Function record layout ✅ (from `fcn.100078d0`) — 33-byte fixed prefix
  (8×u32 + type byte at `+0x1c`) + XOR name + parameter signature;
  deserializer chain traced (`_ReadFunctions` → `fcn.100078d0` →
  `fcn.10007240`/`fcn.10007570` → `fcn.10014bf0`). The cipher is proven
  at instruction level. (Fixed an earlier swap of the `+0x14`/`+0x1c`
  fields.)
- Parameter signature ✅ — `u32 L` + `L`-byte out-param mask +
  `u8 nParams` + `nParams×u32 typeCode` (readers `fcn.10004560` +
  `fcn.10018610`); out-param mask flags Query results, type codes reuse
  the DIVObject taxonomy. **All 1175 records chain.**
- Functions section bounds ✅ — `+606571 .. +682095` (75524 B), exact.
- `_ReadReteNodes` located ✅ — section starts `+682095`, count
  **10759** RETE nodes; section order confirmed from `COsiris::Load`
  (DIVObjects → Functions → ReteNodes → ReteAdaptors → ReteDBases →
  GoalData → GlobalActionList).
- RETE node section ✅ (structure + kinds) — `_ReadReteNodes`: `u32
  count` (10759) + per-node `u8 kind tag (1..8)` → 8 polymorphic
  readers. All 8 classes named: CReteFact/CReteEvent (alpha entries),
  CReteDIVQuery/CReteInternalQuery/CReteRelCondition (condition tests),
  CReteAnd/CReteNAnd (beta joins, left+right CReteConnection),
  CReteRuleActionPart (production RHS). Hierarchy + shared CReteNode
  header (u32 id + flag byte) pinned.
- CReteNode header ✅ — `u8 tag + u32 id + u32 + XOR name + flag byte +
  variable connection list` (validated on node 0 = `CReteEvent NewHour`,
  via `fcn.10020f50`).
- RETE node full byte-walk 🟡 — nodes are variable (no size prefix; the
  base reads a variable connection list via `fcn.10005a70`), so reaching
  `_ReadReteAdaptors` needs every kind's read sequence + sub-readers
  tallied. Ongoing — but lower priority than the engine semantics below
  (the compiled network is regenerable from `story.div`).
- Gameplay event queue ✅ — div.exe defers/batches events through an
  `Events.cpp` queue struct `[0x7447d8]` (`+0` cap, `+4` count, `+8`
  buffer; 8-byte `{handle, argDesc}` entries, grown by `fcn.004fa4f0`);
  the flush `fcn.0050fff0` drains it, calling the imported `COsiris::Event`
  (`0x006063e0`) on the singleton `[0x60640c]` per entry — the single
  div.exe→Osiris assert path, mirror of the DIV Router.
- Event raise wrappers ✅ (structure) — `Events.cpp` (`0x50c000`–`0x510200`)
  is **70 raise wrappers** (≈ the 77 type-1 Events; **70 `inc [0x7447dc]`**
  enqueue sites). Each builds a linked list of 12-byte `COsiArgumentDesc`
  nodes (`+0` type byte, `+4` typed value `(arg<<4)|type`, `+8` next) — the
  same typed-value encoding as handles/Query out-params — one node per
  event parameter, so the type bytes give each event's parameter signature.
  🟡 **Naming dead-end:** div.exe is symbol-stripped; wrapper→event-name
  needs the handle↔function-table correlation, not static strings.
- `COsiris` match/fire engine ✅ (cycle mapped) — `COsiris::Event`
  (`0x10010ba0`) → event-node propagate (vtable +0x28) → assert fact
  (`fcn.10029fc0`) → propagate to children (`fcn.10028c80`) → beta
  And/NAnd joins → `CReteRuleActionPart` fires → DIV call list. Standard
  RETE forward-chaining.
- Beta-join algorithm ✅ — `CReteAnd::Insert` (vtable+0x28, `0x100274c0`)
  scans the opposite input's token-memory array (`[mem+0x10..+0x14]`) and
  per token runs the shared match/emit `fcn.10026300` (insert-dedup
  `fcn.10025ff0` + propagate `fcn.1002b5df`); `CReteNAnd` inverts it
  (emit when no right match — the `"valid test fails: NOT"` path). The
  exact join-key comparison inside `fcn.10026300` (which token columns
  must be equal) is not split out field-by-field 🟡.
- Goal/rule grammar ✅ — `Goal(n){INIT{calls} KB{rules} EXIT{calls}}`;
  rules are `condition → action part`; goal state (`RulesDisabled`,
  completed) gates rule enabling, driven by the `SysCall` verbs
  (`ActivateGoal`/`CompleteGoal`/`SetGoalSleeping`).
- Goal record layout ✅ — `_ReadGoalData` (`fcn.100092f0`, 80 B):
  `u32 id + XOR name + state byte + two ref lists + INIT/EXIT DIV-call
  lists` (shared list reader `fcn.1000d210`, also used by
  `_ReadGlobalActionList`). KB rules live in the RETE, not the goal.
- Tail section readers ✅ — adaptors (`fcn.1000d850`→`fcn.10025d80`),
  dbases (`fcn.1000d900`→`fcn.1002b4d0`), goals, global-action list all
  located (`u32 count` + per-record reader).
- Per-node memory ✅ (structure) — a node's fact database is a
  **deduplicated tuple set**: insert core `fcn.10029fc0` builds the
  tuple, `fcn.10025ff0` walks the `+0x1c`-linked chain with an equality
  test (`fcn.100219d0`) and `new`-links a fresh tuple only if absent,
  then `fcn.10028c80` propagates to children. (The exact tuple record
  layout and the And/NAnd join-key match remain undetailed.)
- Adaptor / DBase record contents ✅ — adaptor (96 B) =
  `COsiColumnIdxValuePairList` (column-remap/constant-bind wiring) +
  flags; dbase (88 B) = `COsiValueTypeList` schema + `COsiVector<CTuple>`
  rows (a typed fact relation). And/NAnd join inputs = two
  `CReteConnection` at node `+0x58` (left) / `~+0x6c` (right).
- RETE compiled-save byte-walk 🟡 (background) — variable-length nodes;
  lower priority since the network is regenerable from `story.div`.
- RETE nodes / adaptors / dbases / goals ❓ — sections after `+682170`
  (the typed RETE network + 80 B Goals + global actions) not yet parsed.
- `COsiris` RETE engine internals ❓ — node matching / event propagation
  not yet reversed (symbol-named, so tractable).
