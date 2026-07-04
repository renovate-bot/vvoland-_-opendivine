# Item registry — `itemstat\itemlink.dat` + `itemstat\itemstat.txt`

How an item **referenced by name** resolves to a concrete item type. When
an [Osiris](../osiris.md) rule says "give the player `IronSword`", a
[treasure table](treasure.md) lists a loot entry, or an
[egg](eggs.md) spawns a monster carrying gear, the item is named by a
**string**. The item registry turns that string into the numeric item
**type** that indexes the [`CItemStatistic`](../items.md) stat table. It is
the name→type indirection layer under the whole item economy.

## The two `itemstat\` files

Loaded **together** at boot (the world-load `fcn.00499990`, `init.cpp`)
by **`fcn.0057b530`**, which is handed both paths at once:

| File | Content |
|---|---|
| `itemstat\itemlink.dat` | the **name → type index** — the lookup table |
| `itemstat\itemstat.txt` | the per-type **stat definitions** (the [`CItemStatistic`](../items.md) schema: damage dice, defense, slot, magic, requirements, …) |

`fcn.0057b530` reads them as text (`fopen "rt"` + `fgets`); a malformed
`itemstat.txt` row logs `"Unknown error parsing itemstat.txt"`, and a bad
link logs `"Failure parsing %s (probably wrong itemlink) (File %s)"`. So
`itemlink.dat` assigns each item name its type id, and `itemstat.txt`
fills in what that type *is*.

## The registry — `[0x750cfc]`

The result is a static registry instance at **`0x750cfc`**:

| Field | Meaning |
|---|---|
| `+0x04` | record **count** |
| `+0x08` | base of the record array |

Each record is **8 bytes**: `{ +0x00 = item type id (int), +0x04 = name
(string pointer) }` — i.e. `itemlink.dat` is a flat list of
`(name, type)` pairs.

## The resolver — `fcn.005799b0`

The universal "look up an item by name". `this = 0x750cfc`; it walks the
`+0x08` array (`[+0x04]` entries, stride 8) and `stricmp`s the query
against each record's `+0x04` name, returning the record's `+0x00` **type
id** on a match, or **`-1`** if the name is unknown (→
`"Item %s is undefined in itemlink.dat"`).

This is the same case-insensitive name-scan idiom used by the other
by-name registries in the engine; the returned type id then indexes the
`itemstat.txt`-built [`CItemStatistic`](../items.md) table for the actual
properties.

## Who resolves item names

`fcn.005799b0` has ~10 callers — every place items are named rather than
typed:

- **[Osiris](../osiris.md)** — `CDIVINITYOsirisObjectFunction::virtual_0`
  (story commands that create/give an item by name).
- **[Treasure](treasure.md)** — the loot roll `fcn.005967a0` and the table
  loaders resolve each entry's item-overrule name; an unknown one logs
  `"Item overrule %s not defined in itemlink.dat for type %d"`.
- **[Item generation](itemgen.md)** — `fcn.005840f0` (the magic-item
  generator resolves its base item by name).
- **[Eggs](eggs.md)** — monster loadout items named in the spawn data.

So `itemlink.dat` is the single chokepoint that keeps item references in
the data files human-readable while the runtime works in compact type ids.

## Status

- File pair ✅ — `itemstat\itemlink.dat` (name→type) + `itemstat\itemstat.txt`
  (stat defs), loaded together by `fcn.0057b530` at boot (`fcn.00499990`),
  parsed as text.
- Registry ✅ — static instance `[0x750cfc]`: `+0x04` count, `+0x08` array
  of 8-byte `{type id, name}` records.
- Resolver ✅ — `fcn.005799b0`: case-insensitive name scan → type id or
  `-1`; consumed by Osiris / treasure / item-gen / eggs.
- `itemlink.dat` on-disk line grammar ✅ (token structure recovered) — the
  line parser is `fcn.0057b360` (`.\WORLD\ItemLink.cpp`): the file text is
  split into lines on `"\n"` (`fcn.004fdc40`), each line is tokenised
  (`fcn.004fdf40`), and a registry record is built from **two fields — a
  numeric type id (`atoi` of one token) and the item name (`strdup` of the
  other)** — stored as the 8-byte `{ +0 type, +4 name }` record. So each
  line is a whitespace-separated `(type id, name)` pair. Residual: the
  exact column *order* (`<id> <name>` vs `<name> <id>`) can't be pinned
  from the binary alone (the two token buffers are adjacent and no
  `itemlink.dat` ships loose to confirm), so it stays a micro-detail 🟡;
  the field set and parse path are settled.

## Citations

```text
div.exe:0x005799b0   fcn.005799b0   item name→type resolver (stricmp scan of [0x750cfc], -1 if unknown).
div.exe:0x0057b530   fcn.0057b530   loader for itemlink.dat + itemstat.txt (text, fgets).
div.exe:0x0057b360   fcn.0057b360   ItemLink.cpp line parser — split on "\n" (fcn.004fdc40), tokenize
                                    (fcn.004fdf40), atoi(type) + strdup(name) → 8-byte {type,name} record.
div.exe:0x00499990   fcn.00499990   world-load: passes itemstat\itemlink.dat + itemstat.txt to fcn.0057b530.
div.exe:0x00750cfc   item registry (static): +0x04 count, +0x08 array of 8-byte {type, name}.
str: itemstat\itemlink.dat · itemstat\itemstat.txt · "Item %s is undefined in itemlink.dat"
str: "Item overrule %s not defined in itemlink.dat for type %d" · "Failure parsing %s (probably wrong itemlink)"
```
