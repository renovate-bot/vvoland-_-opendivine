# GUI (BMGUI plates & controls)

The in-game user interface: a widget toolkit (`.\BMGUI\`) of `CBM*`
**controls** composited into named **plates** (the status panel,
inventory window, automap, etc.). Lower priority for the OpenDivine
port (which builds UI on ebiten), but documented because several
gameplay paths surface through it — opening a chest spawns an
`InventoryPlate`, dialogue draws through the dialog oracles.

## Control toolkit (`CBM*`)

~27 RTTI control classes, an MVC-flavoured toolkit (the `TController` /
`TOracle` suffixes split controller from model/data-source, `View` is
the widget):

| Group | Classes |
|---|---|
| containers / forms | `CBMControl`, `CBMCustomControl`, `CBMControlSetControl`, `CBMSubwindowControl`, `CBMFormControl`, `CBMFloatFormControl`, `CBMSpaceFillControl` |
| buttons | `CBMTextButton`, `CBMClipButtonControl` |
| lists & scroll | `CBMListControl`, `CBMVerticalScrollControl`, `CBMSimpleVerticalScrollControl`, `CBMHorizontalScrollControl` |
| tabs / cycles | `CBMTabControl`, `CBMCycleTabControl`, `CBMCycleControl` |
| input | `CBMStringEditController`, `CBMSliderControl` |
| animation | `CBMAnimationControl`, `CBMAnimationSetControl` |
| text views | `CBMTextViewTController`, `CBMTextViewTOracle`, `CBMDualTextViewTController` |
| inventory | `CBMCInventoryViewer`, `CBMCFlatInventoryViewer` |
| dialogue views | `CBMDialogAnswerTVOracle`, `CBMDialogOptionsTVOracle`, `CBMMonologueTVOracle`, `CBMTalkLogViewOracle` |

Sources: `.\BMGUI\Bmgui.cpp`, `Bmcontrl.cpp`, `TextButton.cpp`,
`EditControl.cpp`, `EnergyBarControl.cpp`, `AnimationControl.cpp`.

## Plates (named windows)

A **plate** is a composited UI window, referenced by name and toggled
via `DIVINITYTogglePlate`. Shipped plates include:

```text
StatusPlate          character stats panel (layout in statuspl.cmp, see formats/cmp.md)
ShieldPlate          equipment/shield panel
InventoryPlate       the inventory grid — "Internal InventoryPlate %d" is spawned
                     when a container/chest is opened (object-interaction.md)
MinimapPlate / AutomapInterfacePlate{640,800,1024}   map UI (per resolution)
UserNotePlate        the player's automap notes (usernotes.bin)
```

The energy-slot / button plates on the HUD (`Combat`, `Diary`,
`Skills`, `Inventory`, `Chest` buttons, etc.) are `CBM*` controls laid
out on these plates; their static positions for the status plate come
from `statuspl.cmp` (30 × 12-byte records, [`formats/cmp.md`](formats/cmp.md)).

### Plate classes (`C*Plate` hierarchy)

Behind the named plates is a **20-class `C*Plate` C++ hierarchy** (RTTI),
all deriving from the base **`CControlPlate`** (vtable `0x60d134`, ctor
`fcn.0044da10`). Each is the behaviour/glue object that wraps a `.bmg`
layout (below) and reads game state to draw a panel:

| Role | Classes |
|---|---|
| base / generic control | `CControlPlate`, `CSpecialControlPlate`, `CClipControlPlate` |
| HUD status | `CStatusPlate` (player stats), `CShieldPlate` / `CEnergySlotPlate` (equipment & spell slots), `CMinimapPlate` (map), `CMonsterInfoPlate` (target readout) |
| text / dialogue | `CMessagePlate` (message log), `CNPCTextPlate`, `CMonologuePlate`, `CDualDialogPlate` |
| windows | `CTradePlate` (shop), `CCharmPlate` (charm), `CSplitPlate` (stack split), `CBookPlate` / `CTextBookPlate` / `CImageBookPlate` (books / journal) |
| buttons | `CTogglePlate`, `CDIVINITYTogglePlate` (the HUD toggle buttons that show/hide the other plates) |

The base `CControlPlate` vtable is ~15 slots; the **draw** virtual is
slot 14 (`+0x38`) — `CShieldPlate`'s is `fcn.00529c10`
(`CShieldPlate::virtual_56`, [`inventory.md`](inventory.md)), which blits
the worn-item icons via the shared sprite lookup. The concrete plates are
constructed by the HUD-build cluster around `0x52cxxx` (e.g. `CStatusPlate`
by `fcn.0052c2e0`), each pairing its RTTI class with the `.bmg` control
tree it owns. So a named plate = a `C*Plate` glue object + its `.bmg`
layout + the `CBM*` controls the layout instantiates.

## Plate-definition language (`.bmg` text files)

A plate's control tree is **not** hard-coded — it is loaded from a
**text `.bmg` definition file** (e.g. `keyplate.bmg`, `energyplate.bmg`)
parsed by a **per-control parser family** (~15 functions in
`0x0045c250`–`0x00460390`, one per `CBM*` control type). The grammar is
**nested**: a *form* contains *lists*, a list contains *controls* — the
parser advances object-by-object, reporting `BMGui:Expected form keyword
or new object` / `…list keyword or new object` when a token fits neither
a property keyword nor the start of a child object.

Each control reads a block of **`keyword value`** properties; the
vocabulary a control parser checks includes `image`, `animation`,
`list`, `index`, `callback`, `button` (a `CBMClipButtonControl`-style
node). Values are typed, and the parser enforces the type with specific
diagnostics:

```text
BMGui:Missing keyword (%d)
BMGui:Expected '%s' (%d)                     // a specific keyword
BMGui:Expected '%s' or '%s' (%d)
BMGui:Expected a number (%d)
BMGui:Expected a number in range %d - %d (line %d)
BMGui:Expected a number > %d (line %d)
BMGui:Expected a nonempty string (%d)
BMGui: Expected a ltr index for tooltip (%d) // a localization-string index
```

So plate UIs are **data-driven**: a `.bmg` file lays out the controls
and their image/animation/callback bindings, and the engine instantiates
the matching `CBM*` objects. Individual plates additionally have C++
glue classes (`.\PLATE\platewin.cpp` plus per-plate `BookPlate.cpp` /
`DiscPlate.cpp` / `optionsplate.cpp` / `researchplate.cpp` /
`TeleportPlate.cpp`) wrapping their `.bmg` layout with behaviour.

### Complete `.bmg` vocabulary (from the 24 shipped plates)

The `.bmg` files are **plaintext, packed in `dat\flat.cmp`** (24 `bmg\*.bmg`,
beside the `.mgc`/`.spl` programs — [`formats/cmp.md`](formats/cmp.md)), so
the shipped plate layouts are directly readable. Enumerated across all 24,
the grammar is `name "<title>" begin form … end` with nested
`begin <type> … end` blocks:

```text
containers : form  floatform  subwindow  tab
controls   : button  clipbutton  checkbox  edit  static  custom  list
             vertscroll  horizscroll  simplevs    { the CBM* node per type }
```

Per-control properties (keyword value), authoritative set:

```text
state sprites : normal  pressed  highlight  disabledlook   { CBM* per-state images }
layout        : position (x,y)  size  xsize  shift  xspacing  yspacing
                area  hotarea  autoarea  row
behaviour     : mouseclick   { the click action — the divevent name dispatched
                               via CEventBMC, see "Event handling" below }
                clicksound  highlsound  hotkey  tooltip (ltr-index)  content  id
auto/dynamic  : autofill  autodelay  dynamic  primary  showlast  showselect
```

So a control's behaviour binding is the **`mouseclick`** keyword, and —
verified across the 24 plates — its value is **`callback id <N>`**, a
**numeric per-plate callback index** (the shipped set uses N ∈ {0–6, 10,
11, 16, 20, 21, 30, 31, 99}), *not* a global named action. On click the
control raises that id through `CEventBMC`, and the **owning plate's C++
handler switches on N** to run the action
([Event handling](#event-handling--divevent) / `DivEventFunctions`). So the
`.bmg` says only *"this button fires callback N"*; the meaning of N is
per-plate, defined in that plate's C++ glue (`platewin.cpp` + the per-plate
class). *(Refines the earlier note that called the value a "divevent name" —
it is a numeric callback id resolved per plate.)*

## Connections

- **Inventory / chests** — using a container opens an `InventoryPlate`
  holding a `CBMCInventoryViewer` ([`object-interaction.md`](object-interaction.md),
  [`inventory.md`](inventory.md)).
- **Dialogue** — the dialog oracles (`CBMDialogAnswerTVOracle` /
  `OptionsTVOracle` / `MonologueTVOracle`) render the
  `GetAnswerText` / `GetQuestion` strings ([`dialogue.md`](dialogue.md)).
- **Automap** — the map plates render the automap tiles
  ([`formats/automap.md`](formats/automap.md)) and user notes.

## Status

- Control taxonomy ✅ — ~27 `CBM*` classes catalogued from RTTI; MVC
  (`Controller`/`Oracle`/view) split identified.
- Plate system ✅ — named composited windows toggled via
  `DIVINITYTogglePlate`; key plates and their roles identified.
- Layout data ✅ — status-plate element positions in `statuspl.cmp`
  (already reversed in `cmp.md`).
- Plate-definition language ✅ — control trees load from text `.bmg`
  files (`keyplate.bmg`/`energyplate.bmg`), parsed by a per-control
  parser family (`0x45c250`–`0x460390`); nested form→list→control
  grammar with typed `keyword value` properties (`image`/`animation`/
  `list`/`index`/`callback`/`button`). `.bmg` byte-level layout ❓ (text,
  not yet sampled against a shipped file).
- Event handling ✅ — the `.\divevent\` subsystem wires a control's
  interaction to an action (see below). The widget *draw / hit-test*
  internals remain ❓ (and are largely moot for the ebiten UI).

## Event handling — `.\divevent\`

A control declared in a `.bmg` with a **`callback`** keyword
(`0x60db74`) is instantiated as a **`CEventBMC`** (vtable `0x60f7cc`,
`.\divevent\eventbmc.cpp`) — an *event-bearing* BMGUI control. The pieces:

| Unit | Role |
|---|---|
| `eventbmc.cpp` (`CEventBMC`) | a control that, on interaction (click/hover), **fires an event** instead of just drawing |
| `divevent.cpp` | the event object that carries the interaction to a handler |
| `DivEventFunctions.cpp` | the named **handler functions** the `callback` resolves to (the action: open Trade, use a slot, toggle a plate, …) |
| `bmclist.cpp` | the per-plate **control list** (the `list`/`index` collection a plate composites) |

So the flow is: a `.bmg` control binds a `callback` name → at load it
becomes a `CEventBMC` in the plate's `bmclist` → on click the control
raises a `divevent` → dispatched to the matching `DivEventFunctions`
handler, which runs the game action (and may toggle another
[plate](#) via `DIVINITYTogglePlate`). This is how the HUD buttons —
Trade/Identify/Repair, the inventory and energy slots, the spellbook —
reach their behaviour. (Handler-name → function table not split
entry-by-entry.)
