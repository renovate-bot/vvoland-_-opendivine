# `sound\music.dat` — music tracks & zones

The music database: the named tracks the engine can stream, and the
named **music zones** that select which track plays where. The
streaming runtime that consumes this (the `SMMusicManager` fade/duck
state machine) is in [`../sound-runtime.md`](../sound-runtime.md); this
doc is the data file behind it.

All integers little-endian.

## Section 1 — track table ✅

```text
u32  count                    (= 43)
Track[count]:
    u32   name_len
    char  name[name_len]      // short id name: "1", "12", "17cAmMix2", "AmbCreepP"
    u32   file_len
    char  file[file_len]      // the .ogg, e.g. "1.ogg", "amb creepP.ogg"
    u32   field               // 0 in every shipped record
```

43 tracks. The `name` is a short handle; `file` is the streamed clip
(under `wav\music\` per [`../sound-runtime.md`](../sound-runtime.md),
played via FMOD `FSOUND_Stream_*`). `name` and `file` can differ
(`AmbCreepP` → `amb creepP.ogg`). Parsing the 43 records consumes
1417 bytes, then **section 2 begins**.

## Section 2 — location tracks 🟡

```text
u32  count                    (= 22)
Track[count]:
    u32   name_len
    char  name[name_len]      // location name, e.g. "BehindDaBridge", "Cellar", "Church"
    u32   file_len
    char  file[file_len]      // "<name>.ogg"   (no trailing field, unlike section 1)
```

A second list of **22 location-named tracks** (`BehindDaBridge →
BehindDaBridge.ogg`, `Cellar → Cellar.ogg`, `Church`, `Dungeon`, …) —
the place-specific music, distinct from section 1's numbered/ambient
tracks. (Correction: an earlier pass mislabelled these as "zones"; they
are tracks. The zone/region → track selection lives in the `MusicV1.0`
region table, `region.001`, [`region.md`](region.md), which references
these track names.) Note the record has **no trailing `u32`** that
section 1 carries.

Parsing section 2 reaches offset 2063; the **third section** is the
zone map below.

## Section 3 — zone → weighted day/night tracks ✅ (model)

```text
u32  count                    (= 153 zones)
Zone[count]:
    u32   name_len
    char  name[name_len]      // "Aleroth", "Aleroth_Cellar", …
    …     day track list      // weighted tracks for daytime
    …     night track list    // weighted tracks for night
    where each track entry ≈ { u32 name_len, char name[], f32 weight }
```

153 zones, each binding a list of **day** tracks and a list of **night**
tracks with **float weights** for random selection. Example, `Aleroth`:

```text
day:   "4" (0.2), "forest" (0.2)
night: "GenDay01" (1.0), "GenNight" (1.0)
```

The track names reference sections 1–2; the zone names are what the
**`MusicV1.0` region table** (`region.001`, [`region.md`](region.md))
matches against. So the music subsystem picks a zone from the player's
region, then a weighted track from the **day or night** list per the
world clock ([`world-clock.md`](../world-clock.md)) — i.e. zones can play
different music by time of day.

**Track entry — verified byte-exact:** `{ u32 name_len, char name[],
f32 weight }`. `Aleroth`'s day list is `count=2` → `"4"` (0.2),
`"forest"` (0.2), and its night `count=0`. The two `u32` count fields
delimit the day and night lists.

**Section-3 "flag bytes" — resolved in [sound.md](sound.md) ✅.** The
"variable trailing `u32` run" that blocked a fixed walk here is not
irregular at all: each zone carries **5 parallel binding lists** (not
day+night 2), and the "trailing `0,0,1`" after `Aleroth` is simply the
empty counts of two further lists plus the final `u32` flag. With the
5-list schema (`{label, List[5], u32 flag}`, each list
`{count, (label,f32 volume)×count}`) the whole section walks
byte-exact to EOF — see sound.md's Section 3 for the schema, the
reader functions (`FUN_00554080`), and the list roles (L0 music,
L3 day / L4 night ambients). This paragraph's "does not walk with any
fixed pattern" conclusion is retracted.

## Status

- Section 1 (tracks) ✅ — `u32 count(43)` + `{name, .ogg file, u32
  field}` records; verified.
- Section 2 (location tracks) ✅ — `u32 count(22)` + `{name, .ogg}`
  records (no trailing field); 22 place-named tracks. *Corrects an
  earlier "zones" mislabel.*
- Section 3 (zone map) ✅ (model) — `u32 count(153)` zones, each with a
  weighted **day** track list and **night** track list (so music can
  change by time of day); the exact per-entry separator/flag fields 🟡.
- Region → track link ✅ (by name) — the `MusicV1.0` regions
  (`region.001`) match zone names; the zone picks a weighted day/night
  track per the world clock.
- The `field` (`+after file`, always 0) ❓ — a track flag/type, unused
  in the shipped data.
