# Divine Divinity binary architecture

`div.exe` is the Divine Divinity (Larian, Steam re-release) main executable. MSVC 2008 (linker 9.0, timestamp 2013-04-19), x86 PE, image base `0x00400000`, entry `0x005e64b1` → `__tmainCRTStartup` → `WinMain`.

It is **not SafeDisc-wrapped** — the entry is a normal CRT init, sections and imports are clean, and `.text` entropy is uniform. Analyze the binary directly. The `CDAPFN0506_PFN_1..5/Test1` exports near `0x00651000` are leftover SafeDisc PFN API descriptors (`CDAPFN_PROPERTIES`, 136 bytes each, array at `0x00648920`); each descriptor's `+0x0C` field points at the real function body in `.text`. The tamper-call at the end of each PFN body is gated by a hard-coded non-zero flag and never fires.

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
