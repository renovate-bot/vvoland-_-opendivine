# Monologues (NPC barks, `monologue.cpp`)

The **monologue** system is the NPC *bark* layer — short, NPC-keyed lines
an NPC speaks on its own (idle chatter, reactions to being looked at),
distinct from the branching conversation trees of `DivDialogSystem.dll`
([`dialogue.md`](dialogue.md)). Each line has optional voice-over audio.

## Data files

| File | Role |
|---|---|
| `dat\mono.dat` | the base monologue script (text) |
| `dat\monologues\%s\mono.dat` | per-campaign/area monologue script (`%s` = set) |
| `dat\monologues\%s\mono_female.dat` | the **female** parallel text (same line count) |
| `dat\mono.cmp` / `localizations\%s\mono.cmp` | the VO **audio** ([`formats/cmp.md`](formats/cmp.md)) |

The male (`mono.dat`) and female (`mono_female.dat`) scripts are parsed in
lockstep and **must have the same number of message items** — a mismatch
logs `"Amount of message items different between male/female monologue
texts"` (handlers `fcn.004fb100` / `fcn.004fb500`). So every line exists in
a male and a female variant, selected by the speaking NPC's gender.

## Loader & parser

Loaded once at boot as the *"Loading monologues"* step of `init.cpp`
([`architecture.md`](architecture.md)): the manager ctor is `fcn.004fc260`
(zero/sentinel-inits its fields, builds a sub-object at `+0x23c`), and the
script is read and parsed via `fcn.004ae140` → the **per-line parser
`fcn.004fc420`** (`monologue.cpp`).

`fcn.004fc420` is a **template-statement line parser** (`monologue.cpp`).
It peels the next line off the buffer (`fcn.004fdc40`, split on `"\n"`,
skipping leading spaces and `/`-comment lines), then matches that line in
turn against a **12-entry keyword-template table** at `div.exe:0x650d2c`
using the format matcher `fcn.004fdf40`. The templates are not plain
strings: they embed slot placeholders — **`#` = an integer slot** (the
case `atoi`s it) and **`$` = a string slot** — with the literal words
matched verbatim. The first template that matches selects a switch case
(`jmp [edi*4 + 0x4fc6a4]`); no match falls through to
`"Line %d - Syntax error in monologues : %s"`.

All twelve statement templates and their handlers, verified from the table
and the switch (index → template → handler):

| # | Template | Handler | Role |
|---|---|---|---|
| 0 | `new message $` | `fcn.004fc2b0` | begin a new message block, `$` = its label |
| 1 | `say time # text $` | `fcn.004fb100` | **add a line**: `#` = display duration, `$` = text. Also hosts the male/female lockstep check (`"Amount of message items different…"`, `0x4fb2ec`) |
| 2 | `set owner $` | `fcn.004fb370` | the **owning NPC** of the block (`fcn.00422950` name lookup; `"Unknown npc %s in monologues"`) |
| 3 | `set lock` | `fcn.004fb3b0` | set a lock flag (no argument) |
| 4 | `set event $` | `fcn.004fb3d0` | **`MonologueEvents`** binding via config-section lookup `fcn.00505650` (the `CameraEvents` pattern in [`minor-mechanics.md`](minor-mechanics.md)) |
| 5 | `wav $` | `fcn.004fb400` | VO clip name for the line ([`mono.cmp`](formats/cmp.md)) |
| 6 | `face $` | `fcn.004fb420` | the NPC whose **portrait/face** is shown (same `fcn.00422950` name lookup + `"Unknown npc"`) |
| 7 | `include $` | `fcn.004fb460` | **include another monologue file** — recurses into the file reader `fcn.004fcab0` |
| 8 | `look $ $` | `fcn.004fb480` | examine-reaction line, two string slots (`"Unknown npc %s in monologues look statement"`) |
| 9 | `say time # text $ index #` | `fcn.004fb500` | timed line variant with an explicit `index` |
| 10 | `scrying $` | `fcn.004fb770` | **scrying-mirror entry** (→ `fcn.004fa540`; see Scrying portraits below) |
| 11 | `description $` | `fcn.004fc1f0` | metadata description field |

So a monologue script is **grouped by NPC** (`set owner`), each block
carrying timed message lines (`say …`), an examine `look` set, an optional
VO clip (`wav`), portrait (`face`), `MonologueEvents` binding (`set event`)
and `description`, with `include` for file composition.

### Scrying-mirror portraits (`scrying $`)

The shipped `dat\monologues\Scrying\` directory holds **portrait `.tga`s**
(`goemoe1..6`, `eolus1..3`, `mardaneus`, `zaxnadrix`, `kroxy`, `bronthion`)
— the faces shown for the scrying / magic-mirror viewing mechanic. These
are the targets of the `scrying $` statement (handler `fcn.004fb770` →
`fcn.004fa540`), tying the monologue grammar to the on-disk portrait set.
(No `mono.dat` text scripts are shipped in this `gamedata\` tree — the
base/per-set monologue scripts live in the `mono.cmp` archives — so the
grammar is recovered from the binary, not from a sample file.)

## Triggering

A monologue is advanced from the **agentscript executor**
(`fcn.004329a0`, [`npc-ai.md`](npc-ai.md)) — at `0x434577` it logs
`"Npc %s triggers next monologue line"`. So an NPC's behaviour script steps
through its monologue lines (and the engine logs `"Start a monologue"` /
`"next monologue line"`), with the matching `mono.cmp` VO clip played per
line and the male/female text chosen by gender. The `look` set is fired by
the examine interaction rather than the behaviour tick.

## Status

- Data files ✅ — `mono.dat` (+ per-set `monologues\%s\`), the
  `mono_female.dat` parallel text (equal item count, enforced), and the
  `mono.cmp` VO audio.
- Loader/parser ✅ — boot step `fcn.004fc260` (manager) → `fcn.004ae140`
  → template-statement parser `fcn.004fc420` (`monologue.cpp`), 12-entry
  template table `0x650d2c` matched by `fcn.004fdf40`, `"Syntax error in
  monologues"` diagnostic.
- Statements ✅ — all **12 statement templates** recovered verbatim with
  their handlers and switch indices (`new message`, `say time # text $`,
  `set owner`, `set lock`, `set event`, `wav`, `face`, `include`,
  `look $ $`, `say … index #`, `scrying`, `description`). `#` = integer
  slot, `$` = string slot. The male/female lockstep check lives in the
  `say` handler `fcn.004fb100`.
- Triggering ✅ — advanced by the agentscript executor `fcn.004329a0`
  (`"triggers next monologue line"`); VO via `mono.cmp`; gender-selected
  text. The selection index / cooldown state on the manager object 🟡
  (runtime, not statically anchorable).

## Citations

```text
div.exe:0x004fc260   monologue-manager ctor (boot "Loading monologues").
div.exe:0x004ae140   reads dat\monologues\%s\mono.dat (+ female) and drives the parser.
div.exe:0x004fc420   template-statement parser — monologue.cpp; "Syntax error in monologues".
div.exe:0x00650d2c   12-entry keyword-template table (new message/say/set owner/…).
div.exe:0x004fc6a4   12-case switch jump table (template index → handler).
div.exe:0x004fc2b0   "new message $"   — begin a message block.
div.exe:0x004fb100   "say time # text $" — add a line; hosts male/female lockstep check.
div.exe:0x004fb500   "say time # text $ index #" — indexed line variant.
div.exe:0x004fb370   "set owner $"     — owning NPC ("Unknown npc %s in monologues").
div.exe:0x004fb3b0   "set lock"        — lock flag.
div.exe:0x004fb3d0   "set event $"     — MonologueEvents config binding (via fcn.00505650).
div.exe:0x004fb400   "wav $"           — VO clip name.
div.exe:0x004fb420   "face $"          — portrait NPC (same name lookup).
div.exe:0x004fb460   "include $"       — recurse into file reader fcn.004fcab0.
div.exe:0x004fb480   "look $ $"        — examine-reaction lines ("... look statement").
div.exe:0x004fb770   "scrying $"       — scrying-mirror portrait (→ fcn.004fa540).
div.exe:0x004fc1f0   "description $"   — metadata.
div.exe:0x004fcab0   monologue file reader (called by boot loader and by "include").
div.exe:0x004329a0   agentscript executor — triggers the next monologue line (@0x434577).
```
