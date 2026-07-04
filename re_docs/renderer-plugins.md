# Renderer plugins (`slash*.dll`)

Divine Divinity's renderer is a **swappable plugin**. `div.exe` picks a
backend at startup, `LoadLibraryA`s the matching `slash<n>.dll`, and
resolves a fixed `DllSlashed*` C API by `GetProcAddress` (so the
plugins never appear in `div.exe`'s static import table). All four
ship; the active one is chosen by the `slashed-*.cfg` files.

| DLL | Identification | Backend | Config |
|---|---|---|---|
| `slash1.dll` | `Direct3D 6 R` | Direct3D 6 | `slashed-d3d6.cfg` |
| `slash2.dll` | `Glide 3.x R` | 3dfx Glide | `slashed-glide.cfg` |
| `slash3.dll` | `DirectX R` | newer DirectX | `slashed-directx.cfg` |
| `slash4.dll` | `Software R` | software (via DirectDraw) | `slashed-software.cfg` |

## Plugin ABI

Every backend exports the **same 15 functions** — verified identical
across all four DLLs:

```text
lifecycle:    DllSlashedInit          DllSlashedShutdown
config:       DllSlashedInternalApplyConfiguration
query:        DllSlashedGetIdentification  DllSlashedGetInfo  DllSlashedGetDate
              DllSlashedGetMajorVersion    DllSlashedGetMinorVersion
              DllSlashedGetResolutions
frame:        DllSlashedStartFrame    DllSlashedEndFrame
glow draw:    DllSlashedGlowDrawLine  DllSlashedGlowDrawRect
              DllSlashedGlowDrawSquare DllSlashedGlowDrawQuad
```

- **lifecycle / config** — bring the device up, apply the resolution
  and options parsed from `slashed-*.cfg`, tear down.
- **query** — `GetResolutions` feeds the options UI; `GetIdentification`
  is the human string in the table above; version/date gate
  compatibility.
- **frame** — `StartFrame` / `EndFrame` bracket one rendered frame: the
  engine begins a frame, draws the world into the plugin's surface, and
  `EndFrame` presents (flip / blit to screen). This is the per-frame
  call from the main update ([`frame-loop.md`](frame-loop.md)).
- **glow draw** — additive **glow / lighting** primitives (line / rect /
  square / quad) the engine overlays for light sources and effects,
  drawn through the backend so each renderer does it natively.

The bulk **sprite blit** (terrain, objects, characters) is not a
separate export — the engine composites into the plugin's frame surface
between `StartFrame` and `EndFrame`; only the glow overlay has dedicated
entry points.

## Relation to the Go port

OpenDivine renders with **ebiten**, so it replaces this whole plugin
layer — none of `slash*.dll` is used. This doc is for understanding the
original's structure (e.g. that "lighting" is an additive overlay pass,
not baked into the sprite blit). The depth-sort and blit the plugins
present are reversed from `div.exe` itself in
[`render-trace.md`](render-trace.md).

## Citations

```text
slash1.dll  "Direct3D 6 R"   ┐
slash2.dll  "Glide 3.x R"     ├ 15 identical DllSlashed* exports each
slash3.dll  "DirectX R"       │ (LoadLibraryA + GetProcAddress, no static imports)
slash4.dll  "Software R"     ┘  ("Software R" links DirectDrawCreate)
```

## Status

- Plugin model ✅ — `LoadLibraryA` + `GetProcAddress` on a fixed
  `DllSlashed*` API; backend chosen by `slashed-*.cfg`.
- ABI ✅ — 15 exports, identical across all four backends; grouped by
  role above.
- Frame protocol ✅ (shape) — `StartFrame` → world composite →
  `EndFrame` present; glow overlay via the `GlowDraw*` primitives.
- Surface / blit details 🟡 — how the engine obtains and writes the
  plugin's frame surface (lock vs shared buffer) is inferred from the
  export shape, not yet traced into a backend.
- Per-backend internals ❓ — the actual D3D6 / Glide / DirectDraw device
  code inside each DLL is not reversed (and is moot for the ebiten
  port).
