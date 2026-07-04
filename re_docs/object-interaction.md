# Object interaction (`CObject::Use`)

How `div.exe` reacts when something *uses* a world object — the player
clicking a door/chest/lever, an inventory item being applied, or an
Osiris script triggering an object. All paths converge on a single
`thiscall` method, **`FUN_00588b00`** (`CObject::Use`, source path
`.\WORLD\objects.cpp`), which dispatches on the object's catalogue
flags and mutates the instance's runtime state.

This builds on [`formats/objects.md`](formats/objects.md): that doc
covers the static object *kind* catalogue (`objects.000`) and the
value-bit unpacker `FUN_005918b0`. This doc covers the *runtime*
instance and the use-time behaviour.

All integers little-endian; image base `0x00400000`.

## Pipeline

Interaction enters as an **agent command message** and fans out to the
two "use" actions, both of which call into `CObject::Use`:

```text
command message (code,source)
  └─ FUN_0050a290   message parser ("Can't parse message cause I dont know what it is (code=%d source=%d)")
       └─ FUN_00509f10   command switch (cmp <code>,0x37; jump table)
            ├─ FUN_00507130  use *world* object   ─┐
            │    └─ FUN_005a3230  resolve object under tile, then Use
            └─ FUN_00507180  use *inventory* object ┤
                 └─ FUN_004b18f0  UseInventoryObject()
                                     │
   Osiris script ────────────────────┤
     method.CDIVINITYOsirisObjectFunction (div.exe:0x00516239) → FUN_005a3230
                                     │
                                     ▼
                         FUN_00588b00  CObject::Use   (SEH frame, handler 0x005fe078)
```

`FUN_005a3230` is the world-side resolver; it guards against bad input
with `"U1:Use has a tile index problem"`, `"U1:Tried to use non
existing object !!!"`, and `"U1:Use has a little tile index problem.
!!! BUGGGER ALERT !!!"` (strings at `0x006212cc`, `0x006212a4`,
`0x00621264`). `FUN_005a3690` (size 0x4c5) is a near-twin resolver that
drives the bit-toggle helper `FUN_0058a630` instead of the inline
toggles — used by the script/lever-style "toggle" entry.

## Runtime instance state word

A placed object instance carries a 32-bit **flags word** plus a packed
value pool immediately after it, read with the same unpacker as the
catalogue: `FUN_005918b0(flags+4, flags, bit_index)`. The runtime
flags word is enumerated verbatim by the instance debug-dumper
**`FUN_005cfe3d`**, which is to this word what `FUN_00581520` is to the
catalogue record.

### Instance layout (confirmed offsets)

Pinned from `FUN_0058a630` and the `Use` body (`ebx` = instance):

```text
instance + 0x18   u16   object index / handle
instance + 0x24   u32   object *kind* index — × 0x94 (148) + catalogue base
                        [<objectmanager> + 0x1c] yields the objects.000 record
instance + 0x2c   u32   dirty / redraw flags (bit 1 = 0x4 marks "needs redraw")
instance + 0x30   u32   runtime state flags word (the sb_* bits below)
instance + 0x34   ...   packed value pool (the *_value fields; unpacker reads here)
```

So `FUN_005918b0(instance+0x34, instance+0x30, bit)` reads a runtime
`sb_*_value`, exactly mirroring the catalogue's `flags_a`/`sub_values`
pair. In `Use`, `lea ebp,[ebx+0x30]` is the flags pointer.

Complete bit map, decoded end-to-end from the dumper. "value" flags
gate a `*_value` field in the value pool (read by `FUN_005918b0` with
the bit index); "bool" flags are read directly as `(word >> bit) & 1`.

| Bit | Mask | Flag | Kind |
|---:|---|---|---|
| 0 | `0x00000001` | `sb_function_parameter` | value |
| 1 | `0x00000002` | `sb_slot` | value |
| 2 | `0x00000004` | `sb_treasure` | value |
| 3 | `0x00000008` | `sb_count` | value |
| 4 | `0x00000010` | `sb_key` | value — required key id |
| 5 | `0x00000020` | `sb_door` | value — marks a door |
| 6 | `0x00000040` | `sb_lever` | value — marks a lever |
| 7 | `0x00000080` | `sb_inside` | bool |
| 8 | `0x00000100` | `sb_osiris` | value |
| 9 | `0x00000200` | `sb_disappears` | bool |
| 10 | `0x00000400` | `sb_chest` | value |
| 11 | `0x00000800` | `sb_light` | value |
| 12 | `0x00001000` | `sb_transforms` | value — transform target kind |
| 13 | `0x00002000` | `sb_wt_use` | bool |
| 14 | `0x00004000` | `sb_use_on` | value |
| 15 | `0x00008000` | `sb_generated` | bool |
| 16 | `0x00010000` | `sb_value` | value |
| 17 | `0x00020000` | `sb_function` | value |
| 18 | `0x00040000` | `sb_move` | bool |
| 19 | `0x00080000` | `sb_item_class` | value |
| 20 | `0x00100000` | *(unused)* | — |
| 21 | `0x00200000` | `sb_locked` | bool — **runtime state** |
| 22 | `0x00400000` | `sb_broken` | bool — **runtime state** |
| 23 | `0x00800000` | `sb_can_carry` | value |
| 24 | `0x01000000` | `sb_walk_through` | bool |
| 25 | `0x02000000` | `sb_closed` | bool — **runtime state**, open/closed |
| 26 | `0x04000000` | `sb_property` | value |
| 27 | `0x08000000` | `sb_stolen` | bool — runtime state |
| 28 | `0x10000000` | `sb_player_block` | bool — runtime state |
| 29 | `0x20000000` | `sb_invisible` | bool — runtime state |
| 30 | `0x40000000` | `sb_inventory` | value |
| 31 | `0x80000000` | `sb_strength` | value |

The low bits mirror the kind (seeded from the catalogue at spawn);
`sb_locked`/`sb_broken`/`sb_closed`/`sb_stolen`/… are mutable runtime
state that `Use` writes. This ordering is the instance word's own — it
differs from the catalogue `flags_a` ordering in
[`formats/objects.md`](formats/objects.md).

Note the distinction inside `Use`: object *kind* properties
(door/lever/chest/value etc.) are read from the **catalogue record**
via `FUN_005918b0` with indices loaded from `DAT_00655970`; the
open/closed/locked/broken *runtime state* is read and written directly
on the **instance flags word** with `or`/`xor`/`test` masks (and via
the helper `FUN_0058a630`, which holds `xor <flags>,0x02000000`
(closed) and `xor <flags>,0x00200000` (locked)).

## The apply triple

Every state change runs the same three steps, so the world stays
consistent:

```text
FUN_0056e2c0   un-occupy: remove the object from the cell/collision grid
               (wrapped in lock acquire/release FUN_00471c00 / FUN_00471d70)
   <toggle the runtime state bit on the instance flags word>
FUN_00572100   re-occupy: re-insert into the grid with the new footprint
FUN_00585900   recompute the object's sprite/animation for the new state
               (calls into the sprite layer FUN_00547a20 / FUN_00547a80 / FUN_00548ad0)
```

This is exactly the door mechanic: an open door is passable and shows
the open sprite; a closed door occupies its cells and blocks. The same
triple is reused by object placement (`FUN_0058584e`, `FUN_00587427`).

## Per-type handlers (inside `FUN_00588b00`)

### Doors — open / close

Two near-identical blocks (`0x005894cd…` and `0x005897df…`) run the
apply triple around a `xor <flags>,0x02000000` toggle of `sb_closed`.
The open/close **sound** is chosen by the door bit:

```text
0x00589691   test byte [flags], 0x20          ; sb_door?
0x0058969c   push 8                            ; door-open sound slot
0x005896a0   push 9                            ; door-close sound slot
0x005896a2   call FUN_00442090                 ; play object sound (fmt "voice\ac%d_%d")
             ; sound-manager singleton in ecx = 0x0064ab34
```

### Locks & keys

`sb_locked` (bit 21, `0x00200000`) is a **hard gate**, not a key
comparison: in `Use`, a locked-and-unbroken object takes the early-out
branch (`test eax,0x200000` → `0x00589dd2`) and never reaches the
open/container code. The required key id is the `sb_key` value (bit 4).

Unlocking therefore happens *outside* the open path — by clearing the
bit. The clear is a `xor <flags>,0x00200000` done inline at `0x005894d2`
or, more generally, in the state-transition helper `FUN_0058a630`
(see *Break / transform* below), which on a break clears both
`sb_locked` and `sb_closed`. The skill-driven unlock sources are **two
RTTI-confirmed `Matter`-school skills**: the thief **`CSurvivorThiefSkill_
LockPick`** (vtable `0x61c0b4`; item `OBJECT_LOCKPICK`) and the wizard
**`CMagicWizardMatterSkill_Unlock`** (vtable `0x61b4dc`) — the Unlock
spell. Both share the skill base (vtable slot 0 `0x543ff0`) and their
apply methods forward to the shared skill dispatcher `fcn.00541840` with a
per-skill type constant (`0x40` for Unlock); Osiris scripts are the third
source.

**Key-item → unlock comparison — isolated** (in the `Use` locked-branch,
`0x00589440`–`0x005894d2`): the handler reads the lock's **required key id**
via the property getter `FUN_005918b0` (→ `esi`), resolves the **player's
inventory** via `FUN_004afb70` (= the `CItemList`) on the singleton
**`[0x658c0c]`**, and
**searches that inventory for an item whose id matches** via
`FUN_004aef00` (`cmp eax, -1` → not found). On a match it falls through to
`xor [flags], 0x00200000` (`0x005894d2`) — clearing `sb_locked`. So the
unlock test is *"does the player carry the item whose id equals the lock's
key value?"*; no match → the branch skips and the object stays locked.
(The door lock itself is the `sb_locked` *property bit*, not a struct
field — the `+0x2ac` "Locked=%d" the dumper prints is a separate
serialization/world-lock counter.) The exact `sb_key` property-arg form
into `FUN_005918b0` is the only fine detail left.

> **`[0x658c0c]` is *not* the player agent — distinguish the two globals.**
> The player *agent/object* (Player.cpp, built by `fcn.004a90e0`) is the
> singleton **`[0x658c04]`** ([`agent.md`](agent.md)). **`[0x658c0c]`** is a
> *separate* small **12-byte** object allocated in `init.cpp`
> (`fcn.004fa4b0(0xc,…)` at `0x4a0caf`, before the player is built at
> `0x4a1561`) with ~184 readers — a player/inventory **controller** handle,
> not the agent. `FUN_004afb70` resolves the active inventory `CItemList`
> from it, which is why the unlock check reads through `[0x658c0c]` rather
> than `[0x658c04]`. Both are written by the boot path (`fcn.00499990` /
> `fcn.004a0b10`); conflating them is easy since both are "the player" in
> casual terms.

### Chests / containers — open inventory plate

When a container is `sb_closed && !sb_broken` it opens its linked
inventory as a GUI window:

```text
0x00589b33   call FUN_005918b0  (bit 0x1e=30)   ; container/inventory link value
0x00589b46   call FUN_004ae360                  ; look up inventory by id
                                                 ; object manager singleton [0x00658c0c]
0x00589b6e   "Internal InventoryPlate %d"  (id + 0x186a0)   ; 0x0060919c
0x00589b96   call FUN_005270c0                  ; open/create the GUI plate
                                                 ; GUI manager singleton ecx = 0x00745454
0x00589b9f   mov dword [plate + 0x18], 1         ; mark plate open
```

### Levers / switches / traps

Levers (`sb_lever`, bit 6) toggle and propagate to a linked target.
The propagation goes through the toggle-style resolver `FUN_005a3690`
(same `"U1:Use…"` tile-resolution diagnostics as `FUN_005a3230`),
which resolves the target tile/object and drives the transition helper
`FUN_0058a630`. A lever wired to a trap with no handler logs
`"Undefined trap (%d) toggled"` (`0x0062018c`). `FUN_005a3690` is
called from combat (`FUN_0042b4b0`), Osiris
(`CDIVINITYOsirisObjectFunction`, `0x00515ec8`), and world events
(`FUN_00517680`, `FUN_00582430`).

(Note: `.\WORLD\ItemLink.cpp` / `itemlink.dat` is a separate system —
an *item-name → id* table consumed by Osiris, not the lever geometry.)

### Break / transform — `FUN_0058a630`

The shared state-transition helper. Given an instance and a target
kind index, it:

```text
recompute sprite (FUN_00585900) + re-occupy grid (FUN_00582630)
spawn/resolve the transformed instance (FUN_005873f0) → edi
or  [edi+0x30], 0x00400000            ; set sb_broken
xor [edi+0x30], 0x02000000  (if set)  ; clear sb_closed  → forced open
xor [edi+0x30], 0x00200000  (if set)  ; clear sb_locked
  if sb_inside (bit 8): FUN_005918b0(+0x34, bit 8) → FUN_0050fa60   ; play break sound
or  [edi+0x2c], 4                     ; mark needs-redraw
```

This is the "break it open" / transform transition: breaking a locked
container forces it open (clears `sb_locked` + `sb_closed`) and marks
it `sb_broken`.

`sb_transforms` (flags bit 12, `0x1000`) morphs the object into another
kind: `Use` reads the transform target via the unpacker (bit 12 value =
new kind index) and re-spawns the object at the same location through
`FUN_005873f0`, whose result takes over `[instance+0x24]`.
`sb_disappears` removes it instead. Using an object flagged as both is
a content error and logs `"Object is being used that disappears and
transforms at the same time !!!"` (`0x00620140`).

### Script-driven (`sb_function` / Osiris)

`sb_function` + `sb_function_parameter` let an object invoke a script
function on use; `sb_osiris` marks objects owned by the Osiris VM
(`"Object now belongs to Osiris (Key=%d)"`, `0x006118c0`).

The invocation site is inside `Use`: in the open/close transition,
when the flag word has bit 8 (`0x100`) set, the object's matching value
is read and fired into the Osiris event system —

```text
0x00589832   test eax, 0x100                  ; object has an event/function?
0x0058983d   push 8
0x00589844   call FUN_005918b0                 ; read the object's bit-8 value (event id)
0x0058984a   call FUN_0050f700                 ; .\OSIRIS\Events.cpp — fire the event
                                                 ; event manager singleton [0x007447dc]
```

The companion transition fires a sound instead via `FUN_0050fa60` at
`0x00589537`. The Osiris binding
`method.CDIVINITYOsirisObjectFunction` (`0x00516239`) calls the same
`FUN_005a3230` resolver, so scripted use and player use share the
`CObject::Use` path; the actual `"Trigger event %d"` /
`"Npc %s triggers event %d"` logging lives in the event executor
(`FUN_00431770` / `FUN_004329a0`).

## Object combination / alchemy (`dat\objects.dat`)

"Use **item** on **object/item**" interactions (alchemy, crafting,
key-on-lock, fill-flask, …) are **data-driven** by the plaintext
**`dat\objects.dat`** ("Extended object functionality data file", packed in
[`global.cmp`](formats/cmp.md), ~2810 data lines). It has two parts:

1. **Binding table — 1362 lines** of
   `use object <A> on object <B> with transform code <T> parameters #,#,#,#`:
   using object **A** (template id) on object **B** fires **transform code
   T**. Many B's share one `(A,T)` — e.g. one reagent usable on a whole
   range of target ids (`use object 4180 on object 3209..3227 → code 1`).
2. **Transform definitions — 223** `start transform code <T> … end transform
   code` blocks giving each code's effect, via sub-blocks **`with using
   object do …`** / **`with used object do …`** containing actions:
   `object disappears`, `create object <id>`, `use used object`, and the
   property setters `make poison duration # hours damage #` / `make food
   boost #` / `make drink boost #` / `make weapon dice # damage # shape #` /
   `make armor class # shape #` / `make useable|moveable` / `set value #` /
   `set function #` / `set slot #` / `set light #` / `set transforms #` /
   `object # is of special type #` (with `clear …` inverses).

So a recipe like `code 20 = {using object disappears; used object
disappears; create object 1861}` is **two items consumed → one produced** —
the alchemy/crafting core (the file's `// ALCHEMY - Bronthion` section
spells out *flask + ingredient = potion* tiers). This is the data behind
the `CObject::Use` path above: the use resolver looks up the `(using, used)`
pair, runs the bound transform's actions on the two objects, and the
property setters are how a produced object gets its consumable/weapon/armor
stats. Item-template ids here are the same space as
[`formats/objects.md`](formats/objects.md) / `itemlink.dat`
([`formats/itemlink.md`](formats/itemlink.md)).

### Alchemy recipe rules & ingredient catalogue (from the `// ALCHEMY` block)

`objects.dat`'s `// ALCHEMY - Bronthion` comment block spells out the
potion-brewing rules the transform codes implement:

```text
small flask  + plant/mushroom/etc = minor potion
normal flask + plant/mushroom/etc = normal potion
big flask    + plant/mushroom/etc = super potion
minor potion + normal/super augmentor = normal potion
normal potion + super augmentor       = super potion
potion + same-or-lower potion         = the lower tier (merge)
```

with the concrete ingredient / vial object-id catalogue:

```text
augmentor plants : weak 3209-3214 · strong 3220-3227
mushrooms        : red 1438/2177-2179 (health) · white 1437/2180-2183 (poison)
                   yellow 2184/2186 (stamina)
plants           : reddish 211 (health) · blueish 210/2176/2185/2187 (mana)
                   whitish 212 (stamina) · drudanae 3216-3218 (drug)
                   rotten food 3778-3782 (poison)
empty vials      : small 4183 · normal 1941 · super 6096
filled small vial: red 4180 health · blue 4179 magic · green 4178 poison
                   yellow 4177 stamina · white 4182 restore · black 4181 shadow
                   orange 4176 strength · lightblue 6114 drug · violet 6125 elixier
```

So filling an empty vial with an ingredient yields the colour/effect potion
of the matching tier, and augmentors/potion-merges promote tiers — the
`make poison/food/drink boost` setters in the command vocabulary are how a
produced potion's consumable effect ([`consumables.md`](consumables.md)) is
stamped. This is the game's full alchemy content, recoverable verbatim.

## Citations

```text
div.exe:0x00588b00   FUN_00588b00   CObject::Use — interaction dispatcher (entry; SEH frame,
                                    handler 0x005fe078). Body analysed from 0x00588b04.
                                    Source path: ".\\WORLD\\objects.cpp".
div.exe:0x005a3230   FUN_005a3230   world-side Use resolver (tile→object); shared by Osiris.
div.exe:0x005a3690   FUN_005a3690   toggle-style Use resolver; drives FUN_0058a630.
div.exe:0x004b18f0   FUN_004b18f0   UseInventoryObject().
div.exe:0x00507130   FUN_00507130   player action: use world object.
div.exe:0x00507180   FUN_00507180   player action: use inventory object.
div.exe:0x00509f10   FUN_00509f10   agent command switch; routes the use-commands.
div.exe:0x0050a290   FUN_0050a290   command message parser.
div.exe:0x00516239   CDIVINITYOsirisObjectFunction → FUN_005a3230 (scripted use).
div.exe:0x005cfe3d   FUN_005cfe3d   runtime instance flags dumper — names every sb_* bit.
div.exe:0x005918b0   FUN_005918b0   flag/value-bit unpacker (see formats/objects.md).
div.exe:0x0056e2c0   FUN_0056e2c0   un-occupy cell/collision grid (lock-guarded).
div.exe:0x00572100   FUN_00572100   re-occupy cell/collision grid.
div.exe:0x00585900   FUN_00585900   recompute object sprite/animation for new state.
div.exe:0x0058a630   FUN_0058a630   break/transform transition: set sb_broken, clear
                                    sb_closed/sb_locked, play break sound, mark redraw.
div.exe:0x005873f0   FUN_005873f0   spawn/resolve the transformed instance.
div.exe:0x00582630   FUN_00582630   re-occupy grid (variant used by the transition helper).
div.exe:0x0050fa60   FUN_0050fa60   play sound by id (break / close-transition sound).
div.exe:0x0050f700   FUN_0050f700   .\OSIRIS\Events.cpp — fire an Osiris event from Use
                                    (object event id); event manager singleton [0x007447dc].
div.exe:0x00431770   FUN_00431770   event executor ("Trigger event %d").
div.exe:0x004329a0   FUN_004329a0   NPC event executor ("Npc %s triggers event %d").
div.exe:0x0042b4b0   FUN_0042b4b0   combat → FUN_005a3690 (objects broken by attacks).
div.exe:0x00442090   FUN_00442090   play object sound sample (fmt "voice\\ac%d_%d").
div.exe:0x004ae360   FUN_004ae360   look up an object's inventory by id.
div.exe:0x005270c0   FUN_005270c0   open/create a GUI plate window.
```

## Status

- Use dispatch entry & call chain ✅ — message switch → use actions →
  `CObject::Use`, including the Osiris path.
- Apply triple (un-occupy / re-occupy / sprite-refresh) ✅ — confirmed
  reused across all state changes and by placement.
- Door open/close + sound slots (8/9) ✅.
- Chest → "Internal InventoryPlate" open ✅ (inventory link = value bit
  30; container id namespaced by `+0x186a0`).
- Instance layout ✅ — kind index `+0x24` (× 0x94 into the catalogue),
  flags word `+0x30`, value pool `+0x34`, redraw flags `+0x2c`,
  confirmed from `FUN_0058a630` and the `Use` body.
- Locked/closed/broken runtime bits ✅ — positions 21/25/22 confirmed
  from the dumper and the inline toggles.
- Lock gating ✅ — `sb_locked` is a hard gate in `Use` (locked +
  unbroken → early-out). Unlock is a bit-clear done by the transition
  helper / lockpick / Osiris.
- Lock/key item match 🟡 — `sb_key` holds the required id, but the
  key-item → unlock comparison site is not yet pinned.
- Lever / break / transform ✅ — propagation via `FUN_005a3690` →
  `FUN_0058a630`; triggered by combat, Osiris, and world events.
- `sb_function` / Osiris invocation ✅ — isolated the call site: `Use`
  reads the object's bit-8 value and fires it through `FUN_0050f700`
  (`.\OSIRIS\Events.cpp`, event manager `[0x007447dc]`); the companion
  transition plays a sound via `FUN_0050fa60`.
- Transform / disappear ✅ — `sb_transforms` (bit 12) re-spawns the
  object as the bit-12 target kind via `FUN_005873f0`; `sb_disappears`
  removes it.
- Full runtime flags-word bit order ✅ — all 32 bits decoded from the
  dumper (table above); bit 20 is unused.
