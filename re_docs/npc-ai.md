# NPC AI / behaviour

NPC decision-making is **not** a hard-coded state machine — it is a
**data-driven behaviour-script language**, `agentscript`
(`.\AGENTS\agentscript.cpp`), the direct analogue of the magic-script
(`mgcscrpt`, [`skills-magic.md`](skills-magic.md)) for combat spells.
Each NPC runs a text script that drives movement, perception, combat
toggles and dialogue, exactly like the patrol routes named in the data
(`Verdistis Patrol1`, `Verdistis Patrol4`, …).

## The agentscript engine

The pipeline has a **compiler** (text → a `u16` opcode program) and a
per-frame **execution** path that steps that program; a separate
**debug disassembler** renders the compiled program back to text.

```text
COMPILE (text → program buffer at agent+0x2b4):
fcn.004314f0   compiler driver — reads the script text a line at a time
               and calls the parser per line, accumulating the emitted
               opcode count, then allocs the program buffer (agent+0x2b4,
               length +0x2b8) and resets the pc (+0x2ba = 0).
fcn.00430010   PARSER — one line → opcodes; a 125-case command switch
               (jump table div.exe:0x004312f4); handles `//` comments,
               `{ }` blocks and the `|` separator; errors
               "Syntax error %s for agent %s".
fcn.0042fba0   InsertProgramLine — splices a line's slots into the
               buffer ("Time out in InsertProgramLine()").
fcn.00431610   label-fixup pass — walks the program and resolves the
               goto opcodes (16/17) to program offsets (fcn.0042feb0).

RUN:
fcn.00431670   StartScriptFrame — scans for opcode 78 (`script frame $`)
               whose operand == the requested frame id, points the pc
               (+0x2ba) just past it and sets agent+0x224 |= 0x8000
               ("script frame %d started"). This is what the Osiris NPC
               Call handler invokes (osiris.md).
fcn.004329a0   the EXECUTOR (9.5 KB) — the per-frame step loop that
               actually performs the commands.
fcn.00431770   DISASSEMBLER (4.2 KB) — renders the program to a text
               buffer (NOT the executor — see below).
```

So a behaviour is **~125 commands** of a small imperative language.
Operands use sigils: **`$`** = a location/variable, **`@`** = an
action/target (e.g. `move to $`, `change action to @`).

### Program format & the disassembler (`fcn.00431770`)

The compiled program is a flat `u16` array at **`agent+0x2b4`**, length
(in slots) at **`+0x2b8`**, and the **program counter at `+0x2ba`** (a
slot index, *not* `+0x2b8` — that is the length).

**`fcn.00431770` is a debug disassembler/lister, not the executor**
(corrects an earlier draft that called it "the interpreter"). It takes an
**output text buffer** and an **instruction count**, then walks the
program from the pc and emits one formatted line per instruction —
header `"\r%c%s - pc =%d (Story=%d)"`, then `"%d :-"` and the decoded
command. It fetches `op = program[pc]`, decrements (opcodes are 1-based),
bounds (`op-1 ≤ 117`), and dispatches a **decode** switch — byte
index-map `0x4328c0` → jump table `0x4327ec` (118 opcodes → 53 cases) —
where **every case only `sprintf`s the operands** (verified on
`move to`/`attack`/`cast spell`/`wait`/`follow npc`: none performs an
action). **66 of the 118 opcodes share a stub at `0x0043272d`** that
prints `"debug code not implemented - function is working"` — i.e. those
opcodes simply have **no disassembly text**; this says nothing about when
they execute. (An earlier draft over-read this stub as proof those 66 are
"parse-time only" — that inference is withdrawn.)

The decode does, however, pin the **on-disk operand encoding** of each
command (the disassembler reads the real program), and how ids resolve to
entities:

- **`move to $`** (op 2 → `0x004318cb`): operand = a **location id**,
  looked up in the named-location table **`[0x750d2c]`** —
  `loc = [[0x750d2c]+4] + (id << 4) + 8`, i.e. **16-byte records** with
  the name at `+8` (the LOCATION registry).
- **`attack $`** (op 91 → `0x0043201d`): operand = a **target npc id**,
  resolved via the shared `CAgentManager [0x658d50]`
  (`target = [[mgr+0xc] + id*4 + 4]`, name at `target+0x21c`).
  `!attack $,$` (op 102) is the two-target "Dynamic attack" variant.
- **`cast spell # on $`** (op 80 → `0x00431d1b`): operands `[spell#,
  targetid]`, width 3 (`[op][#][$]`); target resolved via `CAgentManager`.

So the agent-id and location-id operands index the same `CAgentManager`
and location registry the rest of the engine uses. (`fcn.00431988` is just
the disassembler's shared "copy the trace string & advance pc" tail, not a
spell call.)

### The executor (`fcn.004329a0`)

The actual per-frame interpreter is **`fcn.004329a0`** (9.5 KB,
`agentscript.cpp`) — structurally a twin of the disassembler (fetch
`op = program[pc]`, `dec`, bound `op-1 ≤ 121`, byte index-map `0x4350c0`
→ **122-entry jump table `0x434fcc`**, advance the pc by the `0x64a7b0`
width) but its cases **do real work**: it makes **154** calls to engine
routines — pathfinding/movement (`fcn.0056e6b0` / `fcn.005719f0` /
`fcn.0057bf30`), sleep (`fcn.00429cb0`, the `force sleep` path), object
state (`fcn.005919e0`), trig for facing (`_CIsqrt` / `_CIacos`) — and its
strings are **runtime-event logs**, not a listing: `"Moving %s to location
%s"`, `"No path found - teleporting %s to location %s"`, `"Teleporting %s
because no cell free near other npc %s"`. (This is the distinguishing
test: the disassembler `fcn.00431770` only `sprintf`s decoded operands
into a text buffer; the executor calls subsystems and logs real events.)
On frame end it masks the pc-frame flag with `~0x8000` (clearing the
`agent+0x224` script-frame-running bit `StartScriptFrame` set). It is the
same function already credited below with `set sight #` → `CStats+0x20`
(under [Perception / detection](#perception--detection-fcn004356f0)),
confirming it is where width-≥1 runtime commands take effect. This is the
*script-opcode* layer; the lower-level per-frame movement/animation tick
that carries out the chosen action (`fcn.00411380`, `agentbehaviour.cpp`)
is described under [Behaviour execution](#behaviour-execution-the-per-frame-tick).

**Worked case — `cast spell # on $` (op 80 → `0x00433bd4`).** The executor
case reads the two operands inline (spell# at `program[pc+1]`, target npc
id at `program[pc+2]`), then **gates on `agent+0x25c`** — the magic
component / spellbook, set by the `has magic` command (#77): when it is
null it logs `"Can't cast spell because %s does not have magic component
(use has magic)"` and skips. Otherwise it calls the agent's cast method
**`fcn.0041e4e0`** with the spell index and target; on failure it logs
`"Failed casting %d on %d for %s"`. `fcn.0041e4e0` is the bridge into the
magic system: it drives **`SMagic [0x658c38]`** (the spell-record manager,
[`skills-magic.md`](skills-magic.md)) via its lookup/cast helpers and runs
the faction check (`fcn.004380a0`, the alignment matrix) on the target. So
`cast spell` is a thin opcode that hands off to the same `SMagic`
machinery the player's spellcasting uses — the agentscript→magic seam is
`fcn.004329a0` case 0x433bd4 → `fcn.0041e4e0` → `SMagic [0x658c38]`.

**Worked case — `attack $` (op 91 → `0x00434235`).** It resolves the
target via `CAgentManager` (id at `program[pc+1]`; "Npc %s attacks %s"),
calls the **combat-prep virtual `vtable[+0x6c]`(0) on *both* attacker and
target**, **clears the script-frame bit** (`agent+0x224 &= ~0x8000`,
logged "Slave flag cleared to allow for fight" — so `0x8000` doubles as a
"scripted/slave control" flag that entering combat drops), then calls the
combat-engage method **`fcn.00417050`(targetId)**. `fcn.00417050` resolves
the target through `CAgentManager`, then **gates the attack on `+0x220`
flags**: it proceeds iff `attacker[+0x220] & 0x40` **or not**
`target[+0x220] & 0x10` — i.e. **`+0x220` bit `0x10` on the target means
"protected / cannot be attacked", overridden by bit `0x40` on the
attacker ("may attack protected targets")**. When allowed it establishes
the combat target via `fcn.00423710` (sub-path `fcn.00415610` when the
agent's `+0x254` state is set). This is the agentscript→combat seam:
`fcn.004329a0` case 0x434235 → `fcn.00417050` → target engaged (the melee
resolver `CAgent vtable[+0x28]` then runs from the combat tick). The two
`+0x220` meanings here are **code-anchored** (consumer arithmetic), firming
bits previously only inferred ([`agent.md`](agent.md)).

**Worked case — `move to $` (op 2 → `0x00432e7b`).** It resolves the
location record (`[0x750d2c] + (id<<4)`), converts it to a **destination
cell** (offset helper `fcn.0057bf30` added to the agent's position
`+0x1c`/`+0x20`, then `>>5`), logs `"Moving %s to location %s"`, and calls
the navigation virtual **`CAgent vtable[+8]`(cellX, cellY)** — pathfind &
walk to the cell ([`pathfinding.md`](pathfinding.md)). **If it returns
failure** it logs `"No path found - teleporting %s to location %s"` and
calls **`CAgent vtable[+0xc]`(…)** — the teleport / force-position
fallback. Either way it clears `agent+0x220 & ~0x4` (a movement-state bit)
and advances the pc. So `vtable[+8]` = navigate-to-cell (pathfinding) and
`vtable[+0xc]` = teleport-to are the movement virtuals the script drives;
the same pair recurs in the `move to npc` / `appear near` cases (the
"Teleporting because no cell free near other npc" logs).

### Keyword → case index map (the `0x4312f4` switch)

The parser matches each line against a **125-entry keyword-template
array at `0x64a5b8`** (char pointers, indices 0..124). The match loop
(`0x4300a1`) walks `i = 0..124`, string-compares the line against
`keyword[i]` (`fcn.004fdf40`), and on a hit jumps straight through the
switch by the **same index**: `jmp [i*4 + 0x4312f4]`. So **keyword
array index == switch case == command id** — a flat, dense map. The
templates carry their operand grammar inline: `$` = named reference,
`#` = integer (`#,#` = coordinate pair), `@` = action/behaviour id.

The parallel table at **`0x64a7b0`** is a `u16[125]` indexed by the
same command id (see [below](#compiled-program-slot-widths-0x64a7b0)).

```text
  0 name $                         63 set dialog #
  1 set location $                 64 npc emotions are in $
  2 move to $                      65 set walk speed #
  3 change action to @             66 set gain #
  4 wait # frames                  67 sit for # parse frames
  5 start animation for # frames   68 interested in special object #
  6 reset animation                69 go sit # frames
  7 jump to loop point             70 create group with behaviour #
  8 appear near npc $ angle #       71 add to group from $
  9 wander #                       72 add double level creating class $
 10 move to coords #,#             73 on death reappear at $
 11 end npc                        74 (obsolete)
 12 set loop point                 75 set ai parameter #
 13 set hitpoints #                76 set region sensitive
 14 set alignment $                77 has magic
 15 set aiclass #                  78 script frame $
 16 remove #                       79 script frame end $
 17 goto label #                   80 cast spell # on $
 18 leave to $                     81 cannot die
 19 set fightwalkspeed #           82 can die
 20 set fightspeed #               83 set inventory type #
 21 set attack #                   84 set inventory level #
 22 set defense #                  85 set behavior @
 23 set sight #                    86 behave
 24 set hearing #                  87 set spell level #,#
 25 lock location                  88 set spell knowledge #,#
 26 move to locked location        89 say $
 27 be moody #                     90 force sleep
 28 follow npc $                   91 attack $
 29 set mood bored                 92 move to npc $
 30 set mood bloodlust             93 set seeing triggers event
 31 set mood confused              94 clear seeing triggers event
 32 set mood sleepy                95 look at $
 33 set mood alert                 96 sees all
 34 set armor #                    97 next monologue line
 35 set weapon damage #            98 look direction #
 36 set weapon dice #              99 set tactics @,@
 37 set magicspeed #              100 set fight mode #
 38 set group #                   101 !move to npc $,$
 39 set dexterity #               102 !attack $,$
 40 set strength #                103 end preprocess
 41 set intelligence #            104 set object $ visibility #
 42 appear near location $ angle # 105 add # instances of object # to inventory
 43 move to npc $ range #         106 clear inventory
 44 (obsolete)                    107 visible #
 45 set level #                   108 follow leader
 46 set experience #              109 goto $
 47 sleep for # frames            110 label $
 48 (obsolete)                    111 event #
 49 (obsolete)                    112 debug #
 50 speak with $ about $          113 log $
 51 speak about with partners about $  114 repulsive #
 52 sleep type is # position is #,#,# cell is #,#  115 set treasure $
 53 (obsolete)                    116 regenerate inventory
 54 kill all lights               117 clear generated inventory
 55 (obsolete)                    118 clear path
 56 set mood rabbit               119 running #
 57 (obsolete)                    120 !move to $,$
 58 revive lights                 121 !look at $,$
 59 kitchen is at cell #,#        122 looping action to @
 60 be aware of player            123 strong npc
 61 look for food                 124 boss npc
 62 my house is region $
```

The seven `(obsolete)` slots (44, 48, 49, 53, 55, 57, 74) all point at a
single `"obsolete"` string — retired command ids kept as placeholders so
the indices of later commands stay stable.

### Compiled-program slot widths (`0x64a7b0`)

A behaviour script is parsed into a **`u16` program buffer** (per-agent,
at `agent+0x2b4`, length `+0x2b8`). The table at **`0x64a7b0`** is a
`u16[125]` indexed by the **same command id**, giving each command's
**width in u16 slots** in that buffer. `InsertProgramLine`
(`fcn.0042fba0`) reads `width = u16[id*2 + 0x64a7b0]` to size the line,
shift the trailing slots, and **relocate jump operands**: for the goto
family (command ids `16`, `17`, `109` — `remove #` / `goto label #` /
`goto $`) it bumps the offset operand at `slot+2` by the insertion delta.
(So this is *not* an operand-spec `{count,type}` table, as an earlier
draft guessed — the `0x00020000`/`0x00020002`/`0x00030003` words it saw
were just adjacent `u16` widths read as `u32`.)

The width is **not** simply `sigils+1`; reading it against the templates
shows the real encoding:

- A bare command is `1` slot; each `#`/coordinate operand adds one
  (`sleep type is … cell is #,#` = 6 operands → width 7).
- A **string operand reserves 64 slots** (`say $` and `log $` both have
  width 64) — an inline fixed-size text buffer, not a 1-slot reference.
- **Width 0** marks commands that emit **no runtime instruction** — they
  are applied at parse/preprocess time and leave nothing in the program
  buffer. These are the one-shot agent setup verbs: `name $`, `set
  aiclass/attack/defense/hearing/armor/weapon …/strength/dexterity/
  intelligence/level/experience/gain`, `set ai parameter`, `set region
  sensitive`, `has magic`, `label $`, `end npc`, `strong/boss npc`. (By
  contrast `set sight`/`set hitpoints`/`set walk speed`/`set group`/`set
  dialog` are width 2 — kept as runtime-executable instructions.) The
  width-0-vs-2 split is thus a load-time-config vs runtime-op distinction;
  the exact rule per command is read off this table rather than inferred.

## Command vocabulary (samples, by role)

Recovered from the command strings; this is the NPC's actual "AI":

Operands: **`$`** = a named reference (entity / npc / location / region
/ variable), **`@`** = an action or behaviour id, **`#`** = an integer
literal, **`#,#`** = a coordinate pair. The near-complete vocabulary,
recovered from the command-string block:

```text
Movement     move to $ · move to coords #,# · move to npc $ · move to npc $ range #
             · !move to npc $,$ · move to locked location · goto $ · goto label #
             · wander # · leave to $ · lock location · set location $
             · appear near location $ angle # · appear near npc $ angle #
             · on death reappear at $ · look at $ · look direction #
Action/anim  change action to @ · set behavior @ · reset animation
             · start animation for # frames
Combat       attack $ · !attack $,$ · set attack # · set fight mode [#]
             · clear fight mode · cast spell # on $ · set spell knowledge #,#
             · has magic
Perception   sees all · seeing triggers event · seeing does not trigger event
             · set/clear seeing triggers event · be aware of player
             · look for food · interested in special object #
             · set region sensitive · my house is region $
Mood/state   be moody # · set mood {alert,bloodlust,bored,confused,rabbit,sleepy}
             · force sleep · sleep for # frames · sleep type is #,position #,#,#
             · go sit # frames · sit for # parse frames · (npc) can/cannot die
             · new alignment $ · npc emotions are in $ · be aware of player
Dialogue     say $ · next monologue line · set dialog # · speak with $ about $
             · speak about with partners about $
Inventory    add # instances of object # to inventory · set inventory level/type #
             · clear / regenerate / clear-generated inventory
Group/social add to group from $ · create group with behaviour # · set group #
             · follow leader · follow npc $ · relation $,$,#
World/object set object $ visibility # · visible # · set/clear invisible
             · set hitpoints # · kill all lights · revive lights
             · kitchen is at cell #,# · new entity $
Control flow set loop point · jump to loop point · label $ · start script frame
             · script frame $ · script frame end $ · NpcScriptFrames
             · wait # frames · event # · default # · end npc · end preprocessing
```

These map to the engine's other subsystems: **`set fight mode`** hands
the agent to the melee resolver (`FUN_00417b40`,
[`combat.md`](combat.md)); **`seeing triggers event`** raises an Osiris
event (the perception → story hook, [`osiris.md`](osiris.md));
**`follow leader`** ties into the party; **`move to $`** drives the
pathfinder ([`pathfinding.md`](pathfinding.md)).

## Per-agent state

The agent struct ([`agent.md`](agent.md)) carries the AI state:

| Offset | Field | |
|---|---|---|
| `+0x34` | `Ai class` | selects the behaviour kind / default script |
| `+0x2b4` | ptr | compiled `u16` program buffer |
| `+0x2b8` | u16 | program length (slots) |
| `+0x2ba` | u16 | **program counter** (slot index) |
| `+0x2cc`..`+0x2d4` | `Behavior[3]` | the active behaviour ids |
| past `+0x2dc` | region-list + **behaviour-program** arrays | the loaded script |

## Sleep state (`force sleep` → `FUN_00429cb0`)

The `force sleep` / `sleep type is #, position #,#,#` commands put an NPC
into a persistent **asleep** state (so towns can have NPCs in bed,
distinct from the `sleep for # frames` script-pause). The command handler
calls `FUN_00429cb0` (`.\AGENTS\agentnpc.cpp`), which is **gated on the
agent being "promoted"** (active/loaded) — an inactive NPC logs `"Npc %s
sleep skipped because not promoted"` and the sleep is deferred.

The asleep state is **`agent+0x224` bit `0x200`** (within the flags word
agent.md calls `Parameter 2`): the sleep/wake helpers in the `0x004d0xxx`
region toggle it with `or [+0x224],0x200` / `and [+0x224],~0x200`, and
the sleep action tests it (`test [+0x224],0x200`) to avoid
re-sleeping an already-sleeping NPC. So `agent+0x224` is a multi-purpose
flag word — known bits: **`0x200` asleep**, **`0x400000` sees-all** (the
perception gate, above). The matching `set mood sleepy` only changes the
mood/idle animation; `0x200` is the gameplay sleep state.

## Behaviour execution (the per-frame tick)

`agentbehaviour.cpp` (`fcn.00411380`, `fcn.0040ede0`) is the per-agent
**tick** that *executes* the current scripted action: it reads the
agent's position/midpoint (`+0x1c`/`+0x20`, `+0x24`/`+0x28`), the
`Walkcount` (`+0x278`) and `Ai class` (`+0x34`), steps movement toward
the target, calls the pathfinder (`fcn.004477d0`) and the move helpers
(`fcn.0040f0a0`/`fcn.0040f450`). So `agentscript` chooses *what* to do
(the plan) and `agentbehaviour` carries it out each frame (the
execution), with the command bus / Osiris events linking them.
(`fcn.00411380` is one concrete tick — `CAgentBehaviorLeaveTo`'s vtable
slot 3; the tick is **polymorphic per personality**, all 17 variants
enumerated in [`ai-behaviour.md`](ai-behaviour.md), sharing this same
move-execution machinery.)

## Perception / detection (`fcn.004356f0`)

The NPC's "what can I see" scan, run per agent. The **sight range** is a
stat: the `set sight #` agentscript command (`fcn.004329a0`) stores it at
`CStats+0x20` (`CStats` = `agent+0x2c`); `set hearing #` is the paired
audio range. The scan walks the live-object array (`[0x658d50]`) and, for
each object `O`, applies:

```text
skip if O == self                       (agent at controller+0xd4)
skip if [O+0x220] & 0x100               (un-perceivable / dead)
skip if [O+0x224] & 0x10000100          (invisible / non-target flags)

radius = sight * 32                      (cells → world units)
if not "sees all" ([agent+0x224] & 0x400000)
   and not [O+0x220] & 0x40:
       require faction/relation test fcn.004380a0([agent+0x30],[O+0x30])

if [O+0x220] & 0x20:                     (concealed / in-shadow target)
       day   (GetHour fcn.0050bfe0 ∈ [5,23)):  radius /= 2
       night (hour <5 or ≥23):                 radius /= 4

dist = fcn.0040ecb0(agentPos, Opos)      (agentPos: X=[+0x1c]+[+4], Y=[+0x20]+[+8])
require dist <= radius                    (range gate)
require fcn.0056fbc0(worldmap [0x74eca0], agentPos, Opos)   (line-of-sight)
→ remember O (id [O+0x214]) via fcn.004352e0; the seen-list caps at 16
```

So detection is **`dist ≤ sight·32`, then a world line-of-sight test**,
with two modifiers: the `sees all` flag (`agent+0x224` bit `0x400000`)
bypasses both the faction filter and is set by the `sees all` command;
and a **concealment** flag on the target (`[O+0x220] & 0x20`) halves the
range by day and quarters it at night — the stealth/`Shadows` mechanic,
time-of-day-aware via the [world clock](world-clock.md). The matched
objects feed the `seeing triggers event` Osiris hook and the "Seeing"
runtime state shown by the agent dumper. A separate path (`fcn.004285f0`)
projects `sight+2` cells along a 16-entry direction table (`0x654e50`,
8-byte `dx,dy`) for the directional look/scan ray.

## Alignment & faction relations (`fcn.004380a0`)

The perception scan's faction gate, and ~70 other hostility checks, route
through one predicate. Each agent carries an **alignment entity** pointer
at `agent+0x30` — a `CAlignment` object (`.\AGENTS\alignment.cpp`) with a
name and a numeric id (the agent dumper prints `Alignment : %s (%d)`; the
agentscript command is `set alignment $`, a *named reference*). The
relation manager is the static singleton `[0x658de8]`:

```text
related(A, B):                          // A,B = CAlignment* (agent+0x30, object+0x30)
    if A == 0 or B == 0: return 0
    M = [0x658de8 + 0x424]              // the relation bit-matrix, or null
    if M:
        idA = [A]; idB = [B]            // each entity's id is its first field
        bit = idB * [M+4] + idA         // row-major, [M+4] = row stride
        return ([M] >> bit) & 1         // [M] = bit-array base
    else:                               // no matrix → group-hierarchy fallback
        if A == B: return 0
        return groupDistance(A, B) < 0x19    // fcn.00437fd0, threshold 25
```

So a relation is a single **bit in an id×id matrix** keyed by the two
alignment entities, with a hierarchy-distance fallback (`< 25`) when no
matrix is loaded. In the perception scan a `0` result **skips** the
object, so a set bit marks "this alignment reacts to that one";
`sees all` (`agent+0x224` bit `0x400000`) bypasses the gate entirely.

**Builder & polarity (`fcn.00437ed0`).** Alignments are declared in
`dat\alignment.dat` (`.\AGENTS\alignment.cpp`; `new alignment $` creates
an entity, `relation $,$,#` sets a pairwise relation value `#`). The
setter writes the matrix bit by the same index, gated on the **same `25`
threshold**:

```text
setRelation(A, B, value):
    if value <  0x19 (25):  matrix bit |=  1      // SET   → "reacts"
    if value >= 0x19:       matrix bit &= ~1      // CLEAR (fcn.004375b0)
    then propagate down each entity's child links (+0x0c/+0x10/+0x20)
```

So a **low** relation value (`< 25`) sets the bit and a **high** value
clears it — i.e. the bit means *hostile/reactive* (consistent with the
`enemies` keyword in the grammar and the perception "skip if 0"). The
relation propagates through the alignment hierarchy, so child alignments
inherit a parent's relations. The matrix dimensions are data-driven (one
row per loaded alignment).

## Status

- Architecture ✅ — NPC AI is a **data-driven `agentscript`** text
  language, not a fixed state machine; ~125 commands.
- Engine ✅ — parser `fcn.00430010` (125-case switch `0x4312f4`), compiler
  driver `fcn.004314f0`, line manager `fcn.0042fba0`; keyword-template
  array `0x64a5b8` (125 char ptrs), token matcher `fcn.004fdf40`.
- Command vocabulary ✅ (sampled) — movement / action / combat /
  perception / state / inventory / dialogue / control-flow primitives
  recovered; subsystem tie-ins identified (combat, Osiris, party,
  pathfinding).
- Per-agent state ✅ — `Ai class` `+0x34`, program buffer `+0x2b4`,
  length `+0x2b8`, **program counter `+0x2ba`** (corrected — `+0x2b8` is
  the length, not the cursor), `Behavior[3]` `+0x2cc`, behaviour-program
  arrays past `+0x2dc`.
- Compile pipeline ✅ — compiler driver `fcn.004314f0` (text → parser
  `fcn.00430010` per line → program buffer + label-fixup `fcn.00431610`).
  `StartScriptFrame` (`fcn.00431670`) scans for opcode 78 (`script frame
  $`) and points the pc at it — the runtime entry the Osiris NPC Call
  handler triggers ([`osiris.md`](osiris.md)).
- Disassembler ✅ — `fcn.00431770` is the program **disassembler**
  (text-buffer + instruction-count args; decode switch byte-map `0x4328c0`
  → jump table `0x4327ec`, 118 ops → 53 cases; every case only `sprintf`s,
  verified). **Corrects the earlier "interpreter" label** and the over-read
  that the 66 shared-stub opcodes are "parse-time only" (the stub
  `0x43272d` just means *no disassembly text*). It does pin the real
  operand encodings (`move to $` → location id; `cast spell` → `[#,$]`).
- Executor ✅ (pinned) — `fcn.004329a0` (9.5 KB, `agentscript.cpp`) is the
  real per-frame script interpreter: fetch `program[pc]`, `dec`, bound
  `op-1 ≤ 121`, byte-map `0x4350c0` → 122-entry jump table `0x434fcc`,
  advance pc by the `0x64a7b0` width. Verified to do real work (154 engine
  calls — pathfinding `fcn.0056e6b0`/`fcn.005719f0`, sleep `fcn.00429cb0`,
  object state `fcn.005919e0`, facing `_CIsqrt`/`_CIacos`) with
  runtime-event logs (`"No path found - teleporting %s to location %s"`),
  not `sprintf`-listing. Clears `agent+0x224 & ~0x8000` at frame end; the
  same function handles `set sight #` → `CStats+0x20`. The lower-level
  movement/anim tick is `fcn.00411380` (`agentbehaviour.cpp`).
- `agent+0x224` flag bit ✅ — `0x8000` = **script-frame running** (set by
  `StartScriptFrame`); adds to the known bits (`0x200` asleep, `0x400000`
  sees-all, `0x20000000` promoted).
- agentscript → movement seam ✅ — `move to $` (op2) executor case
  `0x00432e7b`: location `[0x750d2c]+(id<<4)` → dest cell (`fcn.0057bf30` +
  agent pos `+0x1c`/`+0x20`, `>>5`) → `CAgent vtable[+8]`(cellX,cellY)
  (pathfind/walk, [`pathfinding.md`](pathfinding.md)); on failure → log
  "No path found - teleporting" → `CAgent vtable[+0xc]` (teleport). So
  `vtable[+8]` = navigate-to-cell, `vtable[+0xc]` = teleport-to; clears
  `+0x220 & ~0x4` (movement-state bit).
- agentscript → combat seam ✅ — `attack $` (op91) executor case
  `0x00434235`: calls combat-prep `vtable[+0x6c]` on attacker+target,
  clears `agent+0x224 & ~0x8000` ("Slave flag cleared to allow for fight"),
  then `fcn.00417050`(targetId) engages. `fcn.00417050` gates on `+0x220`:
  proceeds iff `attacker & 0x40` or not `target & 0x10` → **`+0x220` bit
  `0x10` = protected/cannot-be-attacked, `0x40` = may-attack-protected**
  (code-anchored, was inferred). Establishes target via `fcn.00423710`.
- agentscript → magic seam ✅ — `cast spell # on $` (op80) executor case
  `0x00433bd4` gates on `agent+0x25c` (magic component, set by `has magic`
  #77; "does not have magic component" log) then calls the agent cast
  method `fcn.0041e4e0` (spell#, target), which drives `SMagic [0x658c38]`
  ([`skills-magic.md`](skills-magic.md)) and faction-checks the target via
  `fcn.004380a0`. Same machinery as player casting.
- Operand encodings ✅ (from the disassembler) — `move to $` (op2,
  `0x4318cb`) operand = location id → named-location table `[0x750d2c]`
  (16-byte records, name `+8`); `attack $` (op91, `0x43201d`) operand =
  npc id → `CAgentManager [0x658d50]` resolve; `cast spell # on $` (op80,
  `0x431d1b`) operands `[#spell,$target]` width 3. These are the on-disk
  program encodings; the disassembler resolves the ids to names via the
  same managers the engine uses elsewhere.
- Behaviour tick ✅ — `agentbehaviour.cpp` (`fcn.00411380`) executes the
  scripted action per frame (move/pathfind).
- Sleep state ✅ — `force sleep` → `FUN_00429cb0` (`agentnpc.cpp`, gated on
  "promoted"); the asleep flag is `agent+0x224` bit `0x200`, toggled by
  the sleep/wake helpers in `0x004d0xxx`. `agent+0x224` flag bits known:
  `0x200` asleep, `0x400000` sees-all. Exact bed-position/wake-condition
  fields 🟡.
- Command vocabulary ✅ (complete) — all **125** keyword templates
  recovered, with their `$`/`@`/`#` operand grammar, and categorised
  (movement/action/combat/perception/mood/dialogue/inventory/group/world/
  control-flow).
- Keyword → case index map ✅ (**resolved** — was a dead-end) — the parser
  iterates a **125-entry char-pointer array at `0x64a5b8`** (indices
  0..124), string-comparing each template against the line (`fcn.004fdf40`)
  in the match loop at `0x4300a1`; on a hit it jumps the switch by the
  same index (`jmp [i*4 + 0x4312f4]`), so **array index == switch case ==
  command id**. Head/tail verified (index 0 = `name $`, 124 = `boss npc`);
  the full table is in the [index map](#keyword--case-index-map-the-0x4312f4-switch)
  above. This corrects the earlier "bounds cannot be recovered" note: the
  match loop hard-bounds the array at 125 (`cmp i, 0x7d`), so no
  pointer-scan heuristic is needed. (The prior speculative "`be moody` =
  index 31" was wrong — it is index 27; index 31 is `set mood confused`.)
- Compiled-program slot widths ✅ — `0x64a7b0` is a `u16[125]` indexed by
  the same command id giving each command's width in the per-agent program
  buffer (`agent+0x2b4`); `InsertProgramLine` (`fcn.0042fba0`) uses it to
  size lines and relocate goto operands (ids 16/17/109). Width = `1 +
  operand slots`, a **string operand = 64 slots** (`say`/`log` = 64), and
  **width 0 = parse-time-only** setup verbs (no runtime instruction). This
  corrects the earlier "operand-spec `{count,type}` metadata" label — the
  packed words it saw were adjacent `u16` widths read as `u32`. The
  width-0 vs width-2 split is load-time config vs runtime op.
- Perception/aggro model ✅ — detection scan `fcn.004356f0`: range gate
  `dist ≤ sight·32` (sight = `CStats+0x20`, set by `set sight #`) then a
  world line-of-sight test (`fcn.0056fbc0`). `sees all` (`agent+0x224`
  bit `0x400000`) bypasses the faction filter; a target concealment flag
  (`[O+0x220] & 0x20`) halves the range by day / quarters it at night
  (`CClock GetHour`), i.e. the time-aware stealth mechanic. Matches feed
  the `seeing triggers event` hook (16-entry seen-list).
- Faction/alignment relations ✅ — `fcn.004380a0` is a bit-matrix lookup
  (`relmgr [0x658de8] + 0x424`, `bit = idB*stride + idA`) over the two
  agents' **CAlignment** entities (`agent+0x30`, `.\AGENTS\alignment.cpp`,
  id at `+0`), with a group-hierarchy-distance fallback (`< 25`). Gates
  perception/hostility; corrects agent.md's `+0x30` "i32" label (it is a
  pointer to the alignment entity).
- Relation builder & polarity ✅ — declared in `dat\alignment.dat`
  (`new alignment $`, `relation $,$,#`); the setter `fcn.00437ed0` sets
  the bit when value `< 25` and clears it (`fcn.004375b0`) when `≥ 25`,
  so a **set bit = hostile/reactive** (low value), propagated down the
  alignment hierarchy. (Pins the previously-🟡 polarity.)
- Perception field semantics ✅ (target `+0x220` bits) — code-anchored from
  the scan `fcn.004356f0`: `0x100` = un-perceivable (dropped at collection
  + skipped at scan), `0x20` = concealed (radius /2 by day, /4 at night via
  `GetHour` `fcn.0050bfe0` on clock `[0x658c1c]`), `0x40` = bypass the
  faction filter (`fcn.004380a0`). The `[O+0x224]` invisibility mask
  `0x10000100` (bits 8 + 28) is the engine-wide **"is invisible → hard-skip"**
  test, read in ~9 sites — the perception scan `fcn.004356f0`, the movement
  path (`0x415d3c`), render/agent (`0x448266`), and the query helper
  `and eax, 0x10000100` (`0x449703`) — so an object carrying it is dropped by
  perception, movement, and drawing alike. It is **set/cleared from script**:
  the agentscript commands **`set invisible` / `clear invisible`** (keywords
  registered by the parser `fcn.00431770`) and **`make object invisible for
  agent`** (runner `fcn.00430010` @ `0x4310b9`); the Invisibility/Ghost
  *skills* (`CMagicWizardBodySpiritSkill_Invisibility` /
  `CSurvivorDivineSkill_Ghost`) additionally grant it, but only through the
  shared-`CSkill` dynamic dispatch (their vtables are all base stubs). The
  **per-bit source split is now pinned** (`or [O+0x224],imm` sites):
  - **bit 8 (`0x100`) = general "invisible"**, set statically from several
    sources — the agentscript runner `fcn.00430010` (`set`/`make object
    invisible`, `0x4349f7`), the **magic effect system `.\magic\SMagic.cpp`
    (`0x4ce0e5`)** (so the Invisibility *spell*'s flag-set *is* static — its
    skill vtable is a stub, but the effect body sets the bit directly),
    Osiris npc-script frames (`0x513ba5`), and the agentnpc helpers
    (`0x4cf0a9`/`0x4cf906`/`0x4d592c`/`0x4d61ce`/`0x4d6c0e`).
  - **bit 28 (`0x10000000`) = a distinct secondary/internal invisibility**,
    set at a single site `0x4d0f6d` (agentnpc cluster).
  Perception, movement, and render all hard-skip on the combined mask
  (`0x10000100`), so an object hidden by *either* bit drops out of all three.
