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

## Observed version strings (in order, from `FUN_00502bf0`)

```text
GlobalVarsV0.935 25-02-2002          read by FUN_004adcb0
AlignmentmanagerV0.935 25-02-2002    read by FUN_00438890
AgentVariablesV0.935 25-02-2002      reads via DAT_00658d50 (NPC manager)
AgentClassesV0.935 25-02-2002        read by FUN_00422dd0
AgentsV0.935 25-02-2002              read by FUN_00559c20
EggsV0.935 25-02-2002                read by FUN_00426c90
MonsterGenV0.935 25-02-2002          read by FUN_00440080
PartyV0.935 25-02-2002               read by FUN_005178b0
SkillsV0.935 25-02-2002              read by FUN_00543620
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

Eggs body byte-exactly equals `global\eggs.000` (815,216 B = 4-byte
count + 8861 × 92-byte records) — the savegame embeds a snapshot of
the global agent table.

## Subsystem block layouts

Block-by-block schema, gathered from each named reader. The full set
needs incremental work; what's confirmed below should be enough to
parse a savegame end-to-end and skip / preserve unknown bits.

### `GlobalVarsV0.935 25-02-2002`

Reader at `div.exe:0x004adcb0` (`FUN_004adcb0`). Reads 5 u32 globals
into `DAT_006ddd28..38` then calls a sub-record reader.

```text
struct GlobalVarsBlock {
    u32  global_var[5];        // 5 named integers (game time, gold count, etc.)
    SubRecord  trailer;        // FUN_005058a0 — count + per-var name/value list
};
```

### `AlignmentmanagerV0.935 25-02-2002`

Reader at `div.exe:0x00438890` (`FUN_00438890`). Source path
`.\AGENTS\alignment.cpp`.

```text
struct AlignmentBlock {
    u32  count_a;              // size of agent_count list
    u32  count_b;              // size of relations list
    u32  third_count;          // observed 0 in shipped templates
    Agent      agents[count_a];     // each is a 0x28 (40) byte CAlignment object
    Relation   relations[count_b];  // each is a 0x18 (24) byte struct + linked-list pointers
};

struct CAlignment {            // 0x28 = 40 bytes
    void*  vftable;
    u32    pad[3];
    u32    fields[4];
    u32    score;              // observed initial value: 0x32 (50)
};

struct Relation {              // 0x18 = 24 bytes; built into a doubly-linked list
    char* name;                // length-prefixed string read via FUN_004f4d10
    u32   field_at_4;
    u32   field_at_8;          // links populate this and the next field at runtime
    u32   prev_relation;
    u32   next_relation;
    u32   field_at_14;
};
```

### `CountersV0.935 25-02-2002`

Reader at `div.exe:0x00510200` (`.\OSIRIS\osicounter.cpp`). Simple
named-counter list.

```text
struct CountersBlock {
    u32  count;
    Counter  counters[count];
};

struct Counter {              // 8 bytes (1 u32 value + 1 u32 string pointer)
    u32   value;
    char* name;               // length-prefixed string (FUN_004f4d10 reader)
};
```

### `ExplosionsV0.935 25-02-2002`

Reader at `div.exe:0x00575f30` (`.\WORLD\explosion.cpp`). Discriminated
union of 5 explosion classes.

```text
struct ExplosionsBlock {
    u32  count;
    Explosion explosions[count];
};

struct Explosion {
    u32  type_tag;            // 1..5
    // type 1: 0x68-byte CExplosion_Gasoline body
    // type 2: 0x98-byte CExplosion_WalkingMine body
    // type 3: 0x78-byte CExplosion_TrailBomb body
    // type 4: 0x70-byte CExplosion_PoisonCloud body
    // type 5: 0x74-byte CExplosion_DamageCloud body
    u8   body[…];             // sized per type_tag
};
```

### Osiris* blocks

`OsirisobjectsV0.935 25-02-2002` (reader `0x00585bb0`) and
`OsirisnamesV0.935 25-02-2002` (reader `0x005860a0`) are the runtime
`(name → handle)` map for the Osiris VM. They serialise the live state
of the tables shipped as `static\osinames.000` / `static\osiobjects.000`
(see [`savegame-aux.md`](savegame-aux.md)).

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

### `MagicV0.935 25-02-2002`

Reader at `div.exe:0x004e3180` (`.\magic\SMagic.cpp`). Loads the
system-magic state.

```text
struct MagicBlock {
    u32         expected_count;     // must match the engine's compiled spell_count;
                                    //   mismatch logs "Amount mismatch in CMagicSemantic::Load()"
    SpellSlot   spells[expected_count];
    char        sentinel_open[]   = "Check teleport regions";   // version-string sentinel
    Block       teleport_regions; // FUN_0058e690 — variable-length sub-record
    char        sentinel_close[]  = "End check teleport regions";
    u32         field_at_4;
    u8          state[0x154];     // 340 bytes of "magic system state" struct
};

struct SpellSlot {
    u32  active;
    i32  type_id;                  // -1 or active==0 → empty slot
    // if type_id != -1 && active != 0:
    u8   body[0x54];               // 84-byte CMagic instance body
};
```

### `EggsV0.935 25-02-2002`

Reader at `div.exe:0x00426c90` (`.\AGENTS\agentmanager.cpp`). Despite
the block name "Eggs", this is the **Agent (NPC) block** — `Eggs` is
the historical-code name for spawn markers that became the agent
manager. Discriminated-union per-slot.

```text
struct EggsBlock {
    u32  agent_count;
    u32  flags_field;
    Agent agents[agent_count];
};

struct Agent {
    char version[]    = "new npc";    // version-string sentinel per slot
    u32  type_tag;                    // 0..3
    // type 0: empty slot
    // type 1: 0x2e0  (736 B) — basic CAgent
    // type 2: 0x304  (772 B) — CAgent variant
    // type 3: 0x620 (1568 B) — CAgent big variant
    u8   body[…];                     // sized per type_tag; virtual Read via vtable[0x68]
};
```

Followed at end-of-block by an optional `dat\classpatch.txt` apply
pass for legacy save migrations.

### `MonsterGenV0.935 25-02-2002`

Reader at `div.exe:0x00440080` (`.\AGENTS\monstergen.cpp`).
Two parallel parallel arrays of generator records.

```text
struct MonsterGenBlock {
    u32       count_a;             // number of "B" records
    u32       count_b;             // number of "A" records (sub-list)
    SubRecA   sub_a[count_b];      // 24 bytes (0x18) each, FUN_0043e610
    SubRecB   sub_b[count_a];      // 40 bytes (0x28) each, FUN_0043eb30
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
    // if tag != 0:
    u8    body[0x184];             // CProjectile virtual-Read body
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

In shipped data both blocks have count=0 → 4-byte body. Per-item
virtual sub-body is invoked via vtable[0] / vtable[3] respectively;
field-level decode TBD.

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

struct Class {
    u8     fixed[0x318];    // FUN_004f4c70(buf, 0x318)
    LPStr  name;            // class name, e.g. "Hero"
    Bytes  behavior;        // virtual-call serialization at param_1+0x120;
                            //   layout depends on the behavior subclass
                            //   (CAgentBehavior… variants).  TBD.
    LPStr  alignment;       // alignment-table key (e.g. "good")
    LPStr  animation_set;   // animation-set tag, used to look up
                            //   per-anim frame ranges (see CharIndex.cpp)
    SlotData slot[19];      // for each anim slot s where
                            //   fixed[0x4c+s] != 0:
                            //     fixed[0x4c+s] u32 frame-offset values
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

**Behavior block (TBD).**  The virtual call at `param_1+0x120` writes
behavior-subclass-specific bytes whose layout we haven't traced
yet.  Iterating past class 0 needs that decoded — until then,
`pkg/assets/agentclass.ReadHero` reads only class 0 (the player).

### Other blocks (skeletons)

The remaining blocks each match the same outer pattern — `u32 count`
or fixed struct, then per-record bodies — but their exact field
layouts are TBD. Per the named readers in `FUN_00502bf0`:

```text
AgentVariables  via DAT_00658d50 (NPC manager)
Agents          via FUN_00559c20   linked NPC instance list
Eggs            via FUN_00426c90
Party           via FUN_005178b0
Skills          via FUN_00543620   destructor; reader actually does nothing
                                    new for this block (skill state lives in
                                    the Agents block)
Traps           via FUN_005945c0   destructor (reader is the matching ctor)
Timers          via FUN_005168c0   destructor; reader nearby
DialogLog       via FUN_00472940   appends one (id, flag) tuple per visited node
NoMagicZones    via FUN_0058dd50
```

A full sub-format breakdown is mechanical from here — each named
function is small and self-contained. The infrastructure to do it is
the same for all of them: open the function, count `fread`/`FUN_004f4c70`
calls, map field offsets.

## Loader citation

```text
div.exe:0x00502bf0   FUN_00502bf0   savegame load orchestrator;
                                    opens data.000, reads "h$wa", then
                                    walks the 24 blocks above; per-region
                                    world.x* / objects.x* / shroud.x* are
                                    loaded by separate per-partition readers
                                    *before* data.000 is opened.
div.exe:0x004f4d70   FUN_004f4d70   read-and-validate version string
                                    (NUL-terminated, byte-for-byte strcmp).
div.exe:0x004f4c70   FUN_004f4c70   thin fread wrapper (size, count) into
                                    a typed slot — used by everyone.
```

## Auxiliary save files

These siblings of `data.000` have their own heads but follow the same
"fread into a fixed struct" pattern:

| File | Purpose | Format |
|---|---|---|
| `info.000` | save-slot metadata for the load screen | 32-byte struct: `u32 size; u32 0; u32 mtime_lo; u32 size2; u32 0;` plus a few flags |
| `items.000` | flat array of item instances | `[u32 count][Item × count]` |
| `mapflags.000` | per-cell scripted flag bitmap | sparse encoding; reader at `FUN_0044ae10` |
| `quest_log.000` | active + completed quests | reader at `FUN_00481b50` |
| `quickinfo.000` | player snapshot for thumbnail | reader unmapped |
| `telpstates.000` | teleporter activation bitmap | already shipped — `[u32 count][record × count]` (172 bytes total in the new-game template) |

These are unblocked for parsing once you actually need to read existing
saves; they're skipped here because the engine boots cleanly without
them.
