# Misc small data files

Small standalone data files in the game root, each fully decoded
against the shipped copy.

## `divinityevent.dat` — input-action names

Plain ASCII: 20 `"quoted"` bindable UI/input action names, one per CRLF
line. **Full list + the `keylist.txt` 1-based-index binding is in
[config.md](config.md)** (its natural home, alongside keybindings) — kept
there to avoid duplication.

## Image / palette files — standard BMP (no custom format)

`00000000.016` / `00000000.256` (and similar) in the game root are plain
**Windows BMP** files (`BM` magic + `BITMAPINFOHEADER`; `.016` = 4-bpp,
`.256` = 8-bpp paletted) — loading/static images, not a Larian format.
No reverse-engineering needed; read with any BMP loader.

## `persist.dat` — persistent entry table

```text
+0x00   u32   count                 — 38 in the shipped file
        Entry[count]:               — 8 × i32, 32 bytes each
            i32 type;    // +0x00  1 or 2
            i32 id;       // +0x04  unique id (e.g. 8758, 8763, 8764, …)
            i32 one;      // +0x08  always 1
            i32 zero;     // +0x0c  always 0
            i32 datasize; // +0x10  payload byte count (8 in every shipped entry)
            i32 x;        // +0x14  payload[0] — X-like coordinate (100..1871, steps of ~49)
            i32 y;        // +0x18  payload[1] — Y-like coordinate (28..979)
            i32 zero2;    // +0x1c  always 0
```

`4 + 38 × 32 = 1220` = the exact file size. ✅ Each entry is a typed,
uniquely-id'd item carrying a `datasize`-byte payload — here a fixed 8-byte
`(x, y)` placement; the descending-X runs are a laid-out row of elements.

**Consumer ✅ — `.\MISC\Persist.cpp`, a generic session-persistence
manager.** The file is read by `fcn.004fe640` (`"rb"`) and written by
`fcn.004fe910` (`"wb"`), both tagged `".\MISC\Persist.cpp"`. The manager
object holds two parallel `count`-length arrays (`+0x18`, `+0x1c` — the
per-entry value arrays) plus the entry count at `+0x04` and the declared
payload size at `+0x24`; the writer emits each entry field-by-field and
guards the payload with `"Persist: DataSize = %d, skipping"` (so an entry
whose declared `datasize` exceeds the slot is dropped — confirming the
`+0x10` field is a payload length, not a constant). `persist.dat` is
**registered in the same config/persistence group as the renderer configs**
(`fcn.00501c20` registers it via `fcn.005018d0` right after `slashed.cfg`,
`slashed-d3d6/-directx/-glide/-software.cfg`, `init.cfg`), so it is the
UI-layout/state blob the engine rewrites on exit and reloads at startup —
the persisted positions of the draggable "Plate" UI elements (cf. the
`*Plate` action names in `divinityevent.dat`, [`config.md`](config.md)).

## `mapids.000` — map id stamp

A single `u32` (`0x01b67819` in the shipped file) — a map/version id
stamp, no record structure.

## Status

- `divinityevent.dat` ✅ — full list in [config.md](config.md).
- `persist.dat` ✅ (format) — `u32 count(38)` + 38 × 32-byte records
  `{kind(1/2), id, 1, 0, type=8, x, y, 0}`, byte-exact (4+38×32=1220);
  38 persistent positioned objects (id + kind + world x,y). The meaning
  of *what* persists 🟡.
- `mapids.000` ✅ — single `u32` id stamp.
- Image/palette files (`.016`/`.256`) ✅ — standard Windows BMP, not a
  custom format.

> **Static data-format catalog: essentially complete.** A sweep of the
> shipped data files finds the remaining ones either already documented
> (see [README](../README.md) formats list) or standard formats (BMP).
> Genuinely-fresh *static* files are exhausted; further depth now lives
> in **code** (engine logic) rather than new file formats.
