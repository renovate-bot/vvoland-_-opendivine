# Savegame format — `<save>\data.000` and friends

Divine Divinity savegames are not a single monolithic file; they are
a directory containing one file per subsystem. The orchestrating file
is `data.000`, which the engine treats as a sequence of length-prefixed
versioned blocks — one per game subsystem.

```text
<save>/
    data.000               ← the master block list (this doc)
    info.000               ← save-slot meta (timestamp, map id)
    items.000              ← item instance pool
    mapflags.000           ← scripted-flag bitmap, per-cell
    quest_log.000          ← active / completed quest entries
    quickinfo.000          ← player snapshot for the load screen
    telpstates.000         ← teleporter activation state
    dialogs.000            ← dialog visit-state (see formats/dialogs-savegame.md)
    story.000              ← Osiris VM dump (see formats/osiris.md)
    world.x0..x4           ← per-region cell + object grid (see formats/world.md)
    objects.x0..x4         ← per-region object instance state
    extfree.x0..x4         ← per-region object free-list bookkeeping
    shroud.x0..x4          ← per-region fog-of-war state
    inv.b0..b2 / .i0..i2   ← party inventory bag/item pairs
    mapv.0..4              ← per-region map version stamps (see formats/mapids.md)
    static/                ← height.x*, dialogs.000, books.000, osinames.000, osiobjects.000
```

## `data.000` — block list

```text
struct DataFile {
    u32     banner_len;        // = strlen+1 (includes NUL); = 98 in shipped template
    char    banner[banner_len]; // "Divinity Save Game (C)Copyright 2001,2002 Larian Studios,
                               //  All rights reserved - V0.935 25-02-2002\0"
    u32     version_flag;      // 4 bytes raw — observed = 1 in shipped template
    Block   blocks[25];         // fixed sequence, terminated by EOF
};

struct Block {
    u32     name_len;          // strlen(name)+1, INCLUDING the trailing NUL
    char    name[name_len];    // version-tag string read+verified by FUN_004f4d70
    u8      body[…];            // subsystem-specific payload — sized only by the
                               //   matching reader on the engine side
};
```

The banner / version-flag / block-name are all written by
`FUN_004f4c90` (length-prefixed) and `FUN_004f4c50` (raw bytes).
The earlier "h$wa" magic claim was wrong — those 4 ASCII bytes
were a Ghidra-decompiled debug trace string, not file data.

> **Last block quirk.** The 25th block name is stored at
> `div.exe:DAT_006173b0` as `"\xA8" + "PlayerInfoV0.935 25-02-2002\0"`.
> The writer's strlen-walking loop starts at `0xA8` (a non-NUL byte),
> so it ends up emitting `nlen = strlen+1 = 29` (one more than other
> blocks) plus the leading 0xA8 in the name buffer. Readers must
> account for the extra byte.

Each block's version string is its **type tag** AND its compatibility
fence: a savegame from a different engine build with a bumped version
string will fail to load.

The **save** side is the mirror function **`FUN_00502170`**: it *emits*
each section's version header (e.g. `AgentClassesV0.935 25-02-2002`)
just before that section's payload, in the same order `FUN_00502bf0`
reads them. So `FUN_00502170` (save) and `FUN_00502bf0` (load) are the
write/read orchestrators of the whole-world serialization; a reader
matches the header strings each writer emits.

## Observed version strings (in order, from `FUN_00502bf0`)

```text
GlobalVarsV0.935 25-02-2002          read by FUN_004adcb0
AlignmentmanagerV0.935 25-02-2002    read by FUN_00438890
AgentVariablesV0.935 25-02-2002      reads via DAT_00658d50 (NPC manager)
AgentClassesV0.935 25-02-2002        read by FUN_00422dd0 (.\AGENTS\agentmanager.cpp) —
                                     the agent-class registry: a count then per-class records
                                     processed by the 0x412xxx agent-class cluster
                                     (fcn.00412690 / 004124b0 / 00412750 / 00412930); the
                                     class templates NPC/monster stats derive from (monsters.md)
AgentsV0.935 25-02-2002              read by FUN_00559c20 (one u32) + FUN_00426c90 — the
                                     agent-manager loop with the "new npc" sentinels; this
                                     block carries the live agent INSTANCES (incl. the player)
EggsV0.935 25-02-2002                read by FUN_0043e020 (.\AGENTS\eggman.cpp) — the egg
                                     spawn-table snapshot (≈ global\eggs.000, 92-byte records)
MonsterGenV0.935 25-02-2002          read by FUN_00440080
PartyV0.935 25-02-2002               read by FUN_005178b0 (.\PARTY\partyman.cpp) —
                                     the party **roster**: per-member records with scalar
                                     fields read into +0xc/+0x10/+0x14 (fcn.004f4c70), the
                                     member set + slots, NOT the player's stat block (no
                                     +0x94 attribute / +0x1c Level / +0x2c CStats reads here)
SkillsV0.935 25-02-2002              read by FUN_00543620 — per-skill state: manager scalars
                                     +0x20/+0x24/+0x28/+0x34, then a count + per-skill u32
                                     array (the rank store, skills-magic.md)
TimeV0.935 25-02-2002                read by FUN_0050bfd0
GameclockV0.935 25-02-2002           read by FUN_00505b90
TrapsV0.935 25-02-2002               read by FUN_005945c0
TimersV0.935 25-02-2002              read by FUN_005168c0
CountersV0.935 25-02-2002            read by FUN_00510200
ExplosionsV0.935 25-02-2002          read by FUN_00575f30
DoorChestListV0.935 25-02-2002       read by FUN_005a0250
DialogLogV0.935 25-02-2002           read by FUN_00472940
NoMagicZonesV0.935 25-02-2002        read by FUN_0058dd50
MagicV0.935 25-02-2002               read by FUN_004e3180
ProjectilesV0.935 25-02-2002         read by FUN_00564540
PainpointsV0.935 25-02-2002          read by FUN_004fd9c0
AnieffectsV0.935 25-02-2002          read by FUN_004ee3f0
OsirisobjectsV0.935 25-02-2002       read by FUN_00585bb0  (handles to Osiris obj names)
OsirisnamesV0.935 25-02-2002         read by FUN_005860a0
PlayerInfoV0.935 25-02-2002          last block; name buffer prefixed with 0xA8 byte
```

The sequence is hard-coded in the savegame loader at
`div.exe:0x00502bf0` — a producer must emit them in this exact order
or the loader's strcmp at the next block fails.

## Validated block sizes (shipped startup `data.000` = 2,902,490 bytes)

`pkg/assets/savedata` walks the file by anchoring on these exact block
names; the body sizes are inferred from offset deltas:

| #  | block                      | body size  | notes                          |
|---:|----------------------------|-----------:|---------------------------------|
|  1 | GlobalVars                 |     90,705 | `FUN_004adcb0`                 |
|  2 | Alignmentmanager           |    485,157 | `FUN_00438890`                 |
|  3 | AgentVariables             |      2,182 |                                 |
|  4 | AgentClasses               |    438,928 |                                 |
|  5 | Agents                     |    832,479 | the heaviest block             |
|  6 | Eggs                       |    815,216 | **= `global\eggs.000` size**   |
|  7 | MonsterGen                 |     19,933 |                                 |
|  8 | Party                      |         40 |                                 |
|  9 | Skills                     |     86,292 |                                 |
| 10 | Time                       |         24 | raw fwrite of 0x18 bytes       |
| 11 | Gameclock                  |         36 | raw fwrite of 0x24 bytes       |
| 12 | Traps                      |     40,100 |                                 |
| 13 | Timers                     |          4 | u32 count = 0                  |
| 14 | Counters                   |        328 |                                 |
| 15 | Explosions                 |          4 | u32 count = 0                  |
| 16 | DoorChestList              |          4 | u32 count = 0                  |
| 17 | DialogLog                  |          4 | u32 count = 0                  |
| 18 | NoMagicZones               |         32 |                                 |
| 19 | Magic                      |      1,472 |                                 |
| 20 | Projectiles                |          4 | u32 count = 0                  |
| 21 | Painpoints                 |          4 | u32 count = 0                  |
| 22 | Anieffects                 |          4 | u32 count = 0                  |
| 23 | Osirisobjects              |     14,472 |                                 |
| 24 | Osirisnames                |     74,165 |                                 |
| 25 | PlayerInfo                 |    rest of file |                            |

Eggs body has the exact shape of `global\eggs.000` (815,216 B = 4-byte
count + 8861 × 92-byte records) — the savegame embeds the egg
spawn-table with live state: only the **flag byte at record offset
`+0xc`** differs from the shipped `eggs.000` (205 of 8861 records in the
startup template; the reader `FUN_0043e020` tests `flags & 4` there).

**Verified absolute offsets** (the version-string start of each block in
the shipped `data.000`, re-checked directly against the file — all 25
blocks present in this exact order, sizes = the offset deltas above):
`GlobalVars@110 · Alignmentmanager@90847 · AgentVariables@576042 ·
AgentClasses@578260 · Agents@1017222 · Eggs@1849729 · MonsterGen@2664971 ·
Party@2684936 · Skills@2685003 · Time@2771323 · Gameclock@2771373 ·
Traps@2771440 · Timers@2811567 · Counters@2811599 · Explosions@2811957 ·
DoorChestList@2811993 · DialogLog@2812032 · NoMagicZones@2812067 ·
Magic@2812133 · Projectiles@2813632 · Painpoints@2813669 · Anieffects@2813705 ·
Osirisobjects@2813741 · Osirisnames@2828248 · PlayerInfo@2902446`. A
reimplementation can seek directly to any block by its version-string
anchor.

## Block & record reading protocol (unified)

Every versioned token in a savegame — both the top-level **block
headers** *and* the **per-record sentinels** inside a block — is read
through the same pair of primitives:

- **`FUN_004f4d70`** — read a NUL-terminated string and **strcmp it
  byte-for-byte** against the expected tag; on mismatch it aborts with
  `Bad Version : %s instead of %s`. (`FUN_004f5180` parses the embedded
  `V0.935`/date digits via `atof` for finer compatibility checks.)
- **`FUN_004f4c70`** — the thin `fread(size, count)` into a typed slot.

These (with the bulk reader `FUN_004f4c00` and the length-prefixed string
reader `FUN_004f4d10`) are the **`.\MISC\divsave.cpp`** primitive family —
the shared serialization library every block reader calls.

A third divsave primitive, **`FUN_004f4de0`** (called from the load
orchestrator `FUN_00502bf0`), is a **block locator/skip**: it `ftell`s the
current position, `fseek`s to EOF to learn the file length, returns, then
reads tags and `_stricmp`s them while `fseek`ing past non-matching blocks —
i.e. it **scans forward for a named block by tag** rather than requiring
strict order. This is what lets the loader tolerate blocks it doesn't
recognise / find an optional block anywhere after the cursor (a
forward-compatibility seam in the format).

This is why agents persist with a literal **`"new npc"` sentinel before
each slot**: the agent-manager reader (`FUN_00426c90`,
`.\AGENTS\agentmanager.cpp`) loops `count` times doing
`FUN_004f4d70("new npc")` + `FUN_004f4c70(body)` per agent — the exact
same verify-then-read used for blocks. The **`Agents`** block is
`FUN_00559c20` (one u32 → `[0x748a74]`, the global id counter)
**followed by** this `FUN_00426c90` loop (dispatched inline by the load
orchestrator after a `fcn.004f5180("V1.0028")` version gate) — i.e. the
`Agents` block carries the live agent instances, player included. The
**`Eggs`** block is a different reader entirely: `FUN_0043e020`
(`.\AGENTS\eggman.cpp`, manager `[0x65921c]`) — the egg spawn-table
snapshot. A reimplementation therefore needs **one** versioned-token
reader and one agent-slot deserializer (the discriminated union under
`Agents` below), reused throughout.

## Subsystem block layouts

Block-by-block schema, gathered from each named reader. **The full set
is now byte-complete and machine-validated**: a Python walker built from
these traces parses the entire shipped startup `data.000` (2,902,490
bytes) field-by-field to exact EOF — every version token strcmp'd, all
850 agents, 382 classes, 782 traps, 8,861 eggs, 96 skills, and the 5
alignment `ftell`-checkpoints verified. The only delegated interiors are
(a) the Magic block's embedded region-file image (geometry stride
tracked in [`region.md`](region.md)) and (b) the per-class CMagic
`vt+0x2c` payloads attached to in-flight projectiles/spells
([`../skills-magic.md`](../skills-magic.md)) — both empty in the shipped
template.

### `GlobalVarsV0.935 25-02-2002` ✅ (byte-validated)

Reader at `div.exe:0x004adcb0` (`FUN_004adcb0`). Reads 5 u32 globals
into `DAT_006ddd28..38` then tail-jumps to the **variable-manager Load
`fcn.005058a0`** (`.\MISC\vars.cpp`) — the same function the
AgentVariables block uses (it is `CVariableManager` vtable slot 1).

```text
struct GlobalVarsBlock {
    u32  global_var[5];          // → 0x6ddd28..0x6ddd38
    VarTables tables;            // fcn.005058a0
};

struct VarTables {               // fcn.005058a0
    u32  table_count;
    u32  markers[table_count];   // saved heap pointers — only null/non-null matters
    VarTable tables[…];           // one per NON-NULL marker, fcn.005053d0 each
};

struct VarTable {                // fcn.005053d0 — a named variable scope
    u32   var_count;
    LPStr scope_name;
    u32   var_markers[var_count]; // saved pointers again (0 = absent slot)
    Var   vars[…];                // one per non-null marker (12-byte object, vars.cpp:171)
};

struct Var {                     // on disk
    LPStr name;
    u32   value;                 // → var+4
    u32   type_or_flags;         // → var+8
};
```

Shipped startup template: 102 tables, 2,343 variables — walked
byte-exactly by the validation parser.

### `AlignmentmanagerV0.935 25-02-2002` ✅ (byte-validated)

Reader at `div.exe:0x00438890` (`FUN_00438890`), per-entity reader
`fcn.004378b0`. Source path `.\AGENTS\alignment.cpp`.

```text
struct AlignmentBlock {
    u32  relation_count;         // → mgr+0
    u32  entity_count;           // → mgr+0x414
    u32  third;                  // → mgr+0xc (1790 in shipped startup)
    Entity   entities[entity_count];    // 0x28-byte CAlignmentEntity each (score init 0x32)
    Relation relations[relation_count];
};

struct Entity {                  // fcn.004378b0 (alignment.cpp:192/196)
    u32   n;                     // → ent+4
    u8    rows[n][8];            // → ent+0xc — the relation bit-matrix rows
    u32   field_20;              // → ent+0x20
    u32   m;                     // → ent+0x10 — sub-entity count (hierarchy children)
    u32   field_24;              // → ent+0x24
    Sub   subs[m];               // each: { u32 k; u8 rows[k][8]; u32 field_10 } (0x18-byte obj)
    u32   checkpoint;            // fcn.004f4f10 — MUST equal the file offset of this
                                 //   field itself (the writer emitted its own ftell);
                                 //   mismatch → "Mismatch in alignment load"
};

struct Relation {                // on disk (in-memory 0x18 obj + entity links)
    LPStr name;                  // → rel+8
    u32   field_14;              // → rel+0x14
    u32   field_4;               // → rel+4
    u32   entity_idx_a;          // resolved to entity ptrs via fcn.00437c10
    u32   entity_idx_b;          //   → rel+0xc / rel+0x10
    u32   value;                 // → rel+0 (the relation score)
};
```

**`fcn.004f4f10` is a divsave primitive**: `ftell`, then `fread(u32)`,
return `read_value == ftell_before` — a **position checkpoint** the
writer plants (its own `ftell` as a u32). The shipped startup save has 5
entities and all 5 checkpoints verify. Shipped: 5 entities, 895
relations.

### `CountersV0.935 25-02-2002` ✅ (byte-validated)

Reader at `div.exe:0x00510200` (`.\OSIRIS\osicounter.cpp:79/82`). Simple
named-counter list; per entry the **value comes first, then the name**:

```text
struct CountersBlock {
    u32  count;                // 11 in shipped startup
    Counter  counters[count];  // 8-byte objects in memory
};

struct Counter {               // on disk
    u32   value;               // → rec+0
    LPStr name;                // → rec+4
};
```

### `ExplosionsV0.935 25-02-2002` ✅ (fully pinned)

Reader at `div.exe:0x00575f30` (`.\WORLD\explosion.cpp:1155..1167`).
Discriminated union of 5 explosion classes; per-slot **pointer markers**
select occupancy, and each object's body is a **raw heap dump** read by
the shared base Read `fcn.00574090` (all five vtables' `+0x10` virtuals
— `0x574710/0x574840/0x574d70/0x575120/0x5751a0` — call it first and
add **no** further file reads, only FX/sound re-attachment).

```text
struct ExplosionsBlock {
    u32  count;
    u32  slot_markers[count];  // saved pointers; 0 = empty slot, no body follows
    Explosion explosions[…];    // one per non-null marker
};

struct Explosion {
    u32  type;                // 1..5 → size 0x68/0x98/0x78/0x70/0x74
                              //   (Gasoline / WalkingMine / TrailBomb /
                              //    PoisonCloud / DamageCloud; vtables
                              //    0x61e268/0x61e280/0x61e298/0x61e2b0/0x61e2c8)
    u8   body[size-4];        // raw dump of this+4..size; MUST start with the
                              //   magic 0xff00aabb at body+0 (else the reader
                              //   fseeks back — fcn.004f4f50/fcn.004f4f70 mark/
                              //   restore — and reads size-0x10 bytes into
                              //   this+0x10: the pre-magic legacy layout)
    LPStr alignment_name;     // ONLY if the baked pointer at body offset 0x30
                              //   (mem +0x34) is non-null; resolved via
                              //   fcn.00437e40 (alignment.md)
};
```

### Osiris* blocks ✅ (byte-validated)

`OsirisobjectsV0.935 25-02-2002` (reader `0x00585bb0`) and
`OsirisnamesV0.935 25-02-2002` (reader `0x005860a0`,
`.\WORLD\objects.cpp:5976/6337`) are the runtime `(name → handle)` map
for the Osiris VM — the live state of the tables shipped as
`static\osinames.000` / `static\osiobjects.000`
(see [`osi-static.md`](osi-static.md)).

```text
Osirisobjects:  u32 n;  u8 recs[n][8];   u32 trailer   (shipped: n=1808, trailer=1808)
Osirisnames:    u32 n;  u8 recs[n][36]                 (shipped: n=2060)
```

Both use the generic stream `fcn.005e5cc4` / raw `fread` rather than the
divsave `[0x6e0124]` context; the Osirisobjects trailer u32 is
EOF-tolerant (a failed read stores 0).

### `TimeV0.935 25-02-2002`

Reader at `div.exe:0x0050bfd0` (`FUN_0050bfd0`). Trivial: `fread` of
exactly **24 bytes** into the time-manager state (game-time
accumulator, day/night phase, etc.). No structure beyond that.

### `GameclockV0.935 25-02-2002`

Reader at `div.exe:0x00505b90`. `fread` of **36 bytes** into the
gameclock state. The reader then resets the live wall-clock fields
(`+0x14` and `+0x1c` get rewritten with the loaded value, `+0x20` /
`+0xc` get a fresh `timeGetTime()` so the elapsed-time calculation
restarts from save-load.)

### `MagicV0.935 25-02-2002` ✅ (byte-validated; slot body corrected)

Reader at `div.exe:0x004e3180` (`.\magic\SMagic.cpp`), per-slot reader
`fcn.004d4340` (SMagic.cpp:0x3670 alloc site). Loads the system-magic
state. The loop runs over the **engine's compiled spell count**
(`mgr+0x17c` = 128), not the file count (a mismatch only logs
`Amount mismatch in CMagicSemantic::Load()`).

```text
struct MagicBlock {
    u32         expected_count;     // shipped: 128
    SpellSlot   spells[compiled_count];
    char        sentinel_open[]   = "Check teleport regions";   // FUN_004f4d70 token
    RegionFile  teleport_regions;   // FUN_0058e690 — an embedded "Divinity RegionsV1.0"
                                    //   region-file image (region.md); 42 bytes/empty
                                    //   in the shipped startup save
    char        sentinel_close[]  = "End check teleport regions";
    u32         field_at_4;
    u8          state[0x154];       // 340 bytes of magic-system state → mgr+0x28
};

struct SpellSlot {                  // slots are IN-PLACE 0x54-stride entries at
                                    //   mgr+0x180; the 0x54 is a memory stride,
                                    //   NOT an on-disk body size
    u32  active;                    // → slot+0x28
    i32  type_id;                   // -1 or active==0 → nothing more for this slot
    // if type_id != -1 && active != 0 (fcn.004d4340):
    u32  fields[7];                 // → +0x14,+0x18,+0x1c,+0x20,+0x24,+0x28,+0x2c
    u32  field_4c;                  // → +0x4c
    u32  data_len;
    u8   data[data_len];            // → alloc'd buffer at +0xc
};
```

### `SkillsV0.935 25-02-2002` ✅ (byte-validated; corrected)

Reader at `div.exe:0x00543620` (`.\Skills\skills.cpp`). This is the
**global skill manager**, *not* the per-agent spell masks. (The earlier
"count-prefixed array of 8-byte skill records" claim was wrong — the
8-byte array at `+0x14` is alloc'd and `memset` locally, never read from
the file.)

```text
u32   slot_count        → +0x08   (96 in shipped startup)
u32                     → +0x20
u32   list_count        → +0x18   (12 in shipped)
u32                     → +0x24
u32                     → +0x00
u32                     → +0x34
u32[3]                  → +0x28,+0x2c,+0x30
u32[list_count]         → +0x10 buffer  (bulk, elem 4)
SkillRec[slot_count]    → per slot: i32 skill_id, then the skill object's
                          Load virtual (vtable[+0x18]) reads its record
```

`skill_id` selects the global skill object `[0x746800 + id*4]`
(the 96-entry table built by the skills.cpp static initializer at
`0x601360..`); `id == -1` allocates a blank 0x78-byte skill
(skills.cpp:1187, vtable `0x61b440`) instead — **both** paths then run
the same Load virtual. The **base Load `fcn.00541560`** (all 96 classes
inherit it; 7 override it with extra tail fields):

```text
u8    dump[0x74]        raw heap dump → skill+4 (baked name ptrs inside)
LPStr name              → +4   (e.g. "Sword Expertise (Passive)")
LPStr description       → +8
u8    recs[n][8]        n = u32 at dump offset 0x68 (mem +0x6c) → +0x68
LPStr rank_text[5]      → +0xc..+0x1c (the 5 per-rank description strings)
```

Subclass tails (Load overrides, all `call fcn.00541560` first; the
slot→class map recovered from the initializer):

```text
slot 33  fcn.00545800   u32 cnt; u32; cnt × 12-byte records → +0x78
slots 35, 38, 50  fcn.00544f90   + u32 → +0x78
slot 44  fcn.005400a0   + u32 → +0x78; u32 → +0x7c
slot 52  fcn.00544750   + u32 → +0x80; u32 → +0x7c; u32 → +0x78
slot 55  fcn.00544a00   = jmp base (no tail)
```

### `AgentVariablesV0.935 25-02-2002` ✅ (byte-validated)

Dispatched at `div.exe:0x00503229`: after the version-string read it loads
**through the agent manager** — `edx=[0x658d50]; ecx=[edx+0x24];
call [[ecx]+4]` — the manager's **variables sub-object** (`mgr+0x24`,
a `CAgentVariableManager`, vtable `0x60bf98`). Its vtable slot 1 is
**`fcn.005058a0`** — the *same* variable-manager Load the GlobalVars
trailer uses, so the body is exactly the `VarTables` structure from the
GlobalVars section (per-agent scopes instead of global ones). Shipped
startup: 6 tables, 69 variables.

### `DialogLogV0.935 25-02-2002` ✅ (fully pinned)

Two readers run back-to-back (load orchestrator `0x005034bf..`):

1. **`fcn.00472940`** — **path-driven, consumes no save bytes**: it
   rebuilds dialogue-database paths (`localizations\%s` with
   `\male`/`\female`, `storyeditor`, `dbackup.000`) and reattaches the
   dialogue DB via the `DivDialogSystem.dll` Save/Load exports.
2. **`fcn.00477cf0`** (`.\diary\DialogLogMan.cpp:618`, manager
   `[0x659f24]`) — the actual block body, the **diary/dialog log**:

```text
u32   count                       (0 in shipped startup → 4-byte body)
Entry entries[count];             // 0x58-byte objects, per-entry reader fcn.00477920
```

Per entry (`fcn.00477920`, version-escaped with `-1` markers):
`u8[12] → +4; u32 id → +0; u32 a` — if `a != -1` it is the (legacy)
count of the first u32-list; one or two further `-1` escapes select
newer layouts with **two counted u32-lists** (counts ≤ 20, into the
vectors at `+0x10` and `+0x28`); then `u32 m (≤ 300)` and `m` diary-line
records (0x20-byte objects, DialogLogMan.cpp:312), each read by
`fcn.004774e0` (newest) or `fcn.00477580` (older): `{u32; u32; counted
string}`.

### `PlayerInfoV0.935 25-02-2002`

The **last** block (its name buffer is the `0xA8`-prefixed
`DAT_006173b0`). Reader `div.exe:0x004a3aa0` on the player context
(`[0x658c04]`): it reads **4 `u32` fields** (`fcn.004f4c70`) into player
offsets **`+0x50`, `+0x47c`, `+0x484`, `+0x534`**, then the load
orchestrator sets `player+0x478` and triggers a **shroud refresh**
(`fcn.0053f810`, [`shroud.md`](shroud.md)) and a UI rebuild
(`fcn.00501f30`). So PlayerInfo is just **four player-state scalars** — the
bulk of the player is the player *agent*, saved in the Eggs block.

This completes all 26 `data.000` blocks (Osirisobjects/Osirisnames are the
combined Osiris\* section).

### `AgentsV0.935 25-02-2002` ✅ (byte-validated; corrected)

**Correction to the earlier claim** (which had Agents/Eggs swapped): the
Agents block is the global counter **plus the agent instances**. The
load orchestrator runs, in sequence after the token:

1. `FUN_00559c20` — one `u32` → `[0x748a74]` (the global agent
   id-counter);
2. `fcn.004f5180("V1.0028")` — sub-version gate on the block tag;
3. `FUN_00426c90` — the agent-manager **"new npc" instance loop** (see
   the next section).

The agent *instances* (all NPCs and the player) are therefore in
`Agents`; the `Eggs` block is the egg spawn-table snapshot
(`FUN_0043e020`), not agents.

### `AnieffectsV0.935 25-02-2002` ✅ (fully pinned)

Reader at `div.exe:0x004ee3f0` (`.\MISC\Anieffect.cpp:304..342`) — the
active **animation effects**. Per entry a `u32` **tag**: `-1` = empty
slot; tags 2..11 select the class (others are skipped); the **complete
type→size table** (all share base ctor `fcn.004edbf0`):

```text
tag:   2     3     4     5     6     7     8     9     10    11
size:  0x9c  0x70  0x70  0x70  0x80  0x70  0x90  0x78  0x78  0x74
vt:    6163fc 616414 609038 6143d4 61648c 61642c 616444 61645c 616474 6164a4
```

All ten vtables share one Read virtual `vt[+4]` = **`fcn.004ec960`**:
a **raw heap dump** `u8[size-4] → this+4` (no further file reads —
the function then re-attaches the imagelist if `0 ≤ this+0x1c < 7` and
constructs a fresh 0xa8 `CAniObject`, else returns 0 and the entry is
destroyed). So per entry on disk = `{u32 tag; u8 body[size-4]}`.

### `AgentClassesV0.935 25-02-2002`

Reader at `div.exe:0x00422dd0` (`.\AGENTS\agentmanager.cpp`) — the loaded
**NPC/agent class templates**. It frees the old class array (entries
destroyed via `fcn.00412690`; count `[esi+0x2c]`, array `[esi+0x28]`),
reads a `u32` **count** (`fcn.004f4c70`), allocates the array
(agentmanager.cpp:1358), then per class `new`s a **792-byte (`0x318`)
agent-class record** (agentmanager.cpp:1361), constructs it
(`fcn.004124b0`), and deserializes it (`fcn.00412750`, with post-init
`fcn.00412930`):

```text
u32   count            → mgr+0x2c
ptr[count] → CAgentClass records, each 0x318 (792) bytes
              (ctor fcn.004124b0, load fcn.00412750)
```

So the block is `{count, count × CAgentClass record}` — the NPC class
definitions (ai-class, stat template, behaviour refs; cf.
[`../npc-ai.md`](../npc-ai.md)). The full per-class byte layout is in
the `AgentClasses` schema section below (the record is 0x318 fixed bytes
+ strings + a CStats template + per-slot arrays, all pinned).

### `PainpointsV0.935 25-02-2002` ✅ (fully pinned)

Reader at `div.exe:0x004fd9c0` (`.\MISC\painpoint.cpp:519/530..542`) —
the "pain-points". Per entry a **`u32` tag**: `-1` = empty slot (nothing
more), else type 0..4 (`new` sizes: type 1 → `0x54`, all others →
`0x4c`; vtables `0x616ddc/0x616df4/0x616e0c/0x616e24/0x616e3c`). All
five classes share one Read virtual `vt[+0x10]` = **`fcn.004fcce0`**:

```text
u32   count
per entry ×count:
    u32 tag             — -1 → empty slot
    u8  body[size-4]    — raw heap dump → this+4 (size = 0x4c or 0x54,
                          from this+0x44 set by the ctor)
    LPStr alignment     — ALWAYS read (len may be 0); resolved via
                          fcn.00437e40 → this+0x3c
```

### `TrapsV0.935 25-02-2002` ✅ (byte-validated; tail pinned)

Reader at `div.exe:0x005945c0` (`.\WORLD\Trapman.cpp:953/956`), per-trap
deserializer **`fcn.00593b60`** (Trapman.cpp:324..348) — the live traps.

```text
struct TrapsBlock {
    u8    header[0x18];        // → mgr+0..0x17; +0x00 = trap_count (782 shipped),
                               //   +0x10 = flagged_count (68 shipped)
    Trap  traps[trap_count];
    Flag  flagged[flagged_count];   // the previously-unpinned TAIL:
                                    //   8 bytes each {u32 a; u32 trap_index};
                                    //   the loader does trap[trap_index].flags |= 4
                                    //   (bit 2 = triggered/disarmed state)
};

struct Trap {                  // fcn.00593b60
    u8   dump[0x24];           // raw 36-byte trap record → trap+0 (traps.md);
                               //   +0xc = effect_count (baked ptr at +0x20 discarded)
    Effect effects[effect_count];
};

struct Effect {                // the CTrapState effect variants (traps.md)
    u32  type;                 // 0..5 → object sizes 0x1c/0x18/0x18/0x18/0x1c/0x1c
                               //   (vtables 0x620914/0x620920/0x62092c/0x620938/
                               //    0x6208d8/0x6208e4)
    u8   body[size-4];         // raw: 24 B for types 0,4,5; 20 B for types 1,2,3
};
```

Shipped startup: 782 traps carrying 438 effect records, 68 flagged
entries — walked byte-exactly.

### `ProjectilesV0.935 25-02-2002` ✅ (fully pinned; body corrected)

Reader at `div.exe:0x00564540` (`.\WEAPON\projectile.cpp:695/704`) — the
in-flight projectiles, via the **generic serialize stream
`fcn.005e5cc4`** (`(dst, count, elemSize, FILE)` = fread semantics).

```text
u32   count
per slot ×count:
    u32 present?               — 0 → null slot, nothing more
    else: new(0x184) projectile (ctor fcn.00563af0, vt 0x61d6a8), then the
          Read virtual vt[+8] = fcn.005647d0
```

The 0x184 is the **in-memory** object size; the on-disk body is
**field-wise** (`fcn.005647d0`), not a raw dump:

```text
u32        → +0x58
u8[0x58]   → +0xbc          u8[0x58] → +0x114        u8[0x54] → +0x68
u32 ×6     → +0x16c,+0x170,+0x174,+0x178,+0x17c,+0x180
u8         → local           u32 n_magic → local (0..3)
n_magic ×  attached-magic objects → +0x5c/+0x60/+0x64:
    fcn.004c79f0  (.\magic\magic.cpp:599..)  reads {u32 len (clamped 255);
        char class_name[len]}  — a strcmp chain over the ~40 CMagic subclass
        names ("CMagicBanish", "CBlessMagic", "CFireBallSpell", …) news the
        matching class (sizes 0x120 etc.),
    then fcn.004c7840 invokes the object's Load virtual vt[+0x2c] for the
        class-specific payload.
```

So a saved projectile = fixed 293 inline bytes + 0–3 attached magic
objects, each `{LPStr class_name; per-class body}`. The per-class CMagic
`vt+0x2c` Load bodies are the delegated remainder (bounded, one small
reader per spell class — [`../skills-magic.md`](../skills-magic.md));
the shipped startup save has `count = 0`.

### `TimersV0.935 25-02-2002`

Reader at `div.exe:0x005168c0` (`.\OSIRIS\ositimer.cpp`) — the story's
**named timers**. It frees the old timer array (each entry a heap object
with a name string at `+4`), reads a `u32` **count** (`fcn.004f4c70` →
`+0x04`, stored to `+0x08`), allocates a `count`-pointer array, and per
entry `new`s an **8-byte timer record** (`ositimer.cpp:57`):

```text
u32     count                → mgr+0x04 / +0x08
ptr[count] → timer records, each 8 bytes:
    +0x00 u32     id / expiry tick   (init -1, read via fcn.004f4c70)
    +0x04 char*   name               (read via fcn.004f4d10 = the length-
                                       prefixed string save reader)
```

So a saved timer is `{u32 value, name}`. `fcn.004f4d10` is the engine's
**string read primitive** (length-prefixed; the counterpart to the
`fcn.004f4c70`/`fcn.004f4c00` scalar/bulk readers) — it recurs in any
block that stores names.

### `NoMagicZonesV0.935 25-02-2002`

Reader at `div.exe:0x0058dd50` — same count-prefixed-array shape as
DoorChestList, with **4-byte** elements. It frees the old buffer
(`+0x28`), reads a `u32` **count** (`fcn.004f4c70` → `+0x2c`), then bulk-
reads `count` × 4-byte elements (`fcn.004f4c00`, element size 4) into the
`+0x28` buffer:

```text
u32       count   → mgr+0x2c
u32[count] zones   → mgr+0x28 buffer   (region ids where magic is suppressed)
```

The "no-magic zones" are **regions** (4-byte region ids, vs DoorChestList's
2-byte object ids — consistent with the region system,
[`region.md`](region.md)) in which spellcasting is blocked.

**Runtime enforcement (precomputed on movement).** The manager lives at the
global **`[0x751614]`** (count `+0x2c`, region-id buffer `+0x28`). The check
is done **on agent move**, not per cast: `fcn.004281f0`
(`.\AGENTS\agentmove.cpp`) resolves the moving agent's current region via the
region lookup `fcn.0058d620` ([`region.md`](region.md) winding test), then
**scans the no-magic region-id list** (`cmp region, [base + i*4]` over the
`+0x28` buffer) and caches the outcome in the agent's region-state fields
`+0x2c0`/`+0x2c4`/`+0x2c8`. That cached "in a no-magic zone" state is what the
agent dumper `fcn.0042d230` prints as **`"No magic"`**, and what the cast
path consults to suppress spellcasting. So the region→magic block is a
**precomputed per-agent flag refreshed each move**, tying the region system
([`region.md`](region.md)) to magic ([`../skills-magic.md`](../skills-magic.md)).

### `DoorChestListV0.935 25-02-2002`

Reader at `div.exe:0x005a0250` — a tiny, fully-mapped block: a `u32`
**count** (`fcn.004f4c70` → `+0x4c`) followed by a **`u16[count]`** array
(`fcn.004f4c00` with element size 2 → `+0x50`):

```text
u32     count         → mgr+0x4c
u16[count] ids         → mgr+0x50   (door/chest object ids)
```

It stores only the **list of door/chest object ids**, not their state —
the per-door open/locked/broken bits live in the object flags word
(`sb_locked`/`sb_closed`/`sb_broken`, [`../object-interaction.md`](../object-interaction.md))
and persist with the object instances. So this block is the registry of
which objects are tracked doors/chests.

### `PartyV0.935 25-02-2002`

Reader at `div.exe:0x005178b0` (`.\PARTY\partyman.cpp`). The party
manager: it frees the old member array, reads a 5-field `u32` header
(`fcn.004f4c70`, ctx `[0x6e0124]`) into `+0x04` (= **member count**),
`+0x08`, `+0x0c`, `+0x10`, `+0x14`, then allocates **`(count+1)` ×
24-byte records** (`partyman.cpp:521`, `fcn.004fa4f0`) and reads each
member:

```text
header:  u32 count(+0x04), +0x08, +0x0c, +0x10, +0x14
record[count] (24 bytes each):
    +0x00 u32   (member agent id / handle)
    +0x04 u32
    +0x08       (reserved — not read from save)
    +0x0c u32   +0x10 u32   +0x14 u32
```

So a party is `{header, (count+1) × 24-byte member records}` — five `u32`
fields per member (the `+0` member reference plus slot/formation/state
words; their individual meanings are the remaining detail, but the count,
stride, and read order are firm).

### Agent instances (`FUN_00426c90` inside the `Agents` block) ✅ (byte-validated)

Reader at `div.exe:0x00426c90` (`.\AGENTS\agentmanager.cpp:1234..1240`).
Discriminated-union per-slot, one `"new npc"` sentinel each.

```text
struct AgentList {
    u32  agent_count;                 // 850 in shipped startup
    u32  flags_field;                 // 0x352 in shipped startup
    Agent agents[agent_count];
};

struct Agent {
    char version[]    = "new npc";    // LPStr token per slot
    u32  type_tag;                    // 0..3; anything else → "Save file corrupt for agent %d"
    // tag 0: empty slot (no body)
    // tag 1: 0x2e0  (736 B) — basic CAgent      (ctor fcn.0042c420, vt 0x609a5c)
    // tag 2: 0x304  (772 B) — CAgent variant    (ctor fcn.00429130, vt 0x60982c)
    // tag 3: 0x620 (1568 B) — party/player class (ctor fcn.0042a7f0, vt 0x6098cc,
    //                          .\AGENTS\agentparty.cpp)
    u8   body[…];                     // via vtable[0x68] Read — tag 1: fcn.0042eac0
                                      //   (CAgent::Read); tag 2: fcn.00429200; tag 3: fcn.0042a990
};
```

Shipped startup: 850 slots = 849 tag-1 NPCs + 1 tag-3 (the player).
After the loop: a distance-based activation pass (no reads) and an
optional `dat\classpatch.txt` apply pass for legacy save migrations.

**`CAgent::Read` (`fcn.0042eac0`, `.\AGENTS\agents.cpp`) — complete byte
sequence.** The leading read is a **raw heap dump** of the whole object:
`fread(type_size − 4)` into `this+4` (`type_size` = the tag's 0x2e0/
0x304/0x620, stored at `+0x2a8`). Every "is this sub-object present?"
decision afterwards branches on the **baked pointers inside that dump**;
counts come from dump fields. In file order:

```text
u8    dump[type_size-4]        → this+4  (coords +0x14..+0x20 clamped to ±0x400 after)
CStats record                  the +0x2c stats object IS deserialized here:
                               dump.+0x28 != 0 → new(0x8c) ctor fcn.0055a190, else
                               new(0x11c) ctor fcn.0055a1f0; then its vtable[+0x30]
                               Read runs — fcn.00559f10 (small): u8[0x88] → stats+4,
                               then n×20-byte records (n = u32 at stats body 0x78);
                               fcn.00559e60 (big): u8[0x118] → stats+4, then n×20
                               (same n offset), zeroing +0xc4/+0xf0 blocks + negative
                               clamps. **This corrects the earlier "CAgent::Read does
                               not deserialize CStats / stats are template-derived"
                               claim — the CStats body is right here in the record.**
LPStr alignment_name           → resolved via fcn.00437e40 → +0x30 (alignment.md)
LPStr name                     → +0x21c
u32   npc_table_index          → resolved via [0x658d50]+0x28 table → +0x228
                               (a manager index, not an inline inventory; items still
                               live in the object/world save and link by reference)
if dump.+0x25c:  u8[12]        → the position/cell sub-object (new 0xc ctor
                               fcn.0041e450; the 12-byte read is fcn.0041e4d0)
(if dump.+0x258: construct the 0xd8 runtime sub-object fcn.00435280 — NO bytes)
if dump.+0x2b4:  u8[2×n]       n = i16 at dump.+0x2b8 — the agent-script variable
                               token stream (+ in-memory pointer fixup fcn.0042fdc0,
                               "ConvertScriptPointersFromReading")
if dump.+0x2b0:  LPStr keyname → the "<class><sex>.key" animation-keytable name
                               (fcn.0042c8e0 loads the .key FILE, heroes.md; only the
                               name string comes from the save)
u32[dump.+0x2c4]               → +0x2c0 bulk
u32[dump.+0x2d8]               loop-point pointer markers → +0x2dc
per non-null marker:           { LPStr label; u32 value }   (fcn.0042ff20 — the
                               behaviour-program loop points, agent.md)
(behaviour factory fcn.0042cb80 → +0x260, cell-grid re-register fcn.004270a0,
 AI data-patch fcn.0042f8f0 — all no-read)
```

**Subclass tails** (missed before — tags 2/3 override `vtable[0x68]`):

```text
tag 2  fcn.00429200:  CAgent::Read, then bulk 2-byte elems × (2·dump.+0x2f4)
                      → +0x2ec buffer (4·[+0x2f4] bytes total)
tag 3  fcn.0042a990:  tag-2 Read, then:
    u32  class_index          (→ fcn.00424580 hero class setup if != -1)
    u32  flag                 (→ +0x610)
    if flag:
        u8[0x200]             → +0x420   (512-byte player state block)
        CStats record         big variant (0x118 + n×20) → +0x428 — the PLAYER's
                              second stats object, read via the same vt[+0x30]
        LPStr alignment_name  → resolved → +0x42c
```

### `EggsV0.935 25-02-2002` ✅ (byte-validated; corrected)

Reader at `div.exe:0x0043e020` (`.\AGENTS\eggman.cpp:57`, manager
`[0x65921c]`) — **not** the agent loop (that is the `Agents` block).
The egg spawn-table snapshot:

```text
u32  count;                    // 8861 in shipped startup
u8   eggs[count][0x5c];        // 92-byte egg records (formats/eggs.md)
```

Body shape = `global\eggs.000` exactly; content differs only in the
per-record **flag byte at `+0xc`** (the reader tests `flags & 4`; 205 of
8861 records differ in the startup template) — the live spawn state.

### `MonsterGenV0.935 25-02-2002` ✅ (byte-validated; corrected)

Reader at `div.exe:0x00440080` (`.\AGENTS\monstergen.cpp:528..540`).
The records are **variable-length** (they contain names and counted
sub-lists), not the flat 24/40-byte structs claimed before; 0x18/0x28
are only the in-memory object sizes.

```text
struct MonsterGenBlock {
    u32   count_b;                 // → mgr+4   (B records; 99 shipped)
    u32   count_a;                 // → mgr+0xc (A records; 158 shipped)
    RecA  a[count_a];              // fcn.0043e610
    RecB  b[count_b];              // fcn.0043eb30
};

struct RecA {                      // fcn.0043e610 (0x18-byte object)
    u32   field_0;                 // (float slot)
    LPStr name;                    // → +4
    u32   n1;                      // → +0x10
    u32   n2;                      // → +0x14
    u8    list1[n1][8];            // → +0x08   (only if n1)
    u8    list2[n2][8];            // → +0x0c   (only if n2)
};

struct RecB {                      // fcn.0043eb30 (0x28-byte object)
    LPStr name;                    // → +0x24
    u32   f0, f4, f8;              // → +0, +4, +8
    u32   nsub;                    // → +0x1c
    u32   f20;                     // → +0x20
    u32   fc;                      // → +0xc
    u8    f10[8];                  // → +0x10 (double)
    RecA  subs[nsub];              // nested A-records (monstergen.cpp:196/200)
};
```

### `DoorChestListV0.935 25-02-2002`

Reader at `div.exe:0x005a0250` (`FUN_005a0250`). Trivial:

```text
struct DoorChestBlock {
    u32   count;
    u16   handles[count];          // door / chest object handles
};
```

### `ProjectilesV0.935 25-02-2002`

Reader at `div.exe:0x00564540` (`.\WEAPON\projectile.cpp`).
Discriminated union — non-zero tag → 0x184 (388-byte) `CProjectile`
body + virtual `Read` via vtable[8].

```text
struct ProjectilesBlock {
    u32   count;
    Projectile  projectiles[count];
};

struct Projectile {
    u32   tag;                     // 0 = empty slot, non-zero = active
    // if tag != 0: the FIELD-WISE body of fcn.005647d0 (see the main
    // Projectiles section above) — 0x184 is only the in-memory size
};
```

### `PainpointsV0.935 25-02-2002` and `AnieffectsV0.935 25-02-2002`

Writers `FUN_004fcfc0` (Painpoints) and `FUN_004ecfa0` (Anieffects)
both emit:

```text
struct PainAniBlock {
    u32   count;
    Item  items[count];   // (u32 marker_or_handle, opt-virtual sub-body if marker != -1)
};
```

In shipped data both blocks have count=0 → 4-byte body. The per-item
bodies are fully decoded in the Painpoints / Anieffects sections above
(shared Read virtuals `fcn.004fcce0` / `fcn.004ec960`, raw
`size-4` heap dumps; painpoints add a trailing alignment LPStr).

### `PartyV0.935 25-02-2002`

Writer `FUN_005177f0`: a 5-u32 header followed by 5-u32 entries.

```text
struct PartyBlock {
    u32   count;        // header[1] = number of party members
    u32   field_b;      // header[2..5] — 4 more party-manager u32s
    u32   field_c;
    u32   field_d;
    u32   field_e;
    Member members[count];
};

struct Member {        // 5 × u32 = 20 bytes on disk; in-memory stride is 0x18 = 24
    u32  a;            // memory +0
    u32  b;            // memory +4
    u32  c;            // memory +12  (gap at +8 not serialized)
    u32  d;            // memory +16
    u32  e;            // memory +20
};
```

Shipped Party body = 40 bytes = 5 × u32 header + 1 × 5-u32 member.

### Trivial 4-byte blocks

`Timers`, `Counters` (just count + name list), `Explosions`,
`DoorChestList`, `DialogLog`, `Projectiles`, `Painpoints`, `Anieffects`
all share the `[u32 count][per-item …]` shape. In the shipped startup
template every one of these has `count = 0`, so the on-disk body is
exactly **4 bytes** of zeros.

### `AgentClassesV0.935 25-02-2002`

The per-character-class definition table.  Loaded by
`FUN_00422dd0` (matching writer `FUN_00422d90`); each class entry
is read by `FUN_00412750` (writer `FUN_004126c0`).  Most relevant
because the renderer's per-anim-slot direction count (the divisor
into each `.key` group's frame range) lives here — see
`re_docs/render-hero.md` and `pkg/assets/agentclass`.

```text
struct AgentClasses {
    u32   count;            // number of classes (382 in shipped startup)
    Class classes[count];   // variable-stride
};

struct Class {              // fcn.00412750, byte-validated across all 382
    u8     fixed[0x318];    // FUN_004f4c70(buf, 0x318) — raw heap dump
    LPStr  name;            // → +0x114; class name, e.g. "Hero"
    CStats stats_template;  // → +0x120 — RESOLVED (was "behavior TBD"):
                            //   fixed[+0x11c] != 0 → small 0x8c class
                            //   (ctor fcn.0055a190, Read fcn.00559f10:
                            //   u8[0x88] + n×20-byte records, n = u32 at
                            //   stats body offset 0x78); else big 0x11c
                            //   class (ctor fcn.0055a1f0, Read fcn.00559e60:
                            //   u8[0x118] + n×20). This is the per-class
                            //   stat template NPC spawns derive from
                            //   (stats.md / monsters.md).
    LPStr  alignment;       // → +0x124 via fcn.00437e40
                            //   ("Alignment %s not found !!!!" on miss)
    LPStr  animation_set;   // → +0x118; per-anim frame-range tag
    SlotData slot[19];      // for each anim slot s where the BAKED POINTER
                            //   u32(fixed[s*4]) != 0:
                            //     fixed[0x4c+s] × u32 frame-offset values
                            //   (count byte from +0x4c+s, presence from the
                            //   pointer slot — both inside the dump)
};
```

`LPStr` is the engine's standard length-prefixed string from
`FUN_004f4c90`: `u32 len; char bytes[len]` (no NUL).

The 0x318 fixed block holds the bulk of class state.  Known field
offsets inside it:

| Offset | Type      | Field                                   |
|-------:|-----------|------------------------------------------|
| `+0x4c` | `u8[19]` | per-anim-slot direction count (≤ 0x14)   |
| `+0x114` | `u32`   | pointer to class name (string above)     |
| `+0x118` | `u32`   | pointer to animation_set tag             |
| `+0x120` | `u32`   | pointer to behavior object (vtable)      |
| `+0x124` | `u32`   | pointer to alignment record              |
| `+0x12c` | `u32`   | tactics                                  |
| `+0x130` | `i16`   | base attribute (set by case 0xc parser)  |
| `+0x132` | `i16`   | base attribute (paired)                  |
| `+0x15c` | `u32`   | flag-set bitmask                         |
| `+0x160` | `u32`   | flag-set bitmask 2                       |
| `+0x310` | `u8`    | bitfield (case 0x39 / 0x3c)              |

The per-slot direction count at `+0x4c+slot` is the `param_7`
divisor passed to `FUN_0050ac30` (`MakeAnimationDirectionsFromKeys`).
For class 0 ("Hero" / the player) the array is:

```text
slot:  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18
       B  A  Q  D  E  F  H  P              G  C  Z         J  M  U
count: 20 20 20 20  5 20 20  0 20 20  0 20 20 20  0  0 20 20 20
```

Per-slot frame-offset array (each `u32`) maps an animation phase
index to a frame-offset within the layer's `.key` group.  We don't
fully consume it yet; the renderer derives frames from the group's
`grp.Start` directly.

**Behavior block — RESOLVED.** The "behavior object" at `+0x120` is in
fact the class's **CStats template** (small 0x8c "Monster statistics"
class or big 0x11c class, selected by `fixed[+0x11c]`), serialized as
`{u8 body[0x88|0x118]; n × 20-byte records}` with `n` at stats body
offset `0x78` — see the `Class` struct above. All 382 shipped classes
walk byte-exactly with this layout, so `pkg/assets/agentclass` can now
iterate past class 0.

(The former "other blocks (skeletons)" list is gone — every block above
now has its full byte layout, validated against the shipped save.)

## Validation

`scratchpad parse_data000.py` (2026-07-02 pass) walks
`main/startup/data.000` with zero slack:

```text
GlobalVars   5 globals + 102 var tables / 2,343 vars
Alignment    5 entities (5 ftell checkpoints OK) + 895 relations
AgentVars    6 tables / 69 vars
AgentClasses 382 classes (incl. CStats templates + slot arrays)
Agents       id-counter + 850 slots (849 tag-1 NPCs + 1 tag-3 player)
Eggs         8,861 × 92 B (= eggs.000 modulo the +0xc flag byte, 205 recs)
MonsterGen   158 A + 99 B records          Party  1 member
Skills       96 records (7 subclass tails) Time/Gameclock 24/36 raw
Traps        782 traps / 438 effects / 68 flagged
Timers 0 · Counters 11 · Explosions 0 · DoorChestList 0 · DialogLog 0
NoMagicZones 7 · Magic 128 slots + 42-byte region blob + 4 + 0x154
Projectiles/Painpoints/Anieffects 0/0/0
Osirisobjects 1,808 × 8 + trailer · Osirisnames 2,060 × 36
PlayerInfo   4 u32 → exact EOF at byte 2,902,490
```

## Loader citation

```text
div.exe:0x00502170   FUN_00502170   savegame SAVE orchestrator — emits the section version
                                    headers in order (mirror of the loader).
div.exe:0x00502bf0   FUN_00502bf0   savegame load orchestrator;
                                    opens data.000 (banner + version flag),
                                    then walks the 25 blocks above; per-region
                                    world.x* / objects.x* / shroud.x* are
                                    loaded by separate per-partition readers
                                    *before* data.000 is opened.
div.exe:0x004f4d70   FUN_004f4d70   read-and-validate version string
                                    (length-prefixed incl. NUL, strcmp;
                                    "Bad Version : %s instead of %s").
div.exe:0x004f4c70   FUN_004f4c70   thin fread wrapper (size, count) into
                                    a typed slot — used by everyone.
div.exe:0x004f4c00   fcn.004f4c00   bulk reader: alloc count×size, fread.
div.exe:0x004f4d10   fcn.004f4d10   LPStr reader: u32 len + len bytes.
div.exe:0x004f4f10   fcn.004f4f10   ftell CHECKPOINT verify: reads a u32 and
                                    compares it to the current file offset
                                    (writer planted its own ftell).
div.exe:0x004f4f50   fcn.004f4f50   mark position (ctx+4 = ftell) — no bytes.
div.exe:0x004f4f70   fcn.004f4f70   fseek back to the mark — no bytes
                                    (the explosions legacy-format fallback).
```

**`objects.x<n>` / `extfree.x<n>` / `world.x<n>` are NOT re-serialized
into `data.000`** — the save path copies them as whole files
(`.\WORLD\Compress.cpp` bundler `fcn.005732a0`, see
[`osi-static.md`](osi-static.md)); their record formats are already
byte-exact there (28-byte object records + free-list), so they need no
per-byte work here. Inventory items ride in those object files, which is
why no agent sub-reader reads items inline.

## Auxiliary save files

These siblings of `data.000` have their own heads but follow the same
"fread into a fixed struct" pattern:

| File | Purpose | Format |
|---|---|---|
| `info.000` | save-slot metadata for the load screen | 32-byte struct: `u32 size; u32 0; u32 mtime_lo; u32 size2; u32 0;` plus a few flags |
| `items.000` | flat array of item instances | `[u32 count][Item × count]` |
| `mapflags.000` | **named scripted map-flags** (quest triggers tied to map areas) | **decoded from the shipped file**: 24-byte header `{u32 0; u32 0; u32 count(106); u32 datasize; u32 dims(10,10 as 2×u16); …}` then `count` records each `{u32 name_len; char name[]; baked-ptr + per-area data}` — e.g. names `"Water supply orcs"`, `"Arm…"`. So it's a list of **named** scripted flags (the story/quest flags pinned to map regions), not an anonymous bitmap; the per-area data is baked-pointer (heap-dump) so its exact cell layout replays the deserialize. Reader `FUN_0044ae10`. |
| `quest_log.000` | active + completed quests | a **versioned `"ML3ID"` container** (reader `FUN_00481b50` accepts both `"ML3ID"` and the older `"ML2ID"` magic at `0x60f0ac`/`0x60f0a4`; writer `FUN_00481ea0`). After the 5-byte magic the body is a **serialized ordered map/tree** of quest entries — the reader rebuilds nodes and walks left/right children via `node[+0x10]`/`node[+0x18]`, with `0xffffffff` child sentinels (visible in the shipped file). So the on-disk form is a flattened red-black/`std::map`-style tree keyed by quest id, not a flat record array; recovering each node's payload bytes replays the node reader (heap-dump-style), the same bound as the other baked-pointer saves. |
| `quickinfo.000` | player snapshot for the load screen | **decoded from the shipped file**: header `{u32 0; char Name[] ("Hero"); … u32 w=158 (0x9e); u32 h=118 (0x76); version strings "V1.0029a" + "V0.935 25-02-2002"}` then a **w×h×2 = 158×118×2 = 37288-byte 16-bit thumbnail** bitmap (header ≈204 B + 37288 = 37492 = exact file size). The default startup thumbnail is all-zero (black). |
| `telpstates.000` | per-teleporter activation state | **decoded + verified against the reader** `FUN_0052f680`: `u32 count` then `count` × `{u32 teleporter_id; u32 state}` (8 B records). The reader reads the count, then per record reads `id`, resolves its slot in the teleporter array via `FUN_0052f430` (`base[id*4]`), and reads `state` into it. The shipped new-game template is `count=21`, ids `696..716` (`0x2b8..0x2cc`) all `state=1`. `4 + 21×8 = 172` = exact file size — so it is a `{id,state}` list, **not** a bitmap. |

These are unblocked for parsing once you actually need to read existing
saves; they're skipped here because the engine boots cleanly without
them.
