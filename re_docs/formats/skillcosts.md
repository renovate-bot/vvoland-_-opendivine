# `dat\wgiaa.000` & `dat\skillcosts.txt` — the skill-cost table

The per-skill **skill-point cost table**: how many points it costs to raise
each skill to each of its five ranks. The data lives in two paired forms
owned by `.\Skills\skills.cpp`:

| File | Form | Role |
|---|---|---|
| `dat\wgiaa.000` | binary, **shipped** | the loaded/cached cost table (97×40 B) |
| `dat\skillcosts.txt` | CSV text, **not shipped** | the human-editable import source |

At runtime the engine reads the binary `wgiaa.000`; `skillcosts.txt` is the
(dev-time) import that regenerates it and is absent from the shipped data.

## `wgiaa.000` — binary layout (byte-exact)

```text
Record[97]:                  (40 bytes — 10 × int32, little-endian)
    i32 cost[5]              skill-point cost to reach ranks 1..5
    i32 reserved[5]          always 0
```

`97 × 40 = 3880` = exact file size. The record is indexed by **skill id**
(0-based), matching the 96-skill tree ([`../skill-tree.md`](../skill-tree.md),
[`../skills-magic.md`](../skills-magic.md)):

- **Record 0** = all `-100` (`0xffffff9c`) — the sentinel for the reserved
  "no skill" slot 0.
- **Records 1..95** = the real skills. 91 carry the five ascending per-rank
  costs in `cost[0..4]`; the other slots are all-zero (undefined skills).
  `reserved[5..9]` is zero in every real record.
- **Record 96** is past the valid range (the loader bounds-checks the skill
  id `< 96`, `0x542922 cmp eax,0x60`) and holds uninitialized slop
  (`0x6e6f7473` = `"ston"` repeated) — not a cost record.

Verified samples (cols 0..4):

```text
skill 1   2  8 14 20 26      skill 2  10 12 14 16 18
skill 3  21 24 27 30 33      skill 5   4  6  8 10 12
skill 40  6  8 10 12 14      skill 95 24 27 30 33 36
```

So each skill defines a strictly increasing cost ladder over its five
ranks; raising a skill from rank `r-1` to `r` costs `cost[r-1]` skill
points.

## `skillcosts.txt` — CSV import format

Parsed by `fcn.00542760` (`fopen "rt"`, `fgets` 2048-byte lines). Each line
is comma-separated (`","` = `0x62b3c0`): a **skill id** followed by its
**five rank costs** (the splitter advances five times, `cmp edi,5`), each
`atoi`-converted. The id is range-checked against `0..95` (`cmp …,0x60`); a
malformed file logs `"Error importing skill costs from %s"`. The five
values land in `cost[0..4]` of that skill's record, leaving `reserved`
zero — exactly the binary layout above.

## Code

```text
div.exe:0x00611020   "dat\wgiaa.000" path literal.
div.exe:0x00542980   wgiaa.000 writer — fwrite(this+0x38, size=40, count=97), "wb".
div.exe:0x00543450   skills.cpp init — fread wgiaa.000 (rb) into +0x38, then import skillcosts.txt.
div.exe:0x00543540   skills.cpp init variant — same fread + import.
div.exe:0x00542760   skillcosts.txt CSV parser; 5 cost columns; "Error importing skill costs from %s".
div.exe:0x004996c5   boot/export path — re-writes wgiaa.000 after the table is loaded.
```

The in-memory table is the 97×40-byte array at **`+0x38`** of the skills
manager object (the `fwrite`/`fread` base). The cost array feeds the skill
trainer / level-up point spend ([`../progression.md`](../progression.md));
the *point pool* it is charged against is the dynamic level-up state and is
not part of this file.

## Status

- `wgiaa.000` ✅ — fully decoded, byte-exact to EOF (97×40 B, 10×int32,
  `cost[5]`+`reserved[5]`); record 0 = `-100` sentinel, records 1..95 the
  five-rank cost ladders, record 96 trailing slop. Directly reimplementable.
- `skillcosts.txt` ✅ — CSV import format (id + 5 costs), bounds and error
  path recovered from `fcn.00542760`. Not shipped, so verified from the
  binary rather than a sample.
