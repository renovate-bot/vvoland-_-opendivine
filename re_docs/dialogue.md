# Dialogue system

Branching NPC conversations. Divine Divinity's dialogue runs in a
**separate DLL**, `DivDialogSystem.dll` (`CDivDialogSystem`) — distinct
from the Osiris story VM. It drives the classic structure: the NPC
speaks an **answer** (with a voice clip), the player chooses from N
**questions** (responses), and each choice walks to the next node;
branches are gated by named **dialogue flags**. All text is wide-char
(UTF-16), so it localizes per language.

## API (`CDivDialogSystem` exports)

```text
LoadDialogSystem(path, callback)   load the dialogue database + an event hook
Load(path) / Save(path)            (de)serialize dialogue state
StartDialog(id)                    begin conversation `id`
GetNumQuestions()                  number of player response options now
GetQuestion(i) → wchar_t*          response text i        (UTF-16)
GetQuestionNodeID(i) / GetAnswerNodeID()   node ids for traversal
GetAnswerText() → wchar_t*         the NPC's current line (UTF-16)
GetAnswerSoundName() → char*       voice clip for the line
SelectQuestion(i)                  pick response i → advance the tree
EventChanged(flag, bool)           set a dialogue flag (gates branches)
ResetDialogState()
```

So the runtime model is: `StartDialog` → loop { read `GetAnswerText` +
play `GetAnswerSoundName`; list `GetQuestion(0..GetNumQuestions-1)`;
`SelectQuestion` } until a leaf, with `EventChanged` toggling the flags
that branch conditions test.

## Data files (`localizations/<lang>/`, `main/startup/`)

### `dialogids.dat` — dialogue flag/event registry ✅

The named **dialogue flags** (the `EventChanged` variables) — game-state
booleans that branch conditions check:

```text
u32  version (= 1)
u32  count   (= 1455)
Record[count]:
    u32   name_len
    char  name[name_len]      // no NUL
    u32   id
```

Real entries: `zandalor_player_is_rescued_by` (id 0), `player_is_survivor`
(2), `player_is_warrior` (3), `player_is_mage` (4), `player_is_female`
(5), `arhu_tells_player_about_secret_entrance_to_castle` (6) — i.e.
class/gender checks and quest-progress flags.

The file holds **two** such tables back-to-back: after the 1455 flags
comes a **second `u32 count` (= 490) + records** block in the identical
`{name_len, name, u32 id}` format, holding **NPC / actor names** and
dialogue actions — `Aenora`, `Albin`, `Alchemist`, `AcceptTradeStolenItems`.
The two tables consume the file exactly (60971 bytes). So `dialogids.dat`
= 1455 branch flags + 490 named actors/actions.

### `dialogtxt.dat` — localized text ✅

The actual conversation lines:

```text
u32  version (= 1)
u32  flag    (= 1)
u32  count   (= 7583)               // number of strings
Pair[count]:                        // string table
    u32  offset                     // byte offset into the blob below
    u32  byte_len
u8   blob[]                         // UTF-16LE strings, concatenated
```

`blob` starts at `12 + count*8`; string *i* is
`blob[offset_i : offset_i + byte_len_i]`. Verified: the last pair ends
exactly at EOF.

**The text is obfuscated**: every byte is `+1` (so decode = subtract 1,
*then* UTF-16LE) — the same trivial cipher Larian uses for `.cmp`
archive names ([`formats/cmp.md`](formats/cmp.md)). Decoded samples:

```text
[200]  "Greetings, my friend. Please tell me your name."
[2]    "Let's just say I'm naturally resourceful..."
[1000] "They do say that the sewers under the town connect to ancient
        dwarf-built passages. …there's a secret underground entrance to
        Stormfist Castle, if you know where to look."
```

7583 strings, fully extractable.

### `dialogs.000` / `dialogs.div` — the node graph ✅ (format decoded)

The on-disk format is now fully decoded end-to-end: the **header + root
index** (below, verified against the shipped file) followed by **per-node
bodies** whose exact field sequence is enumerated further down. Replaying
the per-node reader over the bodies reconstructs the graph — i.e. the
format is reimplementable; only a mechanical full-file byte-walk (vs. the
documented read order) is left.

`dialogs.div` (source) and `dialogs.000` (compiled) hold the **node
graph**. The loader is `CDivDialogSystem::LoadDialogSystem` →
`fcn.10003c00`, which opens `\dialogs.div`, reads the node count, and
**loops a per-node reader `fcn.10003bc0`** — `new(0x68)` (a **104-byte
node**) + the field reader `fcn.10002f90` + list insert.

**On-disk layout (decoded from the shipped `main\startup\dialogs.000`):**

```text
u32  version = 1
u32  count   = 39                       — number of root/index entries
{ u32 node_id; u8 flag } × count        — sorted node-id index (ids
                                          monotonic: 2,3,4,5,48,71,172,…;
                                          flag ∈ {0,1})
… node bodies …                         — read by fcn.10002f90 (the
                                          104-byte node fields, below)
```

The 39 index entries are the **conversation roots** (the `node_id`s
`StartDialog` is called with), each with a `0/1` flag; the per-node bodies
that the field reader consumes follow the index. (Verified against the
shipped file: the index parses to offset 203, after which the first node
body begins.)

Read primitives in this DLL: **`fcn.10001060` = read u32**,
**`fcn.1000586b` = read string-ref** (paired with the temp-free
`fcn.10001040`). The node field reader fills the 104-byte node:

The read order is **exact** (`fcn.10001060` = read u32, `fcn.1000586b` =
read string-ref, `fcn.10002f10`+`fcn.10001c20` = read the `std::string`
name); the full 104-byte node is:

```text
+0x00  u32          scalar A    (read first)
+0x04  u32          scalar B
+0x09  u8           bool        = (next u32 read == 1) — a node flag
+0x0c  std::string  name        "fd_<n>" (SSO: data @+0x0c, len @+0x14, cap @+0x18)
+0x28  u32          +0x2c  u32          — two scalars, loaded-but-unread (see below)
+0x30  u32 count1   +0x34  ptr  list1   — condition list (gate, see below)
+0x38  u32 count2   +0x3c  ptr  list2   — condition list (gate, see below)
+0x40  u32 count3   +0x44  ptr  list3   — condition list
+0x48  u32 count4   +0x4c  ptr  list4   — condition list
+0x50  u8           +0x51  u8
+0x54  u32          weight/priority — **defaults to 1000000 (0xf4240)**
+0x58  u32          +0x5c  u8
+0x60  u32          nLinks (child-link count)
+0x64  string-ref / links — followed by the link list (fcn.100010c0 loop)
```

So a node has the answer-text/voice strings at `+0/+4`, the node-id at
`+0x08`, the inline `std::string` **name** (`fd_<n>`) at `+0x0c` (length at
`+0x14`), then **four `{count, list-ptr}` condition lists** at
`+0x30/+0x34`, `+0x38/+0x3c`, `+0x40/+0x44`, `+0x48/+0x4c` (the loader reads
them in that order: a `u32` count then a list reader), two inert
load-only `u32`s at `+0x28/+0x2c`, **flag bytes** at `+9/+0x50/+0x51/+0x5c`,
and the
**weight/priority** `u32` at `+0x54` (default 1e6). The runtime
answer-text/sound accessors resolve `+0/+4` against the string pool (a
post-load fixup turns the loaded atoms into pointers). **`+0/+4` are
`dialogtxt.dat` indices, not inline strings — verified against the shipped
`dialogs.000`**: an ASCII scan of the whole node region returns *zero*
string runs (the node names `fd_<n>` and voice filenames that inline
strings would show are absent — they survive only in the uncompiled
`localizations/<lang>/dialogs.div` source), and the first node body's
leading `u32` is **7583 = exactly `dialogtxt.dat`'s entry count** — the
one-past-end "no text" sentinel, tying the atom space to `dialogtxt`'s
index range `[0, 7583)`. **Correction:**
`+0x34/+0x3c/+0x44/+0x4c` are *not* "answer text / voice clip" string-refs —
the answer text is `+0x00` and the voice clip `+0x04` (established below);
those four are the **condition-list pointers** the gate evaluator walks.

**Node-id is `node+0x08` (settled from the DLL exports).** Disassembling
the symbol-named getters in `DivDialogSystem.dll` pins it directly: the
real `GetAnswerNodeID` body (`0x10002060`, behind the export thunk at
`0x10005770`) and the real `GetQuestionNodeID` body (`0x10002120`, thunk
`0x100057b0`) both resolve the current/indexed node through a runtime
**node map at `CDivDialogSystem+0xa4`** and return **`[node+0x08]`** —
`GetAnswerNodeID` ends `mov eax,[esi+8]`, `GetQuestionNodeID` ends
`mov eax,[edi+8]; ret 4` (`0x1000222e`). `GetAnswerNodeID` first checks the
**current answer node at `this+0xbc`** (null ⇒ returns `-1`) and reads the
current-answer holder at `this+0xb8`. So the engine-exposed **node-id is
the `u32` at `node+0x08`**, and a child-link's target-id (`link+0`, below)
is therefore a `+0x08` value of some node. So `+0x28/+0x2c/+0x30` are **not**
the node-id; `+0x30`/`+0x38`/`+0x40`/`+0x48` are now pinned as **condition-list
counts** (gate, below) and `+0x28/+0x2c` as two **inert load-only `u32`s**
(authoring metadata, no runtime reader — see below).

### Node gating — `fcn.100040a0` (called from `StartDialog`)

A node is only offered when its **condition lists** pass. `StartDialog`
(`0x100053d2`) calls `fcn.100040a0`, which walks two of the node's
`{count, list}` pairs and evaluates each entry through `fcn.100038e0`:

- **list1** (ptr `+0x34`, count `+0x30`) — a **show-if / required** set:
  the loop bails (node hidden) the moment any entry evaluates **false**.
- **list2** (ptr `+0x3c`, count `+0x38`) — a **hide-if / excluded** set:
  the loop bails the moment any entry evaluates **true**.

Each entry is a 4-byte **condition atom-id** (`list[i]`, stride 4);
`fcn.100038e0` looks it up in a **`std::map` held on the dialog-state
object** (a red-black tree — `+0x08` child links, `+0x0c` key — reached via
`CDivDialogSystem+0xc0`), i.e. the runtime set of dialogue flags/variables.
That `+0xc0` map is constructed at load by the `CDivDialogSystem`
ctor/loader (`fcn.10004c40`, from `LoadDialogSystem`, calls the `std::map`
ctor `fcn.10002690` on the system's member maps at system `+0x28`/`+0x48`/
`+0x68`/`+0xc0`) and torn down by the system dtor (`fcn.10003a10`,
`CDialogSystem::virtual_0`) — these are **system fields, not node fields**
(an offset collision with the node layout). A present key ⇒ the condition is
satisfied. Two boolean node flags also gate here: `+0x51` together with the
`+0x08`/`+0x5c` bytes.

Once a node passes the gate, **`fcn.100041c0` orders the eligible nodes by
the `+0x54` weight/priority** (default `0xf4240`): it walks the collection
comparing `[node+0x54]` between candidates (`cmp ecx,[eax+0x54]` with
`jge`/`jle` at `0x100042aa`/`0x10004386`), so `+0x54` is the selection sort
key. `fcn.100041c0` is called from **both `StartDialog` and `SelectQuestion`**
— it builds/advances the offered-question list, re-applying the `+0x30`/`+0x38`
gates and the `+0x54` order. So the runtime dialogue flow is
**gate-filter (`fcn.100040a0`) → weight-order (`fcn.100041c0`)**.

**All four `{count, list}` pairs are condition lists** evaluated by the same
`fcn.100038e0` (atom-id → dialog-state map lookup). The remaining two
(list3 `+0x40/+0x44`, list4 `+0x48/+0x4c`) are walked by **`fcn.10005030`**,
the question/answer **enumeration** routine (called from `StartDialog` and
`SelectQuestion`): it iterates this node's **child links** (`+0x60` count,
`+0x64` list), resolves each child (`fcn.100037b0`) and display-gates it
through `fcn.100040a0`, then evaluates this node's list3/list4 via
`fcn.100038e0`. So lists 1&2 are the **display gate** (does this node
appear — `fcn.100040a0`), lists 3&4 a **second condition set checked during
question enumeration/selection** (`fcn.10005030`); both pairs read the same
flag map, only the evaluation phase differs. They are conditions, **not**
on-select effects (they evaluate the map, they don't mutate it).

**`node+0x28/+0x2c` are loaded-but-unread** (two `u32`s read by the loader
`fcn.10002f90` via the read-`u32` helper, *not* a container). No function in
`DivDialogSystem.dll` reads them on a node, and no export surfaces them
(the accessors return `+0x00`/`+0x04`/`+0x08`), so they cannot reach the
engine either — i.e. **inert authoring/compile metadata** carried in the
node record but unused at runtime, like the answer-authoring fields noted
elsewhere.

*Offset-collision caveat (verified three times — do not re-misread):*
`fcn.10003a10` (dtor `CDialogSystem::virtual_0`), `fcn.10004c40` (from
`LoadDialogSystem`) and `fcn.10004130` (from the enumeration) all read
`+0x28`/`+0x48` on the **CDivDialogSystem object**, whose `std::map`
members sit at those offsets (`+0x28`/`+0x48`/`+0x68`/`+0xa4` node-map/
`+0xc0` flag-map; ctor `fcn.10002690`, the RB-tree). `fcn.100037b0` is the
**node-map lookup** on that system object — it is *not* a node-field
reader. The same numeric offsets on a dialogue node are unrelated.

**Two string fields labelled** (from the runtime accessors, on the map
node): the **answer text is `node+0`** — `GetAnswerText` (`0x10004780`)
finds the current node and returns `[node+0]`, a wide string; the
**voice-clip name is `node+0x04`** — `GetAnswerSoundName` (`0x10004800`)
reads a `char` `std::string` there (the `cmp [+0x18], 0x10` is the SSO
capacity check, so `+0x04`=data ptr / inline buffer, `+0x14`=length,
`+0x18`=capacity). So a node leads with `wstring answerText` then
`string soundName`; the `+0x28…` u32 group are the id/type/flags scalars
and the later refs are secondary text.

**Child link = 2× u32** (the reader `fcn.100010c0` reads `[link+0]` and
`[link+4]`). `GetQuestionNodeID` returns the **first** u32, so
`+0 = target node id` and `+4` is the gating field (a condition / flag
id — the `dialogids.dat` / Osiris `DIALOG_EVENT` hook).

At runtime the loaded tree lives in a `CDivDialogSystem` **singleton**
(`[0x10017140]`): the node map is at instance `+0xa4`, the question/link
structures at `+0x88`/`+0x9c`, the **current answer node** pointer at
`+0xb8`, and a valid flag at `+0xbc` (`GetAnswerNodeID` returns the
current node's id, the map key, or `-1` when none).

### Branch gating (how questions are filtered)

- **Dialogue flags** are a `std::map<int, bool>` at **`this+0xc0`**.
  `EventChanged(id, value)` (`fcn.10004aa0`) is a one-liner: `ecx += 0xc0`,
  `slot = mapLookup(&id)` (RB-tree find/insert `fcn.100038e0`), then
  `*slot = value`. The `id` is a `dialogids.dat` flag id / Osiris
  `DIALOG_EVENT` id ([above](#dialogidsdat--dialogue-flagevent-registry-)).
  So `EventChanged` **only writes the flag map** — it does not re-filter
  options already on screen.
- **The question set is a second `std::map` at `this+0x9c`** (tree header
  `this+0x88`); each entry's **visible byte is `+0x10`**. `GetNumQuestions`
  (`fcn.100020a0`) and `SelectQuestion` (`fcn.10005480`) both just walk
  that tree (successor helper `fcn.1000639e`) and act on entries with
  `byte[+0x10] != 0` — they **read** precomputed visibility, never
  evaluate a gate at query time.
- **Both maps are MSVC `std::map<int,V>`**, node layout pinned from the
  tree code: `+0x00` left, `+0x04` parent, `+0x08` right, `+0x0c` **key**
  (`int`), `+0x10` **value**, `+0x14` color byte, `+0x15` isNil byte. So
  the flag map (`+0xc0`, `value` = bool) and the question map (`+0x9c`,
  `value` = the `+0x10` visible byte) share this node shape; the successor
  walk `fcn.1000639e` and find/insert helpers (`fcn.100038e0`) are the
  generic `std::_Tree` primitives.
- **`fcn.100037b0` is `std::map::operator[](key)`** — *correcting* the
  earlier "rebuilds the question map" label. It tree-walks by `key`
  (`[node+0xc]`), inserts a zero-value node via `fcn.100035e0`
  (→ RB-rebalance `fcn.10002950`) if absent, and returns **`&value`**
  (`node+0x10`). So it is the subscript primitive the node-entry code uses
  to *assign* each question's visible byte (`question_map[qid] = gate`),
  not the gate evaluator itself.
- **Visibility is recomputed on node entry**: a flag set by `EventChanged`
  takes effect on the **next** node entry (the `+0x10` byte is a per-entry
  snapshot), via `question_map[qid] = gateResult`.
- Enable-pass ✅ (gate confirmed) — **`fcn.10004130`** (DivDialogSystem.dll)
  is called from **`StartDialog`** (`0x1000542c`) and **`SelectQuestion`**
  (`0x1000560f`) — i.e. on every dialogue advance. It walks the node set via
  `operator[]` `fcn.100037b0` and sets each entry's **enable byte at `+9`**:
  it writes `0` (disabled) **iff `[node+0x30] == 0 && [node+0x38] == 0`**,
  else `1` (enabled). So a node shows unless **both** of its condition
  fields `+0x30` / `+0x38` are zero — confirming `+0x30`/`+0x38` as the
  **gate pair** and the on-disk field-read order (the node reader
  `fcn.10002f90` reads `+0x28`,`+0x2c`,`+0x30` as three consecutive `u32`s,
  then a string-ref `+0x34`, then the second gate `u32` `+0x38`).
  **Open point (a) resolved:** `+0x30`/`+0x38` are read **from `dialogs.000`
  by the node reader `fcn.10002f90`** (they are two of the per-node
  `u32`s, decoded above) — so they are **static, file-baked link gates**,
  **not** recomputed from `EventChanged` flags at runtime. `fcn.10004130`
  gates the question purely on these static fields. So within the DLL the
  question visibility is **static**; the `EventChanged` flag map (`+0xc0`,
  write-only in the DLL) integrates **host-side** (div.exe, driven by Osiris
  `DIALOG_EVENT`) — a reimplementation gates on its own flag state, so this
  is not a reimplementation blocker. (b) The `+9`-vs-`+0x10` two-byte
  detail (different objects/passes) is the only un-pinned micro-detail.
- **Flag-map ↔ gate bridge is host-side (narrowed).** `EventChanged`
  (`0x10004aa0`) is confirmed to be the **only** writer of the flag map: it
  does `ecx += 0xc0` then `std::map` find-or-insert (`fcn.100038e0`) and
  stores the `bool` — `flagMap[+0xc0][key] = value`. A pattern sweep of the
  whole DLL for any other `+0xc0` accessor (`add/mov/lea reg,0xc0`) finds
  **none** — so within `DivDialogSystem.dll` the flag map is *write-only*.
  The flags→`+0x30`/`+0x38` derivation therefore does **not** happen via an
  in-DLL flag-map read; it is **host-mediated** — `div.exe` calls
  `EventChanged` to record a flag *and* (separately) populates the nodes'
  gate fields, or passes them at `Load`/`StartDialog`. So open point (a) is
  narrowed from "unconfirmed" to "**not in the DLL** — the bridge lives in
  the `div.exe` story/dialogue glue", which is the correct place to look
  (and is consistent with quest/dialogue flags being Osiris story state,
  [`osiris.md`](osiris.md) / [`variables.md`](variables.md)).
- **Flag source pinned (host side) ✅.** The `div.exe` caller of the
  `EventChanged` import thunk (`fcn.00472550`) is **`fcn.0050c0a0` =
  `CDIVINITYOsirisDialogEventFunction::virtual_0`** — the Osiris
  **DIALOG_EVENT** handler (type 11 in the [osiris.md](osiris.md)
  DIV-function table, vtable slot0 `0x50c0a0`). So the full flag-input
  chain is now end-to-end: a story rule fires a `DIALOG_EVENT`
  DIV-function → that handler (`fcn.0050c0a0`) → the `EventChanged` wrapper
  (`fcn.00472550`) → `DivDialogSystem.dll::EventChanged(flagId, value)` →
  `flagMap[+0xc0][flagId] = value`. This **closes the flag *source***:
  dialogue visibility is driven by the Osiris story exactly as the
  [Journal](quest-log.md) is. The only residual is the *intra-DLL* step
  that turns `flagMap[+0xc0]` into the per-question gate bytes
  (`+0x30`/`+0x38` → enable `+9` → visible `+0x10`), done via generic
  `std::map` machinery without a re-readable `+0xc0` access — an
  implementation detail, not a missing link in the model.

So the full runtime loop is reimplementable: load 104-byte nodes + 8-byte
links into a node map keyed by id; on entering an answer node, evaluate
each child link's `+4` gate against the flag map to set its visible byte;
`GetAnswerText`/`GetNumQuestions`/`GetQuestion` present the line and
visible options; `SelectQuestion` advances; `EventChanged` toggles flags.
Remaining fine detail: which scalar u32 is node id vs. type, which
string-ref is answer-text vs. voice-clip, and the gating expression form.

## Engine bridge — the conversation manager (`.\Dialog\dialogman.cpp`)

The `DivDialogSystem.dll` above is driven from `div.exe` by a thin
**conversation-manager** wrapper unit (`.\Dialog\dialogman.cpp`, the
`fcn.00472xxx` cluster — the same unit as the `DialogLog`
[`formats/savegame.md`](formats/savegame.md)). The boot *"Loading
conversation manager"* banner is a **no-op** (no loader call); the DLL
singleton (`[0x10017140]`) is built on demand. The 13 imported methods are
thunked at IAT `0x606000..0x60602c` (ctor/dtor, `LoadDialogSystem`,
`StartDialog`, `GetQuestion`, `GetAnswerText`, `GetAnswerSoundName`,
`GetNumQuestions`, `SelectQuestion`, `EventChanged`, `Save`, `Load`).

**Opening a conversation** (`fcn.004724c0`): resolve the target NPC through
the agent manager `[0x658d50]` (`mgr+0xc + idx*4`), set the conversation
context on the **player** (`[0x658c04]+0x47c`, via `fcn.004a3ca0(player, 2)`
— the player-state field from PlayerInfo, [`formats/savegame.md`](formats/savegame.md)),
then call `StartDialog(rootNodeId)` with the root from the manager
(`mgr+0xc`).

**Loading the tree** (`fcn.00472940`, the DialogLog/restore path → wrapper
`fcn.00472750`): destruct + reconstruct the DLL singleton and call
`LoadDialogSystem(file, callback)`. The dialog file is **gender-specific**
(a `"\female"` / `"…ale"` suffix is appended, like the monologues
[`monologues.md`](monologues.md)), and the `(int, bool)` **callback** is the
dialogue→engine event hook fired when a node triggers a flag/event.

**Osiris ↔ dialogue is bidirectional:**
- **Osiris → dialogue:** `fcn.0050c0a0` forwards a story flag/event change
  into the DLL — it reads `(eventId, value)` from the Osiris event record
  and calls `EventChanged` (via the `fcn.00472550` thunk), so story
  progress updates which options are enabled (the `EventChanged` flag-map
  write at DLL `this+0xc0`, above).
- **dialogue → Osiris:** the `LoadDialogSystem` callback (and the
  `link+4` gate / `DIALOG_EVENT` ids in `dialogids.dat`) raise story events
  when a node is taken.

The per-turn GUI loop then uses the thin wrappers — `GetNumQuestions` /
`GetQuestion(i)` / `GetQuestionNodeID(i)` to list options, `SelectQuestion(i)`
to choose, `GetAnswerText` / `GetAnswerSoundName` / `GetAnswerNodeID` for the
NPC reply (node-id `+0x08`, above) — and `Save`/`Load` persist the dialog
state with the savegame.

## Status

- Architecture ✅ — branching tree engine in `DivDialogSystem.dll`,
  separate from Osiris; API recovered from exports.
- `dialogids.dat` ✅ — two `{name_len, name, u32 id}` tables: 1455
  branch flags + 490 NPC/actor names; consumes the file exactly.
- `dialogtxt.dat` ✅ — fully decoded: header + `count×{offset,byte_len}`
  table + UTF-16LE blob, obfuscated `+1`/byte; all 7583 lines
  extractable (verified to EOF).
- `dialogs.000` / `dialogs.div` node layout 🟡 — parser pinned:
  `LoadDialogSystem` → ctor `fcn.10004c40` (builds 7 member vectors) →
  loader `fcn.10003c00` (opens `\dialogs.div`, loops per-node reader
  `fcn.10003bc0` = `new(0x68)` + field reader `fcn.10002f90`). The
  **104-byte node** structure is mapped (≈12 u32 fields, several
  string-refs, a weight defaulting to 1e6, and an `nLinks (+0x60)` +
  8-byte-link array `(+0x64)` child list). Read primitives:
  `fcn.10001060` (u32), `fcn.1000586b` (string).
- Child-link record ✅ — 2× u32 (`fcn.100010c0`); `+0` = target node id
  (returned by `GetQuestionNodeID`), `+4` = gating field. Node header
  carries a `std::string` name (`fd_<n>`) at `node+0x14`.
- Runtime instance ✅ — `CDivDialogSystem` singleton `[0x10017140]`: node
  map `+0xa4`, link/question maps `+0x88`/`+0x9c`, current node `+0xb8`,
  valid flag `+0xbc`. `SelectQuestion(i)` → current node = link i target.
- Branch gating ✅ — dialogue flags are a `std::map<int,bool>`
  (`fcn.100038e0`), set by `EventChanged(id,val)`; each question has a
  visible byte at `+0x10` that `GetNumQuestions`/`SelectQuestion` filter
  on; the link `+4` gate is evaluated vs. the flag map on node entry to
  set it. Full runtime loop is reimplementable.
- Node string fields ✅ — `answerText` (wide string) at `node+0`
  (`GetAnswerText` `0x10004780`); `soundName` (`char std::string`) at
  `node+0x04` (`GetAnswerSoundName` `0x10004800`, SSO via `+0x14`/`+0x18`).
- Node field layout ✅ (complete) — the 104-byte node is mapped field by
  field from the reader `fcn.10002f90`: u32 scalars at `+0/+4/+0x28/+0x2c/
  +0x30/+0x38/+0x40/+0x48/+0x58/+0x60`, four string-refs at `+0x34/+0x3c/
  +0x44/+0x4c`, `std::string` name at `+0x0c`, flag bytes at `+9/+0x50/
  +0x51/+0x5c`, weight/priority u32 at `+0x54` (default 1e6), then the
  link list. (Read with `fcn.10001060`=u32 / `fcn.1000586b`=string-ref.)
- Dialogue maps ✅ — both are MSVC `std::map<int,V>` (node `+0x0c` key,
  `+0x10` value, `+0x14`/`+0x15` color/isnil); `fcn.100037b0` =
  `operator[]` (returns `&value@+0x10`), correcting its prior "rebuilds the
  question map" label. The gate→visible assignment is `question_map[qid] =
  gate`; the gate-*expression* computation (the caller feeding `operator[]`)
  is still 🟡.
- Node-id ✅ — the engine-exposed node-id is the `u32` at **`node+0x08`**,
  settled from the `DivDialogSystem.dll` exports: `GetAnswerNodeID`
  (real body `0x10002060`) and `GetQuestionNodeID` (`0x10002120`) both
  resolve the node via the map at `CDivDialogSystem+0xa4` and return
  `[node+0x08]`. So the `+0x28/+0x2c/+0x30` scalars are **not** the id.
- Node-scalar semantics ✅ (resolved by whole-DLL sweep + gate trace) — id
  is `+0x08`; **`+0x30`/`+0x38`/`+0x40`/`+0x48` are the *counts* of four
  condition lists** (`+0x34`/`+0x3c`/`+0x44`/`+0x4c`), not scalar gate flags.
  The display gate `fcn.100040a0` (from `StartDialog`) walks list1
  (`+0x34`, count `+0x30`, all-must-pass) and list2 (`+0x3c`, count `+0x38`,
  none-may-pass); `fcn.10005030` (from `StartDialog`/`SelectQuestion`) walks
  list3/list4 during question enumeration. Each entry is a condition atom-id
  evaluated by `fcn.100038e0` against the dialog-state map at
  `CDivDialogSystem+0xc0`. *(This supersedes an earlier note that read
  `+0x30/+0x38` as scalar flags gated by `fcn.10004130`; full disassembly of
  `fcn.100040a0` shows they are loop counts, and `fcn.10004130` operates on
  the **CDivDialogSystem** object's maps, not the node.)* The two leading u32s
  **`+0x28` / `+0x2c`** are **load/save-only authoring metadata, inert to
  the shipped runtime**: a sweep of every node-field access in
  `DivDialogSystem.dll` finds `+0x28`/`+0x2c` read *only* by the node
  loader `fcn.10002f90` (the `+0x28`/`+0x2c` hits elsewhere belong to other
  objects — a vtable'd STL container in `fcn.100038e0`, and CRT code above
  `0x10009000`). No navigation, gate, visibility, or `SelectQuestion`/
  `StartDialog` path branches on them, and **no export surfaces them**
  (`GetAnswerNodeID`/`GetQuestionNodeID` return `+0x08`, the text/sound
  getters return `+0/+4`), so the host `div.exe` cannot read them either.
  They therefore round-trip load↔save as editor/authoring fields (most
  plausibly the node's graph-canvas position or an authoring tag) without
  affecting dialogue behaviour — not a runtime node-type discriminator.
