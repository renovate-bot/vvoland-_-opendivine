# Agent command messages

The engine's central command bus. Player and AI actions do not call
gameplay code directly — they are encoded as **request messages** and
dispatched through a client/server message protocol. This is why
interaction, combat and movement are event-driven throughout (see
[`object-interaction.md`](object-interaction.md), [`combat.md`](combat.md)):
those subsystems are reached *as messages*, not as direct calls.

Divine Divinity runs this client/server protocol even in single-player
— the parser logs `"… Client received a request …"` and
`"Don't know what to do with message …"` /
`"Can't parse message cause I don't know what it is (code=%d source=%d)"`.

## Message format

A message is a record in the agent's message queue:

```text
Message:
    u16  code        // +0x00 — the command (0..0x37); selects the handler
    …    payload     // command-specific fields (target id, slot, coords, …)
```

Messages also carry a **source** (the originator), reported alongside
`code` in the parse-failure log.

## Dispatch

`FUN_00509f10` uses a **two-level compressed switch**:

```text
code = (u16)msg[0]                              // bounds-checked queue read
if (code > 0x37) → error log (FUN_004f5310)
case = byte_table[0x50a254 + code]              // 56-byte code→case map
jmp  jump_table[0x50a1dc + case*4]              // 30 case targets → Parse<Action>Message
```

So the 56 codes collapse onto 30 cases. One handler carries a stub
string `"ParseDropAndUseOnMessage not implemented yet for …"`, confirming
the `Parse<Action>Message` naming.

Resolving every case's call target gives the **complete code → handler
map**: 28 codes have active handlers, **20 codes route to the error log
`FUN_004f5310`** (unimplemented/invalid in this build), and 8 codes are
ignored (no-op inline cases).

## Command code → handler

| Code(s) | Handler | Verb |
|---|---|---|
| 0 | `FUN_00506b80` | |
| 1 | `FUN_00506c40` | |
| 2 | `FUN_00506d10` | |
| 3 | `FUN_00506f30` | |
| 4 | `FUN_005074a0` | **pick up object** (`"told to take an object but can't find …"`) |
| 5, 54 | `FUN_00508a70` | drop / use-on (`ParseDropAndUseOnMessage`) — **delegates to code 17** (`FUN_00507130`) after target resolution, so "use item on object" and "use object" share the `CObject::Use` path |
| 6 | `FUN_00507ad0` | **take from inventory/container** (`"Trying to take slave from inventory"`) |
| 7 | `FUN_00508d50` | **multiplayer message** (`.\…\MPLAYERMessage.cpp`) |
| 8 | *(inline, no call)* | |
| 11 | `FUN_00506630` | |
| 17 | `FUN_00507130` | **use world object** ([`object-interaction.md`](object-interaction.md)) |
| 19 | `FUN_00507180` | **use inventory object** |
| 20 | `FUN_00508280` | |
| 21 | `FUN_00507f30` | |
| 22 | `FUN_00507ff0` | |
| 26 | `FUN_00506490` | |
| 41 | `FUN_00508400` | |
| 42 | `FUN_00508320` | |
| 45 | `FUN_00506750` | |
| 46 | `FUN_00506a50` | |
| 48 | `FUN_00508510` | |
| 49 | `FUN_00508050` | |
| 50 | `FUN_005081c0` | |
| 51 | `FUN_005080e0` | |
| 52 | `FUN_00507000` | |
| 53 | `FUN_00509740` | |
| 55 | `FUN_00509d40` | |
| 9,10,12,14,15,25,29,40 | *(ignored)* | no-op |
| 13,16,18,23,24,27,28,30–39,43,44,47 | `FUN_004f5310` | unhandled → error log |

Naming the remaining active verbs (move/attack/cast/talk/equip) is
**blocked by the handler shape, not just unfinished**: the unnamed
handlers are thin — they read the payload `[msg+8]`, run a shared
**target-resolution / bounds preamble `FUN_00586d60`** (validates the
message's target coordinate against the map; `shl …,6` = the 64-px world
cell), then call into deeper agent/object functions that carry **no
per-verb log string**. So a verb cannot be read off the handler body by a
string; pinning each needs tracing the message *senders* (the UI/AI that
stamps a given code) instead. Recorded as a dead-end for the
string-based approach. Codes 0/1/2 share `FUN_00586d60` then
`FUN_004df5d0`; code 3 routes through `FUN_004a4dc0`.

## Relation to Osiris events

This is the **agent command** bus (player/AI verbs). It is distinct
from the **Osiris event** system (story/quest events) routed through
the event manager at `[0x7447dc]` — though some commands raise Osiris
events as a side effect (e.g. using a scripted object,
[`object-interaction.md`](object-interaction.md)).

## Citations

```text
div.exe:0x0050a290   FUN_0050a290   request parser ("Client received a request").
div.exe:0x00509f10   FUN_00509f10   command switch on the u16 code (0..0x37 jump table).
div.exe:0x004f5310   FUN_004f5310   diagnostic log (unknown / unhandled message).
```

## Status

- Protocol ✅ — client/server request messages; `u16 code` at `+0x00`
  plus a source; dispatched by `FUN_00509f10`.
- Dispatch ✅ — bounds-checked queue read, `code > 0x37` rejected, jump
  table to `Parse<Action>Message` handlers.
- Full command-code → handler map ✅ — the two-level switch
  (`0x50a254` byte table + `0x50a1dc` jump table) walked entry-by-entry:
  28 active handlers, 20 codes → error log, 8 ignored.
- Verb names 🟡 — confirmed: pick-up (4), drop/use-on (5,54), take from
  container (6), use-world (17), use-inventory (19). The other ~18
  active handlers are thin dispatchers (no distinctive strings); mapped
  by code + address but not yet named.
- Payload layouts ❓ — per-command message field layouts are not yet
  decoded.
