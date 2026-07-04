# Divine Divinity binary architecture

`div.exe` is the Divine Divinity (Larian, Steam re-release) main executable. MSVC 2008 (linker 9.0, timestamp 2013-04-19), x86 PE, image base `0x00400000`, entry `0x005e64b1` → `__tmainCRTStartup` → `WinMain`.

It is **not SafeDisc-wrapped** — the entry is a normal CRT init, sections and imports are clean, and `.text` entropy is uniform. Analyze the binary directly. The `CDAPFN0506_PFN_1..5/Test1` exports near `0x00651000` are leftover SafeDisc PFN API descriptors (`CDAPFN_PROPERTIES`, 136 bytes each, array at `0x00648920`); each descriptor's `+0x0C` field points at the real function body in `.text`. The tamper-call at the end of each PFN body is gated by a hard-coded non-zero flag and never fires.

## Runtime structure

Past CRT init, `WinMain` reaches `main` (`FUN_0040d850`), which runs the
frame loop. Two cross-cutting facts shape everything else:

- **The engine is data/event/script-driven.** Behaviour lives in data
  files (`objects.000`, `props.000`, `traps.dat`, eggs, dialogue,
  magic-script) and is dispatched through an agent **command bus**
  (`FUN_00509f10`) and an **Osiris event manager** (`[0x7447dc]`), not
  direct calls. See [`frame-loop.md`](frame-loop.md) (the per-frame
  update order) and [`messages.md`](messages.md) (the command bus).
- **One pass per frame at 40 Hz.** `main` → `FUN_004ab4a0` ticks every
  subsystem (clock, agents, projectiles, render) once per iteration,
  capped by `game rate 40 fps`.

[`README.md`](README.md) indexes the per-subsystem and per-format notes.

## Game boot sequence (`.\GAME\init.cpp`)

The one-time engine/level boot runs in **`fcn.004a0b10`** (`.\GAME\init.cpp`),
a single ~4.5 KB routine that constructs every manager and loads its data
file in a fixed order, printing a progress banner (`fcn.004ac920`) before
each step, and ends by handing off to the main loop. This is the master
index of the engine's subsystems — the load order below is verbatim from
the banner strings, each paired with the init function it dispatches and
the data file it pulls. (A few subsystems share the generic
manager-construction helper `fcn.0055e0a0`; those rows name it as the
constructor, with the real loader being subsystem-specific.)

| # | Banner | Init fn | Data file | Subsystem doc |
|--:|---|---|---|---|
| 1 | animation maps | `fcn.00439120` | `dat\animidx.dat` | [animation](animation.md) |
| 2 | item statistics | `fcn.0057b530` | `itemstat\itemstat.txt`, `itemlink.dat` | [items](items.md) |
| 3 | Object manager | `fcn.0058a250` | `static\objects.000`, `dat\objects.dat` | [objects.000](formats/objects.md) |
| 4 | inventory managers | `fcn.004b0a60` | — | [inventory](inventory.md) |
| 5 | sprite engine / twilight | `fcn.00597160` | `static\gradsmal.tga` | [render-trace](render-trace.md) |
| 6 | sprite sorter | `fcn.005466d0` | — | [render-trace](render-trace.md) |
| 7 | cell manager | `fcn.00571fa0` | — | [world.x](formats/world.md) |
| 8 | locations + trap locations | `fcn.0057c3d0` | `location.000` | [location](formats/location.md) |
| 9 | camera | `fcn.004f27b0` | `dat\camera.dat` | — |
| 10 | npc manager | `fcn.00422370` | — | [monsters](monsters.md) |
| 11 | magic semantic | `fcn.004cd4b0` | — | [skills-magic](skills-magic.md) |
| 12 | group / conversation / spranim | `fcn.0055e0a0` (ctor) | — | — |
| 13 | treasure | `fcn.00597090` | `dat\treasure\list.dat` | [treasure](formats/treasure.md) |
| 14 | alignments | `fcn.004386e0` | `dat\alignment.dat` | — |
| 15 | npc data | `fcn.00426110` | `dat\npclist.dat` | [monsters](monsters.md) |
| 16 | monster generation | `fcn.0043f2a0` | `dat\monstergen.dat` | [monsters](monsters.md) |
| 17 | **map** (world grid) | `fcn.005a0300` | `world.x<n>` + roofs | [world.x](formats/world.md) |
| 18 | Osiris objects | `fcn.00585c80` | `osiobjects.000`/`osinames.000` | [osi-static](formats/osi-static.md) |
| 19 | shroud | `fcn.0053eff0` | `shroud.x<n>` | [shroud](formats/shroud.md) |
| 20 | effect processor | `fcn.00490300` | — | — |
| 21 | player | `fcn.004a90e0` | — | [agent](agent.md) |
| 22 | traps | `fcn.00593650` | — | [traps](traps.md) |
| 23 | magic system | `fcn.004c99c0` | — | [skills-magic](skills-magic.md) |
| 24 | rain | `fcn.00500f90` | — | — |
| 25 | explosion manager | `fcn.0055e0a0` (ctor) | — | [explosions](explosions.md) |
| 26 | monologues | `fcn.004fc260` | `dat\monologues\%s\mono.dat` | — |
| 27 | birds | `fcn.004f0800` (ctor) / `fcn.004f0fe0` (data) | `dat\birds.cfg` + `dat\birds.000` | [minor-mechanics](minor-mechanics.md#ambient-birds-cbirdmanager) |
| 28 | projectile / particle system | `fcn.004cca70` | — | [projectiles](projectiles.md) |
| 29 | plate system + Automap | `fcn.0044c3b0` | — | [automap](formats/automap.md) |
| 30 | Diary / CharSel / Main Menu | `fcn.00478030`… | — | [gui](gui.md) |
| 31 | books of god | `fcn.0056cf60` | — | [books](formats/books.md) |
| 32 | skills | `fcn.00543450` | `dat\skills.dat`, `localizations\%s\skills.txt` | [skills-magic](skills-magic.md) |
| 33 | teleporter list | `fcn.00530700` | — | — |
| 34 | story map flags | `fcn.0044ae10` | — | [osiris](osiris.md) |
| 35 | **Initializing osiris** | `fcn.00516b90` | `binary.div` | [osiris](osiris.md) |
| — | **Starting game engine…** | `fcn.005a0c00` | → main loop | [frame-loop](frame-loop.md) |

Two of these cross-validate earlier findings: step 35's
`fcn.00516b90` is exactly the `RegisterDIVFunctions` DIV-router builder
decoded in [`osiris.md`](osiris.md), and step 17's `fcn.005a0300` is the
`world.x<n>` grid ctor in [`formats/world.md`](formats/world.md) — so the
boot list is anchored to known code at both ends. The per-map **object
instances** (`objects.x<n>`) are restored within the *Loading map* /
*Loading Osiris objects* steps via the object manager, not by the WORLD
file-plumbing ([`formats/osi-static.md`](formats/osi-static.md)); the
instance record format is still open 🟡.

A couple of banners in the sequence (e.g. *"Loading genetic algorithms"*)
print with **no loader call of their own** — back-to-back with the next
banner — so they are not separate subsystems: the "genetic algorithms"
step is just the monster-generation tables loaded one step earlier
(*"Loading monster generation data"*, [`monsters.md`](monsters.md)), the
engine's procedural-monster ("genetic") placement.

## Renderer plugin DLLs

Loaded at runtime via `LoadLibraryA` based on `slashed-*.cfg`. They are not in `div.exe`'s PE import table — div resolves symbols with `GetProcAddress` for the `DllSlashed*` API, so cross-binary references won't appear as static imports.

| DLL | Identification | Config |
|---|---|---|
| `slash1.dll` | `Direct3D 6 R` | `slashed-d3d6.cfg` |
| `slash2.dll` | `Glide 3.x R` | `slashed-glide.cfg` |
| `slash3.dll` | `DirectX R` (newer DX) | `slashed-directx.cfg` |
| `slash4.dll` | `Software R` | `slashed-software.cfg` |

Plugin ABI: `DllSlashedInit`, `DllSlashedShutdown`, `DllSlashedStartFrame`, `DllSlashedEndFrame`, `DllSlashedGetResolutions`, `DllSlashedGetIdentification`, `DllSlashedGetMajorVersion`, `DllSlashedGetMinorVersion`, `DllSlashedInternalApplyConfiguration`, `DllSlashedGlowDraw{Line,Rect,Square,Quad}`, …

## Subsystem DLLs (statically imported)

- **OsirisDLL.dll** — `COsiris` scripting / story engine. Exports include `Compile`, `InitGame`, `Save`, `Load`, `Event`, `Merge`, `RegisterDIVFunctions`, `GetStoryVersion`, `Minilog_Create`. Story logic is compiled into Osiris bytecode (see `binary.div`).
- **DivDialogSystem.dll** — `CDivDialogSystem`. Exports: `LoadDialogSystem`, `StartDialog`, `GetQuestion`, `GetAnswerText`, `GetAnswerNodeID`, `GetNumQuestions`, `GetAnswerSoundName`, `SelectQuestion`, `EventChanged`, `Save`/`Load`. Branching dialogue trees.
- **binkw32.dll** — RAD Bink video for intro / cutscenes (`static\larian.bik`, `kz.bik`, `scene1.bik`, `scene2.bik`).
- **fmod.dll** — FMOD Sound System. **UPX-packed**; only 3 functions are visible without unpacking.

The **complete** static import set is just: `fmod`, `winmm`, `version`,
`imm32`, `kernel32`, `user32`, `gdi32`, `divdialogsystem`, `osirisdll`,
`binkw32`, `msvcr90`.

## No functional multiplayer (`.\MPLAYER\` is the message system)

Despite `.\MPLAYER\…` source paths and strings like `"multiplayer"`,
`"session $"`, `"connection #"`, and `"Server connected"`, **Divine
Divinity has no working network multiplayer** — and this is provable
statically: the import list above contains **no networking DLL** (no
`ws2_32`/`wsock32`, no `recv`/`send`/`socket`/`connect`), so the executable
cannot open a socket. The `.\MPLAYER\` namespace is instead the in-game
**message / event-passing system** (`Message.cpp`, `fcn.00508d50` /
`fcn.00509930`) behind the **message board** and **rumor board** boot steps
([`architecture.md` boot table](#game-boot-sequence-gameinitcpp)) — i.e.
in-world notifications and agent/system messages, not a netcode layer. The
`"Server connected"` / `"ConnectDuration"` strings are **vestigial**
leftovers of a cut MP mode (referenced only by a status path
`fcn.004a1c80`, never by socket code). So a port needs no networking; the
"MPLAYER" code is a single-player message bus.

## Build fingerprints

- MSVC 2008 (`msvcr90.dll`, `_except_handler4_common`, `__CxxFrameHandler3`)
- Source path leaks: `.\MISC\divversion.cpp`, `.\GLOBAL\globalflat.cpp`, `.\magic\magic.cpp`, `mapgen\divinity.map`
- Game name strings: `Divine Divinity`, `Divine Divinity HD`, `Larian Studios`, `(www.larian.com)`

## Where to look when

| Symptom | Look in |
|---|---|
| Rendering bug | `slash{1,2,3,4}.dll` matching the user's `slashed-*.cfg` |
| Story / scripting | `OsirisDLL.dll` + `binary.div` |
| Dialogue tree | `DivDialogSystem.dll` |
| Cutscene playback | `binkw32.dll` |
| Audio | `fmod.dll` (must unpack UPX first) |
| Savegame | `data.000`, `story.000` under `main/startup/` |
