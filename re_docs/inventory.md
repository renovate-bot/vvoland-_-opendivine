# Inventory & equipment slots

How an agent's carried and worn items are stored, and how the engine
finds the item in a given equipment slot. This is the storage side of
the "Inventory" and "Equip / unequip" STATUS items; the visual side is
in [`clothing.md`](clothing.md).

## Container chain

```text
CAgent ─ +0x23c → inventory  ─ FUN_004afb70 → CItemList
                  (+0x240/+0x244: the "Inventory %d,%d" handle pair,
                   see agent.md)
```

A `CItemList` is a flat, searchable item array:

```text
CItemList:
    +0x04  ptr → array descriptor:
                    [+0x00] base    — first item record
                    [+0x04] length  — capacity
                    [+0x08] stride  — bytes per item record
    +0x0c  i32 count                — live item count
```

Item record (relevant fields, confirmed from the gatherer and the slot
search):

```text
Item:
    +0x08  i32  catalogue kind index — × 0x94 into objects.000; the item's
                 kind (its ClothingCode lives at record+0x3c, see clothing.md)
    +0x0c  i16  slot tag             — which equipment slot this item occupies
```

## Find item by slot (`FUN_00422260`)

Equipment slots are not array positions — each worn item carries a
**slot tag** at `+0x0c`, and the engine finds a slot's item by scanning
the list for a matching tag:

```c
// FUN_00422260(CItemList *this, int index, int wantedSlotTag)
n = this->count;            // [this+0x0c]
desc = this->descriptor;    // [this+0x04]
for (i = index; i < n; i++) {
    item = desc.base + i*desc.stride;     // [desc+0]+i*[desc+8]
    if ((i16)item->slotTag == wantedSlotTag)   // [item+0x0c] == arg
        return item;
}
return 0;
```

## Equipment slots

The clothing gatherer `FUN_0043a5b0` ([`clothing.md`](clothing.md))
composes the body sprite from the items in the **visible** slots — the
tags it searches for are:

```text
{ 0, 2, 3, 4, 7 }
```

with the slot→part map now **proven** (correcting the earlier
"slot 7 = weapon" / "slot 0 first letter" inferences): **0 = helmet,
2 = torso armour, 3 = weapon (`local_2c` — the letters the composer's
"Helmet*" fields actually carry), 4 = shield, 7 = leggings**
([clothing.md](clothing.md) has the evidence chain: two-handed switch
on tag 3, shield-hide on tag 4 via `CItemStatistic+0x88`, and the
shipped `.key` layer inventories). Tags
`1`, `5`, `6` are non-visible carry slots (`5`/`6` = the two
interchangeable ring slots, fallback swap @`0x508e3c`; and
the like).

The item-stat vocabulary (`.\itemstat\itemgenerate.cpp`, a keyword
table at `0x006133xx`) names the slot/property set the item parser
understands, including the wearables:

```text
helmet  armor  gloves  boots  belt  ring  necklace  shield  weapon
```

(plus stat keywords: armorclass, offense, defense, hitpoints, numdice,
dicetype, diceadd, twohanded, onehanded, resistfire, charmlevel, …).

## Equipment-presence bitmask (`agent+0x294`, `FUN_0043b800`)

`FUN_0043b800` scans the same visible slots (via `FUN_00422260`) and
builds a small **bitmask of which slots are currently occupied** at
`agent+0x294` — `=0`, then `|=1` / `|=2` / `|=4` / … as each scanned slot
holds an item. It's a cheap "what is worn right now" cache that the
combat/animation code reads (it forwards into `FUN_004171b0`, the combat
region) rather than re-walking the item list — e.g. *has-shield* /
*has-weapon* gating. (Distinct from `FUN_0043a5b0`, which gathers the
ClothingCode *letters* for the body-sprite composer,
[`clothing.md`](clothing.md).)

## Equipment → stat bonuses (open; not a slot-scan fold)

Tracing how a worn item's `offense`/`defense`/`armorclass` reaches the
effective `CStats` block ([`stats.md`](stats.md)) has now **dead-ended on
a third independent angle**, but each pass has tightened the scope:

1. **Not the boost list.** The boost-add routine `FUN_00559c60`
   ([`stats.md`](stats.md)) has ~42 callers and **none are in the
   equipment/inventory code** — all potions (`FUN_00587a20`) and `CMagic*`
   buffs. (Rejected: "equipment applies bonuses as permanent boosts".)

2. **Not a `Recalculate` fold.** Reading both `Recalculate` slot-0 bodies
   end-to-end ([`stats.md`](stats.md)): the monster slot `FUN_0055a400` is
   a plain base→effective copy; the player slot `FUN_0055a2a0` derives
   `MaxHP`/`MaxMana`/`Offense`/`Defense` from the primary attributes ×
   per-class coefficients — **neither scans worn items**. The earlier
   "(equipment bonuses)" step in stats.md was a guess and has been removed.

3. **Not the `FUN_00422260` slot scanners.** All callers of the
   find-worn-by-slot helper were enumerated and classified, and the whole
   `0x439xxx`–`0x43bxxx` slot-scan cluster is **`CClothingVisualizer`**
   (RTTI-named: `FUN_0043ba90` = `CClothingVisualizer::virtual_12`,
   `FUN_0043afc0` the master appearance refresh that gathers clothing
   letters + retriggers animation via `timeGetTime`/Animan). It is purely
   the **hero sprite-composition** path — and its `+0x23c` is an *agent
   backreference* (offset reuse vs the inventory's `agent+0x23c`), not an
   item list. The other callers are the `+0x294` presence mask
   (`FUN_0043b800`), the equipment-slot **GUI plates**
   (`CShieldPlate::virtual_56` `FUN_00529c10` / `FUN_00529e40`), MPLAYER
   message echo (`FUN_00508d50`/`FUN_00509930`), and `special.cpp`
   (`FUN_00442ec0`). **None aggregate item stats into `CStats`.**

**Net scoping.** Because the player `Recalculate` rebuilds *both* the
derived effective stats *and* their base from the primary attributes
(clobbering anything written into the derived base fields), worn-item
bonuses to derived stats (`+hitpoints`/`+offense`/`+defense`) cannot
simply be added to the `CStats` base block — a recompute would erase them.
They must therefore be applied either (a) to the **primary attribute**
inputs (`Strength`/`Dexterity`/`Intelligence`/`MaxStamina`) at equip time,
or (b) in a **post-recompute additive pass** invoked alongside the boost
folds — neither yet located. The flat-additive item fields
(`armorclass`/`protect`/`armor`) and combat dice are instead read **live**
at hit-time off the worn weapon/armor ([`combat.md`](combat.md)), so part
of "equipment → effect" is by design *not* a fold at all. Next angle:
find writers of the primary-attribute fields (`CStats+0xa4..`) outside
level-up, or a recompute orchestrator that runs an equipment pass.

## Carry weight

Items carry a **`Weight`** value in the `objects.000` catalogue (it is a
named column in the catalogue CSV header, alongside `Id,Name,Weight,…`), so
weight is a per-*kind* property, not per-instance. The agent tracks a
**current carried weight at `agent+0x1b0`**: the object dumper prints it as
`"Current weight = %d"` straight from `[agent+0x1b0]` (`0x0042d6c2`; the
adjacent `[+0x1a4]` is the `"Summoned by"` id and `[+0x1c0]` a float). This
pins the previously-parked `+0x1b0` field.

**No carry limit is enforced.** There is no `"too heavy"` / `"overloaded"`
/ encumbrance string anywhere in the binary, and no pickup gate compares
`+0x1b0` against a maximum — so weight is **tracked and displayed (and fed
to trade pricing, [`items.md`](items.md)), not a movement/pickup
restriction**, matching the game's known lack of encumbrance. 🟡 The exact
accumulation site (where an added item's catalogue weight is summed into
`agent+0x1b0`) is not isolated: `+0x1b0` is reused as a field on the
inventory object (`agent+0x23c`, where it reads as a slot/index compared to
`-1`), so the 51 `+0x1b0` references can't be split agent-vs-inventory
without per-site class checks.

## Citations

```text
div.exe:0x0042d6c2   object dumper — reads carry weight [agent+0x1b0] ("Current weight = %d").
div.exe:0x00422260   FUN_00422260   find worn item by slot tag (linear scan of CItemList).
div.exe:0x0043a5b0   FUN_0043a5b0   clothing gatherer — searches slots {0,2,3,4,7}.
div.exe:0x0043b800   FUN_0043b800   equipment-presence bitmask builder (agent+0x294).
div.exe:0x0043ba90   CClothingVisualizer::virtual_12 — mask + appearance refresh on slot change.
div.exe:0x0043afc0   FUN_0043afc0   master appearance refresh (clothing letters + Animan retrigger).
div.exe:0x00529c10   CShieldPlate::virtual_56 — equipment-slot GUI plate render (not stat).
div.exe:0x004afb70   FUN_004afb70   resolve the CItemList from an agent's inventory.
                                    item-stat keyword table @ 0x006133xx (itemgenerate.cpp).
```

## Status

- `CItemList` layout ✅ — descriptor `{base,len,stride}` at `+0x04`,
  count at `+0x0c`.
- Item slot-tag model ✅ — `item+0x0c` = slot tag; slots are tags, not
  positions; `FUN_00422260` is a tag scan.
- Item kind link ✅ — `item+0x08` indexes `objects.000`.
- Visible equipment slot set ✅ **(map complete)** — `{0,2,3,4,7}` =
  helmet / torso / weapon / shield / leggings; the old "slot 7 =
  weapon" is retracted ([clothing.md](clothing.md)).
- Full item record layout ✅ **(RESOLVED — the on-disk slot record is
  byte-exact)**: the inventory pools are **TRamFile index/bulk pairs**
  (`inv.i<n>`/`inv.b<n>` = startup copies of
  `inventi/inventb` (n=0), `oinventi/oinventb` (n=1),
  `minventi/minventb` (n=2)). The i-file is a flat array of 28-byte
  records `{u32 blockOffset, u32 itemCount, u32 zero, char name[16]}`
  (blob length = next offset − own offset; names are write-only —
  regenerated as "Unnamed %d" on load). The b-file is 40-byte item-slot
  records:
  `{+0x00 u32 x, +0x04 u32 y (inventory-grid position), +0x08 u32 kind
  (objects.000 index, all < 7208), +0x0c i16 slotTag (−1 = not
  equipped), +0x0e u16, +0x10 u16 stack count, +0x12 u16 uninit,
  +0x14 u32 instance-flags dword (bit 21 unidentified; bit 23 cleared
  on load), +0x18 u8[16] value pool}` — validated byte-exact against
  all three shipped pairs (offset chains sum exactly to the b-file
  sizes). Managers: three TInventoryManagers in the global array
  `[0x658c0c]` (0 invent, 1 oinvent, 2 minvent; ctor `fcn.004b0a60`,
  TRamFile vtable `0x61611c`, ReadRecord `0x4ebe80`); record 0 is
  always "Root". The per-instance CItem state (durability/identified/
  boosts) serializes separately in `items.000`
  ([items.md](items.md)). The m/o prefix semantics (merchant vs object
  inventories) stay 🟡.
- Carry slots `{1,5,6,…}` ❓ — exact slot-id → ring/necklace/belt/etc.
  enumeration not yet pinned.
- Carry weight ✅ (field + source) — current carried weight at
  `agent+0x1b0` (dumper `0x0042d6c2` "Current weight = %d"); per-item
  weight is the `objects.000` `Weight` catalogue column. **No encumbrance
  cap** (no too-heavy/overloaded string, no max-weight gate) — display/
  trade only. Accumulation site 🟡 (`+0x1b0` reused on the inventory
  object). Pins the parked carry-weight field.
- Equipment-presence bitmask ✅ — `agent+0x294`, built by `FUN_0043b800`
  from the worn slots; a has-shield/has-weapon cache for combat.
- Equipment → stat bonus path ✅ **(RESOLVED)** — it was hiding in
  **CStats vtable slot 1** (`fcn.0055b9b0`, the recompute the earlier
  sweep read as slot-0-adjacent): a fixpoint loop over the 11-entry
  `CItem*` array at `CStats+0xf0..+0x118` adding item attribute boosts
  (`CItem+0x74..+0x80`) and direct combat-stat boosts
  (`CItem+0x2c..+0x50`) into the effective block — full field map in
  [stats.md](stats.md). The prior negative results stand: flat
  `armorclass`/dice ARE read live in combat, and the boost list is
  potions/spells only.
