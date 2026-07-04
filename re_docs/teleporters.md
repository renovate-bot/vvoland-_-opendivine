# Teleporters (`CTeleportPlate`, `.\plate\TeleportPlate.cpp`)

The fast-travel / scripted-teleport network. A teleporter is a **plate**
object placed in the world; its destination is read from a text table at
boot. This doc covers the data format, the runtime class, and how a plate
fires.

## Data file — `dat\teleporter.txt` (text)

Loaded once at boot as the **"Loading teleporter list"** step of
`.\GAME\init.cpp` ([`architecture.md`](architecture.md)) — driver
`fcn.00530700`, which opens `dat\teleporter.txt` in text mode (`"rt"`) and
hands each line to the per-entry parser `fcn.00530490`. (A second loader
`fcn.00530220` re-reads it, and `fcn.0052fbd0` / `fcn.0052fee0` are the
binary save/restore counterparts that `fopen … "rb"`.)

The parser tokenises each line (`fcn.004f6270` = next whitespace token,
`atol` for the integer columns) into a **0x180-byte record** (allocated by
`fcn.004fa4b0`). Verified field stores:

```text
struct TeleporterEntry {            // 0x180 bytes
    u32  name_id;        // +0x00 — first token is a NAME, interned through
                         //         the string registry [0x750d2c]
                         //         (fcn.0057bf90) to a dense id stored here
    …
    u32  field_18;       // +0x18 — initialised 0
    u32  field_1c;       // +0x1c — initialised 0xF (15) — default/sentinel
    i32  param1;         // +0x20 — atol of token 2  (teleport target X *)
    i32  param2;         // +0x24 — atol of token 3  (teleport target Y *)
    u32  field_28;       // +0x28 — initialised 0
    i32  param3;         // +0x2c — atol of a later token (destination
                         //         map / region id *)
    …                    // +0x30+ runtime sprite/render state (below)
};
```

(`*` the three integers are the destination parameters; that they are
target-X / target-Y / destination-map is inferred from the plate's
walk-on behaviour, not yet pinned to the exact teleport call — see
*Firing* below. A middle token between `+0x24` and `+0x2c` is passed to the
path/name builder `fcn.0041d800`, so one column is a string, e.g. a
destination map name.) Each parsed entry is inserted into the teleporter
table via `fcn.00480c10`.

## Runtime class — `CTeleportPlate`

A teleporter in the world is a `CTeleportPlate` object (vtable at
**`0x619de4`**, ctor `fcn.0052f720`), constructed through the object-class
factory `fcn.0052fa00`, which registers the class name **`"TeleportPlate"`**
and its automap icon (`automap.bmg`, so teleporters show on the automap,
[`formats/automap.md`](formats/automap.md)). The ctor zero-inits
`+0x20..+0x2c` and chains the base `CObject` ctor (`fcn.0045be90`); the
destination parameters above are filled from the `teleporter.txt` entry.

Overridden virtuals (the slots pointing back into the `0x52f…` unit):

| Slot | Off | Fn | Role |
|---:|---|---|---|
| 0 | `+0x00` | `fcn.0052f9e0` | scalar-deleting dtor (`fcn.0052f4a0` + free) |
| 6 | `+0x18` | `fcn.0052f250` | tiny accessor |
| 13 | `+0x34` | `fcn.0052f2e0` | thunk → base draw (`call [vt+?]`) |
| 14 | `+0x38` | `fcn.0052f260` | **render** the plate |

`fcn.0052f260` (`virtual_56`) draws the plate: it blits the base sprite
object at `+0x30` (via its own `vtable[+0x38]`), and when `+0x3c != -1`
draws a second **active/overlay** sprite (icon id at `+0x3c`) at a fixed
pixel offset (`+0xe1` x, `+0x168` y) resolved through the shared sprite
lookup `fcn.004e9850`. So `+0x30` = base sprite, `+0x3c` = overlay/active
sprite id.

## Firing

`CTeleportPlate` is registered as a placeable object class, so a plate is
just a world object sitting on the floor. Stepping onto it triggers it
through the normal **walk-on object** path (`sb_walk_on`, the static
object bit in [`formats/objects.md`](formats/objects.md) / the Use dispatch
in [`object-interaction.md`](object-interaction.md)) — the handler reads
the plate's destination parameters (`+0x20`/`+0x24`/`+0x2c`) and relocates
the agent via the `CAgent` teleport virtual (`vtable+0xc`, the
teleport/relocate slot noted in [`agent.md`](agent.md)). The exact
walk-on→teleport call site is the remaining link 🟡.

## Status

- Data file ✅ — `dat\teleporter.txt`, text, parsed by `fcn.00530490`
  into 0x180-byte entries; loader is the boot step `fcn.00530700`.
- Record fields ✅ (offsets) — `+0` interned name-id, `+0x1c` default 15,
  `+0x20`/`+0x24`/`+0x2c` the three integer parameters; semantics
  (target X/Y / destination) inferred 🟡.
- Class ✅ — `CTeleportPlate` (`.\plate\TeleportPlate.cpp`), vtable
  `0x619de4`, ctor `fcn.0052f720`, factory `fcn.0052fa00`, automap icon.
- Render ✅ — `virtual_56` `fcn.0052f260`: base sprite `+0x30` + overlay
  `+0x3c` at pixel offset `(+0xe1, +0x168)`.
- Firing 🟡 — walk-on → `CAgent` teleport (`vtable+0xc`); exact call site
  not yet isolated.
- Per-save activation state ✅ — which plates the player has unlocked is
  persisted in `telpstates.000`, a `{u32 count}{u32 id; u32 state}[]` list
  read by `fcn.0052f680` (slot resolved via `fcn.0052f430`, `base[id*4]`).
  See [`formats/savegame.md`](formats/savegame.md).

## Citations

```text
div.exe:0x00530700   teleporter-list loader — opens dat\teleporter.txt "rt", per boot step.
div.exe:0x00530490   per-entry parser — tokenise + atol into the 0x180-byte entry.
div.exe:0x0052f720   CTeleportPlate ctor — sets vtable 0x619de4, zero-inits +0x20..+0x2c.
div.exe:0x0052fa00   object-class factory — registers "TeleportPlate" + automap.bmg icon.
div.exe:0x0052f260   CTeleportPlate::virtual_56 — plate render (base +0x30 / overlay +0x3c).
div.exe:0x00619de4   vtable.CTeleportPlate.
div.exe:0x00750d2c   string/name registry used to intern the entry name (fcn.0057bf90).
```
