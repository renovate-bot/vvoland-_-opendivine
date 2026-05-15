# Sound configuration files

Four files under `sound/` configure FMOD's runtime: music playlist,
SFX bank, reverb presets, and reverb-zone bindings. The actual audio
payloads (`.ogg` / `.wav`) live in `dat\sound.cmp` (1457 entries —
see [`cmp.md`](cmp.md)).

All integers little-endian.

## `sound\music.dat` — music + ambient catalogue

Three sequential sections, each starting with a `u32` count:

```text
struct MusicDat {
    // Section 1: orchestrated music tracks
    u32 track_count;                                       // = 43
    Track tracks[track_count];

    // Section 2: looped atmosphere tracks
    u32 ambient_count;                                     // = 22
    Track ambients[ambient_count];

    // Section 3: per-region music/ambient bindings
    u32 region_count;                                      // = 153
    u8  region_body[];                                     // schema TBD — 12,066 bytes in shipped file
};

struct Track {                       // section 1 form
    u32  label_len;
    char label[label_len];           // e.g. "1", "12", "17cAmMix2"
    u32  filename_len;
    char filename[filename_len];     // e.g. "1.ogg", "17cAmMix2.ogg"
    u32  trail;                      // 0 in shipped data — reserved / id placeholder
};
```

Section 2's `Track` records have the **same shape minus the trailing
`u32`** — just `label_len + label + filename_len + filename`.

`pkg/assets/musicdat` validates sections 1 + 2 byte-exact:

| section          | count | examples                                     |
|------------------|------:|---------------------------------------------|
| tracks           |    43 | "1"/"1.ogg", "12"/"12.ogg", "17cAmMix2"/...  |
| ambients         |    22 | "BehindDaBridge", "Cellar", "Forest", ...   |
| regions (header) |   153 | first label = "Aleroth" (7 chars)            |

### Section 3 — per-region bindings ✅

Each region holds **5 parallel assignment lists** plus a trailing flag.
Reader: `FUN_00554080` (`.\SOUND\SMMusicRegion.cpp`); per-list reader
`FUN_00553d80` / `FUN_00553f00` (identical byte-shape, different
in-memory dispatch).

```text
struct Region {
    u32   label_len;
    char  label[label_len];          // NO trailing NUL
    List  lists[5];                  // 5 parallel binding lists
    u32   flag;                      // observed 0/1 — likely indoor/outdoor
};

struct List {
    u32   count;
    Assignment items[count];
};

struct Assignment {
    u32   track_label_len;
    char  track_label[track_label_len];
    u32   value;                     // raw u32; reinterpret as float32 → volume (0.0..1.0)
};
```

Empirically:
- **L3** holds *day* ambient bindings (`GenDay01`, `Cellar`, `Cellar02`, ...).
- **L4** holds *night* ambient bindings (`GenNight`, `ForestNight`, ...).
- **L0** holds the primary music track (e.g. `4`, `forest`, `Hookers3`).
- **L1**, **L2** roles still inferential.

Validated by [`pkg/assets/musicdat`](../../pkg/assets/musicdat) —
all 153 regions parse byte-exact through to EOF (14133 bytes), with
`Aleroth`'s `L0 = [(4, 0.2), (forest, 0.2)]`, `L3 = [(GenDay01, 1.0)]`,
`L4 = [(GenNight, 1.0)]` matching the documented values.

## `sound\nsound.dat` — sound effect bank

Binary table of named SFX classes. Reader at
`div.exe:0x00549ac0` (`.\SOUND\DivSoundManager.cpp`). 1.6 MB in
shipped install. The file has **no count** — it's read until EOF;
each record begins with a 32-bit key.

```text
struct NSoundFile {
    SfxClass classes[];        // sequential, terminator = EOF
};

struct SfxClass {
    u32  key_a;                // composite-key part 1 (≤ 0x3fff)
    u32  key_b;                // composite-key part 2 (≤ 0x3fff)
    u32  key_c;                // composite-key part 3 (≤ 0xf)
    u32  field_14;             // typically 1 / 0
    u32  flags;                // `field_20 == 1` becomes a bool flag
    u32  pad_a;
    u32  pad_b;
    u32  category;             // `field_24` packed: bit 0 = (field_58==1), bit 1 = (field_38==1)
    u32  field_28;             // typically priority
    u32  type_tag;             // 1, 2, or other — sub-record size differs per type
    u32  field_30;             // bool — true iff value == 2
    u32  field_31;             // bool — true iff value == 2
    u32  field_34;
    f32  base_gain;            // field_18 — defaults to 0.0 if fread fails
    u32  variant_count;        // field_38
    Variant variants[variant_count];
    // if type_tag == 2: one extra trailing Variant slot is allocated to
    //   hold a cumulative-weight running total (used by the variant
    //   selector to do weighted random pick); not in file but materialised
    //   from the per-variant weights.
};

struct Variant {
    char  path[];              // NUL-terminated path string (XOR-obfuscated)
                               //   e.g. "\\WAV\\Impact & Swoosh\\Bodyfall01Slide.wav"
                               //   resolved to a u32 file handle by FUN_0054eaa0
    f32   param_a;             // observed: per-variant pitch / random-pitch range
    f32   param_b;             // f3 — secondary param
    f32   param_c;             // f4 — likely an angle (passed through sin/cos)
    f32   weight;              // ONLY present if type_tag == 2 — relative pick weight;
                               //   the engine builds a cumulative array and picks via
                               //   uniform-random-into-CDF.
};
```

The composite 32-bit key is rebuilt via:

```text
sound_key = (key_a & 0x3fff) | ((key_b & 0x3fff) << 14) | (key_c << 28)
```

So it's a 14+14+4 bit pack, capping at 16384/16384/16. Story scripts
fire SFX by looking up this key.

## `sound\reverbregions.dat` — reverb-zone → preset map

Plaintext, DOS line endings (`\r\n`), one entry per line:

```text
object {REGION_<name>, <preset_index>, (<preset_index>, <variant>, <unused>, <unused>)}
```

Example:

```text
object {REGION_Sewers1Map1, 7, (7, 0, 0, 0)}
object {REGION_PetCave,     7, (7, 1, 0, 0)}
object {REGION_IonasDung1,  7, (7, 2, 0, 0)}
```

The first integer is the preset index into `reverbs.dat`'s preset
table; the variant in parens picks one of the named regions in that
preset's region list (see below).

## `sound\reverbs.dat` — reverb preset table

Count-prefixed binary file. The shipped install has 8 presets.

```text
struct ReverbsFile {
    u32     preset_count;       // 8 in shipped install
    Preset  presets[preset_count];
};

struct Preset {
    u32   name_len;
    char  name[name_len];       // NOT NUL-terminated (e.g. "Cave", "CellarCave")
    u32   flag;                 // 0x08 — observed in every preset
    u32   parameters[12];       // 12 × 32-bit fields; mix of i32 / f32 per slot.
                                //   Map roughly to FSOUND_REVERB_PROPERTIES:
                                //   Room (i32), RoomHF (i32), Reserved, DecayTime (f32),
                                //   DecayHFRatio (f32), Reflections (i32),
                                //   ReflectionsDelay (f32), Reverb (i32),
                                //   ReverbDelay (f32), HFReference (f32),
                                //   ScaleHF (f32), ScaleLF (f32) — exact slot
                                //   ordering verifies against FMOD's preset table.
    u32   region_count;
    Region regions[region_count];
};

struct Region {
    u32   name_len;
    char  name[name_len];       // e.g. "REGION_AlerothDungeon1"
};
```

Validated by [`pkg/assets/reverbs`](../pkg/assets/reverbs): 8 presets,
first is "Cave" with 19 regions, second is "CellarCave".

The 8-float reverb-parameter block matches FMOD's `FSOUND_REVERB_*`
preset struct (the values like `0xfffffcd8` in the dump are signed
int reinterpretations of typical reverb parameters).

## Loaders

```text
div.exe:0x00547d40   FUN_00547d40   sound manager init — opens nsound.dat + music.dat
div.exe:0x0054a380   FUN_0054a380   reverb config init — opens reverbs.dat + reverbregions.dat
```
