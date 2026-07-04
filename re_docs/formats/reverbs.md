# `sound\reverbs.dat` — environmental reverb (EAX)

The spatial-audio data: named **reverb environments** (EAX presets) and
the world **regions** that use them, so audio echoes differently in a
cave vs a hall. Consumed by `SMReverbManager`
(`.\SOUND\SMReverbManager.cpp`), which keeps a reverb-region priority
queue (`ReverbRegionID %d`, `id=%d nesting=%d reverb=%s`) — see
[`../sound-runtime.md`](../sound-runtime.md).

All integers little-endian.

## Content

- **Reverb presets** — at least one named preset (`Cave`) followed by a
  fixed block of **EAX 2.0 listener parameters** (≈13 fields, 52 bytes):
  negative millibel levels (`Room ≈ -808`, `Reflections ≈ -777`,
  `Reverb ≈ -302`) and floats (`DecayTime ≈ 2.7`, `DecayHFRatio ≈ 0.75`,
  `EnvironmentSize ≈ 13.6`, `EnvironmentDiffusion = 1.0`,
  `AirAbsorptionHF = 20.0`). These are the classic EAX reverb knobs.

- **Region → reverb map** — ~**90** length-prefixed `REGION_*` names
  (`REGION_AlerothDungeon1`, `REGION_Cave1`, `REGION_DwarvenHalls`, …),
  each binding a world reverb region to a preset. The names match the
  **`ReverbV1.0` region table** (`region.004`, [`region.md`](region.md)),
  so entering a reverb region selects its EAX environment.

## Exact record layout ✅

`sound\reverbs.dat` is fully decoded (all 8 presets parse to exact EOF):

```text
u32  numPresets = 8
per preset:
    u32   namelen; char name[]      "Cave","CellarCave","CellarDung","Dungeon",
                                    "Map4","OutsideGen","PocketUniverse","Sewers"
    u32   envId                     EAX environment preset id (8/8/5/13/7/19/24/21)
    i32   Room                      millibels  (e.g. -808 … -1010)
    i32   RoomHF
    i32   RoomRolloffFactor
    f32   DecayTime                 seconds   (2.7)
    f32   DecayHFRatio              (0.75)
    i32   Reflections               mB
    f32   ReflectionsDelay          s (0.015)
    i32   Reverb                    mB (-302)
    f32   ReverbDelay               s (0.022)
    f32   EnvironmentSize           (13.6)
    f32   EnvironmentDiffusion      (1.0)
    f32   AirAbsorptionHF           (20.0)
    u32   regionCount
    { u32 namelen; char name[] } × regionCount    the REGION_* using this preset
```

So each preset carries a full **EAX 2.0** parameter block (id + 12
params, 13 × u32 = 52 bytes) immediately after its name, then **owns its
own region list** — the region→preset mapping is *implicit* (a region
appears under the preset that uses it). The 8 presets cover **90**
regions total (19+14+34+15+1+4+1+2). The loader is `FUN_0054c1b0`
(`SMReverbManager`), reading the fields with `fread(_, 4, 1, _)`.

## Companion text file — `sound\reverbregions.dat`

A separate, **human-readable** file listing the reverb regions and their
indices (verified against the shipped file: **89** entries). Each line is
the engine's generic object-definition syntax:

```text
object {REGION_Sewers1Map1,7,(7,0,0,0)}
object {REGION_PetCave,7,(7,1,0,0)}
…
object {REGION_BlueBoarCellar,7,(7,88,0,0)}
```

i.e. `object {REGION_<name>, 7, (7, <index>, 0, 0)}` — **type `7`** is the
reverb-region kind ([`region.md`](region.md)'s region-type space) and
`<index>` is a sequential id `0…88`. So `reverbregions.dat` is the
**named-index roster** of reverb regions (CRLF-terminated text), the
readable companion to the binary `reverbs.dat` (whose per-preset region
lists reference these same `REGION_*` names). The `~90` regions in the
binary's preset lists match this 89-entry roster.

## Status

- Purpose ✅ — EAX environmental reverb presets + region→preset
  assignments; the audio side of the `ReverbV1.0` regions.
- Companion roster ✅ — `sound\reverbregions.dat`, 89 `object {REGION_*,7,
  (7,idx,0,0)}` text lines (verified vs the shipped file), the named-index
  list the binary's region map references.
- Record layout ✅ — **fully decoded** (above): `numPresets` + per-preset
  `{name, envId, 12 EAX 2.0 params, regionCount, region names}`, parses
  byte-exact to EOF.
- EAX parameter set ✅ — id + Room/RoomHF/RoomRolloff/DecayTime/
  DecayHFRatio/Reflections/ReflectionsDelay/Reverb/ReverbDelay/
  EnvironmentSize/EnvironmentDiffusion/AirAbsorptionHF (EAX 2.0).
- Region map ✅ — 8 presets own 90 `REGION_*` regions; the mapping is
  implicit by which preset lists each region.

```text
div.exe:0x0054c1b0   FUN_0054c1b0   SMReverbManager loader — opens sound\reverbs.dat,
                                    sequential u32 (fread 4×1) field reads.
```
