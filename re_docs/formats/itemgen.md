# `dat\<lang>\itemgen.cmp` — item-generation affixes

The random-magic-item table: the **prefixes and suffixes** the engine
attaches to generated items, and the stat/skill bonuses each grants —
so a rolled item becomes e.g. *"Deadly Sword of lightning sparks"*. It
is per-language (`dat\English\itemgen.cmp`) because the affix names are
localized. (`cmp.md` flags itemgen.cmp as a magic-sentinel,
type-polymorphic file and points here.)

All integers little-endian, signed where noted.

## Header & records

```text
i32   sentinel = 0xfffffffe (-2)     magic
i32   count    = 306                  affix records
…     Record[count]                   type-polymorphic, variable length
```

Each record is variable-length: a few leading int fields (type / slot
mask / rarity), a **length-prefixed affix name**, then signed
stat-modifier fields. Negative values are penalties — e.g. the
`"Cursed"` prefix carries values like `-100` / `-10`; positive values
on better affixes are the bonuses.

The exact per-type field layout is polymorphic (it varies by affix
kind) and needs the loader to pin; the **affix vocabulary**, however,
decodes cleanly.

## Affix vocabulary (306 records; 351 strings)

Scanning the strings yields **351** name strings, but the header
`count` is **306 records** — so the two differ, and the gap is real:
**there are 306 affix records**, and ~45 of the 351 strings are
**embedded effect-grant names** an affix carries (its on-hit/cast effect),
not separate affixes — e.g. `Deathblow`, `Frost`, `of ice` (the `CMagic*`
on-hit procs / spell grants, cf. [`../items.md`](../items.md)) appear as
in-record sub-strings. So the **affix count is 306** (the header), of which
the string scan below lists **153 prefix-side + 198 suffix-side** *strings*
(affix names + their embedded effect names):

```text
prefixes (stat modifiers):
  Cursed, Wrecked, Destroyed, Broken, Cracked, Worn   ← penalties / damaged
  Monk's, Bull's, Owl's, Spider's, Mole's, Guard's    ← attribute bonuses
  Fine, Sharp, Deadly, Bright, Sturdy, Iron           ← quality / damage

suffixes ("of …", mostly skill/spell grants):
  of swiftness, of poison, of fire sparks, of lightning sparks,
  of lockpicking, of blessing, of pickpocketing, of backstabbing,
  of trap detection, of monster identification, of repairing,
  of meteor strikes, of spikes, of skeletons, of stunning, …
```

The **suffixes map directly onto the skill/magic taxonomy**
([`../skills-magic.md`](../skills-magic.md)): *of lockpicking* → the
LockPick effect, *of blessing* → Bless, *of meteor strikes* →
MeteorStrike, *of skeletons* → SummonSkeleton, *of pickpocketing* →
PickPocket, and so on. So a suffix grants its item the corresponding
`CMagic*` effect; prefixes grant attribute/quality modifiers.

## Status

- Header ✅ — `0xfffffffe` sentinel + `i32 count(306)`.
- Affix vocabulary ✅ — 351 names extracted (153 prefixes + 198
  suffixes); suffixes tie to the skill/magic effect taxonomy.
- Record framing ✅ (shape) — variable-length, type-polymorphic:
  leading ints + length-prefixed name + signed stat-modifier fields
  (negatives = penalties, e.g. `Cursed`).
- Record fields ✅ (decoded from data) — each affix carries a
  **`rarity_weight`** (≈50/100/150) and, for prefixes, a triple read as
  `{group/stat selector, magnitude}` before the length-prefixed name.
  Two clean structures emerge:
  - **Condition prefixes** use the special selector **`-1`** with the
    condition value as the magnitude — an ascending ladder
    `Cursed(-100) · Wrecked · Destroyed · Broken · Cracked · Worn ·
    New(120)` (item wear from broken to pristine).
  - **Attribute/element prefixes** come in **5-tier ladders**, magnitude
    **`100 / 500 / 1000 / 2000 / 5000`** per family, e.g.
    Strength `Bull's→Lion's→Elephant's→Whale's→Dragon's`; thievery
    `Rogue's→…→Master assassin's`; intelligence
    `Apprentice's→…→Sorcerer lord's`; poison `Spider's→…→King cobra's`;
    fire `Fire worm's→…→Fire dragon's`; plus the element-damage prefixes
    (`Iron/Silver/Golden`, Fire/Lightning/Poison/Spiritual damage, Mana/
    Life drain, Stun) on the damage channels.
  - **Suffixes** (`of swiftness`, `of lightning sparks`, …) use a
    *different* record shape — they bind a `CMagic*` effect (the
    suffix→effect taxonomy below) rather than a flat stat delta.
- Type-tag dispatch ✅ — the per-record type tag `0..7` resolves to a
  category via the `0x578ea4` jump table (table below: `0/1`→SkillBoost,
  `2`→PathBoost, `3`→Statsboost, `4`→EffectBoost, `5`→AddCharmQuality,
  `6`→Durability, `7`→Speed), each ctor pinned by its RTTI vtable.
- Per-category field offsets ✅ — each category's read method (`virtual_20`,
  vtable `+0x14`) calls the base reader `fcn.005773c0` then reads its target
  `i32`(s): `+0x40` (SkillBoost/PathBoost/Durability/Speed, shared
  `fcn.00577700`), `+0x40/+0x44/+0x48` (Statsboost `fcn.00577820`), through
  `+0x48` (EffectBoost `fcn.00577900`), `+0x3c` (AddCharmQuality
  `fcn.005779c0`). The magnitude is the `CBE*` expression; only each id's
  *enumeration meaning* stays in the shared `CItemStatistic` id space.
- Generation pipeline ✅ (located) — the affix/generation **descriptions**
  load via `fcn.00578c90` (`.\WORLD\itemgeneration.cpp`, the `0xfffffffe`
  sentinel above); the item **generator** is `fcn.004b2360`
  (`.\itemstat\itemgenerate.cpp`): it builds a generated item, names it
  **`__G%d`**, sets a **"generated" flag at instance `+0x168`**, and
  inits the instance via `fcn.004b6910` (fields at `+0x18`/`+0x30`/
  `+0x4c`/`+0x5c`/`+0x60`/`+0x78`/`+0x7c`/`+0x94`). A rolled item is thus
  a `CItemStatistic` ([`../items.md`](../items.md)) plus generated affix
  state.
- Affix representation ✅ — an affix's modifier magnitude is a
  **boost-expression tree**, not a fixed table value. **Node semantics
  now fully closed** (RTTI→vtable walk; vtables `CBEPrefix 0x61a4bc`,
  `CBEDelayFunction 0x61a4cc`, `CBERandom 0x61a4dc`,
  `CBEBasicUnary 0x61a5e4`, `CBEBasicArithmetic 0x61a5f8`; **slot 1 =
  `Evaluate(ctx) → float`**, recursing into children via their
  `vtbl+4`; slot 2 = the *description printer* that builds the affix
  display text — the earlier reading of `fcn.005375e0`/`fcn.00537410`
  as value methods was off by a slot: those are printers):
  - **`CBEPrefix`** (`0x535a00`) is a **min/max combiner** (not a
    wrapper): children at `+4`/`+8`, op `+0xc`: 0 = `min(A,B)`,
    1 = `max(A,B)`, other → 0.0 (printer emits `"min ("`/`"max ("`).
  - **`CBEBasicArithmetic`** (`0x535920`): op `+0xc`: 0 = `A+B`,
    1 = `A−B`, 2 = `A×B`, 3 = `A/B` with a `|B| < 1e-4` guard.
  - **`CBEBasicUnary`** (`0x5357a0`): op `+0x8`: 0 = `−x`, 1 = `|x|`,
    2 = logical-not (`|x| < 1e-4 → 1 else 0`), 3 = `floor(x)`,
    4 = `ceil(x)`, 5 = bool-vs-0 compare; >5 → 0.0.
  - **`CBERandom`** (`0x535c40`): `r = rand()/32767.0`; if
    `[this+4] ≠ 0` → `2r − 1` (signed −1..1), else 0..1 — the
    `rand()`/`signrand()` split, now exact.
  - **`CBEDelayFunction`** (`0x535aa0`): a deferred named-value node
    (0x400 name buffer, resolves its operand at evaluate time).
  So a magic bonus like "+2–5 fire" is arithmetic over a `CBERandom`
  inside the affix's expression, evaluated at generation time. (This
  is what the docs previously called the "genetic-algorithm" encoding;
  the boot-time "genetic algorithms" banner is a no-op — the real
  mechanism is this boost-expression tree.)
- Affix → **target category ✅** (the `CItemGeneration_*` subclasses) —
  the affix's *target* is encoded in its description **subclass**, of which
  there are exactly **seven** (RTTI), each binding the rolled BE value to a
  category:

  | `CItemGeneration_*` subclass | What the affix modifies |
  |---|---|
  | `Statsboost` | a [`CStats`](../stats.md) attribute (Strength/Agility/…) |
  | `SkillBoost` | a [skill](../skill-tree.md) rank |
  | `PathBoost` | a skill **path/discipline** (a whole tree) |
  | `EffectBoost` | grants a [`CMagic*`](../skills-magic.md) effect (the on-hit / cast specials) |
  | `AddCharmQuality` | charm quality (charm-item potency) |
  | `Durability` | item durability |
  | `Speed` | attack / use speed |

  So a generated affix = a `CItemGeneration_<category>` description (the
  **target**) + a [`CBE*` boost-expression](#) (the **rolled value**). The
  loader `fcn.00578c90` reads a per-record **type tag** (`0..7`) and
  dispatches through an **8-case jump table at `0x578ea4`** to the matching
  category constructor. The complete on-disk **type-tag → category** map
  (verified from the jump table + each ctor's RTTI vtable):

  | type tag | factory | category class |
  |---:|---|---|
  | 0, 1 | `fcn.00577480` | `CItemGeneration_SkillBoost` |
  | 2 | `fcn.00577730` | `CItemGeneration_PathBoost` |
  | 3 | `fcn.00577780` | `CItemGeneration_Statsboost` |
  | 4 | `fcn.00577860` | `CItemGeneration_EffectBoost` |
  | 5 | `fcn.00577950` | `CItemGeneration_AddCharmQuality` |
  | 6 | `fcn.00577a00` | `CItemGeneration_Durability` |
  | 7 | `fcn.00577af0` | `CItemGeneration_Speed` |

  Each ctor zero-inits the shared base struct and sets the defaults
  **`rarity_weight` (`+0x1c`) = 100** (all categories) and, for
  `AddCharmQuality`/`Durability`/`Speed`, a `+0x14 = -1` sentinel
  (`Statsboost` instead sets `+0x18 = 3`). So the on-disk record's leading
  type tag now resolves directly to its category and struct shape;
  `CItemGenerationDescription` is the base record.

  **Per-category read method (`virtual_20`, vtable `+0x14`) and target
  fields** (each calls the shared base reader `fcn.005773c0` first, then
  reads its own `i32` target field(s) via the serialize read-context
  `[0x6e0124]`, 4 bytes each):

  | category | read fn | target field(s) |
  |---|---|---|
  | SkillBoost / PathBoost / Durability / Speed | `fcn.00577700` (shared) | one `i32` at **`+0x40`** (the target id: skill / path / durability / speed param) |
  | Statsboost | `fcn.00577820` | three `i32` at **`+0x40`/`+0x44`/`+0x48`** (attribute id + two params) |
  | EffectBoost | `fcn.00577900` | reads through **`+0x48`** (the `CMagic*` effect id + params) |
  | AddCharmQuality | `fcn.005779c0` | field at **`+0x3c`** |

  So the per-record **target id is the `i32` at `+0x40`** for the four
  simple categories, with Statsboost/EffectBoost reading a small fixed run
  of extra `i32`s and AddCharmQuality using `+0x3c`. The affix **magnitude**
  is the separate `CBE*` boost-expression. `CItemGenerationDescription` is
  the base record (its common fields are read by `fcn.005773c0`).
- Affix → exact field-id ✅ *(recovered this pass)* — category from the type
  tag, target field offsets per category as tabled above (`+0x40` for the
  simple kinds), rarity weight = the ctor default `+0x1c` = 100.
- **Apply side ✅ (RESOLVED — and a vtable correction).**
  `CItemGeneration_*` vtable **slot 0 is not the destructor** — it is
  **`Apply(CItem* item, CItemStatistic* genStat, int flag)`** (`ret
  0xc`), with literal-offset stores onto the new `CItem` instance
  ([items.md](../items.md) for the CItem layout). Called from the
  generator driver `fcn.00579260` (invoked by `fcn.004b2360`) via
  `call [vtbl+0]` with arg1 = the fresh `CItem` (`fcn.004b9020` →
  `fcn.004b8ac0`). Per-category bodies: **Durability `0x577a60`**
  (`add [item+0x18], min+rand%(range+1)`; current = `1+rand%max`,
  clamped), **Speed `0x577b50`** (`add [item+0x5c]`), **Statsboost
  `0x577510`** (16-case jump table `0x577680` mapping stat-id →
  `CItem` offset: 0→`+0x74`, 1→`+0x78`, 2→`+0x80`, 3→`+0x7c`,
  4→`+0x2c`, 5→`+0x30`, 6→`+0x34`, 7→`+0x38`, 8→`+0x3c`, 9→`+0x44`,
  10→`+0x4c`, 11→`+0x50`, 12→`+0x48`, 13→`+0x10`, 14→`+0xc`,
  15→`+0x40` — this *is* the target-id enumeration the previous bullet
  left open), **AddCharmQuality `0x5779e0`**
  (`mov [item+0x14], [this+0x3c]`), **SkillBoost `0x579920`** (appends
  `{skill-id, level}` to the generated `CItemStatistic+0x18` container
  — targets the statistic, not the CItem), **EffectBoost `0x579220`**
  (→ `CItem::AddCharm` `0x4b7ae0`). The display-keyword table is at
  `0x655138` (printer `0x577d03`). Note the `+0x168` "generated" flag
  is on the 0x208-byte `CItemStatistic`, not the instance.
