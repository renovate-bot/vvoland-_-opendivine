# Equipment visual composition (clothing codes)

How worn equipment changes a character's on-screen sprite. When an
agent equips or removes a visible item, the engine rebuilds a composite
sprite by stacking per-bodypart layers; *which* layers it stacks is
driven by a 4-character **clothing code** carried by each item's
catalogue entry. This is the "Equipment influence" path on the STATUS
roadmap.

The layer-*name* builder is already ported in
`internal/game/character/character_compose.go` (the engine's
`FUN_00439b70`). This doc covers the parts around it that were not yet
reversed: where the per-item letters actually come from
(`FUN_0043a5b0`), what the clothing code *is* (grounded in the shipped
`objects.000`), and the compose driver that turns it into pixels
(`FUN_0043afc0`, `.\AGENTS\clothingvisualizer.cpp`).

## The clothing code (`objects.000` `+0x3c`)

Each catalogue entry's `ClothingCode` field (`+0x3c`, 16-byte NUL-padded
string; see [`formats/objects.md`](formats/objects.md)) holds the
visual code. In the shipped catalogue, of 7208 entries:

- **4856** are the 4-char sentinel `"none"` — not visually worn.
- **198** are real wearables (`code != "none"`).
- the rest are empty.

The code is positional. Decoded from the 198 wearables:

| Pos | Meaning | Observed values |
|---|---|---|
| 0 | item **family / slot class** | `1`–`5` = armour tiers (shields, mail, leggings, helmets); `I K M O S W Y w` = weapon families |
| 1 | **hold / wear type** | `H` = held weapon, `J` = arrow, `v` = scroll; a letter for armour |
| 2 | **variant class** | `A`–`O` — the letter the layer composer keys on (`HelmetCls`/`WeaponCls` in the Go port) |
| 3 | numeric **sub-variant** | a digit on 4-char armour codes |

Weapon-family first letters (pos 0), from real items:

```text
I = spear/pike/lance     (IHA)     S = staff/polearm/scythe (SHK/SHI/SHJ)
K = dagger/dirk/sabre    (KHB/KHC) W = bow                  (WHN)
M = mace/maul/axe/scepter(MHM/MHF) Y = crossbow             (YHO)
O = two-handed sword/hammer(OHE/OHL) w = spellscroll        (wvn)
```

> Correction: `character_compose.go`'s `CharacterEquipment` doc comment
> says each letter is "the third character of the equipped item's
> catalogue **name**". It is not — it is a character of the item's
> **`ClothingCode`** (`+0x3c`), a purpose-built positional code, not the
> human-readable Name (`+0x20`). The composer logic is unaffected; only
> the source field is mislabelled.

## Gathering the worn items (`FUN_0043a5b0`)

`FUN_0043a5b0` assembles the per-slot letters the composer consumes:

1. Read the agent's inventory: `inv = [agent+0x23c]`, then the item
   array base/cursor at `[inv+0x240]` / `[inv+0x244]`.
2. Fetch the items in the **visible equipment slots** — inventory slot
   indices `{0, 2, 3, 4, 7}` (queried via `FUN_00422260`). **Slot→part
   map resolved (correcting the earlier "slot 7 = weapon"):**
   **0 = helmet, 2 = torso armour, 3 = weapon, 4 = shield,
   7 = leggings.** Evidence: tag 3's `ClothingCode[0]` feeds the
   two-handed switch (`'N'..'Y'` @`0x43a9ce`) and the attack-anim
   letter; tag 4 is special-cased through `fcn.0042b2c0`, which hides
   the item when the equipped weapon's `CItemStatistic+0x88`
   (the `twohanded` keyword) is set — a two-handed weapon hides the
   *shield*; and the layer inventories in the shipped `.key` files
   match (A = legs, B = torso, C = weapon classes A–O, D = helmet,
   E = the 5 shield tiers). Buffer bindings (both `FUN_0043a5b0` and
   the composer `FUN_00439b70`): tag0→`local_c`, tag2→`local_14`,
   tag3→`local_2c`, tag4→`local_24`, tag7→`local_1c`.
3. For each equipped item, index its catalogue record
   (`record = [catalogueBase ([this+0x1c])] + kind*0x94`) and copy the
   `ClothingCode` bytes from `record+0x3c` into the letter struct
   (`lea ecx,[base+rec+0x3c]; mov dl,[ecx]; …`).

The result is the `(Helmet, HelmetSub, HelmetCls, Torso, Legs, Face,
Weapon, WeaponSub, WeaponCls, WeaponHand)` tuple that
`composeLayerNames` already models.

## Compose driver (`FUN_0043afc0`)

`FUN_0043afc0` (`.\AGENTS\clothingvisualizer.cpp`) is the recompose
entry, run when equipment changes:

```text
FUN_0043a5b0   gather worn-item ClothingCode letters from slots {0,2,3,4,7}
FUN_00439b70   build the up-to-5 .key layer-group names (already ported)
operator new   allocate the composite
FUN_0050ac30 / FUN_004e84e0   resolve + blit each named layer into the composite
```

The composite is built from per-bodypart **components**; the editor
dump labels them `Component %d: dc=%d ac=%d` and
`Component %d: %dhx%dv q=%d` (`%dhx%dv` = horizontal×vertical cell
span). The component record layout is not yet pinned.

## Citations

```text
div.exe:0x0043afc0   FUN_0043afc0   CClothingVisualizer recompose driver (gather→name→blit).
                                    Source: ".\AGENTS\clothingvisualizer.cpp".
div.exe:0x0043a5b0   FUN_0043a5b0   gather worn items' ClothingCode letters (slots {0,2,3,4,7}).
div.exe:0x00439b70   FUN_00439b70   layer-name composer (ported as composeLayerNames).
div.exe:0x00439ae0   FUN_00439ae0   per-slot clothing-code extractor helper.
div.exe:0x00422260   FUN_00422260   inventory slot accessor (get item in equipment slot N).
div.exe:0x0043b5e0   FUN_0043b5e0   second clothingvisualizer entry (recompose variant).
```

## Status

- Clothing code format ✅ — positional 4-char code at `objects.000+0x3c`,
  decoded and grounded in the 198 shipped wearables.
- Equipment-letter source ✅ — `ClothingCode` (`+0x3c`), not Name;
  `character_compose.go` comment is mislabelled (correctable).
- Visible equipment slots ✅ — inventory indices `{0,2,3,4,7}`; map
  resolved: 0 helmet / 2 torso / 3 weapon / 4 shield / 7 leggings
  (supersedes "slot 7 = weapon"); tags 5/6 = the two interchangeable
  ring slots (fallback swap @`0x508e3c`). NOTE the **`CItemStatistic
  +0x84` slot enum is a different id space** (helmet 0, necklace 1,
  weapon 2, armor 3, leg 4, ring 5, boots 6, shield 7, belt 8,
  gloves 9 — parser jump table @`0x4ba51c`); +0x90 = rarity 0-3. Old
  bullet: weapon
  (7) separated.
- Layer-name composition ✅ — already ported (`FUN_00439b70`).
- Component record layout 🟡 — labels (`dc`/`ac`, `h×v`, `q`) known; the
  struct fields and the blit (`FUN_0050ac30`/`FUN_004e84e0`) are not yet
  decoded.
- Slot-index → bodypart mapping 🟡 (bounded; 3 routes checked) — slot set
  `{0,2,3,4,7}` confirmed; slot 0 = helmet and slot 7 = weapon proven (see
  [`inventory.md`](inventory.md)); **2/3/4 → torso/legs/face order is not
  cleanly recoverable statically**, confirmed across three angles: (1) no
  bodypart-name string table sits near the composer (the nearby `helmet`/
  `boots` strings are perception senses, not a clothing table); (2) the
  composer `FUN_00439b70` iterates the slots in order `0,2,3,4,7` but the
  order doesn't label the bodyparts; (3) it extracts each slot's clothing
  char via `FUN_00439ae0` and composes layers, but the slot→layer/bodypart
  assignment carries no labels and the code-position writes
  (`mov byte [esi+ecx], dl`) are generic. So the 2/3/4 order stays inferred
  (torso/legs/face) — a presentation-layer detail with no string/label
  anchor in the shipped binary.
