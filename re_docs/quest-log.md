# Quest log / Diary (`.\Diary\`)

The in-game **journal and diary** — the player's quest tracker and lore
record. It is the `.\Diary\` source cluster, surfaced as two GUI panels:

- **Journal** (`.\Diary\JournalInterface.cpp`) — the **quest log**: the
  active/completed quest entries.
- **Diary** (`.\Diary\Diary.cpp` / `.\Diary\DiaryInterface.cpp`) — the
  character/lore diary.
- **Monster log** (`.\Diary\MonsterLogMan.cpp`) — the **bestiary**: the
  per-monster knowledge entries filled in as the player identifies
  creatures (see [Monster log](#monster-log--bestiary) below).

Both are `C*Plate` HUD windows ([`gui.md`](gui.md)) shipped per screen
resolution — `JournalPlate{640,800,1024}` and `DiaryMainPlate{640,800,1024}`
— laid out from `diary.bmg`. The companion `.\diary\DialogLogMan.cpp` is
the **conversation log** (what NPCs have said), persisted as the `DialogLog`
save block ([`formats/savegame.md`](formats/savegame.md), reader
`FUN_00472940`).

## Data: `dat\diary_flags.txt`

The journal/diary **entries are flag-gated**, defined in the text file
**`dat\diary_flags.txt`**. It is loaded by `fcn.00451230` (the
`JournalInterface` init, in the `0x450…0x451` cluster it **shares with the
automap user-notes** code, [`formats/automap.md`](formats/automap.md)):
the path is built (`dat\` + `\diary_flags.txt`) and the file parsed through
the same block/list-reader family as user notes
(`fcn.004505e0` / `fcn.00450ad0` / `fcn.004508b0`). So a diary/quest entry
is keyed by a **flag id**, and the entry becomes visible when that flag is
set.

## Flag source — Osiris story events

The diary flags are **story flags driven by Osiris**: the same
`dialogids.dat` / `DIALOG_EVENT` flag space that gates dialogue
([`dialogue.md`](dialogue.md)) and that `EventChanged` toggles. When the
story script (`binary.div`, [`osiris.md`](osiris.md)) advances a quest, it
sets the corresponding flag; the Journal then shows that entry's text. So
the quest log is a **read-out of story-flag state**, not an independent
quest engine — quest *logic* lives in the Osiris story, the diary is its
presentation. (This mirrors the dialogue gate: nodes/entries are precomputed
from flags, not evaluated by a dedicated quest VM.)

### Runtime manager & persistence (`.\Diary\QuestLogMan.cpp`)

The journal's entry set is owned at runtime by **`QuestLogMan`**
(`.\Diary\QuestLogMan.cpp`): its add/update entry points `fcn.00482ab0`
and `fcn.00482b60` operate on a tree database through `fcn.00481a10` — the
same `0x481xxx` cluster as the **`"ML3ID"` quest-log reader `fcn.00481b50`
/ writer `fcn.00481ea0`**. That database is **persisted to
`quest_log.000`** (the versioned `ML3ID`/`ML2ID` ordered tree decoded in
[`formats/savegame.md`](formats/savegame.md)). So although *which* entries
are visible is driven by the Osiris story flags above, the accumulated
journal — the entries the player has unlocked, in order — is a real
persisted structure that `QuestLogMan` maintains and the save round-trips,
not something recomputed from scratch each load.

## Monster log / bestiary

The third Diary panel is the **bestiary** (`.\Diary\MonsterLogMan.cpp`).
Its entries are defined in the text file **`dat\diary_monsterlog.txt`**,
loaded by **`fcn.0047ffb0`** (the same `dat\` + `\diary_monsterlog.txt`
path-build + parser idiom as the journal's `diary_flags.txt`). Each entry
is the lore text for one monster type.

Unlike the Journal (gated by Osiris story flags), the monster log is gated
by **knowledge of the creature**: the Survivor Lore skill
**`MonsterIdentification`** ([`skill-tree.md`](skill-tree.md)) — a higher
identification level reveals more of a monster's entry (its stats /
resistances), the bestiary counterpart of item [Identify](items.md). So
the log is a read-out of which monsters the player has identified, over the
`diary_monsterlog.txt` text, exactly as the Journal is a read-out of story
flags over `diary_flags.txt`.

## Character-stats panel (`diary_stats_{m,f}.txt`)

The Diary's **character self-assessment** text is data-driven by
**`dat\<lang>\diary_stats_m.txt` / `diary_stats_f.txt`** (gender-specific;
packed in [`global.cmp`](formats/cmp.md), plaintext). Format: each entry is
a header line **`<category> <threshold>`** followed by the flavour
paragraph shown once the hero's value in that category reaches the
threshold. The English file has **57 entries across 13 categories**, each a
rising ladder:

```text
cat 0  : 0 4 8 16 30 50        { character level milestones }
cat 1-4: 0 25 50 75 100        { the four 0-100 attribute tracks }
cat 5  : 0 125 250 370 500     cat 6: 0 100 200 300 400 500
cat 7-8: 0 100 200             cat 9-10: 0 5 10 15   cat 11-12: 0 8 15
```

So as a stat crosses each threshold the panel swaps in the matching prose
(e.g. cat 0 level-0: *"You are still wet of ear-back and green of foot…"*).
Purely cosmetic flavour — no gameplay effect — gated by the live stat value
against the ladder.

## NPC portrait index (`npc_thumb_idx.txt`)

`dat\npc_thumb_idx.txt` (plaintext in [`global.cmp`](formats/cmp.md)) is the
**name → portrait-thumbnail index** map: **278** lines of
`<NPCName> <index>` (`Anastasia 0` … `vampBossghoul 277`), covering NPCs and
creatures. It resolves a speaker/creature name to its thumbnail slot for the
conversation portrait and the bestiary panel (cf. the `monologues\Scrying\`
portraits in [`monologues.md`](monologues.md)).

## Status

- Subsystem ✅ — `.\Diary\` Journal (quest log) + Diary + **Monster log**
  (bestiary) panels, per-res
  `JournalPlate`/`DiaryMainPlate` plates (`diary.bmg`), plus the
  `DialogLogMan` conversation log (`DialogLog` save block).
- Entry data ✅ (located) — `dat\diary_flags.txt`, flag-gated entries,
  loaded by `fcn.00451230` via the user-note block-reader family.
- Monster log / bestiary ✅ — `.\Diary\MonsterLogMan.cpp`, entries in
  `dat\diary_monsterlog.txt` (loaded by `fcn.0047ffb0`), gated by the Lore
  `MonsterIdentification` skill rather than story flags.
- Flag wiring ✅ (model) — entries gated by Osiris story flags (the
  `dialogids.dat` / `DIALOG_EVENT` space); quest logic is in the Osiris
  story, the diary is the presentation.
- `diary_flags.txt` byte format 🟡 — parsed by the shared block reader;
  the exact per-entry record layout (flag id ↔ text id) not split
  field-by-field.

## Citations

```text
div.exe:0x00451230   fcn.00451230   JournalInterface init — loads dat\diary_flags.txt.
div.exe:0x004505e0   fcn.004505e0   block/list reader (shared with automap user notes).
div.exe:0x00472940   FUN_00472940   DialogLogMan — conversation-log save/restore (DialogLog block).
div.exe:0x0047ffb0   fcn.0047ffb0   MonsterLogMan — loads dat\diary_monsterlog.txt (bestiary).
source: .\Diary\Diary.cpp / DiaryInterface.cpp / JournalInterface.cpp / DialogLogMan.cpp / MonsterLogMan.cpp
```
