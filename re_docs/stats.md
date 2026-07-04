# Character stats struct (`CStats`)

The RPG attribute block carried by every living agent — hitpoints,
mana, offense/defense, level, senses, and elemental resistances —
plus the timed-boost list that modifies them. Recovered from the stats
debug-dumper **`FUN_0055b630`** (`.\stats\stats.cpp`), which `fprintf`s
each field next to its `[esi + offset]` read (`esi` = the `CStats`
`this`).

The struct holds the attributes **twice**: one block of *effective*
(currently in-effect, post-boost) values and a parallel block of *base*
values, followed by the active-boost list. Stat consumers read the
effective block; boosts and level-ups write through to it from base.

## Ownership & class hierarchy (verified)

The `CStats` object is a **sub-object the owning agent points to at
`agent+0x2c`** — verified: four sites construct a stats object via a
per-class ctor and store the result at `[agent+0x2c]`, e.g. `0x425005`
/ `0x425081` (agent init) and **`0x42ebc8` inside `CAgent::Read`**
(`fcn.0042eac0`, the savegame agent-body reader). Combat reads it as
both `[attacker+0x2c]` and `[target+0x2c]` (`Hp` at `CStats+0x04`, see
[`combat.md`](combat.md)).

It is **polymorphic** (MSVC RTTI): base `CAgentStatistics` plus
`CMonsterStatistics` / `CPlayerStatistics`, with vtables at
`0x61cde8` / `0x61ce5c` / `0x61ce98` (ctors near `0x559c60` /
`0x55a190` / `0x55a1f0`). The per-class `vtable[+0x1c]` is the
opposed success-chance check decoded in [`combat.md`](combat.md)
(base 60 player / 50 monster, clamp 20–95 %, time-of-day modified).

### Virtual table (14 slots)

`CAgentStatistics` is an **abstract base**: its slots `0..8` are all the
MSVC pure-virtual stub `_purecall` (`0x5e5e44`), so only the concrete
`CMonsterStatistics` / `CPlayerStatistics` implement them; slots `9..13`
are concrete already in the base. Diffing the three vtables:

| Slot | Off | Player | Monster | Role |
|---:|---|---|---|---|
| 0–4,6,8 | `+0x00..+0x20` | impl | impl | per-class stat behaviour (pure in base) |
| 5 | `+0x14` | `0x55aee0` | `0x55b080` | **magic-status / SMagic-record apply** — the player slot reads `[0x658c38]+0x180 + id*0x54` ([`skills-magic.md`](skills-magic.md)) |
| 7 | `+0x1c` | `0x55b210` | `0x55b390` | **opposed success-chance** check ([`combat.md`](combat.md)) |
| 9 | `+0x24` | `0x559f40` | `0x522460` | small accessor (player reads `+0x74`) |
| 10 | `+0x28` | `0x55a090` | `0x559fd0` | accessor |
| 11,12 | `+0x2c`,`+0x30` | shared-ish | | small get/set (slot 11 is an empty `ret` hook) |
| 13 | `+0x34` | `0x55be30` | `0x55b820` | **attribute accessor / dumper** — prints `Strength/Dexterity/…` from the `+0x80` block (this is the call the agent dumper labels "Agent script", see [`agent.md`](agent.md)) |

So the combat/magic entry points are CStats virtuals: `+0x14`
(magic-status), `+0x1c` (hit chance), `+0x34` (attribute access). The
SMagic reader is **slot 5 (`+0x14`)**, player-specific — *not* a
`+0x50` slot (that offset, in [`combat.md`](combat.md), is a different
object: the per-element to-hit channel).

All integers little-endian; i32 unless noted.

## Layout

### Effective block (`+0x04 .. +0x34`)

| Offset | Field |
|---|---|
| `+0x04` | `Hp` |
| `+0x08` | `Mana` |
| `+0x0c` | `MaxHitpoints` |
| `+0x10` | `MaxMana` |
| `+0x14` | `Offense` |
| `+0x18` | `Defense` |
| `+0x1c` | `Level` |
| `+0x20` | `Sight` |
| `+0x24` | `Hearing` |
| `+0x28` | `ResistanceToLightning` |
| `+0x2c` | `ResistanceToPoison` |
| `+0x30` | `ResistanceToFire` |
| `+0x34` | `ResistanceToSpirit` |

### Base block (`+0x38 .. +0x68`)

The same 13 fields at `+0x38..+0x68`, but **not** in quite the same order
as the effective block: the **`CMonsterStatistics::Recalculate`** virtual
`FUN_0055a400` (slot 0) copies base→effective field-by-field and gives the
authoritative correspondence (the player slot `FUN_0055a2a0` **does not**
straight-copy these — it *derives* them, see *Player derived-stat formula*
below):

```text
eff +0x04 Hp        ← base +0x38      eff +0x20 Sight     ← base +0x54
eff +0x08 Mana      ← base +0x40      eff +0x24 Hearing   ← base +0x58
eff +0x0c MaxHp     ← base +0x3c      eff +0x28 ResLight  ← base +0x5c
eff +0x10 MaxMana   ← base +0x44      eff +0x2c ResPoison ← base +0x60
eff +0x14 Offense   ← base +0x48      eff +0x30 ResFire   ← base +0x64
eff +0x18 Defense   ← base +0x4c      eff +0x34 ResSpirit ← base +0x68
eff +0x1c Level     ← base +0x50
```

So the **base** order is `Hp, MaxHp, Mana, MaxMana, …` (`+0x3c`=MaxHp,
`+0x40`=Mana) — MaxHp and Mana are swapped relative to the effective
block's `Hp, Mana, MaxHp` (`+0x08`=Mana, `+0x0c`=MaxHp). **`+0x04..+0x34`
is the effective block, `+0x38..` the base** — confirmed both by this
copy direction and by the boost fold adding into `+0x04..+0x34`.

This `Recalculate` (slot 0) is the **effective = base reset** the boost
section needs: the recompute is *slot 0 reset* → *boost folds* (slots
`+0x24`/`+0x28`, see *Apply loop*), so a removed boost simply isn't
re-added on the next recompute. The monster slot ends by copying one
attribute, `+0x80 ← +0x88`. (An earlier note here speculated a third
*"equipment bonuses"* fold step after the boosts; that was **never
confirmed and is now removed** — neither `Recalculate` slot scans worn
items, and the equipment→stat path remains separately open, see
[`inventory.md`](inventory.md).)

## Player derived-stat formula (`FUN_0055a2a0`)

`CPlayerStatistics::Recalculate` (slot 0, `FUN_0055a2a0`) is **not** a
base→effective copy like the monster slot. For the four *combat-derived*
stats it **computes** them from the primary attributes scaled by a
**per-class coefficient**, indexed by the class/profession index at
**`CStats+0xa0`** (`c`). The four coefficient tables are consecutive
`float[]` in `.rdata`; the player and monster split here is the engine's
core "monsters have authored stats, players grow them from attributes +
class" rule.

```text
c (CStats+0xa0)                  class 0   class 1   class 2
A  HP per Stamina  (0x6542a8)      6         4         5
B  Mana per Int    (0x6542b4)      3         6         4
C  Offense per Dex (0x654284)      0.8       0.7       0.7
D  Defense per Dex (0x654290)      0.7       0.7       0.8

MaxHitpoints (+0x0c, +0x3c) = round( A[c] * MaxStamina(+0xb4) )
MaxMana      (+0x10, +0x44) = round( B[c] * Intelligence(+0xac) )
Offense      (+0x14)        = round( C[c] * Dexterity(+0xa8) )
Defense      (+0x18)        = round( D[c] * Dexterity(+0xa8) )
ResPoison    (+0x2c)        = floor( Strength(+0xa4) * 0.4 ) + base(+0x60)
ResFire      (+0x30)        = floor( Dexterity(+0xa8) * 0.4 ) + base(+0x64)
```

(`round` is the shared float→int helper `fcn.005e5d40`; the `* 0.4` is the
compiler's `imul 0x66666667; sar edx,2` reciprocal-multiply.) Current
**`Hp`/`Mana` are then set to the new max** (`+0x04 = +0x0c`, `+0x08 =
+0x10`) — i.e. a player recompute (level-up / character init) heals to
full. `Level` (`+0x1c`), `Sight` (`+0x20`), `Hearing` (`+0x24`),
`ResLightning` (`+0x28`) and `ResSpirit` (`+0x34`) are copied straight
from their base fields (not derived). The tables have a 4th trailing slot
whose value isn't a clean coefficient (raw `0x00000002` read as float),
so only classes `0..2` are real — matching the game's **three professions:
Warrior, Wizard, Survivor**. These are the same three classes that ship as
character sprite banks (`war`/`wiz`/`sur` × m/f, [`formats/heroes.md`](formats/heroes.md))
and carry class-specific abilities (`CSpecialMove_Wizard`/`_Survivor`,
`CSurvivorLoreSkill_*`, `CMagicWizardMatterSkill_*` in the RTTI). The
coefficient profile names them: **class 0 = Warrior** (highest HP factor
`A=6×Stamina`, lowest mana `B=3×Int`), **class 1 = Wizard** (highest mana
`B=6×Int`, lowest HP `A=4`), **class 2 = Survivor** (balanced `A=5`/`B=4`,
the dexterity-leaning Offense/Defense `C=0.7`/`D=0.8`). (The 0/1/2 → name
mapping is inferred from the coefficient shape + the class-skill RTTI; the
char-select assignment site is `CStats+0xa0`, written at e.g. `0x42be11`.)
The attribute block is reset effective`+0x80 ← base+0xa4` (player; the
monster slot uses `+0x80 ← +0x88`, a tighter layout), so **`+0x80..` is
the effective attribute block** that boosts (`Type` 10–13) and combat
read.

### Boost list & back-reference

| Offset | Type | Field |
|---|---|---|
| `+0x6c` | — | unaccounted (padding/unused) |
| `+0x70` | ptr | `Backreference` — back to the owning agent |
| `+0x74` | ptr | boost list base |
| `+0x78` | i32 | `BoostlistSize` (live count) |
| `+0x7c` | i32 | boost list **capacity** (grows by 8) |

Each boost-list element is a **20-byte (`0x14`) record** — the dumper
loop (`FUN_0055b630` @ `0x55b7f3`, stride confirmed `0x14` by the manager
`FUN_00559d50`) prints them as `Duration=%d,TimeStarted=%d,Type=%d,
Value=%d`, and the manager adds a fifth field (the boost **id**, searched
at `+0x10`):

```text
struct Boost {        // 0x14 = 20 bytes
    i32 Type;         // +0x00 — which attribute / effect
    i32 Value;        // +0x04 — magnitude applied
    i32 Duration;     // +0x08 — lifetime in clock ticks
    i32 TimeStarted;  // +0x0c — CClock time when applied ([0x658c1c])
    i32 Id;           // +0x10 — boost id (key for refresh/remove)
};
```

A boost expires when `now − TimeStarted ≥ Duration` (time from the world
clock `[0x658c1c]`); while active it adds `Value` to the attribute named
by `Type` in the effective block. `FUN_00559d50` **refreshes** an
existing boost by id — it scans the list (stride `0x14`) for
`record.Id == id`, logging `"Can't find back boost with ID %d"` on miss,
and re-stamps the record's `TimeStarted` (`+0x0c`) to the current clock
plus its `Type`/`Value`. (Earlier notes had this as a 16-byte
`{Duration,TimeStarted,Type,Value}` record; the dumper's field order and
the manager's `0x14` stride correct both the size and the layout.)

**Adding a boost (`FUN_00559c60`).** The counterpart to the refresh —
`AddBoost(id, value, type, duration)`. It first walks the list and
**drops already-expired entries** (`now − TimeStarted > Duration`, skip
`Duration == −1`), then if the live count `+0x78` has reached the
capacity `+0x7c` it **grows the vector by 8** records (realloc,
`.\stats\stats.cpp`), and finally appends the new 20-byte record stamped
`TimeStarted = now`. So the lifecycle is **add → refresh-by-id → fold →
expire-on-next-add/fold**, all keyed by the `Id`.

The **42 `AddBoost` call sites** are the **timed-buff sources**, and they
resolve to two producers (verified — the immediate `push` before each
`AddBoost` is the `Type` arg):

- **Potions** — `FUN_00587a20` (`.\…\potion`): `StrengthPotion` adds a
  **Type 10 (Strength)** boost for `StrengthBoostDuration`, plus
  `MagicPotion` / `HealthPotion` (the instant heals), `InvisibilityPotion`,
  and **`RestorationPotion`** — durations from the `…BoostDuration` props.
- **`CMagic*` buff spells** (the `0x4cd…0x4d5` effect cluster) — the
  signature is **`fcn.004d5620`, the all-attributes buff**: four
  consecutive `AddBoost` calls with **Type 10/11/12/13** =
  Strength/Dexterity/Intelligence/MaxStamina (the `CStrengthSpell`/
  `CBlessMagic` family); `fcn.004ce310` / `fcn.004d5aa0` add the
  combat-stat / single-attribute buffs.

**No caller is in the equipment/inventory code** — so worn-item bonuses do
*not* flow through the boost list (revising an earlier hypothesis; the
equipment→stat path is a separate, still-open mechanism, see
[`inventory.md`](inventory.md)).

**Apply loop & `Type` enumeration.** Active boosts are folded into the
effective block by two `CStats` virtuals that each walk the `0x14`-stride
list, skip expired records (`now − TimeStarted > Duration`; `Duration ==
−1` ⇒ permanent), and `add record.Value` to the effective field selected
by `record.Type`:

| `Type` | += offset | Field | Folder |
|---:|---|---|---|
| 0 | `+0x0c` | `MaxHitpoints` | `FUN_00559fd0` (slot `+0x28`) |
| 1 | `+0x10` | `MaxMana` | ″ |
| 2 | `+0x14` | `Offense` | ″ |
| 3 | `+0x18` | `Defense` | ″ |
| 4 | `+0x20` | `Sight` | ″ |
| 5 | `+0x24` | `Hearing` | ″ |
| 6 | `+0x28` | `ResistanceToLightning` | ″ |
| 7 | `+0x2c` | `ResistanceToPoison` | ″ |
| 8 | `+0x30` | `ResistanceToFire` | ″ |
| 9 | `+0x34` | `ResistanceToSpirit` | ″ |
| 10 | `+0x80` | `Strength` | `FUN_00559f40` (slot `+0x24`, player) |
| 11 | `+0x84` | `Dexterity` | ″ |
| 12 | `+0x88` | `Intelligence` | ″ |
| 13 | `+0x90` | `MaxStamina` | ″ |

So `Type` is a dense index over the boostable stats: the **derived /
max** combat stats (`MaxHitpoints … ResistanceToSpirit`) plus the player
RPG **attributes** — and *not* current `Hp`/`Mana` (`+0x04`/`+0x08`),
`Level` (`+0x1c`), or current `Stamina` (`+0x8c`), which are never
boost targets.

Because this loop **adds boost values into `+0x04..+0x34`**, that block is
confirmed the **effective** one (`+0x38..+0x68` is base) — resolving the
earlier "could be reversed" caveat. Since the fold only ever *adds*, a
reset of effective = base must run before it (otherwise the values would
accumulate every call); that reset is the **`Recalculate`** virtual
(slot 0, `FUN_0055a400`, documented under *Base block* above), so
removing/expiring a boost takes effect by dropping out of the next fold
rather than subtracting in place.

## Agent RPG-attribute block (separate from `CStats`)

`CStats` above is the **combat** block. The owning **agent** carries a
*second*, distinct block of **RPG attributes**, dumped by the agent
debug-printer `FUN_0055be30` (which first calls `FUN_0055b630` for the
`CStats` part, then prints these). Like `CStats` it is stored **twice**
(base + effective), the two blocks **`0x24` apart**:

| Field | base off | effective off |
|---|---|---|
| `Strength`    | `+0x80` | `+0xa4` |
| `Dexterity`   | `+0x84` | `+0xa8` |
| `Intelligence`| `+0x88` | `+0xac` |
| `Stamina`     | `+0x8c` | `+0xb0` |
| `MaxStamina`  | `+0x90` | `+0xb4` |
| **`Experience`** | **`+0x94`** | **`+0xb8`** |
| `Armor`       | `+0x98` | `+0xbc` |
| `Damage`      | `+0x9c` | `+0xc0` |

(offsets are agent-relative.) So **`Experience` is an agent attribute**
(`+0x94` base / `+0xb8` effective), *not* a `CStats` field — while
`Level` is the `CStats` field at `CStats+0x1c`. `Hitpoints`/`MaxHitpoints`
also print from agent `+0x38`/`+0x3c`. This corrects the earlier loose
"Experience in the stat block" note: the two blocks (combat `CStats` vs
agent RPG attributes) are distinct sub-structs.

## Citations

```text
div.exe:0x0055b630   FUN_0055b630   CStats debug-dumper — fprintf of every labelled
                                    attribute; this doc is built from its [esi+off] reads.
                                    Source unit: ".\stats\stats.cpp".
div.exe:0x0055b820   FUN_0055b820   caller of the dumper.
div.exe:0x0055be30   FUN_0055be30   caller of the dumper.
div.exe:0x00559c60   FUN_00559c60   boost add — expire-stale, grow-by-8, append timed record.
                                    Callers: potions (FUN_00587a20) + CMagic buffs (0x4d2xxx).
div.exe:0x00559d50   FUN_00559d50   boost refresh-by-id (scans the 0x14-stride list,
                                    "Can't find back boost with ID %d", re-stamps TimeStarted).
div.exe:0x00559fd0   FUN_00559fd0   boost fold — combat stats (Type 0..9 → +0xc..+0x34).
div.exe:0x00559f40   FUN_00559f40   boost fold — player attributes (Type 10..13 → +0x80..+0x90).
div.exe:0x0055a400   FUN_0055a400   Recalculate (slot 0, monster) — plain effective = base copy,
                                    no formula (monsters carry authored stat values).
div.exe:0x0055a2a0   FUN_0055a2a0   Recalculate (slot 0, player) — DERIVES MaxHP/MaxMana/Offense/
                                    Defense from attributes × per-class coeff (CStats+0xa0 = class).
div.exe:0x006542a8   coeff A  {6,4,5}    HP per MaxStamina, class-indexed.
div.exe:0x006542b4   coeff B  {3,6,4}    Mana per Intelligence.
div.exe:0x00654284   coeff C  {0.8,0.7,0.7}  Offense per Dexterity.
div.exe:0x00654290   coeff D  {0.7,0.7,0.8}  Defense per Dexterity.
div.exe:0x005e5d40   FUN_005e5d40   shared float→int round helper.
```

The owning agent links to its `CStats`; the agent struct
([`agent.md`](agent.md)) carries the spell/treasure/attitude fields,
while the numeric attributes live here.

## Status

- Effective block (`+0x04..+0x34`) ✅ — 13 fields, sequential, confirmed
  from the dumper.
- Base block (`+0x38..+0x68`) ✅ — base-vs-effective assignment now
  **confirmed** by the boost apply loop (`+0x04..+0x34` = effective).
- Boost record shape ✅ — **20-byte (`0x14`)** record
  `{Type+0, Value+4, Duration+8, TimeStarted+0xc, Id+0x10}` (from the
  dumper's print order + the manager's stride; corrects the earlier
  16-byte `{Duration,TimeStarted,Type,Value}` guess). Expiry
  `now−TimeStarted ≥ Duration` off the world clock `[0x658c1c]`; refresh
  by id is `FUN_00559d50`. `Type` enumeration ✅ (0..13 → effective
  field; see apply table).
- Boost lifecycle ✅ — add `FUN_00559c60` (expire-stale → grow-by-8 →
  append, `+0x78` count / `+0x7c` capacity), refresh-by-id `FUN_00559d50`,
  fold (below), inline expiry. Sources are **potions + spell buffs** only,
  not equipment.
- Boost apply loop ✅ — `FUN_00559fd0` (Types 0..9, combat) / `FUN_00559f40`
  (Types 10..13, player attributes) walk the list, skip expired
  (`now−TimeStarted > Duration`, `−1`=permanent), `add Value` to the
  effective field. Confirms `+0x04..+0x34` = effective.
- Recalculate / effective=base reset ✅ — **monster** slot 0
  (`FUN_0055a400`) copies the base block into the effective block
  field-by-field (mapping above; corrects the base block's `MaxHp`/`Mana`
  order). **Player** slot 0 (`FUN_0055a2a0`) instead **derives**
  `MaxHitpoints`/`MaxMana`/`Offense`/`Defense` from the primary attributes
  × a per-class coefficient (`CStats+0xa0` = class; tables `0x6542a8` /
  `0x6542b4` / `0x654284` / `0x654290`), heals to full, and copies the
  remaining fields straight — see *Player derived-stat formula*. This is
  the engine's "monsters authored / players grown from attributes" split.
- Equipment bonus fold ❌ (does not exist here) — neither `Recalculate`
  slot touches worn items; the speculative third "equipment fold" step has
  been removed. Worn-item → stat is a **separate, still-open** path tracked
  in [`inventory.md`](inventory.md).
- Recompute orchestrator 🟡 (body readable; only the caller anonymous) —
  the `CStats` vtable is now enumerated (14 slots, `0..52`); the **slot-0
  recompute/derive body is `CPlayerStatistics::virtual_0` = `fcn.0055a2a0`**
  (monster has its own slot-0), and it is **static and readable**: it stores
  `Level` (`+0x1c`) and `Experience` (`+0x94`) then **float-derives** further
  fields (the `fimul`/`fild` chain — the "players grown from attributes"
  derivation). So the recompute *logic* is recoverable by reading slot 0;
  what stays 🟡 is only the **orchestrator caller** (which invokes
  `[vtable+0]` on a boost/level change) — *(partially superseded: a
  concrete static caller of the recompute is now known — the level-up
  block invokes slot 1 at `0x42ade4`, and equip/unequip call it too)*.
  Corrected slot map (from dumping the vtables `0x61ce98`/`0x61ce5c`):
  **slot 1 (`+0x04`) = the equipment/stat recompute fold**
  (`fcn.0055b9b0` player / `0x55a460` monster copy-only — see below),
  **slot 3 (`+0x0c`) = the damage fold** (`fcn.0055ca10`/`fcn.0055a970`;
  the earlier "slot 12 (+0x30)" label was wrong — that RTTI-era name
  survives only as the historical "virtual_12", combat.md), slot 5
  (`+0x14`, `fcn.0055aee0`) = SMagic-table consumer, slot 52 (`+0x34`
  end, `fcn.0055be30`) = the stat dumper.
- `+0x6c` ✅ **(re-identified — the earlier "requirement block" reading
  was a misread)**: `CStats+0x6c` is the **weapon-mastery chance-to-hit
  cache**, written by `fcn.0042b540` from the slot-1 equipment fold
  (Sword-class weapon → `SwordChanceToHit[rank(skill 0x41)]`, Mace →
  `MaceChanceToHit[rank(skill 0x42)]`, all other classes → 0; only those
  two curves exist) and read as the `+ WeaponChanceToHit` term of the
  to-hit base ([combat.md](combat.md) closed form). The `+0x64..+0x70`
  comparisons inside `fcn.0055b9b0` that suggested a "requirement block
  on CStats" are actually against **`CItem+0x64/68/6c/70`** — the
  *item's* Str/Dex/Int/Sta requirements ([items.md](items.md)) checked
  per equipped slot; the CStats offsets were an offset collision.
- **Slot 1 = the equipment→stat fold ✅ (`fcn.0055b9b0`)** — the
  long-open "equipment fold" residual. Fed by an **11-entry `CItem*`
  array at `CStats+0xf0..+0x118`** (rebuilder `fcn.0041c610` via
  resolver `fcn.0057a970`; equip handler `fcn.0041dd70` with evaluator
  `fcn.0042b1a0`; unequip `fcn.0041cf50`). The body is a **fixpoint
  loop**: reset effective attrs from base, add each active item's
  `CItem+0x74/+0x78/+0x7c/+0x80` → Str/Int/Dex/MaxSta, re-check every
  item's requirements/durability/two-handed rule and restart if a slot
  got disabled; then base→effective copy, boost folds (vt+0x24/+0x28),
  the class-coefficient MaxHp/MaxMana/Offense/Defense derivation, and
  the **direct item combat-stat fold**: `CItem+0x2c→MaxHp, +0x30→
  MaxMana, +0x34→Offense, +0x38→Defense, +0x3c→Sight, +0x40→Hearing,
  +0x44→ResLightning, +0x48→ResPoison, +0x4c→ResFire, +0x50→ResSpirit`,
  plus the `+0x6c` ChanceToHit refresh and item-granted skill
  add/remove (`0x544140`). **Weapon dice and armorclass are
  deliberately NOT folded** — dice are read live per swing and armor is
  read live on the defense side (hit-location roll, combat.md).
- Stat-derivation 🟡 (mechanism found) — the derive virtual
  **`CPlayerStatistics::virtual_0` (`fcn.0055a2a0`, slot 0)** computes derived
  stats via **per-class float-table multipliers**: `fld [0x6542a8 + …]` /
  `fld [0x6542b4 + …]` `fimul` a computed input (from `CStats` `+0x4c`/`+0x50`/
  `+0x58`), indexed by the **class id at `+0xa0`** (`edi = [esi+0xa0]`). This
  is the *same per-class table family* (`0x65429c`/`0x6542a8`/`0x6542b4`) the
  combat damage uses for `mastery_fraction[weapon]` ([`combat.md`](combat.md)),
  so `+0xa0` is the character class/profession that drives both combat
  scaling and stat derivation. So derived stat = `class_table[+0xa0] × input`.
  Remaining: the exact per-table semantics (which derived stat each table
  column yields) and the boost-apply/expire fold (the timed-boost list tick).
- Level-up threshold (re-checked, stays dynamic) — `virtual_0` **does not**
  compute `Level` from `Experience`: it stores `Level` (`+0x1c`) as a *given*
  value and copies `Experience` (`+0xb8`→`+0x94`), then derives the other
  stats. So the XP→Level threshold is *not* in this readable derive virtual —
  confirming [`progression.md`](progression.md)'s finding that it lives in the
  dynamic level-up path (offset-reused, not statically isolable), unlike the
  combat fold which *was* readable.
