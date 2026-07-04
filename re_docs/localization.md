# Localization / text (`.\MISC\translator.cpp`)

How the game serves **localized text and voice**. All player-facing
strings, dialogue, hints, and voice-over are pulled from a per-language
`localizations\<lang>\` directory, resolved through the translator.

## The `localizations\` layout

The configured language name (e.g. `German`) is the `%s` in these paths:

| File | Content |
|---|---|
| `localizations\%s\text.cmp` | the main **UI string table** (the bulk of menu / HUD / system text) |
| `localizations\%s\voice.cmp` | dialogue **voice-over** audio ([`sound-runtime.md`](sound-runtime.md)) |
| `localizations\%s\mono.cmp` | monologue / bark voice-over ([`monologues.md`](monologues.md)) |
| `localizations\%s\skills.txt` | localized skill names / descriptions ([`skills-magic.md`](skills-magic.md)) |
| `localizations\%s\hints.txt` | loading-screen / tooltip hints |

Each has a **non-localized fallback** one level up (`localizations\text.cmp`,
`localizations\hints.txt`) used when no per-language file exists. The
`.cmp` files are the standard packed-archive format ([`formats/cmp.md`](formats/cmp.md));
the branching dialogue text is a separate `dialogtxt.dat` (UTF-16,
[`dialogue.md`](dialogue.md)).

## The translator (`.\MISC\translator.cpp`)

`fcn.00504d50` / `fcn.00504e00` / `fcn.00505020` / `fcn.00505100` are the
translator. The UI string table is **lazy-loaded once** — `fcn.00504e00`
gates on the flag **`[0x744780]`** (`test … 1` / `or … 1`) and `fread`s
`text.cmp` into memory on first use, after which logical string ids resolve
to the loaded localized strings. `fcn.00504d50` builds/escapes a string
(handling embedded `\r`/`\n`). The `language #` script/console verb selects
the active language; an unset one logs `"Empty language field"`.

## ANSI vs Unicode build

The engine warns **"The language you selected required a unicode version of
Divine Divinity. You are currently running an ansi version and a lot of
text might be shown wrong."** — so there are two builds: an **ANSI** build
(the common shipped `div.exe`, single-byte code pages) and a **Unicode**
build for languages that need wide characters (the relevant import is
`VerLanguageNameA`). A faithful port should just treat all text as UTF-8/16
and skip this split.

## Status

- Layout ✅ — `localizations\<lang>\{text,voice,mono}.cmp` +
  `skills.txt`/`hints.txt`, with a non-localized fallback; `.cmp` = packed
  archive ([`formats/cmp.md`](formats/cmp.md)).
- Translator ✅ — `.\MISC\translator.cpp` (`fcn.00504e00`); `text.cmp` is
  lazy-loaded once (flag `[0x744780]`) and id→string resolved; `language #`
  selects the language.
- ANSI/Unicode split ✅ (noted) — two builds; a port can ignore it (use
  UTF).
- `text.cmp` string-table record format 🟡 — the per-string id/offset
  layout inside the `.cmp` not split field-by-field.

## Citations

```text
div.exe:0x00504e00   fcn.00504e00   translator — lazy-load text.cmp (flag [0x744780]).
div.exe:0x00504d50   fcn.00504d50   translator — string build/escape (.\MISC\translator.cpp).
str: localizations\%s\text.cmp · \voice.cmp · \mono.cmp · \skills.txt · \hints.txt
```
