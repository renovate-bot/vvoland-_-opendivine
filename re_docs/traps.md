# Traps

Divine Divinity's trap system (`.\WORLD\Trapman.cpp`). A trap is a
**polymorphic effect** bound to a world location or region; when an
agent enters, the trap fires its effect (operate a door, deal damage,
teleport the victim, …). The system is versioned in saves as
`TrapsV0.935`.

## Trap class hierarchy

A trap's effect is a `CTrapState` subclass. The factory `FUN_00593b60`
selects the subclass from a trap **type code** via a jump table:

```text
cmp type, 5
ja  default            ; >5 → assert (Trapman.cpp:333)
jmp [table + type*4]
```

The six effect types, in jump-table order (the type code is the table
index; vtables at `0x00620914`, 12 bytes apart):

| Type | Class | Effect | Param |
|---:|---|---|---|
| 0 | `CTrapDoorState` | open/close a linked door (the same door state machine as [`object-interaction.md`](object-interaction.md)) | — |
| 1 | `CTrapToggleTrapState` | enable/disable another trap | `+0x0c` = target trap |
| 2 | `CTrapTeleportState` | teleport the victim | `+0x0c` = destination |
| 3 | `CTrapBreakState` | break a linked object | `+0x0c` = target object |
| 4 | `CTrapDamageState` | deal damage to the victim | uses `+0x18` |
| 5 | `CTrapPolymorphState` | polymorph the victim | uses `+0x14` |

`CTrapState` is the common base. Each constructed state object is a
fixed **28-byte (`0x1c`)** allocation:

```text
CTrapState:                  // 0x1c
    +0x00  vtable
    +0x04  state
    +0x08  state
    +0x0c  target / link      (toggle target, teleport destination, break object)
    +0x10  state
    +0x14  Polymorph param
    +0x18  Damage param
```

The link types (1/2/3) write the factory's target argument to `+0x0c`;
Damage and Polymorph carry their magnitude in `+0x18` / `+0x14`.

## Binding to locations & regions (`FUN_00593d60`)

Traps are defined in `traps.dat` and bound at load to named world
**locations** (point triggers) or **regions** (area triggers). Bad
references are reported:

```text
"Location %s is unknown in traps.dat"
"Unknown region %s in region trigger trap"
```

The loader also logs `"Loading trap locations"`, `"Loading trap
regions"`, and `"Loading traps"` (`FUN_004a0b10`).

## Save/restore (`FUN_005945c0`, `.\WORLD\Trapman.cpp`)

**Correction:** `FUN_005945c0` is **not** the runtime trigger — it is the
**Traps save-block reader** (`TrapsV0.935`, called only from the saveman
loader `fcn.00502bf0`, [`formats/savegame.md`](formats/savegame.md)). It
reads a **24-byte header** (`fcn.004f4c70`, the save read-primitive) then
the per-trap array: each trap is a **36-byte (`0x24`) record** whose
runtime CTrapState-chain fields are zero/`1`-initialised (`+0xc`=0,
`+0x10`=1, `+0x14`/`+0x18`/`+0x1c`/`+0x20`=0) and whose effect chain is
**reconstructed via the factory `FUN_00593b60`** (the per-state `new` +
ctor), with the per-trap variant data bulk-read by `fcn.004f4c00`. So the
factory `FUN_00593b60` runs at **load** time to rebuild trap state, not at
fire time — `FUN_005945c0` walking `[trap+0x08]`/`[trap]` is the
restore loop, not a per-fire execution.

The **runtime trigger** (the per-frame detection that an agent has entered
a trap's location/region and the resulting `CTrapState` slot-1 effect
execution) is a **separate** path, now partly located: the trap manager is
the global **`[0x658c18]`** (where the save reader installs the trap list),
and the trap-victim detection (`fcn.004453a0`, which reads `[0x658c18]`)
**finds candidate agents through the spatial cell grid** — the object query
`fcn.005709e0` ([`cell-grid.md`](cell-grid.md)) — rather than scanning all
agents, then gates on the trap's location/region (the
[`region.md`](formats/region.md) winding containment shared with shroud and
no-magic zones) before firing the `CTrapState` slot-1 effect. The exact
per-frame driver and the detect→fire ordering are still fuzzy 🟡
(`fcn.004453a0` is vtable-dispatched, ~2 KB).

### Event-driven fire core (`fcn.00593530`)

Alongside that per-frame proximity path, there is a **shared event-driven
fire backend** — `fcn.00593530` (`.\WORLD\Trapman.cpp`). It takes the trap
and a state index, stores the active index at `[trap+4]`, then **walks the
trap's `CTrapState` effect chain** (`[trap+0x20]` = chain base,
`[trap+0xc]` = count) firing each effect via `fcn.005934a0` (which loads
the effect's vtable and dispatches its slot-1 `Effect`). So one trap can
hold several chained effect states, all fired in sequence. Four entry
points converge on it:

| Entry point | Trigger |
|---|---|
| `fcn.00511b30` → `fcn.005935f0` → core | **Osiris** story DIV-functions (`Unknown trap %d in osiris execute trap` / `repair` / `break`) |
| `fcn.00588b04` → core (×2) | **object interaction / lever** (`trap is lever`, `Undefined trap (%d) toggled` — toggle on/off) |
| `fcn.005936c0` (from `fcn.00505bc0`) → core | a further fire variant |
| `fcn.00593af0` → core | break/region variant |

`fcn.005935f0` is a **state-cycle advancer** in front of the core: it
steps the trap's active-state index (`[trap+4]` vs the `[trap+8]` count,
wrapping via `[trap+0x10]`) so repeated triggers walk through a trap's
states. The upshot: **trap firing is event-driven** — Osiris, lever/use,
and region triggers all funnel into `fcn.00593530`'s effect-chain executor;
it is *not* a per-frame proximity loop that re-evaluates every trap (the
proximity path `fcn.004453a0` only locates the victim agent for
location/region traps, then triggers the same effect machinery).

## Effect virtuals & the damage path

Each `CTrapState` has a **2-slot vtable**: slot 0 the destructor, slot 1
the **effect** `virtual(victim)`; the trigger calls slot 1 on each state.
(Taxonomy-vtable note: Door/Toggle/Teleport/Break sit at `0x620914` /
`0x620920` / `0x62092c` / `0x620938`, but **Damage and Polymorph are a
separate pair at `0x6208d8` / `0x6208e4`** — not one contiguous
`0x0c`-stride block, as the table above implied.)

`CTrapDamageState::Effect` (`FUN_00592f20`, slot 1) is richer than "deal
`+0x18` damage", and the two state words carry distinct meanings:

- **`+0x14` = damage element / type** (a `0..8` switch); the factory
  leaves it `0` for a plain damage trap, so `traps.dat` damage traps hit
  type 0.
- **`+0x18` = magnitude.**

```text
type 0 (plain):   victim.vtable[+0x24](0, magnitude=+0x18, 0, 0)
                  → the CAgent HP-apply (FUN_00417550, combat.md slot +0x24):
                    direct HP loss of +0x18
type 1..8 (elemental):  damage = propBase + (rand() scaled by propRandom),
                  per-element magnitudes pulled from props.000 —
                  FireDamage{Damage,RandomDamage}, LightningDamage{…},
                  SpiritualDamage{…}, PoisonCloud{Amount,Damage,RandomDamage}
                  — lazily cached into [0x7516d4..0x7516f0] (init-bit gate
                  [0x7516f4]).
```

So a basic trap subtracts HP through the **same combat resolver as
melee/explosions**, while the elemental branches read the same damage
props the `CMagic*` spell effects use ([`skills-magic.md`](skills-magic.md),
[`formats/props.md`](formats/props.md)) — `FUN_00592f20` is shared
between traps and the spell-damage path (its other callers are in
`FUN_00587a20`), so traps and spells deal damage through one routine.

Disarming is a separate skill path (`CMagicDisarm`, `Disarm`,
`BoobyTrapLevel`).

## Citations

```text
div.exe:0x00593b60   FUN_00593b60   CTrapState factory — jump table on type 0..5.
div.exe:0x00593d60   FUN_00593d60   bind traps to locations/regions (traps.dat).
div.exe:0x005945c0   FUN_005945c0   trap trigger — run + free the trap's state list.
div.exe:0x00592f20   FUN_00592f20   CTrapDamageState::Effect (slot 1) — HP/elemental damage;
                                    shared with the spell-damage path (callers in FUN_00587a20).
div.exe:0x006208d8   vtable.CTrapDamageState   (+0x6208e4 = CTrapPolymorphState).
div.exe:0x004a0b10   FUN_004a0b10   trap loader ("Loading traps").
div.exe:0x00620914   vtable.CTrapDoorState  (taxonomy vtables, 0x0c apart:
                                    Door, ToggleTrap, Teleport, Break, Damage, Polymorph).
div.exe:0x00593530   FUN_00593530   trap fire core — walks CTrapState chain [trap+0x20]/[trap+0xc],
                                    fires each via fcn.005934a0 (slot-1 Effect vtable dispatch).
div.exe:0x005935f0   FUN_005935f0   state-cycle advancer in front of the core ([trap+4]/[trap+8]/[trap+0x10]).
div.exe:0x00511b30   FUN_00511b30   Osiris trap dispatcher (execute/repair/break trap) → fire core.
div.exe:0x00588b04   FUN_00588b04   object-interaction/lever handler ("trap is lever") → fire core.
```

## Status

- Trap taxonomy ✅ — six `CTrap*State` effect classes + `CTrapState`
  base, selected by a 0..5 type jump table in `FUN_00593b60`.
- `CTrapState` object size/shape ✅ — vtable + four parameter words.
- Location/region binding ✅ — `traps.dat` → named locations/regions
  via `FUN_00593d60`.
- Save/restore ✅ (was mis-labeled "trigger") — `FUN_005945c0`
  (`Trapman.cpp`) is the **Traps save-block reader** (24-byte header + 36-byte
  records, `fcn.004f4c70`/`fcn.004f4c00`), called only from saveman
  `fcn.00502bf0`; it rebuilds each trap's `CTrapState` effect chain via the
  factory `FUN_00593b60` on load. So the factory runs at load, not per-fire.
- Type-code → class mapping ✅ — read from the jump table at `0x593d40`:
  0=Door, 1=ToggleTrap, 2=Teleport, 3=Break, 4=Damage, 5=Polymorph.
- Per-type parameter fields ✅ — 28-byte object; link types use `+0x0c`,
  Damage `+0x18` (magnitude) + `+0x14` (element type), Polymorph `+0x14`.
- Effect virtuals ✅ — slot 1 of the 2-slot `CTrapState` vtable.
  `CTrapDamageState::Effect` (`FUN_00592f20`): type 0 = direct HP loss via
  the victim's combat HP-apply `vtable[+0x24]` (`FUN_00417550`); types
  1..8 = elemental damage with `props.000` magnitudes (Fire/Lightning/
  Spiritual/PoisonCloud `Damage`+`RandomDamage`). Shared with the
  spell-damage path. (Damage/Polymorph vtables are at `0x6208d8`/`0x6208e4`,
  correcting the "all at `0x620914`" note.)
- Trap firing ✅ (event-driven core mapped) — firing converges on the
  shared backend `fcn.00593530` (`Trapman.cpp`), which walks the trap's
  `CTrapState` effect chain (`[trap+0x20]`/count `[trap+0xc]`) and fires
  each via `fcn.005934a0` (slot-1 `Effect` vtable dispatch). Entry points:
  **Osiris** (`fcn.00511b30`→`fcn.005935f0`), **lever/object-use**
  (`fcn.00588b04`), and the `fcn.005936c0`/`fcn.00593af0` variants;
  `fcn.005935f0` cycles the trap's active state index between fires. So
  firing is **event-driven**, not a per-frame all-traps scan.
- Per-frame proximity driver 🟡 (narrowed; cause pinned) — for location/
  region traps the victim is located via `fcn.004453a0` (cell-grid object
  query `fcn.005709e0` + region containment), which then triggers the same
  effect machinery. The region-fire variant **`fcn.00593af0` is now
  characterised**: like the Osiris advancer `fcn.005935f0` it cycles the
  trap's active-state index (`+0x10` vs the `+0x14`/`+0` count, tests
  `byte[trap]&2`) and tail-calls the fire core `fcn.00593530`. It has **no
  static caller** — it is reached by load-then-call from the proximity
  detector, the same anonymous-dispatch pattern as every other trap trigger.
  So the residual is *only* the dispatch edge (which `call [reg]` site
  invokes it), not the fire logic; that edge is not statically xref-able (the
  proven cause, not an untraced gap).
- Save `Traps` block ✅ (shape) — 24-byte header + count × 36-byte trap
  records (`FUN_005945c0`); the on-disk `dat\traps.dat` per-field layout
  still ❓.
