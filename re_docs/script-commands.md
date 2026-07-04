# Internal script command language

Divine Divinity ships a small **internal scripting language** used by
object, agent, magic and region scripts — distinct from the Osiris
story VM. It is parsed and executed by the `.\script\` unit
(`interpr.cpp`, `statment.cpp`) and the magic-script unit
(`.\magic\mgcscrpt.cpp`). The compiled command tables in `.rdata`
expose ~270 distinct statements; this doc catalogues them.

Placeholder convention in the command strings:

- `#` — a numeric argument (id, count, level, angle, …).
- `$` — a token argument (a name, label, coordinate pair, or string).
- A leading `!` marks an imperative form (e.g. `!move to $,$`).

The full extracted list is reproducible with:

```sh
rabin2 -z div.exe | sed -E 's/.*ascii +//' \
  | grep -E '^[a-z][a-z0-9 ]+ (#|\$)( #| \$|$)' | sort -u
```

## Parser / executor

```text
div.exe:0x005378d0   FUN_005378d0   .\script\interpr.cpp — statement interpreter entry
div.exe:0x0053dab0   FUN_0053dab0+  .\script\statment.cpp — statement-handler cluster
                                    (~0x0053dab0..0x0053e5c0, one handler per opcode group)
div.exe:0x00538a10   FUN_00538a10+  .\magic\mgcscrpt.cpp — magic-script handler cluster
```

Each command name is stored next to its handler in a dispatch table;
the parser matches the leading keyword and binds the `#`/`$` arguments.

## Command catalogue (by subsystem)

Representative members of each group; the count is the whole language,
not these excerpts.

### Movement & positioning

`move to $`, `move to $,$`, `move to npc $`, `move to npc $ range #`,
`move to locked location`, `new movement $`, `direct move #`,
`follow npc $`, `appear #`, `appear near location $ angle #`,
`appear near npc $ angle #`, `goto $`, `goto label #`, `leave to $`,
`face $`, `look at $`, `on death reappear at $`.

### Combat & damage

`attack $`, `set attack #`, `fight #`, `second fight #`, `fightrun #`,
`fightstill #`, `fightwalk #`, `set fight mode #`, `set fight range #`,
`set fightspeed #`, `set fightwalkspeed #`, `set fight steps #`,
`set damage # #`, `set damage2 # #`, `set damage2 chance #`,
`set arrow damage # #`, `set special damage # # #`,
`set special damage chance #`, `melee damage amount #`,
`fire damage level #`, `lightning damage level #`,
`poison damage level #`, `ritual damage level #`, `hit #`.

### Magic / spells

`cast spell # on $`, `magic #`, `magical #`, `set magicspeed #`,
`charm id #`, `effect $`, plus the `mgcscrpt.cpp` effect statements.

### Agent / AI behaviour

`add behavior $ probability #`, `add class $ probability #`,
`be moody #`, `npc emotions are in $`, `has action $`,
`interested in special object #`, `die #`, `die size #`, `charm id #`,
`candidate #`, `chance #`.

### Groups

`new group $`, `group $`, `set group #`, `add to group from $`,
`add group $ probability #`, `create group with behaviour #`.

### Object & world

`create object #`, `create object # alchemy level #`,
`object # # #`, `object # # # #`, `object # $`, `set object count #`,
`lock door #`, `break trap #`, `chair #`.

### Inventory & items

`add # instances of object # to inventory`, `set inventory level #`,
`set inventory type #`, `clear inventory`, `regenerate inventory`,
`clear generated inventory`, `gold #`, `cost #`.

### Dialog / presentation / config

`description $`, `description level # $`, `general description $`,
`label $`, `event $`, `language #`, `game rate # fps`, `fog speed #`,
`force compiled roofs #`, `alpha bit #`, `frame #`, `image #`,
`debug #`, `log $`, `commit to disk #`.

## Status

- Command vocabulary ✅ — ~270 statements catalogued from the `.rdata`
  command strings; placeholder convention (`#`/`$`/`!`) confirmed.
- Parser / handler clusters ✅ located (`interpr.cpp`,
  `statment.cpp`, `mgcscrpt.cpp`); per-command handler addresses 🟡 —
  the dispatch table is not yet enumerated entry-by-entry.
- Argument grammar 🟡 — `#`=number, `$`=token confirmed from usage; the
  exact tokenizer rules (coordinate pairs, quoting) are not yet
  decoded.
- Relationship to compiled object/region scripts ❓ — these statements
  back the object `s_function` behaviour and region scripts; how the
  source compiles into the runtime tables is not yet traced.
