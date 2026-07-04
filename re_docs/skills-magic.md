# Skills & magic

The skill/spell system. Divine Divinity's abilities are a C++ class
hierarchy — every skill and spell effect is its own polymorphic class,
named in RTTI as `C<Class><Discipline>Skill_<Name>` (skills) or
`CMagic<Effect>` (the shared effect primitives). This doc catalogues
the taxonomy and how the pieces fit; per-effect mechanics are reversed
separately.

## Skill tree

Three player classes, each with four discipline trees. (Counts from the
shipped RTTI type names; the engine groups them as below.)

### Survivor

| Discipline | Skills |
|---|---|
| Divine | DivineDeath, DivineEye, DragonBreath, Ghost, LightOfHeaven, MassDomination, TeleportJump, Timestop |
| Lore | Alchemy, Blind, Curse, Identify, MonsterIdentification, NecroShift, PoisonBody, TrueSight |
| Talent | Antimagic, Charm, Leadership, Merchant, Rangersight, Shapeshift, Survival, Wisdom |
| Thief | BackStabbing, BoobyTrap, DetectTraps, LockPick, PickPocket, PoisonWeapon, Running, Shadows |

### Wizard

| Discipline | Skills |
|---|---|
| BodySpirit | Bless, BodyShield, Fear, Healing, Invisibility, MagicShield, Strength, WizardEye |
| Elemental | ElementalStrike, Freeze, Lightning, MeteorShower, MeteorStrike, PoisonCloud, Sparks, WallOfSmoke |
| Matter | Banish, Cage, MagicAttractor, MagicWall, Matter, Polymorph, Resurrect, Spikes, SummonedShields, Telekinesis, Unlock |
| Summon | DemonicAide, Life, Mimic, Rats, Skeleton |

Wizard skills are named `CMagicWizard<Discipline>Skill_<Name>` — i.e.
they *are* magic effects, deriving from the same base as the
primitives below.

### Warrior

| Discipline | Skills |
|---|---|
| God | Berserk, FeignDeath, Sanctuary, SpiritualDamage, WarriorPrayer, WarriorSidekick, WarriorTactics, WeaponCharming |
| Knowledge | FireDamage, IncreasedDamage, IncreasedDefense, LightningDamage, PoisonDamage, Repair, Rush, Stun |
| Ranger | Accuracy, BlockArrows, ExplosiveArrows, FrostArrows, IncreasedRange, PiercingArrows, PoisonedArrows, SplittingArrows |
| Specialist | Axe, Bow, Crossbow, Hammer, Mace, Shield, Spear, Sword |

The Ranger arrow skills (Explosive/Frost/Piercing/Poisoned/Splitting)
modify the projectile system; the Specialist skills are weapon-family
proficiencies (matching the `ClothingCode` weapon families in
[`clothing.md`](clothing.md): bow/crossbow/axe/hammer/mace/spear/sword).

### Class gating & the per-class special move

The three trees are **gated by the player class** — the class index at
**`CStats+0xa0`** (`0` = Warrior, `1` = Wizard, `2` = Survivor, named in
[`stats.md`](stats.md) from the stat-derivation coefficients): a character
learns only the four disciplines of its own class. On top of the trees,
each class has **one signature `CSpecialMove`** (base `CSpecialMove`
vtable `0x60a4??`; the three concrete classes are
**`CSpecialMove_Warrior`** `vtable 0x609380`, **`CSpecialMove_Wizard`**
`vtable 0x60936c`, **`CSpecialMove_Survivor`** `vtable 0x609394`, in
`.\AGENTS\special.cpp`). Their vtables share slots 1–3 but each overrides
**slot 0** with the class-specific move (`Warrior 0x442370` / `Wizard
0x4426a0` / `Survivor 0x442270`) — the class "special" the HUD exposes
separately from the skill bar. So the full per-class kit is **4 discipline
trees (8 skills each) + 1 signature special move**, all keyed off the
`CStats+0xa0` class.

## Skill activation (the `CSkill` runtime)

The `C<Class><Discipline>Skill_<Name>` classes above derive from a base
**`CSkill`** and are the *player-facing* learned abilities — distinct from
the `CMagic*` / `CInterpreterSpell` **effect** objects that carry the
behaviour (e.g. the thief skill `CSurvivorThiefSkill_LockPick` vtable
`0x61c0b4` is a different class from the effect `CMagicLockPick`). A skill
is the slot you activate; it then drives its bound effect.

Activation goes through one **shared dispatcher `fcn.00541840`**. Each
skill's activate virtual is a thin wrapper: an optional precondition
check, then `push <targetMode>; call fcn.00541840` — passing a per-skill
**targeting-mode constant**. Two confirmed:

```text
CSurvivorThiefSkill_LockPick : check fcn.00545f60 → fcn.00541840(0x28)
CMagicWizardMatterSkill_Unlock:                     fcn.00541840(0x40)
```

`fcn.00541840(mode)` is the runtime activation core (full body read):

1. **Resolve the casting player** — `[0x658c04]+0x4b4` → a handle →
   `CAgentManager [0x658d50]` (`[[mgr+0xc]+id*4]`).
2. **Set the skill's bit** in the player's **`SpellKnowledge[3]` and
   `SpellLearned[3]`** bitmasks (named from the agent dumper,
   [`agent.md`](agent.md)). The `mode` constant is a *bit index* `0..0x5f`,
   split across the triple by range — `mode<0x20` → `agent+0x58`,
   `0x20..0x3f` → `agent+0x5c`, `≥0x40` → `agent+0x60`
   (`|= 1<<(mode mod 0x20)`) — and the same bit is set in the parallel
   `SpellLearned` triple at `agent+0x64/+0x68/+0x6c`. So the `mode` constant
   is the skill's **global spell/skill bit** (LockPick `0x28`, Unlock
   `0x40`), and `fcn.00541840` marks it both *known* and *learned*. This
   is really the **learn/grant** path — it records the ability and places
   it on the hotbar, rather than firing a one-shot effect.
3. **Record the active skill** — when the player has a skill component
   (`agent+0x25c`): `skillcomp+8 = mode`, `agent+0x360 = skill+0x2c`,
   `agent+0x35c = -1`.
4. **Bind it into the skill hotbar** — `fcn.0051f8f0` fetches the
   **`EnergySlotPlate`** GUI element (the energy/skill slot bar, by name
   through the GUI registry `[0x745454]`, `fcn.005270c0`), and
   `fcn.005207f0(plate, 5, skill+0x54, skill+0x5c, skill+0x58, skill+0x60,
   skill+0x64)` writes the skill's parameters into that plate
   (`fcn.0045ac10`).

So `fcn.00541840` is **skill *activation / slotting***, not the effect
cast: it marks the skill active in the agent bitmasks and places it on the
energy-slot UI. (Corrects the earlier "fires the skill's effect" wording —
the actual `CInterpreterSpell`/`CMagic*` behaviour fires when the slotted
skill is *used*.) The split stands: **`CSkill` = the activatable ability
(preconditions, targeting `mode` bit, parameters, hotbar slot); the bound
`CInterpreterSpell`/`CMagic*` = the behaviour** (the `.mgc` program, below).

## Magic effect primitives (`CMagic*`)

24 shared low-level effects that skills, items and traps invoke:

```text
Backstab  Banish  CharmMonster  Disarm  FireDamage  IceDamage
LightningDamage  LockPick  PickPocket  PoisonDamage  Poisoned  Repair
Resurrect  Shadows  Shield  ShieldMagic  SmokeWall  Spikes
SpiritualDamage  SummonDemon  SummonRats  SummonSkeleton
```

Some carry a separate **visual twin** (`CMagicSmokeWallVis`,
`CMagicSpikesVis`) that renders the effect while the gameplay class
applies it. These primitives are the effect layer shared across the
class skills (e.g. the Thief `LockPick` skill and the Wizard Matter
`Unlock` spell both reach the `CMagicLockPick` / unlock effect; the
`CMagicDisarm` effect backs Thief trap-disarming, tying into
[`traps.md`](traps.md)).

## Effect interface (`CMagic*` vtable)

Each `CMagic*` effect is a C++ object with a **12-entry vtable**.
Diffing the vtables of `CMagicFireDamage`, `CMagicIceDamage` and
`CMagicSummonSkeleton` shows only **two** slots differ per effect:

| Slot | Method | |
|---:|---|---|
| 0 | scalar deleting destructor | per-class (always differs) |
| 1 | `Cast` / init (`fcn.00539820`) — **verified shared by all 44 `CMagic*` classes** (the RTTI disasm aliases ~44 `method.C*.virtual_4` labels onto this one address — concrete proof of the "data-parameterised, not behaviour-overridden" thesis: every effect's Cast is literally the *same* function). It sets the **expiry timestamp** at `spell+0x118 = fcn.004c7170([0x658bf4], spell)·1e7 + 2e6` (`fcn.004c7170` resolves the spell's slot in the magic manager — the `·1e7+2e6` is the duration window; the multiplied value is that manager lookup, *not* a literal cast-time field), then **iterates the spell's sub-part list at `spell+0x04`** (count `[list+4]`), invoking each part's `vtable[+0x18]`. | shared base |
| 2,3 | base lifecycle | shared |
| 4 | `GetClassName` — returns the RTTI name string | per-class (e.g. Fire's returns `"CMagicFireDamage"`) |
| 5–11 | base lifecycle: getters (`+field`), an empty override hook (default `ret`), save/load | shared |

The key architectural finding: **the effect classes are
data-parameterised, not behaviour-overridden.** Fire vs ice vs summon
share the *same* ten behaviour virtuals; only the destructor and the
name getter differ. The actual per-effect behaviour comes from each
effect's stored parameters and its magic-script program — not from
overriding an `Apply` virtual. This is why the engine can express ~60
effects with one shared base.

Effect vtables resolve via the MSVC RTTI chain (type-descriptor
`.?AVCMagic…@@` → complete-object-locator → vtable). `CMagicFireDamage`
vtable is at `0x006149f4`.

## Execution

`CMagicInterpreter` / `CMagicSemantic` run spell behaviour through the
internal **magic-script language** (`.\magic\mgcscrpt.cpp`). So a spell
is data — a magic-script program interpreted at cast time — with the
`CMagic*` classes providing the built-in primitives the script calls.

### Effect ↔ script binding (how a class becomes its behaviour)

The effect classes are **not** special-cased one-offs around a single
`CInterpreterSpell`: **every effect class derives from
`CInterpreterSpell`** (vtable `0x61a938`, ctor `fcn.0053ac00`,
`.\script\mgcscrpt.cpp`), and each ctor binds the object to a **named
program in a `.mgc` script file**. `CMagicLockPick`'s ctor
(`fcn.004ca4c0`) is representative:

```text
CInterpreterSpell::ctor(this, "LockPick", "survivor.mgc")   ; fcn.0053ac00
    → loads/parses the named program (fcn.0053a890),
      stores the interpreter state at this+0x110 (+0x114/+0x11c init 0)
this[0] = &CMagicLockPick::vtable                            ; override the base vtable
registry = GetMagicEffectRegistry()                          ; fcn.004c77d0 → singleton [0x6dfe28]
registry.Register(this, id=38, category=2, 0)                ; fcn.004c7510
```

So the **only** thing the C++ subclass adds over the base is its own
vtable (destructor + `GetClassName`) and a numeric **(id, category)**
self-registration; *all* runtime behaviour is the bound mgcscrpt
program. Confirmed bindings (verified from the three `fcn.004c7510`
callers, all `survivor.mgc`, all **category 2**): `LockPick` **id 38**
(`fcn.004ca4c0`), `PickPocket` **id 51** (`0x33`, `fcn.004ca390`), `Repair`
**id 275** (`0x113`, `fcn.004ca5d0`); `Disarm`/`CharmMonster` likewise the
thief/utility group. The registry singleton `[0x6dfe28]` is an 84-byte
object created lazily on first effect. **Note — this `category` (the
`{0,1,2}` kind from the `0x655a98` table) is *not* the `magic.cmp` `f0`**
(a 0–66 running index, [`formats/magic.md`](formats/magic.md)): the two
"category" fields are different keys, so `magic.cmp f0 ≠` the Register
category.

The `.mgc` script files the effects bind to are packed — **as plaintext
mgcscrpt source, not compiled bytecode** — in **`dat\flat.cmp`** (the
Family-A archive, [`formats/cmp.md`](formats/cmp.md)), *not* in `magic.cmp`
(which is only the 96×28 numeric parameter table, [`formats/magic.md`](formats/magic.md)
— correcting the earlier "packed in magic.cmp" note). `flat.cmp` carries
**38 `magic\*.mgc` programs** alongside the `bmg\*.bmg` GUI files:

```text
banish bless deathfx default double eagleeye fear freeze[-old] healing
invisibility lightning magicshield mdamage mgctest mimic msummon
newmagicshield newshield old-healing panic panicapp polymorph portal
projectile ptdemo pushaway shield splinedemo steal strength survivor
telekin test treasure unlock vampire wpmsimple   (+ ptdemo/splinedemo demos)
```

**The programs are directly readable source**, recoverable by extracting
the entry from `flat.cmp` (raw Family-A payload). **Scope (verified by
reading them): the `.mgc` programs are the spell *presentation/visual*
layer — sprite & particle emission and projected-coordinate geometry — not
the gameplay outcome.** `survivor.mgc` is decisive: the thief programs
`"LockPick Source"` / `"PickPocket Source"` are **empty `begin end`**, and
`"Disarm Target"` / `"Backstab Target"` / `"CharmMonster Target"` only
`EmitSpriteList(...)` the effect sprite — i.e. the **lockpick/disarm
difficulty and success rolls are *not* in the script**; they live in the
C++ effect class + the `magic.cmp` parameters + `props.000` curves. So a
`.mgc` gives a spell's exact *visuals* verbatim, while damage/difficulty/
chance stay in code/data. The language is a Pascal-like tree-walked script
([`script-language.md`](script-language.md)); e.g. `bless.mgc` in full
(all visual — lightcone sprites + particle setup):

```text
name "Bless Target"
begin
  { Compute where the projected 2D coordinate is }
  ProjZ := ((TargetZ - CenterZ) * 0.5);
  ProjX := ((ScreenWidth*0.5) + ((TargetX - CenterX) + ProjZ));
  ...
  if ((LocalTime < 9) and (ProjY < 1000)) then
  begin
    EmitSpriteList ("bless.spl", "Bless Lightcone Part 1 Appear", X, Y, Z, -10000);
    while (ProjY > 0) do begin … ProjY := (ProjY - TilePieceHeight); … end
  end
  …
  ParticleType [Count] := (rand () * 2);
end
```

So the grammar is `name "<title>"  begin … end` with `:=` assignment,
`if/then/else`, `while/do`, `{ }` comments, indexed arrays
(`ParticleType[Count]`), built-ins (`EmitSpriteList`,
[sprite lists](sprite-builders.md); `rand()`), and **named host bindings**
(`Source{X,Y,Z}`, `Target{X,Y,Z}`, `Center*`, `Screen{Width,Height}`,
`LocalTime`, …). The bindings are why the script never names raw struct
offsets — the VM maps these names to host fields, so a reimplementation
reads the `.mgc` text verbatim and supplies the same named bindings.

**Complete built-in vocabulary (enumerated from all 38 shipped programs).**
Every function call across the `.mgc` set, by frequency — and they are
**entirely visual / SFX / math, with no gameplay primitive** (no `Damage`,
`SetHp`, stat-setter, etc.), which is the authoritative confirmation that
`.mgc` is the presentation layer:

```text
sprites/draw : EmitSpriteList(166) EmitDummySpriteList(59) DrawStar(30)
               GetRandomSpriteOffset(6) DrawDot(1) DrawQuad(1)
light        : EmitLight(40) EmitLightWithColor(6)
sound        : PlaySoundObject(33) SetSoundObjectPosition(24)
               AddSoundObject(15) RemoveSoundObject(15)
laser/beam   : StartLaser(13) EndLaser(11) LaserNode(3)
math/curve   : rand(142) signrand(57) sin(12) sqrt(11) floor(8)
               spline5(12) cos(1)
timing       : delay(16)
```

(So the `mgcscrpt` *command-vocabulary* listed from the binary's keyword
table in [`script-commands.md`](script-commands.md) — `damage … level #`,
stat-setters, skill-def — is the *language's* full keyword set; the
**shipped spell programs use only the visual subset above**, with the
damage/stat side handled host-C++/`magic.cmp`/`props.000` as scoped above.)

**Complete host-binding set (the VM's read-only inputs, all 43 enumerated
from the programs).** These are the engine-state variables a `.mgc` reads
(never assigns); a reimplementation supplies exactly these — and they too
are **purely visual/effect state**, no gameplay inputs:

```text
position : SourceX/Y/Z  TargetX/Y/Z  CenterX/Y/Z  SpriteX/Y/Z
           SourceSpriteX/Y/Z  HandX0  SpriteOffsetX/Y
           SpecialPointX/Y  Source/TargetSpecialPointX/Y   (arrays, [i])
screen   : ScreenWidth  ScreenHeight   system: SystemIsHardware
time     : LocalTime  GlobalTime  SpellTime
duration : Cast/After/Connect/Execute/Impact Duration
colour   : Source/TargetColorRed/Green/Blue
handles  : SoundHandle  IceSprite
```

So the `mgcscrpt` script API is now fully specified from source on both
sides — **26 built-in calls + 43 host bindings**, all presentation-layer.

### Native per-spell modules (`.\magic\`)

Several primitives also have a **same-named native C++ source module**
under `.\magic\` (their `.cpp` paths survive as linetrack literals). These
hold the bespoke per-effect *behaviour* a data program can't express on its
own — chiefly the per-frame **effect geometry/state** (arc segments, spike
fields, area spread) kept on the effect object at `+0x120`+ and driven via
the [effect processor](screen-effects.md), as opposed to the shared
data-parameterised `CMagic*` gameplay vtable above. Verified modules and
their effects:

```text
bliksem.cpp    → lightning (CMagicLightningDamage)   [Dutch "bliksem" = lightning]
dragonfire.cpp → fire       (CMagicFireDamage)
ice.cpp        → ice/freeze (CMagicIceDamage)
fireball.cpp   → fireball projectile
spikes.cpp     → CMagicSpikes / CMagicSpikesVis (spike field)
mdamage.cpp    → the generic magic-damage effect (mdamage.mgc)
bless.cpp · strength.cpp · eagleeye.cpp   → the buff spells
panic.cpp · panicapp.cpp                   → fear / panic
particle.cpp · shapeset.cpp · generic.cpp  → shared effect helpers
teleport.cpp   → spell-teleport
```

E.g. `bliksem.cpp`'s arc updater `fcn.004be140` is a virtual (no direct
xref) that maintains the lightning-arc state at `esi+0x120`+. So a spell's
**magnitude** is data (the `mgcscrpt` `… damage level #` curve), while its
**spatial/visual form** is this native module — the two layers are
separate.

### Magic-script command vocabulary

Recovered from the command-string blocks (operands: `#` integer,
`$` reference, `level #` = the spell level 1–5 that indexes the
[`formats/props.md`](formats/props.md) per-level curves):

```text
Damage/effect  fire damage level # · lightning damage level # ·
               poison damage level # · ritual damage level # ·
               polymorph level # in $ · cast spell # on $ · charm id #
Spell setup    set spell knowledge #[,#] · set spell level #,# · magic #
Create         create object # · create object # alchemy level # · create food
Stat setters   set armor/attack/defense/dexterity/shield/sight/hearing/status # ·
               set hitpoints # # · set weapon damage # · set weapon dice # ·
               set fightspeed # · set fightwalkspeed # · set magicspeed # ·
               set aiclass # · set alignment $ · set level # · set head #
Skill def      description [level #] $ · requires $ at level # · min level # ·
               level # · level step # · has action $ · borrow action $ from $
```

The **damage verbs all take `level #`** — the same 1–5 skill level whose
numbers live in `props.000` (e.g. `fire damage level 3` reads the
level-3 entry of the relevant curve).

**The complete per-skill parameter binding is in `skills.txt`
(`localizations\<lang>\skills.txt`).** Each skill's description text embeds
**`\v'SP_<skill>','<PropName>'` template references** that bind that skill's
per-rank parameters to **named `props.000` curves**, resolved at the
skill's skill-point level (`SP_<skill>`). The shipped English `skills.txt`
carries **168 distinct `\v` references across 91 skill groups** — i.e. the
*whole* skill/spell roster's effect parameters are data-driven and
enumerable. Examples:

```text
SP_Sword     → SwordChanceToHit · SwordRecuperationTime · SwordDamageModifier
SP_Axe       → AxeExtraDamage · AxeRecuperationTime
SP_Accuracy  → AccuracyDamageBoost · AccuracyRandomDamageBoost
SP_Banish    → BanishChanceTable · BanishDamageTable · BanishRandomDamage
SP_Bless     → BlessBoostDuration · BlessBoostSize
SP_FireDamage→ (3 curves)   SP_Mace/Hammer/Spear/Polymorph/Curse → (4 each)
```

So a skill's damage/chance/duration/etc. is `props[<PropName>][SP_level]`
(with the `base + <X>random` pairing where present), and the binding of
*which* curve to *which* skill is the `skills.txt` `\v` reference set. The
**167 referenced props cover every tunable parameter class** — not just
damage: **cost** (`MimicCost`), **mana** (`InvisibilityManaConsumption`,
`FeignDeathManaLost`), **range** (`IncreasedRangeBoost`, `TrueSightRange`),
**duration** (`*BoostDuration`, `BlindDuration`, `CageDuration`, …),
**recuperation** (`{Sword,Axe,Mace,Hammer,Spear,Bow,Crossbow}RecuperationTime`),
**chance** (`*ChanceTable`), and **damage** (`*Damage`/`*RandomDamage`). So
**every gameplay-tunable spell/skill parameter is a `props.000` curve**
named by `skills.txt` — the spell/skill parameterisation is *fully*
recovered.

**Correction — `magic.cmp` is NOT the spell cost/range table.** Since
cost/range/duration/mana/etc. are all props (above), the 96×7-`i32`
`magic.cmp` ([`formats/magic.md`](formats/magic.md)) is a *different*
per-spell table — its column shapes (`f0` category id, `f5` power-of-two
flag, small `f3/f4/f6`) read as a per-spell **classification / element-flag /
type** record (96 = the skill roster), **not** the tunable parameters. That
re-scopes the magic.cmp column-role 🟡 from "cost/range params" (wrong) to
"per-spell class/element/type flags".

The bindings are not
hardcoded in `div.exe` (this is why prop names like `FireDamageDamage` /
`AccuracyDamageBoost` are absent from the binary's string table while
`WisdomExperienceBoostList` / `MerchantPriceDifference`, consumed directly
by C++, are present). **Net: the skill/spell effect parameterisation is
fully recoverable** = `skills.txt` (curve names per skill) + `props.000`
(the per-level values); only the runtime accumulation/apply (anonymous
load-then-call, [combat.md](combat.md)) stays dynamic.

(Scope, verified: the `\v'group','prop'` substitution is
**skill-description-specific** — 842 occurrences in `skills.txt`, **zero**
in `books.txt` / the diary text — i.e. it is the skill-tooltip dynamic-value
display, *not* a general engine text-variable system; the parameter→curve
binding it encodes is the gameplay-relevant part, the resolver itself is
tooltip presentation.)

**Spell/skill damage magnitude formula (recovered from `props.000`):** the
magnitude is **`<spell>damage[level] + rand(0 .. <spell>randomdamage[level])`**
— a **base curve plus a uniform-random-range curve**, both per-level (1–5).
`props.000` carries the pair for **20 confirmed spell/skill damage sources**
— `lightningdamage`+`lightningrandomdamage`, `chainlightningdamage`+`…random`,
`burndamage`, `poisonclouddamage`, `divinedeathdamage`, `elementalstrikedamage`,
`meteorshower/strikedamage`, `frostarrows/poisonedarrows/piercingarrowsdamage`,
the weapon `…extradamage` skills (`axe`/`hammer`/`crossbow`), `shieldbashdamage`,
… each with its `…randomdamage` twin. So a spell's damage is **fully
data-driven** (base+random, level-indexed) — not an un-recovered formula;
this is the same *base + random* shape as the weapon dice (`numdice·dicetype
+ diceadd`, [combat](combat.md)), just curve-driven instead of dice-driven.
The remaining spell residue is only the `magic.cmp` cost/range column roles
and the (load-then-call, anonymous) apply site, **not** the damage magnitude.

The **stat-setter** group is
shared with the agent setup / [`script-commands.md`](script-commands.md)
statement language. The **skill-def** sub-language (`description`,
`requires … at level #`, `has action`) is how skills declare their
per-level effects and prerequisites.

### Elemental damage formula (`FUN_00592f20`)

The actual per-element damage arithmetic lives in one routine shared by
**spell damage and the trap damage effect** ([`traps.md`](traps.md)):
`FUN_00592f20`, a `0..8` switch on the element. Each per-element prop is
a **per-level array** lazily resolved once via the props lookup
`FUN_00500f10` and cached at `[0x7516d4..0x7516f0]` —
`FireDamage{Damage,RandomDamage}`, `LightningDamage{…}`,
`SpiritualDamage{…}`, `PoisonCloud{Amount,Damage,RandomDamage}`. The
routine indexes them by a **level/tier** `idx = magnitude / 20` (the
`imul 0x66666667; sar edx,3` divide-by-20 idiom on the caller's magnitude
word) and forms:

```text
damage ≈ Damage[idx] + ( rand() reduced into [0, RandomDamage[idx]) )
```

i.e. a per-level **base + bounded random roll**. It then **spawns an area
effect** for that element — `FUN_005761ec` (thunk `FUN_00576200`), the
shared explosion/area-effect create fed by the area-effect manager
`[0x658c54]` (built in `compilestart.cpp`/map-load like the other
singletons) — tagged with the element id (e.g. `4` = poison cloud). So
the elemental "damage" verbs don't subtract HP directly; they drop a
damaging **area effect** ([`explosions.md`](explosions.md)) whose magnitude
is `props.000` base + roll. (The exact secondary scaling of the roll by
the magnitude is read but not fully reduced 🟡.)

## Full effect-class roster (44, from the cast vtables)

Enumerating every class that shares the `Cast` virtual gives **44**
effect classes (the earlier "24 `CMagic*`" count missed the
non-`CMagic`-prefixed ones). `CInterpreterSpell` is the **data-driven
base they all derive from** — it runs an arbitrary `mgcscrpt` program —
so every one of these "a spell is a magic-script" (see the binding
section above); the subclasses differ only in vtable identity and the
named program they bind:

```text
Damage    CMagicFireDamage CMagicIceDamage CMagicLightningDamage
          CMagicPoisonDamage CMagicSpiritualDamage CMagicPoisoned CMagicPoisoned2
          CVampireMagic CVampireHitMagic CDeathEffect1..4Magic
Heal/cure CHealingMagic CMagicResurrect
Buff      CBlessMagic CStrengthSpell CShieldMagic CShieldImpactMagic
          CMagicShieldMagic CInvisibilityMagic CEagleEyeSpell CDoubleMagic
Control   CFearMagic CPanicSpell CPanicReappearanceSpell CMagicCharmMonster
          CMagicBanish CPushAwayMagic CPolymorphMagic CMagicShadows
Summon    CMagicSummonRats CMagicSummonSkeleton CMagicSummonDemon
Thief     CMagicBackstab CMagicDisarm CMagicPickPocket CMagicLockPick CMagicRepair
Teleport  CTeleportMagic CTeleportAppearMagic CTeleportDisappearMagic CPathTraceMagic
Scripted  CInterpreterSpell    (runs a mgcscrpt program)
```

## The `SMagic` manager `[0x658c38]` (`.\magic\SMagic.cpp`)

One of two magic globals — the other is **`[0x658bf4]`**, the
active-effect manager (the `Cast`/init registers timed effects there,
above). `SMagic` is the per-object spell/magic **record table**, built
in `.\GAME\compilestart.cpp` (`fcn.004cd4b0`) and rebuilt at map load
(`fcn.004a0b10`). Reading the builder gives the real layout (an earlier
note mis-attributed the `0x154`/340-byte figure to the singleton itself
— it is one of its sub-allocations):

```text
SMagic manager @ [0x658c38]:
    +0x00   ptr   → a 340-byte (0x154) sub-object (alloc fcn.0058d4d0(5))
    +0x14   i32   = -1 (init)
    +0x18   i32   = -1 (init)
    +0x28 … +0x17c   340 bytes (0x154) of inline state, zeroed
    +0x17c  u32   record capacity = 128 (0x80)
    +0x180  ptr   → heap array of 128 × 84-byte records (0x2a00 = 10752 B, memset 0)
    +0x184  u32   used / next-free record index (init 0)
    +0x188/+0x18c/+0x190  i32 = 0 (init)

record (0x54 = 84 bytes), fields seen across the consumers:
    +0x04   i32   sub-entry count
    +0x08   ptr   → object holding 88-byte (0x58) entries from +0x20
                    (entry+0x00 = id key); flag byte at [+0x3c] & 4 gates it
    +0x0c   ptr   → magic-effect data object
                    (+0x04 = id/category, +0x44 = mutable counter, +0x4c = param)
    +0x28   i32   nonzero = record valid/active (the lookup skips zero)
```

The lookup `fcn.004d3fd0(this, id, startIdx, category)` scans records
`[startIdx … capacity)`, matching `rec+0x28 != 0`, `[[rec+0xc]+4] ==
category`, the `rec+8` flag, then the `rec+8` 88-byte entry array
(`rec+4` entries, key at entry+0) → returns the record index. The stat
methods consult it: the **`CPlayerStatistics` vtable slot 5 (`+0x14`)**,
`fcn.0055aee0` (a CStats virtual, [`stats.md`](stats.md)), does
`rec = [0x658c38]+0x180 + lookup(obj+0x70)*0x54`, then applies via
`fcn.004e2750`, decrementing `[[rec+0xc]+0x44]` — tying the combat/stat
path to the magic table. *(Earlier text put this at `vtable[+0x50]`; it
is slot 5 = `+0x14`, and player-specific — the monster slot 5 is a
different function.)*

## Effect-apply callbacks are static & enumerable (`fcn.004e27b0`)

A significant narrowing of the "effect execution is anonymous load-then-call"
wall: the per-effect apply logic is **not** behind a C++ vtable — it is a set
of **C function-pointer callbacks** installed on each effect descriptor, and
the registration site **`fcn.004e27b0`** enumerates them. For each effect it
writes a callback group onto the descriptor:

```text
[effect+0x30] = phase fn (update / per-tick)
[effect+0x34] = APPLY fn        ← the effect body
[effect+0x38] = phase fn (remove / undo / expire)
[effect+0x3c]/[+0x40] = extra phase fns (some effects)
```

Crucially `fcn.004e27b0` is a **`switch` keyed by the effect-type-id** — it
reads the id from the descriptor (`[arg+4]`), bounds it `≤ 0x52` (82), and
jumps through an **83-entry table at `0x4e302c`**; each `case` installs that
one effect type's callback set and returns. So **case N = effect-type-id N →
its `{apply, update, remove}` bodies**: the id→body mapping is *directly
enumerable* from the jump table, not just an unordered list.

**The full 83-entry table, enumerated** (walking `0x4e302c` and reading the
`mov [eax+0x30/34/38], imm32` installs in each case stub):

```text
eid apply update remove   eid apply update remove   eid apply update remove
 0 4cdc10 4d5620 4d5800   28 4d00f0 4cff20 4d0270   56 4d26d0 4dd290 4e2680
 1 4ce7a0 4d6650 4ce8f0   29 4d8880 4d8790 4d08e0   57 4d2a50 4dd450 4d0270
 2 4cdd50 4cdcc0 4cddb0   30 4d0a20 4d0930 4d0b10   58 4dd6d0 4dd570 4d0410
 3 4d5aa0 4d59e0 4cde20   31 4d0d30 4d0ba0 4cf230   59 4dd740 4d32b0 4d2a80
 4 4cded0 4d5c70 4cdfa0   32 4d7810 4e1d90 4cfb90   60 4d31c0 4dd9f0 4d3400
 5 4d5f00 4d5de0 4d60e0   33 4d7ff0 4d7f40 4cfd60   61 4ddaf0 4d32b0 4d3400
 6 4ce090 4cdff0 4e1c40   34 4d8ab0 4d89f0 4d0d90   62 4d32d0 4dde20 4d3400
 7 4ce310 4d63e0 4ce4c0   35 4d2a50 4cebf0 4cec80   63 4de360 4de250 4d1f50
 8 4ce570 4d64b0 4ce5b0   36 4d9420 4d91a0 4cf230   64 4d34a0 4df1d0 4d3400
 9 4ce960 4d67b0 4ce9d0   37 4d9550 4d9490 4cf230   65 4d0660 4d8690 4d0270
10 4ce620 4d6590 4ce6e0   38   —      —      —      66 4da150 4d1520 4d15c0
11 4d6270 4d6180 4ce2a0   39 4d0df0 4d95d0 4d9810   67 4dc4b0 4dc380 4d3400
12 468f20 4d3d00 4d3d40   40 4d99a0 4d98d0 4d0f90   68 4dc810 4d1f30 4dc9e0
13 4d6b40 4d6930 4cea30   41 4d9ad0 4d9a00 4d1290   69 4da1c0 4d1930 4d19d0
14 4cead0 4d6d00 4d6dd0   42 4d0860 4d06f0 4d0270   70 4de1d0 4ddf80 4d3450
15 4cead0 4d6e60 4d6f30   43 4d0590 4d0500 4d0640   71 4d0280 4d8290 4d0380
16 4d6fc0 4cecb0 4ced40   44 4d9d50 4d9b30 4d12e0   72   —      —      —
17 4d2a50 4cedb0 4cfd60   45 4d13f0 4d1350 4d1430   73 — / 74 — / 75 —
18 4d35e0 4df270 4df3d0   46 4da300 4da230 4d1d20   76 4ddc40 4d32b0 4d3400
19 4d7270 4d7240 4cf230   47 4da800 4da670 4d1db0   77 — / 78 — / 79 —
20 4d7450 4d73b0 4cfd60   48 4dbb90 4dba40 4d3400   80 4e2240 4d83a0 4d03a0
21 4d7640 4d75a0 4cfd60   49 4db710 4db5d0 4db970   81 4e24b0 4d8450 4d0410
22 4d2a50 4d7240 4cfd60   50 4dab30 4da970 4d1e40   82 4d0460 4d8560 4d0410
23 4d7a70 4d78c0 4cf230   51 4dc140 4dbeb0 4d1ee0
24 4d7cd0 4cfc30 4cfd60   52 4dcfd0 4dcd20 4d3400
25 4cfdb0 4cff20 4d0270   53 4db140 4daf80 4db3f0
26 4cff70 4cff20 4d0270   54 4dca70 4dca30 4d1f50
27 4e2060 4d8180 4d0270   55 4d1fa0 4dd1c0 4d2690
```

So **71 distinct `[+0x34]` apply bodies** across the 83 ids, all **static**
in the `0x4cd000…0x4dd000` (`.\magic\SMagic.cpp`) cluster (one outlier:
eid 12 apply `0x468f20`). Eight ids (**38, 72–75, 77–79**) install nothing —
unused/reserved slots (38 is the `switch` default). A few apply bodies are
shared: `0x4d2a50` by ids `{17,22,35,57}` and `0x4cead0` by `{14,15}`.
*(Correction to the first enumeration: eid 80 does install a remove fn,
`0x4d03a0`.)* `eid 6` apply `fcn.004ce090` sets the target's
`+0x224 |= 0x100` hidden bit — but eid 6 is **Panic**, not Invisibility
(see the name table below; the earlier Invisibility label is retracted —
Invisibility is **eid 13**).

These never showed up by xref because the *call sites* are load-then-call
(`call [effect+0x34]`) — but the targets are fully enumerated above. So the
effect *execution* (flag sets, stat scaling, per-phase apply/update/remove)
is **statically recoverable**, and — combined with the proof that
**`magic.cmp` `f0` = this same effect-id** ([`formats/magic.md`](formats/magic.md),
the band-lookup `fcn.004c70c0`) — a reimplementation can chain
`magic.cmp` params → effect-id → these apply bodies end-to-end.

**Spell → effect-id binding — recovered for ~50 skills (not just 3).** The
key is the **skill activate stubs**: each `C*Skill_*` class's vtable **slot
1** is a tiny `push <const>; call fcn.00541840; ret` stub, and that constant
is the skill's **SpellKnowledge bit — which for the spell skills *equals*
the effect-id**. Decoding all 52 `fcn.00541840` callers and resolving each
owner via its RTTI gives a static skill→effect-id table (Bless 0,
Lightning 1, Fear 3, Healing 4, Freeze 5, Strength 7, …, Sparks 61,
ElementalStrike 62, PoisonCloud 58, MagicWall 60, Matter 29, …), cross-
checked by the props strings each apply body reads. *(Caveat: the constant
is strictly the skill-bit; a few utility/thief skills diverge from the
effect-id — LockPick bit 40 vs eid 38, Unlock bit 64 vs charm-id 40,
Shapeshift bit 82 vs charm-id 50.)* So the binding is via the **activate
constant**, not the `fcn.004c7510` ctor registration the earlier "only 3 of
96" note relied on.

**What the apply bodies do — they toggle the agent status-flag words.**
Sampling the cluster, the apply/remove bodies set/clear bits in the agent
flags at `+0x220`/`+0x224` ([`agent.md`](agent.md)): e.g. **eid 6
(Invisibility)** apply `or [+0x224],0x100` (`0x4ce0e5`) — the documented
`hidden` bit, the same one the `visibility set to` script command toggles —
and the cluster also sets `+0x220 |= 0x200` (`0x4cf6d4`) and clears
`+0x220 & ~0x800` / `+0x224 & ~0x100` in the matching remove bodies. So an
effect's gameplay action is, in large part, **a status-bit set on apply and
the inverse clear on remove**, against the agent flag words combat and
perception already read — i.e. the effect table and the agent-flag model
join up. Sweeping the apply bodies for direct `or [+0x224], imm` writes
gives the cleanly-attributable `+0x224` setters:

| effect-id | apply body | bit set | agent.md bit |
|---:|---|---|---|
| 5 | `fcn.004d5f00` | `0x200` | **`asleep`** — set by **Freeze** (the bit doubles as frozen/incapacitated) |
| 6 | `fcn.004ce090` | `0x100` | **`hidden`** — set by **Panic** (the panicked target vanishes; remove `0x4e1c40` reappears it) |
| 13 | `fcn.004d6b40` | `0x100` | **`hidden`** — **Invisibility** |

*(Names from the resolved effect-id table below. Two earlier readings
are retracted: eid 6 is Panic, not Invisibility — Invisibility is
eid 13; and the "eid 13 repositions the agent" split was a misread —
`fcn.0041e480` is a per-component method called by nearly all effect
bodies, not evidence of relocation.)*

(That sweep only catches *direct* `+0x224` or-writes in the apply bodies;
`+0x220` bits and writes routed through helpers aren't in it, so this is a
floor on the effect→bit map, not the whole of it.)

**The complete set of status-bit ops in the cluster** (`0x4cd000…0x4dd000`,
both set and the matching clear) is:

```text
+0x224 |= 0x100        / &= ~0x100        hidden            (agent.md)
+0x224 |= 0x200        / &= ~0x200        asleep            (agent.md)
+0x224 |= 0x10000000   (set, @0x4d0f6d)   non-targetable    (agent.md mask half)
+0x220 |= 0x200        / &= ~0x200        perception-targeting-skip (see below)
+0x220                 &= ~0x800          (control marker — consumer unknown)
```

So the magic effects drive **both halves of the `0x10000100` perception-skip
mask** — `0x100` (hidden, eids 6/13) *and* `0x10000000` (non-targetable) —
plus the `asleep` state and two `+0x220` status bits. **The two `+0x220`
bits are now characterized**: `0x200` is a **third perception/targeting-skip
bit** — set by the SummonedShields body (`or [+0x220],0x200` @`0x4cf6d4`,
pushing `SummonedShieldsLevelBoost`) and in combat @`0x44a722`, and **read
by the perception/targeting evaluator** (`fcn.0044a600` @`0x44aacb`:
`test [+0x220],0x200` immediately after the `+0x224 & 0x10000100`
hidden|non-targetable test → sets the skip result `+0x3c |= 8`). `0x800` is
only ever *cleared* (in two effect-remove bodies `0x4cde4a`/`0x4d1dda`,
gated on `+0x218==1`) with no locatable set-site or reader — a control/
effect-active marker whose consumer isn't provable statically. **Caveat on
attribution:** mapping each of these writes to a *specific* effect-id by
nearest-address is **unreliable** — the apply/update/remove bodies interleave
in the cluster and several effects share a body, so address-bucketing
mis-assigns (it disagreed with the per-body sweep). Only per-body
disassembly gives a sound effect-id→bit link; the table above is therefore
the trustworthy floor.

**The binding mechanism is the effect factory `fcn.004df5d0`.** This large
(~9.5 KB) dispatcher is what *creates* an effect from a type-id: it
bounds-checks the id (`cmp …,0x52`/`0x63` — the same `≤82` ceiling as the
registration switch, plus a `≤99` path) and builds the effect object,
registering its callbacks through `fcn.004e27b0`. Every effect is spawned by
a caller passing its **effect-type-id** to `fcn.004df5d0` — e.g. the on-hit
proc resolver `fcn.004c6500` ([`combat.md`](combat.md)) calls it to spawn the
Fire/Lightning/Poison damage effects when a proc rolls under its chance, and
the spell-cast paths call it with their spell's effect-type-id. So the
spell→effect binding flows through `fcn.004df5d0`. **Record-layout
correction (re-derived while resolving the names):** the effect-source
record carries the **effect-type-id at `rec+0x04`** — it drives both the
factory's own 83-case switch (jump table `0x4e1af4`) and the
registration switch — while **`rec+0x14` is the 0–100 band-axis
*magnitude*** (clamped to 99, passed as the *value* to the `magic.cmp`
band lookup `fcn.004c70c0`) and `rec+0x00` is the record byte size (the
record is memcpy'd via `0x4fa4b0`). The earlier "type-id read from
`[source+0x14]`" was a misread of the magnitude clamp. Every factory
case N pushes the literal **N** as the group id to `fcn.004c70c0`,
re-proving `effect-id = magic.cmp f0` per-case. The full effect
pipeline is therefore *entirely* data + static-body:
*spell record `+0x04` (effect-type-id) → factory `fcn.004df5d0` →
registration switch `fcn.004e27b0`[id] → `{apply +0x34, update +0x30,
remove +0x38}` static bodies → invoked by load-then-call
`call [effect+0x34]`*. The former residual — attaching a human name to
each id — is **resolved**: see the table below.

## Effect-id → name table ✅ (RESOLVED)

The binding fell to a **direct name→id table in the binary**: the charm
parser in `.\itemstat\itemstatistic.cpp` — a `stricmp` chain at
**`0x4b2610..0x4b364e`** mapping human spell names to the numeric ids
of this same effect-id space (proven by four independent anchors, eids
55/56/44/23). Two more naming keys: each eid's *update* callback
constructs its named `.mgc` visual (`push "<name>"; push "<file>.mgc";
call CInterpreterSpell::ctor`), and the props-name strings in the apply
bodies tie to `skills.txt`'s `\v'SP_<skill>','<Prop>'` bindings.

Evidence classes: **P** proven (prop/string/OSI evidence in the body),
**C** charm-chain name (direct name→id code), **V** `.mgc` ctor visual
binding, **I** inferred, **U** unknown.

```text
eid name                        ev   eid name                        ev
 0  Bless                       V    42  Backstab                    C+V
 1  Lightning                   C+V  43  Poisoned (status; proc 0x4c6848) C+V
 2  Mimic ("Summon lesser…")    V/C  44  NecroShift                  C+P
 3  Fear                        C+V  45  Identify (charm form)       C
 4  Healing / Cure              C+V  46  Curse (stat back-boost debuff) C
 5  Freeze (sets 0x200)         C+V  47  Blind                       C
 6  Panic (hides target)        C+V  48  Leadership (party aura)     C
 7  Strength ("… of the lion")  C+V  49  Charm                       C
 8  Telekinesis                 C+V  50  Shapeshift ("frog" default) C+P
 9  Teleport                    C+V  51  Teleport jump               C
10  Wizard's/Eagle Eye          P+C  52  Divine Death                C+P
11  Panic-vanish (scripted)     V/I  53  Ghost                       C+P
12  no-op placeholder           P    54  Mass Domination (BurnDamage) C+P
13  Invisibility                C+V  55  Light of Heaven             C+P
14  Shield ("Orb of protection") C+V 56  Divine Eye (+0x224|=0x300)  C+P
15  Magic Shield ("Orb of magic") C+V 57 no-op timer sub-phase (shared 0x4d2a50) U
16  Fire ball                   C    58  Poison Cloud (area DoT ticker) P
17  Death Effect (deathfx.mgc)  V    59  Stun (magnitude 10)         C
18  Magic Attractor             P    60  Magic Wall (agent-area retarget) P
19  reveal-invisible (TrueSight?) I  61  Sparks (SparksDamage + type-13 proj) P
20  Summon Skeleton             V    62  Elemental Strike (rand-element DoT) P
21  Summon Demon                C+V  63  Polymorph (OSI class variant) P
22  no-op timer sub-phase (0x4d2a50) U 64 one-shot explosion spawner I
23  Polymorph                   P    65  Deathblow (charm id 71→65)  P
24  Drain Life (Vampire)        P+V  66  Repair (charm form)         V
25  Fire damage (proc/charm)    C    67  Berserk (FlashStaminaDrain) P
26  Lightning damage            C    68  Warrior Prayer (Boomerang*) P
27  Poison damage               C    69  LockPick                    I
28  Spiritual damage            C    70  Warrior Sidekick (shadow)   P
29  Matter (DiscsDamage, type-12 proj) P 71 Deathblow (charm; kill-gate + OSIRIS event) P
30  Frost / Ice (CSimpleIceMagic) C+P 72–75,77–79 weapon-proficiency charm
31  minimal no-op (caster +0x25c only) U        ids (Sword/Mace/Axe/Hammer/
32  Resurrect ("Resurrect2")    V                Bow/Crossbow/Shield) — no
33  Banish                      V                callbacks by design    C
34  Smoke wall                  C    76  Spear throw (SpearThrow*)   P
35  Shield Impact               V    80  Timestop (TimeStopDuration) P
36  Meteor shower               C    81  Dragon Breath (ChainLightning*) P
37  Spikes ("Spike Element")    V    82  mass item-spawner (itemlink type 0x57) U
39  Shadows (0x10000000 non-targetable) V+P
40  Unlock                      C    41  Steal / PickPocket          C+V
```

Charm-only ids beyond the switch range: Staff 83, Dagger 84, Life drain
86, Mana drain 87. A block of legacy charm names maps to **−1**
(unimplemented: Create food, Light, Magic arrow, Sleep, Heal, …).
`magic.cmp` carries groups for f0 ∈ {0–11, 13–17, 19–56, 60, 66} —
the absent ids (12, 18, 57–59, 61–65, 67–71, 76, 80–82) are exactly
the effects parameterised by `props.000` curves instead (Timestop,
DragonBreath, Berserk, WarriorPrayer, Sidekick, SpearThrow,
MagicAttractor, …); f0=38 has a record but no body (switch default).
The "unused" switch ids 72–75/77–79 are the weapon-proficiency charm
ids — they install no callbacks *by design*, closing that puzzle too.

Two utility skills are **not** effect-table entries but direct action
bodies in the same cluster (dispatched from the utility-skill handler
sites `0x4a604c..0x4a818a`): **Identify `fcn.004d0ff0`** (clears the
unidentified flags bit via `fcn.00591940`, [items.md](items.md)) and
**Repair `fcn.004d1620`** (props `RepairQuality` × skill rank →
`CItem+0x1c`, [items.md](items.md)).

## Summoning (`CMagicSummon*` → `msummon.mgc`)

The summon spells — **`CMagicSummonRats` / `CMagicSummonSkeleton` /
`CMagicSummonDemon` / `CMagicSummonMimic` / `…DemonicAide`** — are *not*
special-cased: each is a `CInterpreterSpell` whose ctor binds it to the
shared script program **`msummon.mgc`** with a display name (e.g.
`CMagicSummonRats` `fcn.004ca7f0` → `CInterpreterSpell` base `fcn.0053ac00`
with `"msummon.mgc"`, `"Summon Rats"`), exactly the data-driven effect↔script
model above. Casting runs that `mgcscrpt` program, which **creates a normal
monster agent** (the standard `fcn.005068f0` create → deterministic
class stats, [`monsters.md`](monsters.md)) and the agent-magic dispatch
**stamps the new creature's summoned-by backref `agent+0x1a4`** (written at
`0x0041ddc2`, [`agent.md`](agent.md)) so it points at the summoner.

**Count and strength are `props.000` curves, bound by name** — *not* logic
buried in the script. The per-class C++ is identity-only (the
`CMagicSummon*` methods are just the 6-byte class-name getters
`0x4ca810`/`0x4ca840`/`0x4ca870` + a dtor), and `skills.txt` binds each
summon's numbers via `\v'SP_<spell>','<param>'`, resolved against the
shipped `props.000` (parsed byte-exact — all 9005 bytes, 192 props, see
[`formats/props.md`](formats/props.md)). The verified per-skill-level (1–5)
values:

| Spell | count curve | strength curve |
|---|---|---|
| Summon Rats | `RatsAmount` `[1,2,3,4,5]` | `RatsHitpoints` `[20,30,40,50,60]` |
| Summon Skeleton | `SkeletonAmount` `[1,2,3,4,5]` | `SkeletonLevel` `[4,8,12,16,20]` |
| Summon Demon (DemonicAide) | — (always 1) | `DemonicAideLevel` `[20,26,32,38,44]` |
| Summon Mimic | — (always 1) | `MimicCost` `[10,20,30,40,50]` (% of caster abilities inherited) |

Two passives modify summons: **Master Summoner** adds a level boost to
non-animal summons (`SummonedShieldsLevelBoost` `[2,3,4,5,6]`), and
**Leadership** caps how many summoned creatures can join the party
(`LeadershipLevel` `[1,2,3,4,5]`).

So a **summoned creature is a temporary monster** tagged by `+0x1a4`, **not**
a persistent **Party** roster member (the `Party` save-block is the
companion roster, [`formats/savegame.md`](formats/savegame.md)) — summons
and the party are distinct. **There is no duration/lifetime curve** — no
`<spell>Duration` prop exists for any summon (contrast the explicit
`ShapeShiftDuration`/`PolymorphDuration` below), so summons **persist** until
killed or re-cast; the earlier "spawn count / lifetime is in the script"
note was wrong on both points — the count is a props curve and there is no
lifetime.

## Polymorph / Shapeshift (`.\magic\shapeset.cpp`)

Two skills change an agent's **form** by swapping its `currentshape`:

- **`Polymorph`** (Wizard Matter, the offensive `CPolymorphMagic`
  vtable `0x615098`) — turns a *target* into a (usually weaker) creature.
- **`Shapeshift`** (Survivor Talent, self) — turns the *player* into a
  monster form.

Both run through `.\magic\shapeset.cpp` (and the `polymorph.mgc` script,
the same `CInterpreterSpell`+`.mgc` model as summons). The shape is the
object/agent's **`currentshape`** field; the new form's attack comes from
the **`shapedamage`** item-stat column and the set of allowed forms is
bounded by **`maxshape`** ([`items.md`](items.md) / objects.000 columns).
The behaviour is props-tuned: **`ShapeShiftDuration`** / `PolymorphDuration`
(how long the form lasts), **`PolymorphResistanceLevel`** (the target's
chance to resist), and `PolymorphBaseDamage` / `PolymorphRandomDamage`. So
polymorph/shapeshift = *set `currentshape` to a monster kind for a
props-duration, re-deriving sprite + `shapedamage` attack from that form* —
the exact swap (stat/sprite re-bind) lives in the `polymorph.mgc` script
(the `mgcscrpt` layer) 🟡.

## Citations

```text
RTTI type descriptors (.data .?AVC...@@):
  C<Class><Discipline>Skill_<Name>   — 3 classes × 4 disciplines of skills
  CMagic<Effect>                     — 24 shared effect primitives (+ *Vis twins)
  CMagicEffect / CMagicSemantic / CMagicInterpreter — base + script execution
Magic managers: [0x658bf4] active-effect mgr · [0x658c38] SMagic table (84-B records)
                · [0x658c54] area-effect/explosion mgr (FUN_005761ec spawn, FUN_00576200 thunk)
Damage routine: FUN_00592f20 (shared with traps) · props lookup FUN_00500f10
Source units: .\magic\magic.cpp, .\magic\mgcscrpt.cpp, .\magic\SMagic.cpp,
              .\Skills\skills.cpp, .\Skills\skillthief.cpp
```

## Status

- Skill tree ✅ — full class/discipline/skill catalogue from RTTI
  (Survivor/Wizard/Warrior × four disciplines each).
- Skill activation ✅ (full core) — `CSkill` subclasses forward through one
  dispatcher `fcn.00541840(modeBit)` with a per-skill bit (LockPick `0x28`
  after check `fcn.00545f60`; Unlock `0x40`). `fcn.00541840` resolves the
  caster (`[0x658c04]+0x4b4` → `CAgentManager [0x658d50]`), **sets the
  skill's bit in `SpellKnowledge[3]` (`agent+0x58/+0x5c/+0x60`) and
  `SpellLearned[3]` (`+0x64/+0x68/+0x6c`)** ([`agent.md`](agent.md)) — i.e.
  marks the ability known+learned (the learn/grant path) — records
  active-skill state (`agent+0x25c` skillcomp
  `+8=mode`, `+0x35c=-1`, `+0x360=skill+0x2c`), and **slots the skill into
  the `EnergySlotPlate` hotbar GUI** (`fcn.0051f8f0` → `fcn.005207f0` →
  `fcn.0045ac10`, GUI registry `[0x745454]`). This is activation/slotting,
  **not** the effect cast (corrects an earlier "fires the effect" note);
  the bound `CInterpreterSpell`/`.mgc` runs when the slotted skill is used.
- `SMagic` table layout ✅ — manager `[0x658c38]` holds a **128 × 84-byte
  record array** at `+0x180` (capacity `+0x17c`=128, used `+0x184`), each
  record nesting an 88-byte entry array (`+0x08`) and an effect-data
  object (`+0x0c`); recovered from the builder `fcn.004cd4b0` and the
  lookup/apply consumers (`fcn.004d3fd0`/`fcn.0055aee0`/`fcn.004e2750`).
  Corrects the earlier "340-byte singleton" conflation. Per-record field
  *semantics* (what each entry/effect object means) 🟡.
- Effect primitives ✅ — **44 effect classes** enumerated from the cast
  vtables (damage/heal/buff/control/summon/thief/teleport groups), incl.
  the non-`CMagic*` ones; `CInterpreterSpell` is the data-driven
  mgcscrpt class. Plus `*Vis` visual twins.
- Execution model ✅ — spells run as magic-script via
  `CMagicInterpreter`; the command vocabulary is enumerated (damage
  verbs take `level #` → `props.000` curves; spell-setup, create/alchemy,
  stat-setters, and a skill-def sub-language), ties to
  `script-commands.md`.
- Effect interface ✅ — 12-virtual vtable; only the destructor (slot 0)
  and `GetClassName` (slot 4) are per-effect; the ten behaviour
  virtuals are shared, so effects are data-parameterised not
  behaviour-overridden. Slot 1 is the substantial `Cast`/init.
- Per-effect level values ✅ — the per-skill-level (1–5) numbers
  (backstab %, poison-cloud damage, bless duration, summon counts, …)
  live in `dat\props.000`, resolved by name; see
  [`formats/props.md`](formats/props.md).
- Elemental damage formula ✅ — `FUN_00592f20` (shared with traps) indexes
  the per-element `Damage[]`/`RandomDamage[]` prop arrays by
  `idx = magnitude/20`, forms `Damage[idx] + bounded-rand(RandomDamage[idx])`,
  and spawns an **area effect** for that element via the area-effect
  manager `[0x658c54]` (`FUN_005761ec`) — so elemental verbs drop a damaging
  cloud rather than subtracting HP inline. The exact roll-by-magnitude
  scaling is read but not fully reduced 🟡.
- Skill → effect binding ✅ — every effect class **derives from
  `CInterpreterSpell`** and binds in its ctor to a named program in a
  `.mgc` script file (`CMagicLockPick` → `("survivor.mgc","LockPick")`
  via `fcn.0053ac00`), storing the interpreter at `this+0x110`, then
  self-registers by numeric `(id, category)` (LockPick 38, Repair 275;
  category 2) into the effect registry singleton `[0x6dfe28]`. The
  subclass adds only its vtable + the id; all behaviour is the bound
  mgcscrpt program.
- Per-effect formulas (lockpick/disarm/pickpocket difficulty, etc.) 🟡 —
  these live **in the `.mgc` programs** (e.g. `survivor.mgc`'s `LockPick`
  program), packed in `dat\magic.cmp`, not in `div.exe` code. Reaching
  the concrete numbers needs the `magic.cmp` unpack + the mgcscrpt
  program text, not further static disassembly — recorded as a
  data-located dead-end for the code trace.
- Skill level/cost/cooldown storage 🟡 (registry side mapped) — the skill
  **definitions/symbols** live in the global registry **`[0x7467fc]`**
  (the generic by-name/by-id entity registry, looked up via `fcn.00543900`
  / `fcn.005438c0`); the `mgcscrpt` **`SkillLevel`** reference token is
  resolved through it by **`fcn.004ae030`** (so a spell-script expression
  like `… SkillLevel` reads the caster's rank for the skill via this
  registry), and **`CSkill`** (vtable `0x61b440`, `.\Skills\skills.cpp`) is
  the skill-**definition** class. *(Ruled out as the rank store: `CSkill`
  is the global per-skill-**type** definition held in the registry
  `[0x7467fc]` — its ctor `fcn.00541350` zero-inits a fixed field block
  `+0x20..+0x70` (getter `0x541910` reads `+0x44`), one object per skill
  type rather than per agent, so the per-(agent, skill) rank is not a
  `CSkill` field.)* The **per-agent learned-skill set** is the
  **`SpellLearned[3]` /
  `SpellKnowledge[3]`** fields on `CStats` (`+0x64`/`+0x68`/`+0x6c` and
  `+0x58`/`+0x5c`/`+0x60`, [`agent.md`](agent.md)): **3 × `i32` = 96 bits =
  exactly the 96 skills** of the [skill tree](skill-tree.md). It is a
  **one-bit-per-skill** set (`SpellKnowledge` = available, `SpellLearned` =
  actually taken) — 96 bits is precisely the skill count and is too small
  to hold ranks, so it can only be a yes/no mask. **Bit-test confirmed:**
  `fcn.00444ac0` gates on a specific skill with **`[CStats+0x64] & 0x20`**
  (SpellLearned, bit 5 of dword 0) and **`[CStats+0x58] & 0x20`**
  (SpellKnowledge, bit 5) — single-bit tests by skill index, so the fields
  are genuinely per-skill bitmasks (`dword = skill>>5`, `bit = skill&0x1f`),
  not packed values. The per-skill **rank value** (1–5) is **not** in these
  fields.
  Note `fcn.004ae030` is the **parse-time** `mgcscrpt` symbol resolver —
  it `stricmp`s the `SkillLevel` keyword and resolves the named skill
  through the registry `[0x7467fc]` (`fcn.00543900`, by-name), i.e. it
  binds the *script token* to a skill, **not** the runtime rank. The
  runtime rank (which the `props.000` per-rank curves index by level−1) is
  read during cast *evaluation* from a per-(agent, skill) store that the
  direct code probes did not cleanly isolate (not adjacent to the
  `SpellLearned[3]` bitmask, nor in the symbol registry).
  - **Located via the save path (this pass):** the rank store *must*
    round-trip a save, and it does — through the **`SkillsV0.935` save
    block**, reader **`fcn.00543620`** (`.\Skills\skills.cpp`,
    [`formats/savegame.md`](formats/savegame.md)). It reads the skills
    manager's scalars (`+0x20`/`+0x24`/`+0x28`/`+0x34` via `fcn.004f4c70`)
    and a **bulk per-skill state block** (`+0x10` via `fcn.004f4c00`) into an
    allocated list (`+0xc`/`+0x14`). Structure (verified further): the
    `+0x10` read takes a **count**, then the reader allocates a **per-skill
    `u32` array** (`count × 4`, count from `[esi+8]`) — i.e. the saved skill
    state is **one `u32` per skill**. So the per-(agent, skill) **rank is a
    `u32` in that SkillsV0 skill-state array**, not in a `CStats`/agent field
    — which is exactly why the field-offset probes (which scanned the
    agent/CStats) missed it. The only residue is whether that `u32` is the
    bare rank or packs rank+cooldown/known bits (a regenerable serialization
    detail, not worth a byte-grind); the **store is now located and its
    shape pinned** (count-prefixed per-skill `u32` array), no longer an
    unanchored gap.
- **Two skill-effect modes (from the same bitmask).** Not every skill is
  rank-scaled: `fcn.00444ac0` fires several **fixed-chance** procs gated
  *only* on the learned bit — e.g. `[CStats+0x64] & 0x20` (skill bit 5)
  then `rand()%N < 10` (a flat 10% proc), and `& 8` (bit 3) → `fcn.004d3fd0`
  — with **no rank read**. So the `SpellLearned` bitmask drives a class of
  **binary** skill effects directly (on/off passives & procs at a constant
  rate), while the **rank value** matters only for the **`props.000`-scaled
  spell skills** (the ones whose magnitude indexes a per-rank curve). That
  splits the skills into *bit-gated fixed-effect* vs *rank-scaled*, and
  narrows the open rank store to just the latter group.
