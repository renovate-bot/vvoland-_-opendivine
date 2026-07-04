# Fight controllers (`CAgentFight` / `CClientFight` / `CPartyFight`)

The object that drives a combatant's **attack state machine** — choosing
and timing swings, tracking the current target, and turning an intent to
attack into the resolved hit. [`combat.md`](combat.md) documents the
damage math; this documents the controller *around* it and the
client/server split that [`messages.md`](messages.md) routes.

Every fighting [`CAgent`](agent.md) owns a **Fight controller** sub-object
(vtable-tagged, re-stamped/cleared when the agent dies — see
[`death.md`](death.md), where `Die` resets the `CAgentFight` sub-object).
Three concrete kinds:

| Class | vtable | Role |
|---|---|---|
| `CAgentFight` | `0x609020` | the base / NPC controller — server-side attack resolution (the swing sweep `fcn.00417b40`, the HP-apply `fcn.00417550`) |
| `CClientFight` | `0x60c2c8` | the **client** controller (`.\AGENTS\clientfight.cpp`) — turns player input into attack **request messages** |
| `CPartyFight` | `0x60cbb4` | the **party-member** controller — player/companion combat, including the class [special move](skills-magic.md) the HUD exposes |

## The client/server flow

Divine Divinity runs a [client/server message protocol](messages.md) even
in single-player. Combat follows it:

1. The player's **`CClientFight`** turns a click-to-attack into an attack
   **request message** (the client side never resolves damage itself).
2. The message is routed by the agent command bus (`FUN_00509f10`,
   [`messages.md`](messages.md)) to the server.
3. The target/attacker's **`CAgentFight`** / **`CPartyFight`** resolves it —
   the to-hit gate, the per-element [damage](combat.md) sweep
   (`fcn.00417b40`, the `CNpc`/`CPartyMember` fight virtual), the
   [HP-apply](combat.md) (`fcn.00417550`), and the resulting [floating
   damage text](floating-text.md) / [hit FX](effects.md).

So the split is **input vs. resolution**: `CClientFight` is the requester,
`CAgentFight`/`CPartyFight` the authority. This is why the combat code is
message-driven rather than a direct call, and why a single-player build
still carries the protocol.

## How it connects

- **Damage math** — the controller's resolution path is [`combat.md`](combat.md)
  (to-hit `≤ defense/5`, the four-component element loop, the polymorphic
  stat method).
- **Death** — when Hp hits 0 the agent's `Die` ([`death.md`](death.md))
  tears the `CAgentFight` sub-object down (resets vtable `0x609020`, clears
  the attack-state fields).
- **Messages** — the request/resolve hop is the client/server bus
  ([`messages.md`](messages.md)).
- **Special moves** — `CPartyFight` is where the per-class signature move
  (`CSpecialMove_*`, [`skills-magic.md`](skills-magic.md)) is launched.

## Status

- Hierarchy ✅ — `CAgentFight` (`0x609020`, base/NPC) / `CClientFight`
  (`0x60c2c8`, `clientfight.cpp`, client input) / `CPartyFight`
  (`0x60cbb4`, party). Each agent owns one as a sub-object.
- Client/server flow ✅ — `CClientFight` requests via the message bus
  (`FUN_00509f10`); `CAgentFight`/`CPartyFight` resolve (`fcn.00417b40` /
  `fcn.00417550`); torn down by `Die`.
- Fight-object agent offset / per-controller state fields 🟡 — the exact
  `CAgent` field that holds the controller and its internal attack-state
  layout are not split field-by-field (the swing fields appear in
  [`combat.md`](combat.md)).

## Citations

```text
vtables: CAgentFight 0x609020 · CClientFight 0x60c2c8 · CPartyFight 0x60cbb4
div.exe:0x00417b40   fcn.00417b40   CNpc/CPartyMember fight virtual — swing sweep (combat.md).
div.exe:0x00417550   fcn.00417550   HP-apply (combat.md).
div.exe:0x00509f10   FUN_00509f10   agent command bus — routes the attack request (messages.md).
str: .\AGENTS\clientfight.cpp
```
