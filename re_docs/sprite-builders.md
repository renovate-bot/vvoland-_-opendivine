# Procedural sprite/particle builders (`.\script\sprlist.cpp`)

The system that lays out **particle patterns** — the spirals of sparks, the
expanding rings, the pulsing discs that spell effects draw. Where
[`effects.md`](effects.md) (`CAniEffect`) is a *single* animated sprite,
this builds and animates a whole **arrangement** of sprites procedurally
(a ring, a spiral, a pulse), and lives in the
[script subsystem](script-language.md) (`.\script\sprlist.cpp`) so spell
scripts can spawn them.

## The container — `CSpriteList`

`CSpriteList` (vtable `0x61ab8c`) holds a collection of sprite elements and
ticks/draws them together. Its elements derive from **`CSLAbstractSprite`**
(`0x61aad4`):

- **`CSpriteParticle`** (`0x615068`) — a single moving particle (position +
  velocity + lifetime, the same shape as a [`CAniEffect`](effects.md)).
- **`CSLImage`** — a static image element in the list.

The list is drawn through the world's **sorted-sprite** render path
(`CSortedSprite`, [`render-trace.md`](render-trace.md)) so the particles
depth-sort against the scene like any other sprite.

## The builders — `CSLManagedBuilder` family

A **builder** populates a `CSpriteList` with a procedural pattern. Base
**`CSLManagedBuilder`** (`0x61ad54`); the concrete patterns:

| Builder | Pattern |
|---|---|
| `CSLSpiralBuilder` (`0x615fb4`) | sprites laid along a **spiral** |
| `CSLPulseSpiralBuilder` | a spiral that **pulses** (animated radius/density) |
| `CSLPulseRingBuilder` (`0x615f98`) | an expanding/pulsing **ring** |
| `CSLCustomSpriteBuilder` | a caller-defined arrangement |

The geometry comes from the helper shapes `CSpiralDisc` and `CSpline5` (a
5-point spline), and the trajectory of an emitted particle can use the same
[`CPath*`](projectile-types.md) curves (helix/spiral/orbit) that projectiles
use — so a "spiral of homing sparks" is a spiral builder feeding particles
onto orbit paths.

## How it fits

- **Spell visuals** — the elemental/buff spells ([`skills-magic.md`](skills-magic.md))
  invoke these builders from their `.mgc` scripts (the
  [script VM](script-language.md)) to render their signature particle
  patterns (a ring of fire, a spiral of frost), distinct from the single
  impact flash that [`effects.md`](effects.md)'s `fcn.00416050` drops.
- **Render** — emitted sprites go through the [sorted-sprite](render-trace.md)
  pipeline; they are purely cosmetic (no [combat](combat.md) effect — the
  damage is the separate [pain-point](painpoints.md) / spell path).

## The `.spl` sprite-list data format (the `EmitSpriteList` argument)

A `.mgc` spell program (visual layer, [`skills-magic.md`](skills-magic.md))
draws via `EmitSpriteList("<file>.spl", "<entry name>", X, Y, Z, layer)`.
Those `.spl` files are **plaintext sprite-list definitions packed in
`dat\flat.cmp`** (30 `magic\*.spl`, beside the `.mgc` programs and the
`.bmg` GUI — [`formats/cmp.md`](formats/cmp.md)), the **data** the
`CSpriteList` runtime above consumes. Format (recovered by extracting them,
e.g. `bless.spl`):

```text
name "Bless Lightcone Part 1 Appear"
begin list
  begin animation
    list 2          { source CPacked imagelist id }
    index 17        { sprite / animation index within that list }
    bpp16add        { blend mode }
    once            { playback flag }
  end animation
  XPos := -(Width*0.5);   { per-element transform, computed from bindings }
  YPos := -50;
end list
```

Grammar (vocabulary enumerated across all 30): each named entry is a
`begin list … end list` holding one or more element blocks —
**`begin {image | animation | prefab | clip} … end`** — where an element
declares **`list <imagelist#>`** + **`index <sprite#>`**, a **blend mode**
(`normal` / `bpp16add` additive / `bpp8` palette), and **flags**
(`once`, `reverse`, `directional`). It then assigns the element's
**transform** with `:=`: position `XPos`/`YPos`, colour `Red`/`Green`/`Blue`/
`Alpha`, and `Clip{XPos,YPos,Width,Height}` — computed from the host
bindings `Width`/`Height`/`SpecialPointX,Y`/`AnimDeltaX,Y` and `rand()`/
`signrand()`/`delay()`. So the visual pipeline is fully readable end to end:
**`.mgc` program → `EmitSpriteList` → `.spl` entry → `list`/`index` into a
CPacked imagelist → sorted-sprite blit**, every stage plaintext-or-decoded.

## Status

- Container ✅ — `CSpriteList` (`0x61ab8c`) of `CSLAbstractSprite`
  (`0x61aad4`) elements: `CSpriteParticle` (`0x615068`) / `CSLImage`.
- Builders ✅ — base `CSLManagedBuilder` (`0x61ad54`) +
  Spiral/PulseSpiral/PulseRing/Custom procedural patterns; geometry via
  `CSpiralDisc` / `CSpline5`.
- Integration ✅ (cross-referenced) — `.\script\sprlist.cpp`, invoked from
  magic scripts; drawn via the sorted-sprite render path.
- Per-builder parameter layout 🟡 — the exact spiral pitch / ring radius /
  pulse-rate fields per builder are not split out (these are cosmetic
  tuning values).

## Citations

```text
vtables: CSpriteList 0x61ab8c · CSLAbstractSprite 0x61aad4 · CSLManagedBuilder 0x61ad54
         CSLSpiralBuilder 0x615fb4 · CSLPulseRingBuilder 0x615f98 · CSpriteParticle 0x615068
classes: CSpriteList · CSpriteParticle · CSLImage · CSL{Spiral,PulseSpiral,PulseRing,Custom}Builder
         CSpiralDisc · CSpline5 · CSortedSprite
str: .\script\sprlist.cpp
```
