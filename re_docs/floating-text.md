# Floating feedback text (`.\MISC\floattext.cpp`)

The pop-up text that rises above a character — **damage numbers**, the
**XP gained** on a kill, and short status messages. A small, gameplay-
visible feedback system used across combat and progression but, until now,
only mentioned in passing (the XP float in [`progression.md`](progression.md)).

## The spawner — `fcn.004499e0`

One function pops a floating string above an entity. It takes a formatted
text buffer (a ~`0x410`-byte stack string), reads the target agent's
screen position from `[agent+0x23c] → +0x1c/+0x20`, and allocates a
floating-text record (via the `floattext.cpp` allocator `fcn.004f6770`)
into the manager at **`[0x6e0140]`**. The record then drifts upward and
fades over its lifetime, drawn as a [GUI](gui.md) overlay above the world.

The displayed string is built through the **localized-string formatter**
`fcn.00504cf0` ([`localization.md`](localization.md), table `[0x744784]`),
so the feedback text — "Miss", a status word, a number — is pulled from
the language tables, not hard-coded.

## Who pops text

`fcn.004499e0` has many callers, all the moments the game wants to tell the
player something happened at a spot in the world:

- **Combat** — `fcn.0041dd70` / `fcn.0041c250` / `fcn.0041e4e0` (the
  [agent-fight](combat.md) cluster): damage taken, blocks/misses, status
  procs above the struck agent.
- **Progression** — `fcn.0042aaf0` (the party [XP awarder](progression.md))
  pops the **experience gained** over the character on a kill;
  `fcn.0042af30` for related rewards.

So the same call is the unified "show feedback here" primitive — combat
results and reward notifications share it.

## How it fits

- **Position** — anchored to the agent's screen coordinates
  (`[agent+0x23c]`), so the text tracks the entity it belongs to.
- **Text** — formatted through [localization](localization.md), so it is
  language-correct.
- **Render** — a fading overlay in the [GUI](gui.md) layer, ticked from the
  per-frame update like the other transient visuals.

## Status

- Spawner ✅ — `fcn.004499e0`: positions a formatted string at the target
  agent (`[agent+0x23c]`), allocates into manager `[0x6e0140]` via the
  `floattext.cpp` allocator `fcn.004f6770`.
- Text source ✅ — the localized-string formatter `fcn.00504cf0`
  (`[0x744784]`).
- Consumers ✅ — the combat cluster (`fcn.0041dd70`/`0041c250`/`0041e4e0`,
  damage/status) and progression (`fcn.0042aaf0` XP, `fcn.0042af30`).
- Record fields / rise / colour ✅ **RESOLVED** — the initializer
  `fcn.004f6770(mgr, x, y, text)` sets **`+0x00 = worldX`,
  `+0x04 = worldY`** (the spawner passes
  `x = ai[+0x1c]+ai[+4]`, `y = ai[+0x20]−ai[+0x14]+ai[+8]`,
  `ai = [agent+0x23c]` — world coordinates; camera applied at draw),
  **`+0x08/+0x0c` = text pixel width/height** (measurer
  `fcn.004f6ac0`; also used by the overlap adjuster `fcn.004f6570`),
  `+0x10` = strdup'd text, `+0x14` = 40-tick lifetime, `+0x18` =
  active. *(The earlier "+0x04..+0x0c zeroed at init" applied only to
  fresh slot allocation.)* **Rise**: the manager tick `fcn.004f66e0`
  (per frame @`0x4aaab1`) does `+0x14--` and **`dec [rec+4]`** —
  worldY − 1 per tick for 40 ticks, then the text is freed; **no alpha
  fade**, the popup just dies. **Colour is not in the record**: the
  draw `fcn.004f6670(camX,camY)` renders via
  `fcn.004f68e0(x−camX, y−camY, id, "\r%c%s", 4, text)` — the char
  after `\r` selects **GUI font #(char−1)** (bounds-checked fetch from
  the font array `[fontMgr[0x6592c0]+0x78]`), so default floating text
  uses **font index 3** with the colour baked into the bitmap font.
  The colour-variant spawner `fcn.00449a60` prepends
  `sprintf("\r%c", code)` — callers `0x4176d3` (code 0xe → font 13,
  msg 61) and `0x417c7c` (code 0x13 → font 18, msg 62).

## Citations

```text
div.exe:0x004499e0   fcn.004499e0   floating-text spawner (pos [agent+0x23c], mgr [0x6e0140]).
div.exe:0x004f6770   fcn.004f6770   floattext record init — 28-byte record into mgr [0x6e0140];
                                    +0x10 text (strdup), +0x14 lifetime 0x28(40), +0x18 flag=1.
div.exe:0x00504cf0   fcn.00504cf0   localized-string formatter (table [0x744784]).
div.exe:0x006e0140   floating-text manager (active feedback strings).
str: .\MISC\floattext.cpp
```
