# The skill tree (RTTI roster)

The complete Divine Divinity **skill tree**, recovered from the RTTI class
names in `div.exe`. Every learnable skill is a C++ class named
`C<Class><Discipline>Skill_<Name>` (Warrior/Survivor) or
`CMagicWizard<School>Skill_<Name>` (Wizard), so the binary spells out the
whole tree: **three character classes, four disciplines each, eight skills
per discipline** (Wizard's Matter/Summon split 11/5 instead of 8/8). This
is the structured taxonomy behind the in-game skill screen; the *effect*
framework these plug into (the data-parameterised `CMagic*` vtables, the
`mgcscrpt` vocabulary, per-level `props.000` curves) is in
[`skills-magic.md`](skills-magic.md), and the three classes are
[`stats.md`](stats.md)'s Warrior / Wizard / Survivor.

## Warrior

| Discipline | Skills |
|---|---|
| **Specialist** (weapon mastery) | Axe · Bow · Crossbow · Hammer · Mace · Shield · Spear · Sword |
| **Knowledge** | FireDamage · IncreasedDamage · IncreasedDefense · LightningDamage · PoisonDamage · Repair · Rush · Stun |
| **Ranger** | Accuracy · BlockArrows · ExplosiveArrows · FrostArrows · IncreasedRange · PiercingArrows · PoisonedArrows · SplittingArrows |
| **God** | Berserk · FeignDeath · Sanctuary · SpiritualDamage · WarriorPrayer · WarriorSidekick · WarriorTactics · WeaponCharming |

Class prefix `CWarrior<Discipline>Skill_…` (e.g. `CWarriorSpecialistSkill_Sword`).

## Wizard

| School | Skills |
|---|---|
| **Elemental** | ElementalStrike · Freeze · Lightning · MeteorShower · MeteorStrike · PoisonCloud · Sparks · WallOfSmoke |
| **BodySpirit** | Bless · BodyShield · Fear · Healing · Invisibility · MagicShield · Strength · WizardEye |
| **Matter** | Banish · Cage · MagicAttractor · MagicWall · Matter · Polymorph · Resurrect · Spikes · SummonedShields · Telekinesis · Unlock |
| **Summon** | DemonicAide · Life · Mimic · Rats · Skeleton |

Class prefix `CMagicWizard<School>Skill_…` (e.g.
`CMagicWizardElementalSkill_Freeze`). The Wizard skills are **directly**
[`CMagic*`](skills-magic.md) effect classes — each one *is* a castable
spell.

## Survivor

| Discipline | Skills |
|---|---|
| **Thief** | BackStabbing · BoobyTrap · DetectTraps · LockPick · PickPocket · PoisonWeapon · Running · Shadows |
| **Lore** | Alchemy · Blind · Curse · Identify · MonsterIdentification · NecroShift · PoisonBody · TrueSight |
| **Talent** | Antimagic · Charm · Leadership · Merchant · Rangersight · Shapeshift · Survival · Wisdom |
| **Divine** | DivineDeath · DivineEye · DragonBreath · Ghost · LightOfHeaven · MassDomination · TeleportJump · Timestop |

Class prefix `CSurvivor<Discipline>Skill_…` (e.g.
`CSurvivorThiefSkill_LockPick`).

## How a skill is implemented

- **Wizard** skills are themselves `CMagic*` effects (the
  `CMagicWizard<School>Skill_…` class names) — casting one runs the spell
  through `CMagicInterpreter` ([`skills-magic.md`](skills-magic.md)).
- **Warrior / Survivor** skills are `C<Class><Discipline>Skill_…` classes.
  Active ones delegate to a matching `CMagic*` effect that already exists
  in the roster — e.g. `CSurvivorThiefSkill_LockPick` → `CMagicLockPick`,
  `…_PickPocket` → `CMagicPickPocket`, `…_BackStabbing` → `CMagicBackstab`,
  `CWarriorKnowledgeSkill_Repair` → `CMagicRepair`,
  `…_Stun`/`…_FireDamage`/`…_LightningDamage`/`…_PoisonDamage` →
  the on-hit damage `CMagic*` variants. Passive ones (the Specialist
  weapon masteries, Wisdom, Survival, Running) instead feed the
  [stat](stats.md) / [combat](combat.md) math (to-hit, damage, carry,
  speed) rather than casting anything.
- Per-rank scaling for every skill comes from the named
  [`props.000`](formats/props.md) curves (e.g. `CharmDuration`,
  `StunDuration`, `RepairQuality`, `BlindDuration`, `SummonedShieldsLevelBoost`),
  indexed by skill level − 1.

This is why several gameplay systems documented separately are really
*skills*: [Charm/Summon](party.md) (Talent/Summon), [Polymorph &
Shapeshift](skills-magic.md) (Matter / Talent), [traps](traps.md)
(BoobyTrap/DetectTraps), [LockPick](object-interaction.md), Identify
([items](items.md)), and the elemental [pain-point](painpoints.md) clouds
(Elemental school).

## Status

- Full roster ✅ — **96 skills**, three classes × four disciplines,
  enumerated from RTTI (`CWarrior*Skill_`, `CMagicWizard*Skill_`,
  `CSurvivor*Skill_`).
- Class/discipline structure ✅ — Warrior {Specialist, Knowledge, Ranger,
  God}; Wizard {Elemental, BodySpirit, Matter, Summon}; Survivor {Thief,
  Lore, Talent, Divine}.
- Implementation mapping ✅ — Wizard skills are `CMagic*` spells;
  Warrior/Survivor active skills delegate to a `CMagic*` effect, passives
  feed the stat/combat math; per-rank values from `props.000`.
- Per-skill point-cost per rank ✅ — the five-rank skill-point cost ladder
  for all 96 skills is the `dat\wgiaa.000` table
  ([`formats/skillcosts.md`](formats/skillcosts.md)).
- Per-skill effect parameters ✅ — name + **type (Passive/Active)** + the
  per-rank parameter→`props.000` curve bindings are in
  `localizations\<lang>\skills.txt` (the `\v'SP_<skill>','<PropName>'`
  reference set, [`skills-magic.md`](skills-magic.md)).
- Per-skill prerequisite tree 🟡 — the in-UI dependency graph is **not** in
  the shipped data: `skills.txt` carries each skill's name/type/parameters
  but **no** prerequisite/dependency fields (its "must have…" lines are
  gameplay target-resistance text, not tree edges). So the dependency graph
  lives in the **unshipped `skills.dat`** (the boot loader reads `skills.dat`
  + `skills.txt`, [architecture.md](architecture.md) step 32) or is
  hardcoded — i.e. it is an *unshipped-source* gap, not an undecoded shipped
  format.

## Citations

```text
RTTI class families (.data .?AVC… type descriptors):
  CWarrior{Specialist,Knowledge,Ranger,God}Skill_*       (32)
  CMagicWizard{Elemental,BodySpirit,Matter,Summon}Skill_* (32)
  CSurvivor{Thief,Lore,Talent,Divine}Skill_*             (32)
delegate examples: CSurvivorThiefSkill_LockPick↔CMagicLockPick · …_PickPocket↔CMagicPickPocket
                   CWarriorKnowledgeSkill_Repair↔CMagicRepair · …_BackStabbing↔CMagicBackstab
```
