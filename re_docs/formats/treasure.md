# `dat\treasure.cmp` — treasure (loot) tables

The loot system's compiled data: a set of named **treasure tables**,
each a weighted list of item drops, that monsters and containers roll
against. Unlike the string-indexed `.cmp` archives ([`cmp.md`](cmp.md)
Family A), this file is a **serialized in-memory tree** — it was dumped
straight from the heap, so it carries baked runtime pointers rather than
file-relative offsets. (`cmp.md` flags treasure.cmp as neither Family A
nor B and points here.)

All integers little-endian.

## Top level

```text
u32   name_len + "Divinity Treasure tables V1.0\0"   (length-prefixed magic/version)
u32   count                                          — number of treasure tables (89)
u32   table_ptr[count]                               — baked 32-bit runtime pointers
…     table data                                     — the pointed-to table structures
```

The shipped file: `count = 89`; the table data begins at file offset
`394` (right after the pointer array).

### Baked pointers

The `table_ptr[]` entries are runtime heap addresses from when the table
was serialized, **not** file offsets. For the tables laid out
contiguously, a pointer maps to a file offset by a single fixup base:

```text
file_offset = ptr − (table_ptr[0] − 394)        // base = 0x057a79f6 in the shipped file
```

This resolves the leading tables, but **not all** pointers convert: per
`cmp.md`, some pointer slots are reused as **occupancy flags**
(non-null = "this slot is present") rather than real addresses, which is
why the array is non-monotonic and some values fall outside the file.
Distinguishing real pointers from flags needs the loader.

## Treasure table record

Each table (e.g. the first one resolves cleanly) is:

```text
i32   header[4]              // e.g. {995, 1, 2, 0} — table id / flags
u32   name_len + name        // length-prefixed, e.g. "base"
u32   entry_count            // e.g. 34
u32   entry_ptr[entry_count] // baked pointers to the drop entries
```

So the structure is a **three-level tree**: `tables[89] → table{name,
entries} → entry → item-drop definition`. Table 0 is named **`base`**
(the default drop table) and holds **34** entries.

## Status

- Top level ✅ — version-string header + `u32 count(89)` + baked
  pointer array; data starts at offset 394.
- Pointer fixup ✅ (table level) — verified against the **shipped
  `dat\treasure.cmp`** (`"Divinity Treasure tables V1.0"`, 89 tables, data
  at offset 394): the single-base fixup `file_off = ptr − (ptrs[0] − 394)`
  (base `0x057a79f6`) resolves the **table-header** pointers (43/89 land in
  the file; the rest are occupancy/empty slots). Table 0 reads exactly as
  documented — `header = [995,1,2,0]`, name `"base"`, `entry_count = 34`.
- Table record ✅ (verified) — `{header[4], name, entry_count, entry_ptr[]}`;
  table 0 = `"base"` / 34 entries, confirmed byte-for-byte from the
  shipped file.
- **Why the leaf 🟡 is bounded** — the per-table **`entry_ptr[]` values
  point into *separately-allocated* heap regions** (a different base than
  the table headers): with the table base they resolve out-of-file. So
  `treasure.cmp` is a **multi-base heap dump** — walking the 148-byte
  entries statically would require reconstructing the original allocator's
  several base addresses, which only the loader has. This is *why* the leaf
  fields below are pinned from the **code** (the roll/loader reading them),
  not from the data file — and why a full static byte-walk of the entries
  is not feasible from the dump alone.
- Full table enumeration 🟡 — only the cleanly-converting tables are
  walked so far; the per-table header layout varies.
- Drop-entry record ✅ (superseded below) — the `148-byte (0x94)` /
  `base[+0x1c] + i*0x94` figure here describes the **table** array level,
  **not** the drop-entry: the actual `CTreasureBaseTable` entry is a
  416-byte (`0x1a0`) heap object and the drop-record leaf is 28 bytes (see
  the loader-confirmed field maps below). Two **roll-critical fields** are
  pinned at
  `+0x0c` and `+0x10` (both `u32`, read by the entry loader `fcn.00594fc0`
  via `fread` `fcn.004f4c70`). *(Correction, refined: `fcn.005918b0` is
  **not** an "item creator" **nor a category enum** — it is a
  **variable-width bitmask field decoder** (277 callers). The table at
  **`0x655a98`** is **32 `int32`s, one per mask bit, each `∈ {0,1,2}` = the
  byte width of that bit's payload**. Given `(mask, bit)` it computes
  `offset = Σ table[i]` over the *lower* set bits of the mask, then reads
  `table[bit]` bytes at that offset: `0` = field absent, `1` = a `u8`,
  `2` = a **big-endian `u16`** (`(hi<<8)|lo`). The `cmp ecx,1`/`cmp ecx,2`
  branch is on that width. The full width table:*
  `[0]2 [1]1 [2]1 [3]2 [4]2 [5]2 [6]2 [7]0 [8]2 [9]0 [10]2 [11]1 [12]2
  [13-15]0 [16]2 [17]1 [18]0 [19]2 [20]2 [21]0 [22]0 [23]2 [24-25]0
  [26]2 [27]2 [28-29]0 [30]2 [31]2`. *So the roll's `fcn.005918b0(+0xc,
  +0x10)` decodes a variable-width drop-**kind** field packed under a
  presence bitmask (`+0xc` = mask, `+0x10` = the packed bytes). The actual
  item **instantiation** is the separate `fcn.005919e0`.)*
  The remaining ~136 bytes hold the chance/weight + value-range sub-data
  (not split field-by-field). The *table* loader `fcn.00595630` reads the
  table's `+0x10`/`+0x14`/`+0x18` u32s and its name (`+0x0c`, via
  `fcn.004f4d10`) then calls `fcn.00594fc0` for the entries.
- Roll logic ✅ (located) — `.\WORLD\treasure.cpp` roll is
  **`fcn.005967a0`**: it **selects an entry by chance/weight**
  (`fcn.00595830` iterates the table's `count` (`table+4`) entries,
  testing each via `fcn.005954b0` with MSVCRT `rand`), classifies the
  entry's drop **kind** via the bit-resolver `fcn.005918b0(entry+0xc,
  entry+0x10)` (above), then **instantiates** the dropped object via
  `fcn.005919e0`. (treasure loaders: `fcn.005948a0`, `fcn.00595630`,
  `fcn.00595970` — the last opens the file `"rb"`.)
- **Value-range roll ✅ (recovered statically from `fcn.005954b0`)** —
  resolves part of the "~136 bytes of value-range sub-data" above. The
  selector `fcn.005954b0` receives the entry, walks the **sub-array of
  drop-records hung off it** (`base = entry[+8]`, `count = entry[+0xc]`,
  the array the entry loader `fcn.00594fc0` `fread`s into `entry+8` /
  `entry+0xc`), and **skips records whose `+0x04 == 0xffffffff`** (an
  invalid/terminator sentinel). For the chosen drop-record it computes the
  rolled amount as an **inclusive uniform range**:

  ```text
  amount = min + rand() % (max - min + 1)     ; min = rec[+0x14], max = rec[+0x18]
  ```

  with `rec[+0x10]` used as a further `rand` divisor (a secondary
  sub-chance) and the result written back through the caller's out-params.
  So the per-drop **quantity/value is a `[min, max]` uniform roll at record
  `+0x14`/`+0x18`** — the value-range sub-fields are no longer opaque.
- **Drop-record loader & layout ✅ (`fcn.00594810`, confirms the above
  independently)** — `fcn.00594fc0` reads the entry's record `count`, then
  loops allocating **`count` × 28-byte (`0x1c`)** drop-records (zeroing
  `+0xc`/`+0x10`) and loads each with `fcn.00594810`. That per-record loader
  reads, in order: scalars via `fcn.004f4c70` into **`+0x14` (value min)**
  and **`+0x18` (value max)** — the *same* offsets the roll
  (`fcn.005954b0`) reads as the range, so the value-range labelling is now
  confirmed from **both** the loader and the roll — a bulk field via
  `fcn.004f4c00` into **`+0xc`** (the kind-mask data fed to `fcn.005918b0`),
  and a length-prefixed name string via `fcn.004f4d10`. So the **leaf
  drop-record is a 28-byte struct** `{… +0xc kind-mask, +0x10 bit/divisor,
  +0x14 valMin, +0x18 valMax, name}`.

  **Level note (the earlier "remaining nuance"):** the `+0xc`/`+0x10` kind
  selector and the `+0x14`/`+0x18` value range live on this **28-byte leaf
  drop-record**, not on a single flat 148-byte record. The `0x94`-stride
  array (`base[+0x1c] + i*0x94`) noted above is a **distinct, higher level**
  (the table-entry container), each of which hangs a `+8`/`+0xc`
  count-prefixed sub-array of these 28-byte drop-records.

  **Container-entry field map ✅ (`fcn.00594fc0`, read-ctx `[0x6e0124]`):**
  the entry is a **`0x1a0` (416-byte) heap object** (allocated `push 0x1a0`
  in the table loop, vtable `CTreasureBaseTable` at `+0x00`), but
  `fcn.00594fc0` **reads only four fields from disk**: a 4-byte scalar into
  **`+0x04`** (entry id/flags), **`+0x0c` = drop-record count**, **`+0x10` =
  a scalar**, and bulk-builds the **`+0x08` = drop-record sub-array**
  (`count` × the 28-byte records loaded by `fcn.00594810` above). So the
  three treasure levels are mapped: **table** → container entries `{+0x00
  vtable, +0x04 id, +0x08 drop-record[], +0x0c count, +0x10 scalar}` →
  **28-byte drop-records** `{+0x04 valid, +0x0c kind-mask, +0x10 bit, +0x14
  valMin, +0x18 valMax, name}`.

  **Correcting the old "148-byte (0x94)" figure:** that was *not* the entry
  size — the `CTreasureBaseTable` entry is **416 bytes** and only `+0x04..
  +0x10` are serialized; the rest is **runtime state** (the drop-record
  container internals, etc.), *not* unrecovered on-disk data. So there is
  **no remaining on-disk gap at the entry level** — the serialized fields
  are all known. (`0x94`/148 referred to the higher **table** record, the
  `89`-table array level, not this entry.)

  **Middle level is `CTreasureBaseTable`** (vtable `0x6209b4`, ctor
  `virtual_0` = `fcn.00594f30`): the table loader **`fcn.00595630`** reads
  the table header scalars + **name** (via `fcn.004f4d10`) + entry **count**,
  then loops **`count`** times allocating a `CTreasureBaseTable` and loading
  it with `fcn.00594fc0` (which in turn builds that entry's 28-byte
  drop-record sub-array). So the table→entry→drop-record nesting is
  loader-confirmed: `fcn.00595630` (per table) → `fcn.00594fc0` (per
  `CTreasureBaseTable` entry) → `fcn.00594810` (per 28-byte drop-record).
- Instantiation = a **world object with semantic properties**, not a
  pre-filled inventory item — `fcn.005919e0` builds the dropped object and
  sets its [objects.000](objects.md) `s_*` semantic properties (its sub-
  setters `fcn.00591940`/`fcn.00591810`/`fcn.00591870`, with the bounds
  check `"Object semantics property overflow"`). So the treasure path
  produces an **object + its semantics**; the **item runtime-instance**
  mutable fields (identified / durability / charges / stack count) are
  *not* set here — they are applied when the object is taken into an
  [inventory](../inventory.md) (which is why the item-instance struct
  is a separate, pickup-side gap, [items](../items.md)).
- Treasure type → merchant stock + service costs ✅ — the merchant
  **setup** (`fcn.00514860`) resolves the treasure-table by **name →
  index** via `fcn.00594ed0` (`.\WORLD\treasure.cpp`; logs `Warning -
  treasure table %s does not exist` on miss) and stores that **table
  index at agent `+0x1b4`** — i.e. the merchant's *stock* table. The
  service costs are then computed and stored alongside: **`+0x1b8` /
  `+0x1bc` = Identify / Heal / Repair cost**, produced by the agenttrade
  price routine **`fcn.004372e0`** (same `MerchantPriceDifference`
  family as the buy/sell pricing, [`../items.md`](../items.md)), reading
  a trade sub-object at agent `+0x4b4`. The debug line `Treasure type =
  %d IdentifyCost = %d HealCost = %d, RepairCost = %d` (dumper
  `fcn.0042d230`) prints exactly these `+0x1b4..+0x1bc` fields. So
  "treasure type" doubles as the merchant's loot/stock table id *and*
  the basis for its service prices.
