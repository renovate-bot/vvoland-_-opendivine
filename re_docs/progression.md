# Experience & leveling

How the hero gains XP and levels up. The per-skill-level tuning curves
are in [`formats/props.md`](formats/props.md); this doc covers the
**XP-award path** and the level mechanism.

## XP award (`fcn.0042aaf0`)

On a kill, the award function computes the reward and grants it:

1. **Base reward** — each agent/monster (egg) carries an XP-reward
   value; the debug totals (`Total experience generated eggs`,
   `Total experience manual eggs`, `Total experience story`) sum these
   by source, and `Accumulated experience` is the running total.
2. **Level-difference scaling** (`0x42ab82..0x42abc2`) —
   `XP′ = XP ± trunc(XP·0.5·Δ²/144)` where `Δ = monsterLevel −
   playerLevel` (FP constants `144.0` @ `0x609970`, `0.5` @ `0x6094f0`);
   `+` when the monster is at/above the player's level, `−` otherwise;
   a negative result means **no award**. I.e. `XP·(1 ± Δ²/288)`.
3. **Anti-grind mechanic** — the awarder tracks a **per-monster-type
   kill counter at `+0x108`** against **`0x46` = 70**
   (`cmp dword [eax+0x108], 0x46`), with a flag at `+0x10c`.
   *(Corrected: not a cutoff.)* Past 70 kills of a type, each further
   kill has a **1 % chance (`rand()%100`) of a one-time ×10 XP jackpot**
   (`reward·10`, sets the `+0x10c` done-flag; floating-text key 115
   instead of the normal 117).
4. **Wisdom boost** — the Wisdom skill is looked up on the skills
   manager `[0x7467fc]` by the find-by-id `fcn.005438c0` with **id
   `0x37` = Wisdom** (`0x42ac96`); if learned, `fcn.00544e30` applies
   `XP += XP·WisdomExperienceBoostList[rank]/100` with
   `rank = [skill+0x20]/20` (byte-exact confirmed). *(Corrected: the
   `fcn.005438c0` call in this path finds the Wisdom **skill** on the
   skills registry, not a party member — see [combat.md](combat.md)
   for the registry identity.)*
5. **Grant + level-up** — the boosted share is added by
   **`fcn.0055ae50` = `CStats::AddExperience(delta)`** (its only static
   caller is this awarder, `0x42ad04`): `add [ecx+0x94], delta` then
   the closed-form level computation (below), returning the new level.
   `fcn.0042aaf0` then runs the level-up block inline (grants, refill,
   FX — below).

The script command **`set experience #`** ([`npc-ai.md`](npc-ai.md) /
script vocabulary) sets a character's XP directly (quest rewards,
debug).

**Party distribution.** `fcn.0042aaf0` is actually the *party* awarder:
it loops the party, finds each member by id via the list-search
`fcn.005438c0` and adds the (Wisdom-boosted) share to it, then refreshes
the party UI (`fcn.00504cf0` → per-member
`fcn.00504bb0`) and shows the floating XP text (`fcn.004499e0`). So a
kill's reward is split/applied across the active party, not just the
killer.

## Level — the XP→level formula ✅ (RESOLVED)

**`XPforLevel(L) = 1000·(L³ − L)/3 = 1000·(L−1)·L·(L+1)/3`**

Thresholds: L2 = 2 000, L3 = 8 000, L4 = 20 000, L5 = 40 000,
L6 = 70 000, L7 = 112 000, L8 = 168 000, L9 = 240 000, L10 = 330 000, …

Proven at **three independent code sites** (this overturns the earlier
"dynamic-only" park — the whole formula is static):

1. **`fcn.0055ae50` = `CStats::AddExperience(delta) → newLevel`** —
   adds to `[this+0x94]`, then computes the **Cardano closed-form
   root** of the inverted cubic: `t = XP·3.0·0.001` (consts `0x60c400`
   = 3.0, `0x60c3f8` = 0.001); if `t² − 4/27 ≤ 0` (`0x60c3f0` = 4/27,
   i.e. XP ≤ 128) → level 1; else
   `u = cbrt(0.5·(t + √(t² − 4/27)))` (`_CIsqrt` `0x5e5ce6`, `_CIpow`
   `0x5e5ec2` with exponent 1/3 @ `0x60c3e8`), and
   **`Level = trunc(u + 1/(3u))`** (`_ftol` truncation). Algebraically
   `u + 1/(3u)` is the real root of `x³ − x = 3·XP/1000` — the exact
   inverse of the formula. Returns 0 when total XP is 0.
   *FP caveat:* at exact thresholds the root can land at `L − ε`;
   a reimplementation should use the **integer inverse** below.
2. **`0x5206a8..0x5206f4`** (status-plate XP bar) — the integer form:
   `XP(L) = (L·L·1000 − 1000)·L / 3` (÷3 via `imul 0x55555556`),
   computed for `L` and `L+1`, progress =
   `([CStats+0x94] − XP(L)) / (XP(L+1) − XP(L))`. Byte-exact
   confirmation of `1000(L³−L)/3` with **no offset**.
3. **`fcn.00541760`** (skill-record method; callers `0x511369`,
   `0x520715`) — required XP for a skill's next rank =
   `1000(x³−x)/3 + 2000` with `x = [rec+0x70] + rank·[rec+0x74]`
   (per-skill base level + step, rank = `[rec+0x20]/20`) — the same
   curve reused as skill level-requirements expressed in XP.

Two corrections to earlier claims in this doc:
- **`Level++` is not in an unreachable recompute** — it is **inline in
  `fcn.0042aaf0` at `0x42adc5`** (`mov [edi+0x1c], newLevel`,
  `edi = [agent+0x2c]` = the CStats pointer).
- **`Experience +0x94` is CStats-relative** (the `this` of
  `AddExperience` is `[agent+0x2c]`), not a separate agent-block field
  as an earlier pass concluded.

## Level-up grants ✅ (RESOLVED)

In `fcn.0042aaf0`'s level-up block, with `gained = newLevel − oldLevel`:

- **Attribute points: 5 per level** — `0x42ad9f`:
  `lea ecx,[esi+esi*4]` (= `gained·5`) → `0x52b820(charId, gained·5)`
  adds to the pool `[0x7454c0][idx]`; the per-attribute cap record
  `[0x7454bc]+idx·16` (4 ints: Str/Dex/Int/Sta) is *set* to `gained·5`.
  Persisted as `[agent+0x41c]` (re-mirrored at `0x5039cd`).
  **Spend path**: StatusPlate handlers `0x52bbf0` ("+": pending++ in
  `[0x7454b8]+idx·16`, pool−1) / `0x52bc20` ("−") / `0x52bb50` (apply →
  message op `0x2f` via `0x509240`).
- **Skill points: 1 per level** via `0x5421f0(gained)` →
  `[skillMgr [0x7467fc]]+0x20 += n` (sounds 307-309/221-223), **plus
  +1 for each new level divisible by 5** (loop `0x42ad26`, `% 5` — it
  iterates `newLevel..newLevel+gained−1`, exact for normal 1-level
  gains), **plus on difficulty 3** (`[0x658c04]+0x50 == 3`, the
  4-position `DIFFICULTY` option) a **30 % chance** (`rand()%100 < 30`)
  of `+gained` again. **Spend path**: `0x5441e0` (pool > 0 guard) →
  rank-up `0x5440c0` (`[skill+0x20] += 20`, clamp 0..99; rank =
  value/20), `dec pool` at `0x544287`, floating-text key 193.
- **Other grants**: script-command arm `0x512499` (`pool += N`) and a
  consumable at `0x587b28` (+1 skill point, capped at 80).
- **Side effects** (`0x42adbb..`): Level write; `Hp = MaxHp`,
  `Mana = MaxMana`, `Stamina = MaxStamina`; CStats vtable slot 1
  recompute; sounds 303-305; FX id `0x22` on the agent visual; sound
  `0x1a`.

So a level-up grants **points to allocate** (5 attribute + 1(+bonus)
skill) and refills the pools — HP/mana *maxima* are still re-derived
from attributes via the `CStats` derivation ([`stats.md`](stats.md));
there is no per-level growth table (confirmed: no such `props.000`
curve exists).

- Level-up is surfaced in the UI by the status plate (`fcn.0052e7b0`,
  `StatusPlate Level Up` / `Level Down` buttons; the debug `Cheat One
  Level Up` registered in `fcn.00489cf0` — the menu *builder*; the
  actual spend handlers are the `0x52bbf0`/`0x52bc20`/`0x52bb50` and
  `0x5441e0` functions above, found via the pool globals).
- There is **no named `ExpForLevel` table** in the data — consistent
  with the now-proven computed formula.

## Status

- XP-award path ✅ — `fcn.0042aaf0` is the **party** awarder: per-kill
  reward + anti-grind cap (per-type kill counter `+0x108`, limit 70) +
  Wisdom boost (`fcn.00544e30`, props `WisdomExperienceBoostList`) +
  per-member find-by-id + add (`fcn.005438c0` = the list-search), then party-UI refresh
  (`fcn.00504cf0`→`fcn.00504bb0`) and floating XP text (`fcn.004499e0`).
- XP sources ✅ — per-agent reward value, story/quest awards, and the
  `set experience #` script command; Wisdom is the XP-gain skill.
- `Level` field ✅ — `CStats+0x1c`; level-up UI located.
- Experience field ✅ — `+0x94` in the stat block (from the dumper
  `fcn.0055be30`; full order recorded above).
- Level-up threshold formula ✅ **RESOLVED** —
  `XPforLevel(L) = 1000·(L³−L)/3`, proven at three independent sites
  (`CStats::AddExperience` `fcn.0055ae50` Cardano closed form; the
  status-plate XP-bar integer form `0x5206a8`; the skill-requirement
  reuse `fcn.00541760`). *(The long-standing "dynamic-only /
  closed-as-impractical" verdict recorded here was **wrong**: the
  earlier sweeps searched for a threshold *comparison* near the
  `+0x94`/`+0x1c` offsets, but the engine computes the level as a
  **closed-form cube root of the XP** — there is no compare-against-
  table site to find. The winning angle was following the XP awarder's
  callee `fcn.0055ae50` and recognizing the Cardano constants
  3.0/0.001/4/27/⅓. The old dead-end notes are superseded and dropped;
  the `Level++` is inline in `fcn.0042aaf0` at `0x42adc5`.)*
- Level-up grants ✅ **RESOLVED** — 5 attribute points + 1 skill point
  per level (+1 per level divisible by 5; +30 % chance of double on
  difficulty 3), pools/caps/spend handlers all pinned (section above).
  Confirmed no `props.000` growth table exists (191-prop sweep) —
  the counts are hardcoded constants in `fcn.0042aaf0`, now read.
