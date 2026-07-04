# Items — `CItemStatistic` & the item data model

The item system has two layers: a **static stat definition**
(`CItemStatistic`, loaded from the item-statistics table —
`.\…\itemstatistic.cpp`, "Loading item statistics") and the **runtime
instance** carried in inventory ([`inventory.md`](inventory.md)). This
doc pins the static property schema, recovered from the item table's
column names; affixes ([`formats/itemgen.md`](formats/itemgen.md)) and
treasure drops ([`formats/treasure.md`](formats/treasure.md)) both
reference and modify these properties.

## `CItemStatistic` property schema

Every column of the item-statistics table (each a discrete keyword the
loader matches), grouped:

```text
Identity   item · type · slot · itemlevel · level · picture · visuals
Slot/kind  weapon · shield · helmet · gloves · boots · belt · ring ·
           necklace · armor · onehanded · twohanded
Weapon kind  Staff · Dagger · Crossbow · Spear · Hammer · Mace  (+ sword/bow)
Damage     numdice · dicetype · diceadd        (the dice combat rolls)
           shapedamage · uponhit               (shape-change & on-hit dmg)
Defense    offense · defense · armorclass · protect · armor · resistfire (+ resists)
Bonuses    hitpoints · speed · sight · hearing · minstamina · maxshape
Magic      charm · charmlevel · cast · spell(s) · skill(s)
Requirement require · allow · using · canremove · after · handicap · modifier
Rarity     common · rare · artefact
On-hit special  Deathblow · Frost · Stun · "Life drain" · "Mana drain"
```

### Bindings to the rest of the engine

- **Combat dice** — `numdice` / `dicetype` / `diceadd` are exactly the
  fields the damage calculator reads ([`combat.md`](combat.md),
  `fcn.004edef0`); `offense`/`defense`/`armorclass`/`protect` and the
  resists feed the to-hit gate and the subtraction mitigation.
- **On-hit specials** — the `uponhit` column selects `Deathblow` /
  `Frost` / `Stun` / life-/mana-drain procs.
- **Magic items** — `cast` / `spell` / `charm` bind a `CMagic*` effect
  ([`skills-magic.md`](skills-magic.md)) to the item (wands, charms,
  cast-on-use); `charmlevel` is the effect level.
- **Weapon-type via `charm`** — verified against the shipped
  `itemstat\itemstat.txt` (153 `begin item` + 70 `begin charm` blocks), a
  weapon's `charm "<name>"` doubles as its **weapon-class tag**: the most
  common charm names are the nine weapon types **Sword / Staff / Mace / Axe
  / Spear / Bow / Crossbow / Hammer / Dagger** (each defined by a
  `begin charm … end charm` block), separate from the **skill-charms** a
  weapon can also carry (e.g. `Knife` = `numdice 1 dicetype 9 diceadd 3`
  with `charm "Backstab"` + `charm "Dagger"`). So `charm` is the general
  item→`CMagic*` binding *and* the proficiency/animation weapon-class tag —
  the weapon-class charms back the `CMagic*` weapon-handling effects. The
  **charm-definition block** (all 70 share the shape)
  `begin charm  name "<charm>"  type <cast>  attacking  [picture #]
  "<CMagic effect name>"  end charm` binds the charm name to the named
  `CMagic*` effect fired on attack (`attacking` on every one; `picture` on
  50 of them) — so an item's `charm "X"` resolves to charm-def X → its
  `CMagic*` effect. That closes the `itemstat.txt` schema: 153 item blocks
  + 70 charm blocks, both fully field-mapped.
- **Requirements** — `level` / `itemlevel` / `minstamina` / `require`
  gate equipping; `slot` + the kind flags decide which equipment slot.

## Runtime instance — `CItem` ✅ (RESOLVED)

The runtime item instance is a **0xbc-byte class `CItem`** (RTTI
`.?AVCItem@@`, vtable `0x6138c4`, 1 virtual), allocated in
`fcn.004b8ac0` (`itemstatistic.cpp:1781`, `push 0xbc`) and
**copy-constructed (`fcn.004b67b0`) from a prototype `CItem` embedded
at `CItemStatistic+0x94`** (prototype default-ctor `fcn.004b66e0`;
`CItemStatistic+0x118 = self` is the prototype's `+0x84` back-pointer —
i.e. **`CItem+0x84` = owning `CItemStatistic*`**). World/inventory
records reach the `CItem` through the resolver
**`fcn.0057a970(mgr [0x750cfc], id, &flags, …)`**, which requires
**bit 19 (`0x80000`) of the per-record flags dword = "is an item"**;
that flags dword sits at **`+0x30` of a world-object record** and
**`+0x14` of the inventory slot record** (the dword
`TInventory::SplitItem` copies) — inside the bits-10..23 band the
`fcn.005990c0` fixer clears. Key `CItem` fields, all with literal-offset
writers (the "heap-dump-bound" verdict below is retracted):

```text
+0x10  armorclass           (defense-side hit-location roll adds it, combat.md)
+0x14  charm quality        (CItemGeneration_AddCharmQuality::Apply 0x5779e0)
+0x18  max durability       (i32; grown by CItemGeneration_Durability::Apply 0x577a60)
+0x1c  current durability   (i32; 3 writers — see below)
+0x2c..+0x50, +0x74..+0x80   stat-boost targets (Statsboost 16-case table 0x577680:
                             stat-id 0→+0x74, 1→+0x78, 2→+0x80, 3→+0x7c, 4→+0x2c,
                             5→+0x30, 6→+0x34, 7→+0x38, 8→+0x3c, 9→+0x44, 10→+0x4c,
                             11→+0x50, 12→+0x48, 13→+0x10, 14→+0xc, 15→+0x40; all add)
+0x54  charm-slot count     (clamped ≤ 5, 0x57a81a)
+0x58  identified (cached bool; see below)
+0x5c  speed                (CItemGeneration_Speed::Apply 0x577b50, add)
+0x64/+0x68/+0x6c/+0x70  Str/Dex/Int/Sta equip requirements (checked per
                     slot by the equipment fold fcn.0055b9b0 and the
                     equip evaluator 0x42b1a0 against CStats +0x80..+0x90)
+0x84  owning CItemStatistic*
+0xa0  attached-charm list  (CItem::AddCharm 0x4b7ae0)
```

**Durability** — `+0x18` max / `+0x1c` current, three independent
writers: (1) `CItemGeneration_Durability::Apply` `0x577a60` (roll
`min[+0x3c]+rand%(range+1)` → `add [item+0x18]`; if max > 3,
`[+0x1c] = 1 + rand%max`, floor/clamp `[+0x1c] ≤ [+0x18]`); (2) the
**Repair skill body `fcn.004d1620`** (props `RepairQuality` indexed by
Repair rank ÷ 5, `[+0x1c] = max×quality%/100` if greater); (3) the
trade-plate stock normalizer `fcn.0041c800` (merchant stock set to full:
`[+0x1c] = [+0x18]`). There is **no separate max field elsewhere** —
`RepairQuality` is the props curve, not a `CItemStatistic` max.

**Identified** — two-level: the **unidentified bit is bit 21
(`0x200000`) of the record flags dword** (set by the random-loot
spawner `or [obj+0x30], 0x200000` at `0x5c342b`; toggled off by the
**Identify skill body `fcn.004d0ff0`** via the bit-helper
`fcn.00591940(…, 0x15)`), plus a **cached bool `CItem+0x58`** (zeroed
in the prototype ctor; forced 1 by the trade-open walker `fcn.0041c720`
— merchant items are always identified; gating the equip/benefit
evaluator `fcn.0042b1a0`).

**Charges — positively do not exist as a field.** No charges keyword in
the itemstat schema, no "charge" strings, and no `dec/sub` on any
`CItem` offset attributable to item use. What the managers' UI calls
charges is the **charm system**: `+0x14` quality, `+0x54` slot count
(≤5), `+0xa0` attached-charm list (appended by `CItem::AddCharm`
`0x4b7ae0` from the EffectBoost apply `0x579220` and the save-rebuild
re-attach `0x57a8a5`) — a list-length-vs-capacity budget, not a
decrementing counter.

The classes `CItem` / `CItemCharm` / `CItemGeneration` /
`CItemGenerationDescription` (`itemgenerate.cpp` / `itemgeneration.cpp`
/ `ItemLink.cpp`) implement generation, charm behaviour, and the
static↔instance link. The static item **definition** schema
(`CItemStatistic`, below + [`formats/itemlink.md`](formats/itemlink.md))
was already fully recovered.

## On-disk serialization — `items.000` ✅ (byte-exact)

The global item-instance table (`.\WORLD\ItemLink.cpp`, manager
`this=0x750cfc`, registry slot `reg+0x0c` = `[0x750d44]`; Load
`fcn.00579ba0`, Save `fcn.00579a70`). The shipped
`main\startup\items.000` (213,425 B) parses byte-exact:

```text
i32  magic          -100 (old) / -101 (new); shipped -100; Save writes -101
u32  nextGenId      → [0x6ddd74]   (521 — matches max "__G520")
u32  handleCount    → mgr+0x28     (803)
u32  handle[handleCount]           handle → item id; 0 = empty (598 nonzero)
u32  nextItemId     → container+0x30 (1324 = max entry id)
u32  entryCount     (598)
entry × entryCount:
  u8  marker        ≠0 ⇒ the entry is this single byte (none shipped)
  u32 id            unique, 2..1324
  u8  hasGenStat    → if set: generated CItemStatistic blob (fcn.004bb170)
  CItem blob        (fcn.004b8990)
```

**CItem blob** (`fcn.004b8990` + body `fcn.004b7450`; strings are
`u32 len + bytes`, no NUL): statName string (`__G%d` for generated),
21×u32 → `CItem+0x04..+0x54`, u8 identified → `+0x58`, 10×u32 →
`+0x5c..+0x80`, statName again (→ `+0x84` CItemStatistic* lookup;
failure tolerated only for `__G*`), then two lists
`i32 n; n×{string charmName, u32 value, u8 flag}` → `+0x88` and the
`+0xa0` attached-charm list. All counts are signed; `≤0` = empty.
The **generated-CItemStatistic blob** (`fcn.004bb170`, 0x16c object —
confirming the prototype-CItem-at-`+0x94` model) and its `Item94`
payload reader (`fcn.004b7060`, 30×u32 + charm list + u32) are
enumerated in the loader chain; id→CItem* `fcn.004b8360`, reverse
`fcn.004b81b0`, string reader `fcn.004b5910`.

**The `inv.i*` "heap-dump tree" claim is retracted**: the starting
inventories are plain **TRamFile index/bulk pairs** — the "baked
pointers" were uninitialized bytes in the fixed 16-byte name buffer.
Full record layouts (28-byte index / 40-byte slot) in
[inventory.md](inventory.md).

## Trade / shop pricing (`.\AGENTS\agenttrade.cpp`) ✅ (closed form)

The buy/sell math is fully reduced (identical factor code in
`fcn.004368c0` @`0x436a0c`, `fcn.00435f80` @`0x4360b4`, `fcn.004365d0`
@`0x4366ae`):

```text
tier       = clamp(TradersTongue.charge, 0, 99) / 20      ; skill id 0x32
f          = MPD[tier] × 0.01                             ; MPD = props
                                                          ;   MerchantPriceDifference
                                                          ;   = {5,10,15,20,25}
sellFactor = max(RelativeSellPrice − f, 1.0)              ; floor at 1.0
buyFactor  = min(RelativeBuyPrice  + f, 1.0)              ; cap  at 1.0
```

Per trade-offer entry: `count = fcn.00586430(name, kind)`; if
`count > 1` the contribution is `unitValue(fcn.004af960) ×
trunc(count × factor)` — **the factor applies only to stacks ≥ 2**
(the gold/stackable side); single items contribute `unitValue × count`
unscaled. All rounding is `fcn.005e5d40` = `_ftol2` — **truncation
toward zero**. Settlement (`0x436c0a`):
`this.gold(+0x290) += totalOther(buyFactor) − totalThis(sellFactor)`,
the other agent gets the negation. `fcn.00435f80`'s extra flag selects
*whose* factor pair is used (perspective); `fcn.004365d0` is the
evaluation-only variant.

**Field correction:** `RelativeSellPrice` / `RelativeBuyPrice` are
**floats at `agent+0x208` / `agent+0x20c`** — not ints at
`+0x23c`/`+0x240`. The agent dumper (`fcn.0042d230` @`0x42d70a`)
pushes both as doubles against `%d` specifiers — a source-level printf
bug, which is why dumped values looked like garbage and the offsets
were misread. `agent+0x23c` in agenttrade is a lazily-allocated trade
sub-object (alloc @`0x436915`), `+0x240/+0x244` the trade-offer list
refs, `+0x290` gold. This complements the **treasure-type**
identify/heal/repair costs
([`formats/treasure.md`](formats/treasure.md)) — together the merchant
economy.

## Status

- Trade/shop pricing ✅ **RESOLVED** — closed-form factors (Trader's
  Tongue tier × MerchantPriceDifference {5..25}%, sell floored / buy
  capped at 1.0, factor applied to stacks ≥ 2 with trunc-toward-zero),
  fields corrected to floats `agent+0x208/+0x20c` (the dumper's %d was
  a printf bug).
- Static stat schema ✅ — full `CItemStatistic` property vocabulary
  recovered (identity/slot/weapon-kind/damage/defense/bonus/magic/
  requirement/rarity/on-hit), with the combat & magic bindings mapped.
  - **Parser chain located** (where the in-memory `CItemStatistic` offsets
    are assigned): `itemstat\itemstat.txt` → registry loader **`fcn.0057b530`**
    (`.\WORLD\ItemLink.cpp`, *"Unknown error parsing itemstat.txt"*) →
    per-line parser **`fcn.0057b360`**, which tokenises (`fcn.004fdc40`) and
    **template-matches each line via `fcn.004fdf40`** — the *same* `#`/`$`
    slot matcher the monologue parser uses ([monologues.md](monologues.md)) —
    then dispatches the matched field through **`fcn.0057b2e0`** to its
    `CItemStatistic` offset. So the field-name → in-memory-offset mapping
    (incl. `numdice`/`dicetype`/`diceadd`) lives in the `fcn.0057b2e0`
    field dispatch; the offsets are template-parser-assigned, not a fixed
    on-disk record (itemstat.txt is keyword text, not packed).
- Combat dice tie-in ✅ — `numdice`/`dicetype`/`diceadd` are the dice the
  damage calc rolls; defense fields feed mitigation.
  - *Identify-flag route — resolved (superseding the earlier dead-end):*
    the `CSurvivorLoreSkill_Identify` vtable is indeed all shared
    `CSkill` base methods, but the actual item mutation is the static
    **action body `fcn.004d0ff0`** reached from the utility-skill
    dispatcher — see the Runtime instance section for the flags bit 21
    + `CItem+0x58` details.
- Item classes ✅ — `CItem` / `CItemCharm` / `CItemGeneration` /
  `CItemGenerationDescription` enumerated (generation + charm + link).
- Runtime instance fields ✅ **RESOLVED** — the `CItem` layout is
  pinned (section above): durability `+0x18`/`+0x1c`, identified =
  record-flags bit 21 + cached `CItem+0x58`, charm quality `+0x14` /
  slots `+0x54` / list `+0xa0` (charges as a counter **don't exist**),
  stat-boost targets per the `0x577680` table. The slot-record facts
  below remain valid: the **stack
  count is the `u16` at instance `+0x0e`**, recovered from
  **`TInventory::SplitItem`** (`fcn.004b14b0`): the splitter reads the
  source count via `movsx …, word [esi+0x0e]` and the
  `"Left and right don't match up in TInventory::SplitItem() - Count case"`
  guard (`0x613120`) is the over/under-split check on it. When it carves
  off the new stack it copies the core item record into the new instance —
  `word [esi+0x0c]`, `word [esi+0x0e]` (count), `word [esi+0x10]`, and
  `dword [esi+0x14]` — the 40-byte inventory **slot record**, now fully
  decoded byte-exact ([inventory.md](inventory.md)): `+0x0c` = the i16
  slotTag, `+0x0e/+0x10` the u16 pair around the stack count, `+0x14` =
  the instance-flags dword. The CItem state fields (identified /
  durability / boosts) are offset-pinned in the CItem section above and
  serialize in `items.000` (below).
  - **Instance flags word ✅ (semantics) / 🟡 (offset)** — item instances
    carry a **32-bit flags word**: the shipped "Item instance bug" fixer
    (`fcn.005990c0`, logs *"Item instance flags have been cleared - Item
    instance bug fixed…"*) clears the buggy flags with **`AND 0xff0003ff`**,
    i.e. it zeroes **bits 10–23** while preserving bits 0–9 and 24–31. So
    bits 10–23 of that word are the per-instance flags (the corrupted set);
    the exact struct offset is left 🟡 because the fixer addresses it via an
    array form (`[esi + i*8 + 0x10]`) that doesn't map cleanly onto the
    `SplitItem` instance base above.
  - **Durability — resolved (superseding two recorded dead-ends).** The
    earlier routes (the `CMagicRepair` class's own overrides; the
    Identify skill vtable) were genuinely empty — the durability
    arithmetic lives in the static Repair **action body**
    `fcn.004d1620` and the itemgen **`CItemGeneration_Durability::Apply`**
    `0x577a60`, both with literal `[item+0x18]`/`[item+0x1c]` stores
    (Runtime instance section above). `RepairQuality` is the props
    *quality* curve the repair uses, not a max property.
  - **Item field decoder is shared with treasure ✅** — the item
    comparator **`fcn.004ae900`** (called from the `0x4b…` item cluster
    incl. `SplitItem`) passes each item's leading `{[+0] mask, [+4] data}`
    pair to **`fcn.005918b0`** — the *same* **variable-width bitmask field
    decoder** (table `0x655a98` = 32 per-bit byte-widths `∈ {0,1,2}`, big
    -endian) the treasure drop-roll uses
    ([`formats/treasure.md`](formats/treasure.md)). *(Refinement: it is a
    field **decoder**, not a category enum — it locates and reads the
    bit's variable-width value out of the packed `+4` blob under the `+0`
    presence mask.)* So the comparator reads the item's kind field via the
    same routine; the `{mask, data}` pair leads the inventory item/slot
    record (the value-pool scheme, same as the objects.x records).
  - **The "resists offset-pinning" verdict is retracted — resolved.**
    The earlier structural argument ("the writes hide in the shared
    data-parameterised `CMagic*` vtable, dynamic-only") was wrong on
    both counts: **Identify and Repair are plain static action bodies
    with literal-offset stores** — `fcn.004d0ff0` (Identify: skill
    object id `0x27` via `fcn.005438c0`, level gate `fcn.00544ff0`,
    failure float-texts 98/99/101, flags-bit toggle via
    `fcn.00591940(…, 0x15)`) and `fcn.004d1620` (Repair: props
    `RepairQuality`, `mov [item+0x1c], eax`) — dispatched from the
    utility-skill handler sites (`0x4a604c..0x4a818a`), not through the
    effect table. The missing structural insight was the **`CItem`
    prototype embedding** (`CItemStatistic+0x94`, copy-ctor
    `fcn.004b67b0`) and the **resolver `fcn.0057a970`** (record → CItem
    via the bit-19 "is an item" gate) — once found, every field has a
    recognisable call site (see the Runtime instance section). Also
    overturned: `CItemGeneration_*` vtable **slot 0 is not a dtor — it
    is `Apply(CItem*, CItemStatistic*, int)`** (`ret 0xc`), which is
    where the durability/statsboost/speed/charm-quality offsets came
    from ([`formats/itemgen.md`](formats/itemgen.md)).
