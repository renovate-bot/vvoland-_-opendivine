# Combat resolution

How a melee attack is resolved. This documents the combat **entry
points and control flow**; the exact damage numbers are not a single
inline formula — Divine Divinity resolves combat through its agent
message + event systems and parameterises it from item/character data,
so the arithmetic is spread across event handlers and data, not one
function. That structural fact is the main finding here.

## Melee resolver (`FUN_00417b40`)

The core melee-hit function (`.\AGENTS\agentfight.cpp`). It:

1. Reads the attacker's `CStats` — Offense (`+0x14`), and the
   surrounding stat block (`+0x1c`, `+0x20`); see [`stats.md`](stats.md).
2. Runs the **to-hit gate** (now pinned, `0x00417fc8`–`0x00418022`):
   it calls a virtual on the attacker's stat block —
   `(*[stats])[+0x50]` with the weapon/offense field `[stat+0x2c][+0x80]`
   — producing a **chance value** in `ecx`, then computes
   **`defenseStat[+0xc] / 5`** in `eax` (the compiler-emitted
   `imul 0x66666667; sar edx,3` = ÷10, then `add eax,edx` doubles it →
   ÷5) and does `cmp ecx, eax; jg skip`. So the gate is
   **`chance(offense, weapon) ≤ defense/5` → fire the result event**
   (`fcn.0050c6a0` → Osiris). The `defense/5` term matches the
   percentage feel of the per-weapon `MaceChanceToHit` /
   `SwordChanceToHit` properties; the exact `chance(...)` is the
   `+0x50` virtual (offense + weapon `*ChanceToHit`), left as the one
   unresolved input. (Note: the nearby FPU block at `0x00417cb2` —
   `fild [+0x278]`·`[+0x22c]`·`[+0x230]` → `[+0x1c]`/`[+0x20]` — is
   **locomotion** (Walkcount·CellDx/Dy → midpoint), not damage.)
3. Iterates the **four damage components** in a repeated block
   (`read stat → FUN_00416050 → FUN_0054a1b0`), once per element —
   matching the four elemental channels of `CStats`
   (fire/lightning/poison/spirit resistances reduce the matching
   component).
4. Dispatches the result as **events**, not direct writes: it builds
   combat-event objects (`FUN_00438cc0` packs a 0x14-byte event for the
   manager at `0x659210`) and fires Osiris events (`FUN_0050c6a0` →
   event manager `[0x7447dc]`, the same one used by
   [`object-interaction.md`](object-interaction.md) and
   [`world-clock.md`](world-clock.md)).

So the hit is *computed* here (to-hit, per-element damage) but *applied*
through the message/event pipeline, which is why no single function
holds the whole damage equation.

## Inputs

- **Attacker**: `CStats` Offense, plus weapon `CItemStatistic` dice
  (`numdice` / `dicetype` / `diceadd`) and per-weapon chance-to-hit
  (the `*ChanceToHit` keywords).
- **Defender**: `CStats` Defense, armour class, and the four elemental
  resistances (which scale the matching damage component).
- **Modifiers**: Warrior Knowledge/Specialist/Ranger skills
  (`IncreasedDamage`, `*ChanceToHit`, arrow effects) and item special
  damage (`set special damage # # #`, `set damage2 # #`), see
  [`skills-magic.md`](skills-magic.md).

## Correction: the "4 components" loop spawns visual FX, not damage

⚠️ An earlier draft of this doc called `FUN_00416050` → `fcn.004edef0`
the "damage calculator". **That was wrong** — reading it properly:
`fcn.004edef0` sets `vtable.CAniEffect_PlayAnimation` and its RTTI is
`CAniEffect_PlayAnimationAdditiveAttachedToNpcCenter`, i.e. it
**constructs the per-hit visual animation effect**; the integer math
there (`-(w/2) - off`) is the *center-attach position offset*, not
damage. Likewise `FUN_0054a1b0` is FPU + render-manager calls (the
floating damage number / blood splat). So the melee resolver's
"four components" loop is **four visual channels** (per-element hit
FX), **not** four damage rolls. The `numdice/dicetype/diceadd` fields
feed the calc, but the arithmetic is *not* in `fcn.004edef0`.

## What `FUN_00417b40` actually does (and where damage is *not*)

Reading it through, the melee resolver:

1. runs the **to-hit gate** (below);
2. **spawns the per-element hit FX** (`FUN_00416050`→`fcn.004edef0`
   animation; `FUN_0054a1b0` floating-damage/blood);
3. creates an **attractor** (`FUN_00438cc0` → manager `[0x659210]`,
   `FUN_00438b20`) — the source string is **`.\AGENTS\attractor.cpp`**,
   so this is the AI **attractor** system (an attention/aggro point that
   pulls agents), *not* a combat-damage queue. (`fcn.00411a70`, which
   does cell-range math over `[0x659210]`, is an **attractor spatial
   query**, not a damage consumer — an earlier note of mine that called
   it the damage handler was wrong.)
4. fires an **Osiris event** (`fcn.0050c6a0` → event manager `0x7447dc`).

## Damage roll + HP apply — **found** (`fcn.00417550`, agentfight.cpp)

The real damage application is **`fcn.00417550`** (`.\AGENTS\agentfight.cpp`,
490 insns, the agentfight routine that actually calls `rand`). It applies
damage to the **target's `CStats`** — reached as **`agent+0x2c`** (the
agent's pointer to its `CStats`; the Hp it decrements is `CStats+0x04`,
the effective-block `Hp` from [`stats.md`](stats.md)). Per element:

```text
ecx = target->CStats           ; [esi+0x2c]
edx = attacker->CStats->vtable  ; [[edi+0x2c]]
dmg = (*edx[+0x1c])(...)        ; per-element DAMAGE method (polymorphic),
                                ;   fed the dice rolls (rand) + imul/idiv scaling
sub dword [ecx+4], dmg          ; target Hp -= dmg     (Hp = CStats+0x04)
```

It happens **twice in a row** (two components subtracted from the same
`Hp`). `agent+0x2c` is the **`CStats` back-pointer**, used by both the
attacker (`[edi+0x2c]`) and the target (`[esi+0x2c]`), with `Hp` at
`CStats+0x04`.

**Correction to my previous tick — `vt[+0x1c]` is *not* the damage
calculator.** Verified by re-reading: the value subtracted (`var_28h`)
is **read, never written** anywhere in `fcn.00417550`, and the
function's `rand` draws all occur **after** the subtract. So the damage
value arrives **already computed (from the caller)**; `fcn.00417550`
*applies* it. The `attacker.CStats.vtable[+0x1c]` call happens **after**
the `sub`, fed the already-determined damage and the target — i.e. it is
a **post-damage hook** (on-damaged / threat / lifesteal-style effect),
not the roll. (The later `rand%n vs 50` patterns are secondary
crit/status chances.)

**RTTI resolved (prior "dead-end" fixed).** r2's `/x` search failed, but
a direct `pefile` walk of the RTTI chain (data-scan for a method/COL
address → COL `+0x0c` → type-descriptor `+8` name) works. Results:

- The **combat methods are `CAgent` virtuals**, shared by the concrete
  agent classes **`CNpc`** (vtable `0x60982c`) and **`CPartyMember`**
  (vtable `0x6098cc`). In that agent vtable, **`fcn.00417550` (HP-apply)
  = slot `+0x24`** and **`fcn.00417b40` (melee resolver) = slot `+0x28`**
  — which is why neither has a direct `call` xref (virtual dispatch).
- The **stat-class vtables** are `CAgentStatistics` `0x61cde8`,
  **`CMonsterStatistics` `0x61ce5c`**, **`CPlayerStatistics` `0x61ce98`**.
  Their **`vtable[+0x1c]`** methods are per-class: monster `fcn.0055b390`,
  player `fcn.0055b210`. Each runs in **two stages**: (1) the opposed
  clamped-% chance roll documented below, then (2) the `imul …,0x54` table
  lookup — now **identified**: it is the **`SMagic` manager**
  (`[0x658c38]`, `.\magic\SMagic.cpp`, [`skills-magic.md`](skills-magic.md))
  record apply, `rec = [0x658c38]+0x180 + fcn.004d3fd0(agent+0x70)·0x54`
  (`fcn.004d3fd0` = the keyed lookup, tag `0xe`), which mutates the record
  field at `rec[+0xc]+0x44`. So the per-class stat method's second stage is
  the **magic-status / SMagic-record application** (the same mechanism
  stats.md indexes for the `+0x14` status slot), not an unidentified
  combat-stat formula — the earlier "complex, remaining" flag is resolved
  to an existing subsystem.

⚠️ **Correction:** the to-hit gate's `vtable[+0x50]` call is **not** on
`CStats` — it's on a **per-element object** (`[attacker+0xc][elemIdx]`),
fed a stat attribute (`[attacker+0x2c]+0x80`). So "CStats+0x50 = chance"
was imprecise; the chance method lives on the per-element channel object,
and `CStats.vtable[+0x1c]` (`fcn.0055b210`/`fcn.0055b390`) is the
per-class stat method invoked from the HP-apply path.

### The `CStats.vtable[+0x1c]` opposed success-chance check (real numbers)

Reading the bodies — player `fcn.0055b210`, monster `fcn.0055b390`
(same shape, different base constant) — this method is **not** a damage
value; it computes a **clamped percentage chance** from an opposed
comparison of two stat blocks (`this` vs the `arg` stat object), then a
`rand` roll decides success:

```text
chance% = clamp(
            BASE                         ; player 60, monster 50
          + g(this.+0x14)               ; an attribute (fild this.+0x14, via a vtable[+4] getter)
          + (argVal - other.+0x1c) / 2  ; opposed stat difference, halved (sar 1)
          - other.+0x18
          + timeOfDayMod                ; fcn.004a6fd0 → CClock [0x658c1c] → GetHour (fcn.0050bfe0)
          - 10 ,
          20, 95 )                       ; clamp consts 0x64a46c=20 / 0x64a470=95
success = rand()·1e-4 (·0.01 …) < chance ; rand scale 9.999e-5 @0x6105a0
```

So the engine uses a **base±stat-difference, time-of-day-modified,
20–95%-clamped opposed roll**. The base differs by actor class (player
`60` @`0x64a3cc`, monster `50` @`0x64a460`); the floor/ceiling are a
hard **20%/95%**. The day/night term means this check (a to-hit /
perception-style opposed check — exact semantic of `+0x14`/`+0x18`/
`+0x1c` as Sight/Level/etc. not yet pinned) varies with the world clock.
This is the first combat **formula with concrete constants** recovered.

**Net, verified:** to-hit = `attacker.vt[+0x50](attr) ≤ defense/5`
(`FUN_00417b40`); HP-apply = `sub [target.CStats+0x04], dmg` ×2
(`fcn.00417550`), with `dmg` computed upstream. **Still open:** the
upstream damage *value* computation (the caller of `fcn.00417550`, not
cleanly xref'd) and the per-class `+0x1c`/`+0x50` method bodies.

### HP-apply internals — the "×2" is two components, armor-branched

Reading `fcn.00417550`'s body resolves the structure of the "×2 sub"
(without changing the still-open *value* question):

- The two subtractions are **two distinct damage components**, not the
  same value applied twice: the primary `dmg` (the incoming parameter,
  stack `var_28h` — re-confirming it arrives **already computed from the
  caller**) and a secondary `var_1ch_2` that **defaults to `-1`** (no
  second component) unless set.
- Each component is applied via one of **two branches keyed on the
  target's defense/armor field** (`[esi+0x28]` / the per-component
  `var_1ch`/`var_10h` flags):
  - **unarmored** (defense == 0): a plain `sub [target.CStats+0x04], dmg`.
  - **armored** (defense != 0): instead of subtracting raw, it calls the
    attacker CStats virtual — **`vtable[+0x1c]`** for the primary,
    **`vtable[+0x18]`** for the secondary — as `(target.CStats, dmg, …)`.
    `vtable[+0x1c]` is the same per-class opposed-resolution method
    documented above (`fcn.0055b390` monster / `fcn.0055b210` player), so
    the armored path **routes damage through the opposed
    base±stat-difference, clamped roll** (i.e. armor/resist mitigation)
    rather than a flat hit. `vtable[+0x18]` is its sibling for the second
    channel.
- ⚠️ The helper `fcn.00438cc0` → `fcn.00438b20`, called conditionally at
  `0x4175f6` (gated on the global `[0x65b3d0]`), is a **container/list
  build** (`malloc(0x40 + n*4)`, pointer fixups at `+0xc`), **not** a
  damage computation — flagged so it is not misread as the dmg value.

So the *mitigation* side is now structural: two channels, each
plain-subtract when the target is unarmored or routed through the
`+0x1c`/`+0x18` opposed-resolution virtual when armored. The raw
*pre-mitigation* `dmg` value is still the open piece (it enters
`fcn.00417550` as a parameter).

### The pre-mitigation base is known static data (weapon dice)

The unknown is only the *runtime roll arithmetic*, not the *inputs*: a
weapon's base damage is the **`numdice` / `dicetype` / `diceadd`** dice from
its `CItemStatistic`, which are **concrete per-weapon values in the shipped
`itemstat\itemstat.txt`** ([`items.md`](items.md)) — e.g. `Knife` is
`numdice 1  dicetype 9  diceadd 3` = **1d9+3**, `Dagger`/etc. likewise. So
the pre-mitigation `dmg` fed to `fcn.00417550` is
`roll = Σ_{numdice} rand(1..dicetype) + diceadd`, plus the attacker's
offense/strength contribution, computed in the rand-driven fight
resolution. **What remains genuinely open is just that combination
expression** (the exact rand form and how the offense/strength term is
folded in) — which runs at attack time and the constraint bars from dynamic
tracing; the *dice inputs* are fully recoverable static data, so a
reimplementation can roll `NdT+A` from `itemstat.txt` and tune the bonus
term. This narrows the "damage value" gap from "unknown" to "known dice
base + an un-pinned runtime bonus/roll fold".

**The *additive components* are also recovered from `props.000` (this
pass).** Beyond the weapon dice, the other melee-damage terms are
data-driven per-rank curves, each a **base + uniform-random** pair (the
engine's standard `X` / `Xrandom` damage shape, [skills-magic.md](skills-magic.md)):
- **Weapon-mastery extra damage** — `axeextradamage` / `hammerextradamage` /
  `crossbowextradamage` / `spearextradamage`, each with its
  `…extrarandomdamage` twin: the Specialist weapon-skill bonus is
  `<weapon>extradamage[rank] + rand(0..<weapon>extrarandomdamage[rank])`.
- **Accuracy boost** — `accuracydamageboost` + `accuracyrandomdamageboost`.
- **Per-weapon to-hit** — `swordchancetohit` / `macechancetohit` (feed the
  `≤ defense/5` gate above); **Deathblow** — `deathblowchance` /
  `hammerdeathblowchance` (the on-kill proc).
So melee damage = `weapon_dice + Σ(mastery/accuracy props base+random) +
offense/strength term`, with **every component now identified** (itemstat
dice or named `props.000` curves) — only the runtime *summation/fold* is the
anonymous (load-then-call) residual. A reimplementation has all the inputs;
just the exact accumulation order is left to tune.

**On-hit special-damage procs — resolver fully pinned (`fcn.004c6500`).**
The dispatch is `lea eax,[esi-0x19]; cmp eax,0x3e; ja default; movzx
eax,byte[eax+0x4c6ca8]; jmp [eax*4+0x4c6c70]` — a **63-entry byte-index
map at `0x4c6ca8`** feeding a **14-pointer handler table at `0x4c6c70`**,
i.e. **13 distinct handlers + default** (correcting the earlier "63-case /
30 handlers" — the channel-id range `0x19..0x57` indexes through the byte
map). The 13 handlers:

| id | channel | props curve | body |
|---|---|---|---|
| 0x19 | Fire | `FireDamageChance` (`[0x6dfe10]`) | `rand()%100 < curve[val/20]` → spawn eid `0x44` |
| 0x1a | Lightning | `LightningDamageChance` | same %-roll → eid `0x44` |
| 0x1b | Poison | `PoisonDamageChance` | same %-roll → eid `0x44` |
| 0x1c | Spiritual | *(none — caller rolls)* | unconditional spawn eid `0x44` |
| 0x1d | — | — | eid `0x48` |
| 0x1e | generic | — | eid `0x44` |
| 0x2a | — | — | partial desc → shared tail, eid `0x44` |
| 0x2b | Poisonousbody | — | `rand()%3+1`, `val²`, eid `0x4c` |
| 0x3b | Stun | — | eid `0x44` (direct `fcn.004df5d0` @`0x4c6924`) |
| 0x41 | — | — | eid `0x48` |
| 0x47 | Deathblow (WarriorTactics) | `DeathblowChance` + `DeathblowSpiritualResistance` | double-gate → eid `0x44` (below) |
| 0x56 | HP leech | — | attacker deals dmg, `add [attacker.CStats+4], dealt` (clamp max) — **no factory** |
| 0x57 | Mana leech | — | same on the mana fields — no factory |

So only the four named `*DamageChance` channels (Fire/Lightning/Poison +
Deathblow) roll a props chance internally; the rest either spawn
unconditionally (the caller/fold already rolled) or transfer HP/mana
directly (0x56/0x57, no effect). Chance roll is uniformly `rand()%100`,
level index uniformly `val/20`. **WarriorTactics (case 0x47)** is the
Deathblow instant-kill: proc iff `rand()%100 < DeathblowChance[rank]`
**and** the target's spiritual resistance (`target.CStats[+0x2c]→[+0x34]`)
`< DeathblowSpiritualResistance[rank]`, then spawns the kill effect.

**The procs spawn effects — combat ties into the effect pipeline.** When a
channel fires, the case **creates the corresponding `CMagic` effect via the
effect factory `fcn.004df5d0`** on the **SMagic manager singleton
`[0x658c38]`** (`call fcn.004df5d0` at `0x4c658d`/`0x4c6924`, then zeroes the
new effect's `+0x24`). So an on-hit elemental proc is not a bare HP
subtraction — a successful `FireDamageChance` roll spawns the Fire damage
effect, which then runs through the **effect-id → apply-body** pipeline
([`skills-magic.md`](skills-magic.md), `fcn.004e27b0`). This unifies the two
systems: the slot-12 damage fold rolls the channel, `fcn.004c6500` gates it
on the props curve, and `fcn.004df5d0` instantiates the effect that applies
it — the same factory the spell-cast path uses. **Effect-descriptor
constants** (the 6-word desc each spawning case builds): `[+0]` = a
**type/category tag** (`0x44`='D' Damage, `0x48`='H', `0x4c`='L' — *not* a
size), `[+4]` = the effect-id (the factory's 83-case switch key), `[+8]`/
`[+0xc]` = target/attacker ids, `[+0x10]` = a **flags byte** (`0xa`; the
factory tests bit `0x8` — *not* a size), `[+0x14]` = the magnitude
(clamped `[0,99]`).

**Convergence point (verified).** Each channel case, once its
`rand()%100 < <Channel>DamageChance[level]` roll passes, builds a small
on-hit effect **descriptor on the stack** (a shared base constant
`[desc+0] = 0x44` and a `0xa` parameter, the effect-id at `[desc+0x14]`) and
**`jmp`s to the shared tail `0x4c6587`** — `mov ecx,[0x658c38]; call
fcn.004df5d0` — so Fire (case 25), Lightning, Poison and the generic
effect case (30) all spawn through that single factory call. The roll itself
uses the same `÷20` level index as the slot-12 fold (`rand()%100` then
`idiv 100`, curve indexed by `idx = stat/20` via `0x66666667;sar3`), so the
on-hit proc and the damage fold share both the level-index arithmetic and
the effect-spawn tail. It is called from the `CStats`
combat-resolution functions `fcn.0055a970` and `fcn.0055ca10` — which, by
RTTI naming, "`virtual_12`" — but **the slot number was wrong: dumping the
vtables (`0x61ce98` player / `0x61ce5c` monster) places the fold at slot 3
(`+0x0c`)**, with separate monster (`fcn.0055a970`, 1243 B; twin at slot 2
`0x55d610`) and player (`fcn.0055ca10`, 3065 B — richer) implementations. This *locates the
"anonymous fold"*: it is not unreachable, it is a per-stat-class virtual whose
**bodies are static and readable**; only the dispatch (which of the two
classes) is virtual, and both are enumerable. The slot-12 body folds the
integer stat-field terms (attacker/target `+0x20`), a `rand()` random
component, and **float scaling with the tuning constants `50.0` / `20.0` /
`95.0`** (`[0x64a460]`/`[0x64a46c]`/`[0x64a470]`) plus a `×0.01` percent factor
(`[0x609508]`), then rolls the on-hit procs via `fcn.004c6500`. So the on-hit
elemental/deathblow specials are fully data-driven percentage rolls, and the
base-damage fold is now pinned to the slot-3 virtuals — the residual shrinks
to the exact FP accumulation order *within* those two readable bodies, not an
unreachable dispatch.

**The FP fold IS reducible from readable code (correction).** An earlier
note here claimed the operands route through anonymous sub-dispatch and so
the FP order couldn't be closed — that was **wrong, and is corrected**. The
entry does `call edx` twice (`0x55ca42`/`0x55ca4f`, slot-1 `[obj_vtable+4]`
on the attacker `this` and the `arg_48h` stat object), but **both call
results are immediately discarded** — `eax` is overwritten right after each
(`mov eax,[ebx]` at `0x55ca48`, `mov eax,[edi+0x1c]` at `0x55ca54`). So those
calls are **stat-refresh side-effects** (recompute the object's cached
fields), *not* operand-getters. The damage operands are then **direct field
reads**: `fild [edi+0x14]`, `[edi+0x1c] - [ebx+0x1c]` (the attacker-vs-target
stat difference), plus the float tuning consts `[0x64a3cc]`/`[0x64a46c]`/
`[0x64a470]` (50/20/95) and the props/procs already enumerated. So
`damage = f(operands)` **is** statically closable — every operand is a
readable field read or a named props value; the only work left is the
tedious manual reduction of the readable FP accumulation order, **not** an
anonymous-dispatch wall. (This narrows the combat residual: the fold is
readable end-to-end; it just hasn't been hand-reduced to a closed
expression.)

**Recovered damage-fold structure (`fcn.0055ca10`, operands + constants).**
Tracing the readable FP spine, the player fold's operands are all direct
reads on the attacker (`edi`) and target (`ebx`) stat objects, combined with
fixed tuning constants:
- **Stat operands** — attacker `+0x14` (base term), `+0x1c` (offense),
  `+0x6c`, `+0x80` (= **Strength**, RPG-attr block), `+0xa0` (a skill/weapon
  index); target `+0x18`, `+0x1c` (defense terms). The `+0x1c` term is an
  **attacker−target difference** (`[edi+0x1c] − [ebx+0x1c]`).
- **Tuning constants** — `60.0` (`[0x64a3cc]`), `20.0` (`[0x64a46c]`), `95.0`
  (`[0x64a470]`), `50.0` (`[0x64a460]`), the percent factor `×0.01`
  (`[0x609508]`), and `×0.0001` (`[0x6105a0]`).
- **Mastery term** — `[0x65429c + attacker[+0xa0]*4]` (a float weapon/skill
  mastery table indexed by the attacker's skill id) is `fimul`'d by attacker
  `+0x80` (**Strength**) — i.e. `mastery[skill] × Strength`.
- **Random + procs** — a `rand()` spread, then the on-hit proc rolls
  (`fcn.004c6500`, above).
So the backbone is roughly `base(attacker+0x14, +60, −target+0x18, +attacker+0x6c)`
→ conditional `20/95` scaling `×0.01` → `+ base×0.0001`
→ `+ mastery[skill]×Strength` → `+ rand` → `+ props/procs`. Every operand is
a named field or constant; the exact closed-form term composition is the
remaining manual-reduction detail (not a dispatch gap).

**Per-element term arithmetic backbone (monster body, `fcn.0055a970`).**
Reading one element block of the per-channel loop, the integer term has a
concrete recovered shape:

```text
a = fcn.005438c0([0x7467fc], 0x20)     ; registry lookup by id (per-element id)
b = fcn.005438c0([0x7467fc], 0x51)     ; second lookup
idx = a[+0x20] / 20                    ; 0x66666667; sar edx,3  (=÷20, verified)
val = table[0x748a78][idx]             ; a per-level curve indexed by idx
val = (val × b[+0x20]) / 100           ; 0x51eb851f; sar edx,5  (=÷100, verified)
fcn.004c6500(this=0x6dfe18, …, [edi+0x70], 0x3b, val, …)   ; the on-hit proc resolver
```

The two divisors are unambiguous from the standard magic-number sequences
(`2^35 / 0x66666667 ≈ 20`, `2^37 / 0x51eb851f ≈ 100`).

**The registry and its ids — RESOLVED ✅ (the "dynamic-bound" verdict
was wrong).** `[0x7467fc]` is the **skills-manager singleton**
(`.\Skills\skills.cpp`; sole construction write `0x49a631` in
`compilestart.cpp` — `new(0xf60)` → ctor `fcn.00543540` which also
freads `dat\wgiaa.000`, [formats/skillcosts.md](formats/skillcosts.md)),
and `fcn.005438c0` is `CSkillManager::FindSkillById`. The entries are
**`CSkill` objects (0x78 B)**: `+0x20` = charge−20 (so `/20` = rank
index 0..4), **`+0x30` = the 1-based skill id** (= 1 + index into the
hardcoded 96-entry name table at `0x6529b8`; assigned in
`AddSkillByName` `fcn.00541a70`, the sole vector builder — all 96
`CSkill` subclasses are pre-constructed by static initializers into
`g_skills[96]` at `0x746800`), `+0x34` = learned flag (find-by-id
returns NULL unless set), `+0x4c` = charge 0–100. So the "per-channel
ids" are **skill ids, fully static**: `0x20` = Summonmimic, `0x51` =
Stun, `0x55` = FireDamage, `0x41–0x45` = the Sword/Mace/Axe/Hammer/
Spear masteries, `0x48` Shield, `0x4a` Accuracy, `0x52/0x53`
Increased-Damage/Defense, `0x54/0x56` Poison-/Lightning-Damage, `0x5f`
WarriorTactics, `0x60` SpiritualDamage, `0x37` Wisdom (the XP path).
`0x3b` is **not** a registry id — it is the Stun *case constant* for
`fcn.004c6500` (skill→proc-case pairs: Stun→0x3b, Fire→0x19,
Lightning→0x1a, Poison→0x1b, Spiritual→0x1c, Poisonousbody→0x2b,
WarriorTactics→0x47).

**And the curve table:** `[0x748a78]` is the cached props curve
**`MimicCost`** (static initializer `0x603f80` via `fcn.00500f10`);
sibling `[0x748a7c]` = `AccuracyDamageBoost`. So the block quoted above
is really **`MimicCost[SummonmimicRank] × copiedSkill.charge / 100`** —
the monster-side loop is the **summoned Mimic's copied-skill proc pass**
(each player combat skill fires at MimicCost% strength), not a generic
elemental loop. The generic elemental procs remain the
`<Element>DamageChance` rolls in `fcn.004c6500` (above), now keyed by
the skill ids listed here.

**Base term reduced (verified instruction-by-instruction).** The opening of
the fold computes, with the integer side `eax = ([edi+0x1c] − [ebx+0x1c])`
then `cdq; sub eax,edx; sar eax,1` (a signed **÷2**) and the FP side
`fild [edi+0x14]; fadd 60; fisub [ebx+0x18]; fiadd eax; fiadd [edi+0x6c]`:

> **base = attacker[+0x14] + 60 − target[+0x18] + (attacker.Level −
> target.Level)/2 + attacker[+0x6c]**

where `+0x1c` is the **`CStats` Level** ([`stats.md`](stats.md)), so the
`(attacker.Level − target.Level)/2` is an explicit **level-difference** term.

**This `base` is the TO-HIT CHANCE, not damage magnitude (refines the
framing).** Immediately after, the fold **clamps `base` to `[20, 95]`** —
`fcom 20` forces a minimum of 20 (`if base<20 → 20`), `fcom 95` forces a
maximum of 95 (`if base>95 → 95`) — then `fmul ×0.01` converts it to a
fraction. A value clamped to `[20,95]` and scaled by `1/100` is a
**percentage in `[0.20, 0.95]`**: i.e. `fcn.0055ca10`'s leading computation
is the **to-hit probability**, `hit% = clamp(base, 20, 95)`, rolled against
the subsequent `rand()`. So the engine guarantees a **20% floor / 95% ceiling
to-hit**, modulated by the level difference and the stat terms. The **damage
magnitude** is then the *separate* `mastery[skill]×Strength` + props/procs
terms (the `base×0.0001` and the proc rolls), not this clamped expression.
The `cdq` confirms the two opening `call edx` results are discarded (`edx`
is the sign-extension, not a call return) — the stat-refresh-only finding.

**Monster body (`fcn.0055a970`) — same to-hit mechanism, thinner base
(verified order).** The monster slot-12 mirrors the player's to-hit fold but
takes its base as an **argument** rather than composing it from stat reads.
The exact accumulation order, instruction-by-instruction:

```text
fild [ebp+0x14]          ; base value (passed in as an arg)
fadd [0x64a460]          ; + 50.0  (monster base; player uses 60.0 @0x64a3cc)
; clamp to [20,95] via the same two-step compare as the player body:
  max(_, [0x64a46c]=20.0)
  min(_, [0x64a470]=95.0)
fmul [0x609508]          ; × 0.01   → chance fraction in [0.20, 0.95]
roll = (rand() % ecx) × [0x6105a0](≈9.999e-5)   ; the 0.01%-granularity roll
hit  = roll < chance
```

So both stat classes share the **`clamp(base,20,95) × 0.01` vs
`rand()·1e-4`** to-hit, differing only in the base (player: the composed
`+0x14 + 60 − target+0x18 + leveldiff/2 + +0x6c`; monster: `arg + 50`). After
the gate the monster body runs the **per-element damage loop**: for each of
the ~6 channels it scales by the attacker stat field `[esi+0x20]`
(`imul ecx,[esi+0x20]`) and rolls the on-hit proc via `fcn.004c6500`. This
closes the monster side's accumulation order (the player side is reduced
above); the remaining open detail is only the exact integer composition of
each per-element damage term.

**To-hit roll mechanism (closes the to-hit side).** The chance is rolled at
**0.01% granularity**: `roll = (rand() % 10000) × 0.0001` (a `[0,1)`
fraction), compared (`fcompp`) against the clamped `hit_chance =
clamp(base,20,95) × 0.01`. So a swing **hits iff `(rand() % 10000) <
clamp(base,20,95) × 100`** — equivalently `roll% < clamp(base,20,95)` with
hundredth-percent resolution. On a hit the fold proceeds to the damage
magnitude; on a miss it skips it. So the full to-hit side is closed:
`base` (the level/stat expression above) → clamp `[20,95]` → compare vs a
`rand()%10000` roll.

**Damage-magnitude core recovered: `mastery_fraction[weapon] × Strength`.**
After the hit-chance clamp, the fold computes the magnitude as
`fld [0x65429c + attacker[+0xa0]*4]` (a per-weapon **mastery fraction** from
the float table at `0x65429c`) `fimul [edi+0x80]` (× attacker **Strength**).
The `[0x65429c]` table holds **exactly three** fractions — `[0]=0.40`,
`[1]=0.10`, `[2]=0.142857 (1/7)` (the following words are integers, a
different table) — so `+0xa0` is the **3-valued character class** (Divine
Divinity's Warrior / Survivor / Mage), not a per-weapon index. So **base
Strength-damage = class_fraction[+0xa0] × Strength** — `0.40×Str`, `0.10×Str`,
or `(1/7)×Str` by class (the highest `0.40` is the Strength/melee class, the
lowest `0.10` the caster, by gameplay sense; exact index→class names not
asserted). I.e.
then modulated by the **player singleton `[0x658c04]`** (`.\GAME\Player.cpp`,
built by `fcn.004a90e0`) via a player flag at `player+0x88`, and a weapon-type
field `[edi+0x70]` (compared vs 4), and finally combined with the `props`-driven
extras (weapon-mastery `extradamage`, accuracy, `IncreasedDamage`, the
on-hit elemental/deathblow procs). So the full magnitude is
`mastery_fraction[weapon]×Strength + Σ(props base+random) + procs` — every
factor now a named table entry, field, or props curve.

**Damage assembly branches by weapon-type to fetch operands (structure
pinned; corrects an over-label).** After `mastery_fraction[class]×Strength`
is computed and rounded (`fcn.005e5d40`), the fold reads the **weapon-type
field `[edi+0x70]`** and branches (`cmp edx,1`, `cmp edx,4`, …). The
functions those branches call are **field getters / lookups, not
damage-finishing handlers** (correcting an earlier "type-specific damage
handler" label): `fcn.005172d0` is a generic id→index list-search (24-byte
entries, 13 callers), `fcn.0055a4c0` reads an array field `[obj+idx*4+0xf0]`,
`fcn.005177d0` resolves a handle via the agent manager `[0x658d50]`, and
`fcn.004a6a60` reads a field `[obj+0x4b4]`. So the weapon-type branch
**selects which extra operands to fetch** (per-type resist/ammo/handle
lookups), and the damage *composition itself stays in the readable fold*
using those operand values + the props extras + on-hit procs — it is **not**
deferred into opaque per-type math functions. So the magnitude is readable
end-to-end (core `class-fraction×Strength`, weapon-type-selected operand
getters, props, procs); only the exact closed-form arithmetic order is left
to hand-reduce.

**Player slot-12 (`fcn.0055ca10`) skill-augment terms — enumerated.** Reading
the player damage virtual's prop names confirms the fold is **entirely
data-driven** (every augment is a named `props.000` curve, same base+random /
chance shapes as elsewhere):
- **Increased Damage** (warrior skill) — `IncreasedDamageDamage[rank] +
  rand(0..IncreasedDamageRandomDamage[rank])` (the canonical base+random pair,
  read consecutively).
- **Shield Bash** — `ShieldBashChance[rank]` (proc %) + `ShieldBashDamage[rank]`.
- **Poison Damage** — `PoisonDamageDamage[rank]` + `PoisonDamageSpeed[rank]`
  (the DoT magnitude + tick rate).
Together with the weapon dice (itemstat), the mastery/accuracy curves above,
and the on-hit procs (`fcn.004c6500`), **every input to the player melee-damage
formula is now an identified `props.000`/itemstat value** — the slot-12 body
just sums them (the only un-reduced piece being the FP accumulation order).

**Why the fold can't be statically located (structural, verified twice
over).** `fcn.00417550` is confirmed the agent vtable's slot `+0x24`
(`vtable 0x60982c`: `[0x60982c+0x24]=[0x609850]=fcn.00417550`, with the
melee resolver `fcn.00417b40` in the next slot `+0x28`). It has **zero
direct callers**, and — verified by opcode search — **the binary contains
zero `call dword [reg+0x24]` instructions** (and none for the other
registers): this build dispatches *every* virtual as **load-then-call**
(`mov reg,[vtbl+0x24]; call reg`), so HP-apply's invocation site is an
anonymous indirect `call reg` indistinguishable from every other virtual
call — neither an xref nor an opcode search can isolate it. The dmg-compute
sits just above that anonymous indirect call. That is the *structural*
reason the fold resists static RE (not a search-not-yet-done): pinning it
requires dynamic tracing of a live attack, which the constraint bars. The
inputs (dice + the documented offense/defense stats) and the apply
(`sub [CStats+4], dmg`, armor-branched) are both fully mapped; only the
runtime edge between them is dynamic.

*(Superseded: earlier drafts called `fcn.004edef0` / the attractor
system the damage path — both wrong; `fcn.004edef0` is hit animation,
`0x659210` is the AI attractor.)*

**RNG** (whichever routine does the eventual roll): MSVCRT `rand` (thunk
`div.exe:0x5e5dec`) wrapped by a `random.cpp` helper cluster
(`0x0040f550`–`0x00411250`); the bounded primitive masks
`rand() & 0x8000000f` (0..15, `+1` → 1..16), chained for larger ranges.

**To-hit virtual** (refined): the `+0x50` call in the gate is on a
**per-element channel object** (`ecx = [ [attacker+0xc] + elemIdx*4 ]`),
so it is *polymorphic* — `vtable[0x50]` differs per damage channel and
isn't a single static address; it takes an attribute value
(`[attacker+0x2c]+0x80`, the Strength-block base) and returns the chance
compared against `defenseField/5`.

**Decompiled one concrete `+0x50` body (confirms dynamic-only).** The
`CMonsterStatistics` `+0x50` method is **`fcn.0055aee0`**: it indexes an
**84-byte (`0x54`) record** (`imul 0x54` + `[ctx+0x180]`) and runs a chain
of **`rand()` calls, each `idiv`'d and the remainder compared** against the
stat fields `+0x28`/`+0x2c`/`+0x30` — i.e. it is **probabilistic
chance/value computation over per-record stat fields, not a closed-form
`XdY+Z` dice expression**. So the "damage/chance" the combat gate reads is
produced by per-subclass `rand`-driven methods that only yield a concrete
number at runtime; this is the fourth independent angle (after the
`vtable+0x50` slot-inconsistency, the `Experience`/`Level` offset reuse,
and the `[stat+0x2c]+0x80` feed) confirming the exact arithmetic is a
**dynamic-only** result, not statically reducible. The *inputs*
(weapon `numdice`/`dicetype`/`diceadd` from [`items.md`](items.md), the
Strength-block attribute, `defense/5` gate) and the *RNG* (`rand()&…`
masking) are all pinned; only the closed-form combination is dynamic.

## The damage fold — closed form ✅ (RESOLVED)

The player fold `fcn.0055ca10` is fully reduced. Signature:
`__thiscall int fold(CStats* target, int unused, int forceHit)`
(`ret 0xc`; arg2 is overwritten before any read at `0x55ca68`, arg3 read
at `0x55cad4`). **The fold computes AND applies the damage itself**
(`sub [targetStats+4], dmg` at `0x55d5eb`) and returns 1 = landed /
0 = miss/absorbed/<1 — so `fcn.00417550`'s direct `sub` is only the
no-attacker fallback path; when an attacker exists it dispatches the
fold via `mov eax,[attackerStats]; mov eax,[eax+0xc]; call eax`
(`0x417640→0x417689`) — a concrete static load-then-call site.
Constants (hex-dumped): `[0x64a3cc]`=60.0, `[0x64a460]`=50.0,
`[0x64a46c]`=20.0, `[0x64a470]`=95.0, `[0x609508]`=0.01,
`[0x6105a0]`=9.9999e-5, `[0x65429c]`={0.40, 0.10, 1/7};
`fcn.005e5d40` = `_ftol2_sse` → every rounding below is **truncation
toward zero**.

**TO-HIT** (`0x55ca51–0x55cb00`):

```text
base   = Offense(+0x14) + 60 − target.Defense(+0x18)
         + trunc((Level − target.Level)/2)          ; cdq;sub;sar 1
         + WeaponChanceToHit(+0x6c)
chance = clamp(base, 20, 95) × 0.01
HIT    ⇔ forceHit ≠ 0  OR  (rand()%10000) × 9.9999e-5 ≤ chance
```

`CStats+0x6c` is hereby identified (it was "unaccounted" in stats.md):
the **weapon-mastery chance-to-hit cache**, written by `fcn.0042b540`
from the equipment fold — Sword-class weapon →
`SwordChanceToHit[rank(skill 0x41)]`, Mace → `MaceChanceToHit[rank
(skill 0x42)]`, every other class → 0 (only those two curves exist).
The monster bodies compose the **same base with 50.0 and no `+0x6c`
term** — correcting the earlier "monster base = arg + 50" reading:
`fild [ebp+0x14]` there is `this->Offense` (ebp=ecx), not an argument.

**DAMAGE** (evaluation order; `dmg` lives in `[esp+0x4c]`):

1. Core: `dmg = trunc(classFrac[+0xa0] × Strength(+0x80))`,
   classFrac = {Warrior 0.40, Wizard 0.10, Survivor 1/7} `@0x65429c`.
2. Roster gate (`0x55cb1d`): `idx = find(CStats+0x70)` in the
   controlled-actor roster `[0x658c04]+0x4b4` (24-byte entries,
   `fcn.005172d0`); miss or `flags&0x38` → skip to step 10. idx 0 =
   hero; idx>0 = the Mimic copy (MimicCost scaling below). *(This is
   the loop earlier misread as "per-element".)*
3. Equipment dice, all 11 slots (`item = [CStats + i·4 + 0xf0]`),
   active iff `fcn.0055a4c0`: attr requirements `+0x80/84/88/90 ≥
   CItem+0x64/68/6c/70`, durability `CItem+0x1c > 0`, shield slot dead
   if the weapon is two-handed (`CItemStatistic+0x88` byte). Per active
   item: 2 % durability loss (`rand()%50<1`, suppressed by agent
   `+0x224 & 0x20`); `dmg += Σ_{k=1..numdice}(rand()%dicetype + 1)`;
   `dmg += diceadd`; then **keyword procs** — both special vectors
   `CItem+0xac/+0xb0` and `+0x94/+0x98` (12-byte `{obj,val}` elems),
   keyword → `id = fcn.004b2610(name)` → `fcn.004c6500(attackerId,
   targetId, id, val, dmg, 0)`.
4. Poisonweapon (hero): skill `0x22` charge →
   `fcn.004c6500(case 0x2b, charge, PoisonDamageDamage[charge/20]×4,
   PoisonDamageSpeed[charge/20])`.
5. Skill procs (skill→case): `0x51`→0x3b Stun, `0x55`→0x19 Fire,
   `0x56`→0x1a Lightning, `0x54`→0x1b Poison, `0x5f`→0x47
   WarriorTactics, `0x60`→0x1c Spiritual; value = hero `[skill+0x20]`,
   mimic `MimicCost[mimicRank] × ([skill+0x20]/20) / 100`.
6. Shield Bash (hero, shield equipped, skill `0x48`):
   `rand()%100 < ShieldBashChance[rank]` →
   `dmg += ShieldBashDamage[rank]`.
7. Increased Damage (skill `0x52`): `dmg += IncreasedDamageDamage[rank]
   + rand()%IncreasedDamageRandomDamage[rank]` (mimic ×MimicCost%).
8. Weapon mastery `fcn.0055c250(&dmg)` (hero): the weapon's keyword
   class (`fcn.004b2610`: Sword=0x48, Mace=0x49, Axe=0x4a, Hammer=0x4b,
   Spear=0x4c, Bow=0x4d, Crossbow=0x4e, Shield=0x4f, Staff=0x53,
   Dagger=0x54) selects: Sword/Mace `dmg += dmg×<W>DamageModifier[rank]
   /100` (Mace also `+ rand()%MaceExtraRandomDamage[rank]`);
   Axe/Hammer/Spear `dmg += <W>ExtraDamage[rank] +
   rand()%<W>ExtraRandomDamage[rank]`; Bow/Crossbow/Staff/Dagger: none.
   *(This is the switch earlier misread as a "weapon-type +0x70 branch
   / player +0x88 flag" — the +0x88 is the CItemStatistic two-handed
   byte.)*
9. Dodge: second roll `rand()%10000 × 1e-4 ≤ chance` →
   `dmg −= trunc(target.Dexterity(+0x84)/10)`.
10. `if dmg < 1 → return 0`.
11. Magic damage-shield: SMagic `[0x658c38]` record type `0xe` on the
    target (`fcn.004d3fd0`) absorbs: `pool −= dmg`; pool ≥ 0 →
    **return 0**; else `dmg = −pool`.
12. `[targetStats+0x04] −= dmg; return 1`.

**Defense side / armor** (read live, not folded into stats):
`fcn.0055a7a0` rolls a hit location over the weight table
`0x6542c0 = {20,0,25,0,30,0,0,10,5,5,5}%` and adds the struck item's
`CItem+0x10` armorclass, with a 1-in-20 durability loss.

**Ranged/projectile impact** (closes the fire-time path): damage is
rolled at fire time in `fcn.00418e70` (dice via `fcn.0055a530` over the
equipment channels + Accuracy skill `0x4a`, or inline
`rand()%[agent+0x1f8]+1+[agent+0x1f4]`), posted as a clock-scheduled
shot (callback `0x415740`, param+0 = dmg) → arrow-type dispatcher
`0x55f580` (table `0x748b00[28]`) → spawner `0x564d50`
(**proj+0x10 = dmg**, +0x14 = mode, on-hit callback proj+0x3c =
`fcn.00564220`). On hit, `0x564220` calls the target's
**vt+0x28 = `FUN_00417b40`** (mode≠0) or **vt+0x24 = `fcn.00417550`**
(mode 0) with the carried damage (`0x5642f4–0x564315`) — concrete
static call sites of both "unlocatable" combat virtuals. Elemental
arrow procs go through `fcn.004c6500` → the SMagic factory.

**Both former soft spots are now resolved (PROVEN):**

`fcn.00417550(target, attacker, dmg, selector, forceHit)` picks which
attacker-`CStats` virtual applies the damage, from two orthogonal gates —
the target's armor field `[target+0x28]` (armored vs unarmored) and the
`selector` arg:

| `[target+0x28]` | `selector==0` | `selector!=0` |
|---|---|---|
| armored (`≠0`) | vt+0x0c (slot 3 = the damage **fold**, recomputes) | vt+0x1c (slot 7 = **apply-precomputed**) |
| unarmored (`==0`) | vt+0x08 (slot 2; player = a `ret 0` stub) | vt+0x18 (slot 6) |

So **`selector==0` = "recompute-and-apply"** (invoke the full fold, slot
3, ignoring the passed `dmg` — the melee path), and **`selector!=0` =
"apply this precomputed `dmg`"** via the opposed-apply method (slot 7,
`fcn.0055b210`): it re-runs the clamped `[20,95]%` opposed hit chance
(unless `forceHit`), a Dexterity dodge `dmg -= rand()%(Dex/10)+1`, the
magic damage-shield absorption (SMagic type `0xe`), then
`sub [target.CStats+4], dmg`. The projectile path calls it with
`selector=1, forceHit=0`. `attacker==null` is the plain no-attacker
`sub` fallback. (Both branches deal damage — not damage-type/heal
variants.)

The **`[agent+0x28]` branch in `fcn.00418e70`** (ranged fire-time roll)
selects the damage *source*: `[agent+0x28]==0` → `fcn.0055a530` (the
equipped-weapon dice + Accuracy roll); `≠0` → the inline **innate ranged
spec** `dmg = [agent+0x1f4] + rand(1..[agent+0x1f8])` (creature
spit/breath, base `+0x1f4` + spread `+0x1f8`). It is not a weapon-type
flag.

The **monster per-element integer term** (`fcn.0055a970` loop) is the
summoned-Mimic copied-skill pass, closed form:
`val = MimicCost[(Summonmimic.charge−20)/20] × (copiedSkill.charge−20) /
100` (`[0x748a78]` = MimicCost props curve, `[0x7467fc]` = the skills
manager; both `/20` and `/100` magic-number divisions verified) — each
copied combat skill fires at `MimicCost%` strength through
`fcn.004c6500`.

## Citations

```text
div.exe:0x00417b40   FUN_00417b40   melee-hit resolver (agentfight.cpp): stats → to-hit gate →
                                    spawn 4 per-element visual FX → pack+fire combat event.
div.exe:0x00416050   FUN_00416050   per-element VISUAL-FX spawn (forwards into fcn.004edef0).
div.exe:0x004edef0   fcn.004edef0   CAniEffect_PlayAnimation…AttachedToNpcCenter ctor — the
                                    per-hit ANIMATION effect (NOT damage; was mislabelled).
div.exe:0x004eca40   fcn.004eca40   FX center/offset helper (the +0x3c/+0x40 it reads are the
                                    effect anchor, NOT defender armour — earlier note wrong).
div.exe:0x005e5d40   fcn.005e5d40   __ftol-style double→int finaliser (cvttsd2si) — FX coords.
div.exe:0x0054a1b0   FUN_0054a1b0   per-element floating-damage/blood FX (render mgr 0x7469ec).
div.exe:0x00438cc0   FUN_00438cc0   build a 0x14-byte ATTRACTOR object (.\AGENTS\attractor.cpp),
                                    insert into the attractor system 0x659210 — NOT a damage event.
div.exe:0x00438b20   FUN_00438b20   attractor insert (slotted array).
div.exe:0x00411a70   fcn.00411a70   attractor spatial query (cell-range scan of 0x659210) — NOT
                                    the damage consumer (earlier mislabel, corrected).
div.exe:0x0040f550   random.cpp     RNG helper cluster (..0x00411250) over rand (rand()&0x8000000f).
div.exe:0x0050c6a0   FUN_0050c6a0   fire an Osiris combat event (event manager 0x7447dc).
div.exe:0x00417550   fcn.00417550   DAMAGE APPLY (agentfight.cpp): sub [target.CStats+4], dmg ×2
                                    (Hp=CStats+0x04; agent+0x2c=CStats ptr). dmg arrives from the
                                    caller; vt[+0x1c] here is a post-subtract hook, not the calc.
```

## Status

- Melee entry point ✅ — `FUN_00417b40`: stat reads, to-hit gate, then
  it **spawns four per-element visual FX and fires a combat event** (it
  does *not* roll/apply damage inline).
- **Corrections ✅ (this pass)** — (1) `fcn.004edef0`/`FUN_00416050`/
  `FUN_0054a1b0` are per-element **visual-effect** spawns (hit animation,
  floating damage/blood), *not* the damage calculator (and the `+0x3c/
  +0x40` reads are the FX anchor, not armour). (2) `FUN_00438cc0` /
  `0x659210` is the **attractor** system (`.\AGENTS\attractor.cpp`), not
  a combat-event manager; `fcn.00411a70` is its spatial query, not a
  damage consumer. Two earlier "damage calculator/consumer" claims were
  wrong and are retracted here.
- HP apply ✅ — **`fcn.00417550`** (agentfight.cpp): `sub
  [target.CStats+0x04], dmg` ×2 (Hp = `CStats+0x04`; `agent+0x2c` =
  `CStats` ptr). Solid and verified.
- Virtual dispatch ✅ — HP-apply (`fcn.00417550`) and resolver
  (`fcn.00417b40`) are **`CAgent` virtuals** (slots `+0x24`/`+0x28` in
  the `CNpc` `0x60982c` / `CPartyMember` `0x6098cc` vtables) — hence no
  direct call xref. RTTI resolved via a `pefile` COL walk (r2's `/x`
  RTTI search had failed; **dead-end now fixed**).
- Stat method addresses ✅ — `CMonsterStatistics` vtable `0x61ce5c` /
  `CPlayerStatistics` `0x61ce98`; their `vtable[+0x1c]` = `fcn.0055b390`
  (monster) / `fcn.0055b210` (player).
- `CStats.vtable[+0x1c]` formula ✅ — it's an **opposed success-chance
  check**, *not* damage: `clamp(BASE + g(+0x14) + (arg − other.+0x1c)/2
  − other.+0x18 + timeOfDayMod − 10, 20, 95)` then a `rand` roll. BASE =
  60 (player) / 50 (monster); clamp **[20, 95]%**; time-of-day via
  `CClock` `GetHour`. First combat formula with concrete constants.
- Damage values are **caller-supplied args** ✅ — re-verified:
  `fcn.00417550` reads the two subtracted values from `[esp+0x28]`/
  `[esp+0x2c]` with **no prior write** in the function, so the caller
  computes the damage and this virtual only applies it.
- Finding the caller 🟡 (method dead-ended) — `fcn.00417550` is virtual
  (slot `+0x24`); a byte-scan for `call [reg+0x24]` returns **0** in the
  whole binary, so it's invoked via the register-indirect form
  (`mov r,[vt+0x24]; call r`), which a simple pattern can't isolate.
  Needs a smarter scan (track `mov r,[vt+0x24]`→`call r`) — open.
- `0x658c38` manager identified ✅ — it's the **SMagic manager**
  (`.\magic\SMagic.cpp`), a 340-byte (`0x154`) singleton built in
  `.\GAME\compilestart.cpp` (`fcn.004cd4b0`), holding an **84-byte
  (`0x54`) record table at `+0x180`** indexed by a per-object id
  (`+0x70`). So the `CMonsterStatistics vtable[+0x50]` method
  (`fcn.0055aee0`) **consults the magic table** — i.e. `+0x50` is
  **magic-related**, not the plain melee to-hit chance (a further reason
  the earlier "+0x50 = to-hit" guess was loose). The melee to-hit gate
  proper is still the `defense/5` comparison in `FUN_00417b40`.
  **`fcn.0055aee0` body (read in full)**: it takes the opponent stats as
  `arg`, reads both parties' magic id at `+0x70`, looks the opponent's up
  in `SMagic` via `fcn.004d3fd0(SMagic, type=0xf, id, 0)` → a record index
  (`-1` = none), resolves the `0x54`-byte record (`idx*0x54 + [SMagic+0x180]`,
  sub-object at `record+0xc`), and computes a magnitude into `record+0x44`
  through `fcn.004e2750([ebx+0x70], [edi+0x70], record)` — i.e. an opposed
  **magic-attack-vs-resist** modifier keyed by both sides' magic ids, then
  sign-normalised. So `+0x50` is the **magic channel's** opposed
  computation, fully analogous to (but separate from) the melee `+0x1c`
  opposed-chance check. This **resolves** the parked "+0x50 per-class
  body" question for `CMonsterStatistics`.
- To-hit gate ✅ (structure) — a **per-element channel object**'s
  `vtable[+0x50]` chance method (fed an attribute value from the agent's
  Strength block at `+0x2c→+0x80`) compared `≤ defenseField[+0xc]/5`
  (`imul 0x66666667; sar 3; ×2`). The `+0x50` method is **polymorphic
  per channel** (no single static address) — approximable as
  `offenseChance ≤ defense/5`.
- RNG ✅ — MSVCRT `rand` (`0x5e5dec`) via `random.cpp`
  (`0x40f550`–`0x411250`); `rand() & 0x8000000f` (0..15, `+1` → 1..16).
- Ranged/spell damage ❓ — projectile and magic damage paths
  ([`projectiles.md`](projectiles.md), [`skills-magic.md`](skills-magic.md))
  not yet tied in.
