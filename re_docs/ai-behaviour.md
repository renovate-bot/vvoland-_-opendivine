# Native AI behaviours (`CAgentBehavior` / `CGroupBehavior`)

Beneath the data-driven `agentscript` layer ([`npc-ai.md`](npc-ai.md))
the engine has a **compiled** AI layer: a hierarchy of C++ *behaviour*
classes that drive what an agent does on its own (idle routines, combat
tactics) and how a **group** of agents coordinates. An agent that isn't
running a specific script falls back to its behaviour object; groups add a
second behaviour that steers the whole pack. Source unit
`.\AGENTS\agentgroup.cpp` (groups) + `.\AGENTS\agentfight.cpp` (combat).

## Two families (MSVC RTTI)

### Individual agent behaviours — `CAgentBehavior`

The per-agent "personality" — the idle/ambient routine each NPC class
runs. Recovered concrete classes:

```text
CAgentBehaviorAnimal              CAgentBehaviorMarketBuyer
CAgentBehaviorCitizen             CAgentBehaviorMineDropper
CAgentBehaviorCitizenSpecial      CAgentBehaviorRabbit
CAgentBehaviorDefaultMonster      CAgentBehaviorRat
CAgentBehaviorDefaultMonsterSpecial   CAgentBehaviorRoomCrawler
CAgentBehaviorDungeonCrawler      CAgentBehaviorSoldierGuard
CAgentBehaviorGuard               CAgentBehaviorStrictGuard
CAgentBehaviorImp                 CAgentBehaviorTavernGuest
CAgentBehaviorLeaveTo
```

So a citizen wanders/visits the market (`MarketBuyer`, `TavernGuest`), a
guard patrols (`Guard` / `SoldierGuard` / `StrictGuard`), vermin scurry
(`Rat` / `Rabbit`), dungeon monsters roam (`DungeonCrawler` /
`RoomCrawler` / `DefaultMonster`), with `*Special` variants and utility
behaviours (`LeaveTo` = walk off to a target, `MineDropper`, `Imp`).

**Storage & assignment.** The agent holds its `CAgentBehavior` object at
**`agent+0x260`**. The factory `fcn.0042cb80` `new`s the right subclass for
the agent's class and stores it there; it runs from agent construction /
**`CAgent::Read`** (`fcn.0042eac0`, savegame load) and at runtime from the
**`agentscript` executor** (`fcn.004329a0`, [`npc-ai.md`](npc-ai.md)) — the
"individual behavior override" a script can apply. Vtables (RTTI-walked):
base `0x608578`, `Citizen` `0x6085cc`, `DungeonCrawler` `0x6085f4`,
`Guard` `0x608630`, `Rat` `0x6086bc` (unit `.\AGENTS\agentbehaviour.cpp`).
The full per-class table is enumerated below; note `fcn.00411380`
(referenced from [`animation.md`](animation.md) as "the agentbehaviour
tick") is specifically **`CAgentBehaviorLeaveTo::virtual_12`** — a behaviour
tick, not a dedicated animation-advance routine.

**The personality tick is vtable slot 3 (`+0x0c`)** — the slot that varies
across subclasses (slot 0 = dtor `0x412250`, slot 2 `0x40ed40` shared;
`Guard` additionally overrides slot 1 `0x40fb20`):

| Class | slot-3 tick fn |
|---|---|
| (base) | `0x522460` |
| `Citizen` | `fcn.0040f740` |
| `Guard` | `fcn.00412100` |
| `Rat` | `fcn.00410d60` |

Every tick opens with the same contract: gate on **`agent+0x218 == 1`**
(active/visible — the same gate as the animation tick), read the group ref
`agent+0x258`, and run the **alignment/relation check `fcn.004380a0`** (to
decide hostility toward what it perceives, [`npc-ai.md`](npc-ai.md)), then
branch to the per-class routine. The routines call a set of **shared
low-level helpers** rather than each having a bespoke loop: `fcn.0040eff0`
is an **agent-target resolve/validate** helper (it re-looks the handle
through `CAgentManager [0x658d50]` and checks the target's
active-flag `+0x218` / current behaviour `+0x2e4`), and `fcn.0040ecb0` is a
small **coordinate min/max** helper — both used across personalities, so
neither *is* "wander" or "patrol" by itself. The distinguishing logic lives
in the larger per-class bodies (e.g. `Citizen` `fcn.0040f740` →
`fcn.0040f6b0`/`fcn.0040f060`/`fcn.0040f250`; `Guard` `fcn.00412100`
calls the coord helper repeatedly to bound its post). So the individual
behaviour is a small per-class state machine over a shared
perceive→align→resolve-target→move spine.

**Register identity (verified):** the tick is called with `this` = the
**`CAgentBehavior` object** (not the agent); the agent is at
**`behaviour+0x04`**, a per-behaviour tick counter is incremented at
`behaviour+0x38`, and every tick gates on the **agent's** `+0x218 == 1`
(active/visible) then reads its group ref `+0x258`. (Confirmed in
`Citizen` `0x40f740`: `mov eax,[edi+4]; cmp byte [eax+0x218],1`, and in
`LeaveTo` `0x411380`.)

**Complete slot-3 tick roster** (all 18 classes, RTTI-walked from the code
vtables `0x6085xx`/`0x6086xx`; slot-0 dtor `0x412250` and the base no-op
tick `0x522460` shared):

| Class | vtable | slot-3 tick | family (by callees) |
|---|---|---|---|
| `CAgentBehavior` (base) | `0x608578` | `0x522460` | no-op (`ret`) |
| `Animal` | `0x608658` | `fcn.0040fe40` | perceive + move-spine + own `fcn.0040fdb0` |
| `Rabbit` | `0x60866c` | `fcn.00410090` | perceive + move-spine + own `fcn.00410000` |
| `Citizen` | `0x6085cc` | `fcn.0040f740` | perceive + move-spine (`fcn.0040f6b0`/`0f060`/`0f250`) |
| `CitizenSpecial` | `0x608714` | `fcn.004118b0` | perceive + roam-anchor `fcn.0040f250` + own `fcn.00411820` |
| `Guard` | `0x608630` | `fcn.00412100` | perceive + coord-bound post |
| `StrictGuard` | `0x608644` | `fcn.0040fc40` | **post-only** — coord `fcn.0040ecb0` + post-anchor `fcn.0040f0a0`, no perception |
| `SoldierGuard` | `0x608680` | `fcn.00410310` | perceive + post-anchor `fcn.0040f0a0` + own `fcn.00410280` |
| `TavernGuest` | `0x608694` | `fcn.00410660` | perceive + post-anchor + own `fcn.004105d0` |
| `MarketBuyer` | `0x6086d0` | `fcn.00410fe0` | perceive + post-anchor + own `fcn.00410f50` |
| `Imp` | `0x6086a8` | `fcn.00410a70` | perceive + post-anchor + own `fcn.004109e0` |
| `Rat` | `0x6086bc` | `fcn.00410d60` | perceive + move-spine |
| `DefaultMonster` | `0x6085e0` | `fcn.00411b50` | perceive + roam-anchor `fcn.0040f250` + own `fcn.00411a70` |
| `DefaultMonsterSpecial` | `0x608700` | `fcn.00412280` | perceive + roam-anchor + own `fcn.00411750` (+shared `fcn.00411a70`) |
| `DungeonCrawler` | `0x6085f4` | `fcn.00411cf0` | **rand-driven area roam**, no perception |
| `RoomCrawler` | `0x608608` | `fcn.00411cf0` | **same body as DungeonCrawler** |
| `LeaveTo` | `0x6086ec` | `fcn.00411380` | **walk to a target/exit**, no perception |
| `MineDropper` | `0x60861c` | `fcn.00411ff0` | minimal periodic action (`rand` + `fcn.00576200`), no move-spine |

So the personalities cluster into three families, distinguished by which
shared helper they route movement through:

- **Perceive→move** (call the alignment check `fcn.004380a0`): they resolve
  a target (`fcn.0040eff0`) and move via the spine `fcn.0040f060`, then run
  one per-class routine. Within this family the *anchor* differs: the
  citizen/guard set (`SoldierGuard`/`TavernGuest`/`MarketBuyer`/`Imp`) routes
  through the **post-anchor** helper `fcn.0040f0a0` (idle around a fixed
  home/post), while the monster set (`CitizenSpecial`/`DefaultMonster`/
  `DefaultMonsterSpecial`) routes through the **roam-anchor** helper
  `fcn.0040f250` (roam an area); `Animal`/`Rabbit`/`Rat` use the plain
  move-spine (free wander).
- **Move-only (no perception)**: `DungeonCrawler` (= `RoomCrawler`,
  rand-driven roam through `fcn.004475f0`/`00447780`/`004477d0` + post-anchor
  `fcn.0040f0a0`), `LeaveTo` (the same roam machinery without the `rand`
  pick — walk to a designated point), and `StrictGuard` (hold post: only
  coord + post-anchor, never leaves).
- **Action-only**: `MineDropper` — a small `rand`-gated periodic action
  (`fcn.00576200`), no move-spine.

The two anchor helpers `fcn.0040f0a0` (post/home) and `fcn.0040f250`
(monster roam) are near-identical: both walk toward an anchor point
(`+0x1c`/`+0x20`) using the shared pathing primitives `fcn.004270a0` /
`fcn.00427630` / `fcn.0043bcc0` and the round helper `fcn.005e5d40`, driving
the agent's walk slots around `+0x270`–`+0x280` (move target
`+0x270`/`+0x27c`/`+0x280`, `Walkcount` `+0x278`; see
[`npc-ai.md`](npc-ai.md)); they differ only in which anchor they target.
This is the same move-execution machinery [`npc-ai.md`](npc-ai.md) documents
on `fcn.00411380`: that doc picks one tick (LeaveTo's) as representative,
while the roster above shows the tick is polymorphic per personality.

**The per-class "own" helpers are not bespoke logic.** Eight of the
distinguishing routines listed in the roster (`fcn.0040fdb0` Animal,
`fcn.00410000` Rabbit, `fcn.00410280` SoldierGuard, `fcn.004105d0`
TavernGuest, `fcn.004109e0` Imp, `fcn.00410f50` MarketBuyer, `fcn.00411750`
DefaultMonsterSpecial, `fcn.00411820` CitizenSpecial) are **byte-identical
142-byte copies of one weighted candidate-selection routine** — the
compiler emitted a separate copy per class with only the personality's
weight-table address baked in. Each scans a candidate-point array inside the
behaviour object (`this + arg·32 + 0x18`, parallel to a per-class static
weight table in `.data`), tracks the minimum of the cross-metric
`tableWeight·(candidate[+0x1c]+1) − candidate·(scalar+1)`, and returns the
**index of the best-scoring candidate**. The personality-specific behaviour
comes from a **static weight table**, one per class — **verified layout: a
32-byte record = 7 `int32` slot-weights (`+0x00`) + 1 `int32` per-row
scalar (`+0x1c`)**; the `arg` (candidate category `0..2`) selects the row,
so each class table is **3 rows** and the class base is the *row-0* address
(the doc's earlier `0x6497e4` values were the `+0x1c` scalar, not the row
base — corrected below):

| Class | own-helper | row base (`+0x00`) | scalar (`+0x1c`) |
|---|---|---|---|
| `Animal` | `fcn.0040fdb0` | `0x6497c8` | `0x6497e4` |
| `Rabbit` | `fcn.00410000` | `0x649828` | `0x649844` |
| `SoldierGuard` | `fcn.00410280` | `0x649888` | `0x6498a4` |
| `TavernGuest` | `fcn.004105d0` | `0x6498e8` | `0x649904` |
| `Imp` | `fcn.004109e0` | `0x649948` | `0x649964` |
| `MarketBuyer` | `fcn.00410f50` | `0x649a08` | `0x649a24` |
| `DefaultMonsterSpecial` | `fcn.00411750` | `0x649a68` | `0x649a84` |
| `CitizenSpecial` | `fcn.00411820` | `0x649ac8` | `0x649ae4` |

Bases are `0x60` apart (3×32 B). Sample decoded rows `{7 weights | scalar}`:
`DefaultMonsterSpecial` row 0 = `{70,20,0,0,0,0,10 | 100}`; `SoldierGuard`
rows 0/1 = `{7,0,0,0,0,0,0 | 7}`; `Animal` row 0 = `{6,2,0,0,0,0,1 | 9}`.
(The `0x6499a8–0x6499e8` block is an unreferenced byte-copy of the Animal
table — dead data.)

Only `DefaultMonster`'s `fcn.00411a70` (219 B) and the move-only ticks
(`DungeonCrawler`/`RoomCrawler` roam, `LeaveTo`, `StrictGuard`,
`MineDropper`) are genuinely distinct bodies.

### Group behaviours — `CGroupBehavior`

The pack-coordination tactic shared by a group's members:

| Class | Name (data) | Tactic |
|---|---|---|
| `CStupidMonsterBehavior` | `Stupid` | no coordination — each acts alone |
| `CAmbushMonsterBehavior` | `Ambush` | wait concealed, strike when the target nears |
| `CWarMonsterBehavior` | `War` | organised assault |
| `CSurroundMonsterBehavior` | `Surround` | flank / encircle the target |
| `CLockUpEnemyMonsterBehavior` | `Lockup` | pin / block the target in |
| `CPatrolGroupBehavior` | (patrol) | move along a route together |
| `CTalkingGroupBehavior` | (talking) | conversational/idle group |

The data names (`War` / `Ambush` / `Lockup` / `Surround` / `Stupid`) live
as literals at `0x60906f..0x609090`, next to `agentgroup.cpp`. All derive
from the base **`CGroupBehavior`** (vtable `0x6090b4`); a behaviour's
`virtual_0` is the scalar-deleting dtor (e.g. `CWarMonsterBehavior` at
`0x41be20`).

### The tactic virtual (vtable slot 2, `+0x08`)

Diffing the vtables pins the per-tactic method to **slot 2 (`+0x08`)** —
the only slot that differs across the concrete classes (slot 1 `0x41a880`
and slot 3 `0x41a580` are shared overrides):

| Class | slot-2 tactic fn |
|---|---|
| (base `CGroupBehavior`) | `0x4e3f70` |
| `CStupidMonsterBehavior` | `fcn.0041a8a0` |
| `CAmbushMonsterBehavior` | `fcn.0041ab60` |
| `CWarMonsterBehavior` | `fcn.0041b440` |
| `CSurroundMonsterBehavior` | `fcn.0041ae40` |
| `CLockUpEnemyMonsterBehavior` | `fcn.0041b1a0` |
| `CPatrolGroupBehavior` | `fcn.0041bae0` |
| `CTalkingGroupBehavior` | `fcn.0041b6e0` |

All three share the same opening contract on the member agent: read the
**group reference at `agent+0x258`**, set the **engage flags**
(`agent+0x224 |= 0x800000` and `agent+0x220 |= 2`, the behaviour/combat
bits in [`agent.md`](agent.md)), take the **group target at `+0xc8`** (on
the group/leader object), and drive movement through the agent's
target/destination slots `+0x270` / `+0x27c` / `+0x280` via the shared
positioning helper `fcn.00414ef0`. They then differ by tactic:

- **Stupid** (`fcn.0041a8a0`) — minimal: each member just engages the
  group target on its own (no inter-member coordination).
- **Ambush** (`fcn.0041ab60`) — gated on a readiness field (`[target+0x40]`):
  hold until the target is in range, then engage.
- **War** (`fcn.0041b440`) — organised: calls the coordination helpers
  `fcn.00435390` / `fcn.004353d0` and **iterates the group members** with a
  callback (`fcn.00419940`) to assign the assault, rather than acting
  per-member-independently.
- **Surround** (`fcn.0041ae40`) / **Lockup** (`fcn.0041b1a0`) — same
  coordinated shape as War (group `+0x258`, target `+0xc8`, move helper
  `fcn.00414ef0`) but iterate the members with a different callback
  (`fcn.00419a40`) to assign **encircle** (Surround) vs **pin/block-in**
  (Lockup) positions around the target.
- **Patrol** (`fcn.0041bae0`, 39 bytes) / **Talking** (`fcn.0041b6e0`,
  19 bytes) — **passive**: they only touch the move target `+0x270` (Patrol
  walks the group along its route; Talking is an idle/conversational group)
  with no target engagement or member coordination.

So a group's tactic is one shared `CGroupBehavior` singleton whose slot-2
virtual runs against each member agent during its tick, reading the group
ref (`+0x258`) and target (`+0xc8`) and writing the member's engage flags
and move target.

## Construction & selection

The group-behaviour objects are built once as **singletons** by the
factory `fcn.0041be60` (`agentgroup.cpp`), which `new`s each class and
stores its vtable (`CStupidMonsterBehavior` → `0x609128`,
`CAmbushMonsterBehavior` → `0x60913c`, `CWarMonsterBehavior` → `0x609150`,
…). A group then points at the shared behaviour for its tactic.

Group definitions come from the **monster generator**
([`monsters.md`](monsters.md)): `fcn.0043ef40` parses the per-region group
list, matching the behaviour **by name** — an unknown name logs
`"Unknown group behavior %s in monster generator"`, and a region that
references a missing group logs `"Group %s not defined for region %s"`. So
groups are **per-region**, each carrying a named `CGroupBehavior`, a member
class distribution (`"add group \"%s\" probability"`,
`"Incorrect distribution in group"`), and the spawned monsters join it.

## Membership & runtime

An agent records its group as **`Group` / `GroupIndex`** (the agent dumper
prints `"Group=%d GroupIndex=%d"`, `agentgroup.cpp`). `agentscript`
([`npc-ai.md`](npc-ai.md)) manipulates groups at runtime with the verbs
**`create group with behaviour`**, **`add to group from`**, and
**`set group #`**, so scripted encounters can form a pack and assign its
tactic on the fly. Each frame an agent runs its individual `CAgentBehavior`
while the group's `CGroupBehavior` arbitrates shared targeting/positioning.

## Status

- Behaviour class hierarchy ✅ — both families fully enumerated from RTTI:
  17 individual `CAgentBehavior*` personalities + 7 `CGroupBehavior*`
  tactics (base `CGroupBehavior` vtable `0x6090b4`).
- Group-behaviour factory ✅ — singletons built by `fcn.0041be60`
  (vtables `0x609128`/`0x60913c`/`0x609150`/…).
- Selection ✅ — per-region group lists in the monster generator,
  behaviour matched **by name** (`fcn.0043ef40`); names `War`/`Ambush`/
  `Lockup`/`Surround`/`Stupid` at `0x60906f`.
- Membership ✅ — agent `Group`/`GroupIndex`; `agentscript` `create group
  with behaviour` / `add to group from` / `set group #`.
- Per-tactic logic ✅ (slot pinned) — the tactic is **vtable slot 2
  (`+0x08`)**: Stupid `fcn.0041a8a0` (independent engage), Ambush
  `fcn.0041ab60` (range-gated wait-then-strike), War `fcn.0041b440`
  (member-iterating coordinated assault via `fcn.00435390`/`fcn.004353d0`).
  All read the group ref `agent+0x258` and target `+0xc8`, set engage flags
  `+0x224|=0x800000` / `+0x220|=2`, and move via `+0x270/+0x27c/+0x280`
  (`fcn.00414ef0`). All 7 tactic bodies now enumerated: Surround
  `fcn.0041ae40` / Lockup `fcn.0041b1a0` (coordinated member-iteration like
  War, callback `fcn.00419a40` — encircle / pin), and Patrol `fcn.0041bae0`
  (39 B) / Talking `fcn.0041b6e0` (19 B) are passive (route-follow / idle,
  touch only the `+0x270` move target).
- Individual behaviour runtime ✅ — the `CAgentBehavior` object lives at
  **`agent+0x260`**, assigned by the factory `fcn.0042cb80` (from
  `CAgent::Read` `fcn.0042eac0` and the agentscript executor `fcn.004329a0`);
  the personality tick is **vtable slot 3 (`+0x0c`)** (`Citizen`
  `fcn.0040f740`, `Guard` `fcn.00412100`, `Rat` `fcn.00410d60`), each gating
  on `agent+0x218==1`, running the alignment check `fcn.004380a0`, then a
  per-class routine over shared helpers (`fcn.0040eff0` agent-target
  resolve, `fcn.0040ecb0` coordinate min/max).
- Per-personality bodies ✅ — **all 18 slot-3 ticks enumerated** (table
  above) and grouped into three families by their movement helper:
  perceive→move (post-anchor `fcn.0040f0a0` for the citizen/guard set,
  roam-anchor `fcn.0040f250` for the monster set, plain spine `fcn.0040f060`
  for Animal/Rabbit/Rat), move-only (`DungeonCrawler`=`RoomCrawler`
  rand-roam, `LeaveTo` walk-to-target, `StrictGuard` hold-post), and
  action-only (`MineDropper`). The tick receives `this`=behaviour with the
  agent at `+0x04`; `fcn.00411380` is `LeaveTo`'s tick (not an
  animation-advance routine). The per-class "own" helpers are **8 identical
  copies of one weighted candidate-selector** keyed by a per-class static
  weight table (`.data` `0x6497c8`–`0x649ae4`, 32-byte records = 7 int32
  weights + a +0x1c scalar, 3 rows/class); only `DefaultMonster`
  `fcn.00411a70` and the move-only ticks are distinct code.

## Citations

```text
div.exe:0x0041be60   group-behaviour factory — news the CGroupBehavior singletons.
div.exe:0x0041be20   CWarMonsterBehavior::virtual_0 (scalar-deleting dtor; base CGroupBehavior).
div.exe:0x0043ef40   monster-gen group parser — behaviour-by-name; "Unknown group behavior %s".
div.exe:0x006090b4   vtable.CGroupBehavior (base).
div.exe:0x00609128   vtable.CStupidMonsterBehavior · 0x60913c CAmbush · 0x609150 CWar.
div.exe:0x0060906f   group-behaviour name literals (War/Ambush/Lockup/Surround/Stupid).
div.exe:0x00608578   vtable.CAgentBehavior (base; slot-3 tick at +0xc).
div.exe:0x0040f740   CAgentBehaviorCitizen::virtual_12 (tick; this=behaviour, agent=[this+4]).
div.exe:0x00411380   CAgentBehaviorLeaveTo::virtual_12 (walk-to-target tick).
div.exe:0x00411cf0   CAgentBehaviorDungeonCrawler::virtual_12 (= RoomCrawler; rand-driven area roam).
div.exe:0x0040f0a0   post/home-anchor walk helper (citizen/guard family).
div.exe:0x0040f250   monster roam-anchor walk helper (default-monster family).
div.exe:0x0040f060   move-spine helper; 0x40eff0 target-resolve; 0x40ecb0 coord min/max.
```
