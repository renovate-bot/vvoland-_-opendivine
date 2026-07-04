# Screen effects / transformations (`.\effect\effect.cpp`)

The **effect processor** is the full-screen post-process layer: it applies
image-space *transformations* to the rendered frame — the green wash when
poisoned, an ice/frozen overlay, a pulsating ring for a spell — on top of
the normal scene render ([`render-trace.md`](render-trace.md)). It is
distinct from the per-object sprite [`animation.md`](animation.md) and the
attached `CExplosion`/gore effects ([`minor-mechanics.md`](minor-mechanics.md)).

## The processor

Constructed by `fcn.00490300` (`.\effect\effect.cpp`) — called both at the
boot *"Loading effect processor"* step of `init.cpp`
([`architecture.md`](architecture.md)) and from the **magic manager**
(`fcn.004e3180`, [`skills-magic.md`](skills-magic.md)), which is what binds
status/spell effects to their screen visuals.

The processor object is a single root pointer (the ctor allocates a 4-byte
slot and zeroes it) onto which active effects hang as a **region-keyed
binary tree**: a node carries region bounds (compared in the lookup) plus
left/right child (`+0x10`/`+0x14`) and parent (`+0x18`) links, and the
effects that share a region are kept on a **sibling chain** (links on the
effect at `+0x24`/`+0x28`). The shared lookup `fcn.0048ff10` (find node by
the region/coordinate key tuple) backs three operations:

| Op | Fn | Action |
|---|---|---|
| **add** | `fcn.00490330` | find-or-create the node, link the new effect onto its sibling chain |
| **apply** | `fcn.004903a0` | find the node, then walk its sibling chain calling each effect's **apply** virtual (`virtual_12`, `+0x0c`); this is the per-frame post-process |
| **remove** | `fcn.00491150` | find the node, then `fcn.004900d0` unlinks it from the tree and tears the effect down |

So **apply and teardown are distinct paths** — earlier notes here conflated
them. The per-effect apply (`fcn.004903a0`, `call [vtable+0xc]` at
`0x490403`) is reached from the **render path** (via `fcn.0059e200`, called
from the per-frame functions `fcn.0049c7d0` / `fcn.004a3fd0`), and post-
processes the current frame buffer once per matching effect. The teardown
path (`fcn.004900d0`) instead relinks the node out of the BST, calls the
effect's `virtual_8` (free its work buffers, returning *ready-to-remove*),
then its `virtual_0` (the scalar-deleting **destructor**), and frees the
node (`fcn.004fa540`); a `virtual_8` that reports *not ready* trips the
`"EFFECT.CPP : Delete failed"` diagnostic. The recursive walk
`fcn.00490420` → `fcn.004902c0` → `fcn.00490240` is the **delete-all /
prune** traversal over that same teardown path, invoked by the owning magic
effects (`ice.cpp` `fcn.004c6120`, `mdamage.cpp` `fcn.004c9c00`) — so a
transformation's **lifetime is owned by its magic effect**, which adds it at
cast and removes it at expiry, not by a self-timer in the transformation.

## Transformation class hierarchy

All screen effects derive from the abstract base **`CAbstractTransformation`**
(a `CEffect`); the recovered concrete classes (MSVC RTTI type descriptors)
are:

| Class | Screen effect |
|---|---|
| `CAbstractTransformation` | abstract base (the effect interface) |
| `CIceTransformation` | frozen / ice overlay |
| `CPoisonedTransformation` | poison green-wash tint |
| `CPulsatingRingTransformation` | pulsating ring (spell / charm visual) |

Each is a `CEffect` that renders into an **image buffer** it owns — a null
buffer asserts `"CEffect:NewImage == NULL (10)"` (in
`CIceTransformation::apply`).

### Effect interface (vtable)

The `CAbstractTransformation` vtable is **five slots** (verified by walking
the RTTI vtables, e.g. `CIceTransformation` `0x6144ac`):

| Slot | Off | Role | Ice / Poisoned / PulsatingRing |
|---:|---|---|---|
| `virtual_0` | `+0x00` | **scalar-deleting destructor** — shared `fcn.004c62e0` (resets vtable to the base, `if (arg & 1) operator delete this`) | `0x4c62e0` (all share it) |
| `virtual_4` | `+0x04` | **build** — allocate the work/overlay image from the frame geometry (`w`/`h` stashed at `+0x08`/`+0x0c`), zero the phase fields; called once at construction | `0x490430` / `0x490f10` / — |
| `virtual_8` | `+0x08` | **teardown step** — free the work buffers, return *1 = ready-to-remove* / *0 = still active* | `0x4907a0` / `0x490c00` / `0x490c00` |
| `virtual_12` | `+0x0c` | **apply** — post-process the current frame buffer | `0x490800` / `0x490fb0` / `0x490c90` |
| `virtual_16` | `+0x10` | small accessor / cleanup | `0x490b30` / `0x490f70` / `0x490c50` |

The **apply** virtual is the heart: `CPoisonedTransformation::virtual_12`
(`0x490fb0`) locks the frame buffer and `memcpy`/blits a recoloured copy
(`fcn.00558300` / `fcn.00558290` = buffer lock/blit helpers) to tint the
screen; `CIceTransformation::virtual_12` (`0x490800`) draws its prebuilt
overlay. The overlay/work image is built up-front by `virtual_4` (Ice
`0x490430` allocates a `2·w·h` image at `effect.cpp:1151`, Poisoned
`0x490f10` a `3·w·2` buffer), so apply only re-processes pixels each frame;
`virtual_8` frees that image when the effect is removed.

## Triggering

The transformation construction sites are pinned (vtables RTTI-walked:
`CPoisonedTransformation` `0x6149dc`, `CIceTransformation` `0x6144ac`,
`CPulsatingRingTransformation` `0x60cde0`; each `new`s the object, then
registers it on the processor tree via the **add helper `fcn.00490330`**):

| Transformation | Constructed in | Source unit | Trigger |
|---|---|---|---|
| `CPoisonedTransformation` | `fcn.004c9c00` | `.\magic\mdamage.cpp` | a **poison damage** magic effect |
| `CIceTransformation` | `fcn.004c6120` | `.\magic\ice.cpp` | an **ice / frost** spell effect |
| `CPulsatingRingTransformation` | `fcn.00448bb0` / `fcn.005226e0` | `.\AGENTS\visualizer.cpp` (+ a stats-area site) | an agent **visualizer / status ring** (not a magic-damage effect) |

So the **elemental status visuals come from the magic effect classes**: a
poison-damage effect (`mdamage.cpp`) and an ice effect (`ice.cpp`) each
build their transformation at apply time and hand it to the processor —
matching the magic-manager (`fcn.004e3180`) also owning the processor. The
pulsating ring is instead an **agent-visualizer** effect (`visualizer.cpp`)
rather than a spell. Each transformation runs until its `virtual_4` phase
elapses, then drops out of the per-frame walk.

## Status

- Processor ✅ — `.\effect\effect.cpp`, ctor `fcn.00490300` (boot +
  magic-manager `fcn.004e3180`); a **region-keyed tree** (lookup
  `fcn.0048ff10`) with three ops: **add** `fcn.00490330`, **apply**
  `fcn.004903a0` (per-frame, calls `virtual_12`, reached from render
  `fcn.0059e200`), **remove** `fcn.00491150` → `fcn.004900d0` (BST unlink +
  `virtual_8` teardown + `virtual_0` destructor). The recursive
  `fcn.00490420`→`fcn.004902c0`→`fcn.00490240` walk is the **prune/delete**
  pass, not the apply (corrects an earlier mislabel).
- Class set ✅ — `CAbstractTransformation` base + `CIceTransformation` /
  `CPoisonedTransformation` / `CPulsatingRingTransformation` (RTTI).
- Interface ✅ — 5 vtable slots: `virtual_0` scalar-deleting destructor
  (shared `0x4c62e0`), `virtual_4` build (allocate the work image from the
  frame geometry), `virtual_8` teardown step (free buffers, return
  ready-to-remove), `virtual_12` apply (frame-buffer post-process; poison =
  recolour blit, ice = prebuilt overlay), `virtual_16` accessor; effects own
  a `CEffect` image buffer built by `virtual_4`.
- Triggering ✅ — construction sites pinned: `CPoisonedTransformation`
  ← `.\magic\mdamage.cpp` (`fcn.004c9c00`), `CIceTransformation` ←
  `.\magic\ice.cpp` (`fcn.004c6120`), `CPulsatingRingTransformation` ←
  `.\AGENTS\visualizer.cpp` (`fcn.00448bb0`); each `new`s the object and
  registers it via the processor add helper `fcn.00490330`. So elemental
  status visuals come from the magic effect classes; the ring is an
  agent-visualizer effect.

## Citations

```text
div.exe:0x00490300   effect-processor ctor (.\effect\effect.cpp); boot + magic-manager.
div.exe:0x0048ff10   processor tree lookup — find node by region/coord key (shared by add/apply/remove).
div.exe:0x00490330   add — insert effect onto its region node's sibling chain.
div.exe:0x004903a0   apply — find node, walk sibling chain calling virtual_12 (call [vtable+0xc] @0x490403); reached from render fcn.0059e200.
div.exe:0x00491150   remove — find node, then fcn.004900d0 (unlink + teardown).
div.exe:0x004900d0   BST node unlink + free: calls virtual_8 (teardown) then virtual_0 (destructor); "EFFECT.CPP : Delete failed" guard.
div.exe:0x00490420   prune/delete walk (thunk → 0x4902c0 → fcn.00490240, recursive), over the teardown path.
div.exe:0x00490240   recursive effect-tree walk (delete pass).
div.exe:0x004c62e0   CAbstractTransformation::virtual_0 — shared scalar-deleting destructor.
div.exe:0x00490430   CIceTransformation::virtual_4 — build overlay image (effect.cpp:1151).
div.exe:0x004907a0   CIceTransformation::virtual_8 — teardown step (free buffers, return ready-to-remove).
div.exe:0x00490800   CIceTransformation::virtual_12 (apply; "CEffect:NewImage == NULL").
div.exe:0x00490fb0   CPoisonedTransformation::virtual_12 (apply; buffer recolour blit).
div.exe:0x00490c90   CPulsatingRingTransformation::virtual_12 (apply).
div.exe:0x004e3180   magic manager — also constructs the effect processor.
div.exe:0x00490330   effect-processor add — links a new transformation onto the tree.
div.exe:0x004c9c00   .\magic\mdamage.cpp — news CPoisonedTransformation (poison damage).
div.exe:0x004c6120   .\magic\ice.cpp — news CIceTransformation (ice/frost).
div.exe:0x00448bb0   .\AGENTS\visualizer.cpp — news CPulsatingRingTransformation (status ring).
```
