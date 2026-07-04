# `*.fnt` — bitmap fonts

The engine's UI fonts (`fonts\*.fnt`, `.\font\Font.cpp`) — bitmap fonts:
a glyph atlas plus per-glyph metrics, rendered for all in-game text
(menus, dialogue, status plate). 61 ship (sizes/colours, e.g.
`dialog_red.fnt`, `fontor5.fnt`).

## Layout

```text
+0x00  char  magic[11] = "ML_FONT_01\0"
+0x0b  u8    version   = 1
+0x0c  u16   0
+0x0e  u32   = 9          (line height / baseline metric)
+0x12  u32   = 124        (glyph count / max char code)
+0x16  …     (132-byte zero region — sparse per-char presence/metric slots)
+0x9a  u32   glyph[ ]     per-glyph table: a u32 each (atlas offset / packed
                          metric); ~124 entries, values run into the atlas
                          and up to near EOF (e.g. 21155 in a 21734-byte file)
…      u8    atlas[]      8-bit pixels: 0 = transparent, non-zero = ink/alpha
                          (dominant 0/1/2 and 0xe5/0xfb/0xff — anti-aliased)
```

The glyph table at **`+0x9a` (154)** is an array of `u32`s (offsets into
the 8-bit atlas / packed metrics); most low char codes are `0` (unused),
the printable range carries real values. The atlas is 8-bit
anti-aliased glyphs over transparency. The exact per-glyph record
(rect + advance-width packing) is not fully split out — but this format
is **UI-only and the Go port uses its own fonts**, so it is documented
for completeness, not needed for the port.

## Status

- Identity ✅ — `ML_FONT_01` bitmap-font container; 8-bit anti-aliased
  glyph atlas.
- Header ✅ — `magic[11]` + `u8 version=1` + `u16 0` + `u32=9`
  (line/baseline metric) + `u32=124` (glyph count/max code).
- Glyph table ✅ (located) — at `+0x9a`, a `u32`-per-glyph array of atlas
  offsets/packed metrics (~124 entries, sparse for unused codes); the
  exact rect+advance packing left unsplit (low priority).
- Priority — **low**: UI-only. The OpenDivine Go port renders text with
  its own fonts ([`internal/game/mainmenu/font.go`]), so this format is
  documented for completeness, not for the port.
