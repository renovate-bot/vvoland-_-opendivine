# The script language VM (`.\script\`)

Beneath the [magic-script](skills-magic.md) and the
[script-command](script-commands.md) vocabularies sits a real **embedded
scripting language** with variables, expressions, and control flow
(`if`/`while`), implemented as an **AST-walking interpreter**. A `.mgc`
spell script is a *program* in this language — which is why spells can
branch, loop, and compute, not just fire fixed verbs. The source units are
`.\script\expr.cpp` (expressions), `.\script\statment.cpp` (statements),
and `.\script\interpr.cpp` (the interpreter), with `.\script\mgcscrpt.cpp`
the magic-script dialect built on top.

## Expressions — `expr.cpp` (`CExpression`)

`CExpression` (base vtable `0x61a49c`) has one real virtual:
**`Evaluate`** (slot 0, base `fcn.00536670`) — returns the expression's
value. The concrete node types:

| Class | Node |
|---|---|
| `CConstExpression` | a literal constant |
| `CVariableExpression` | a variable read (resolved through `CVariableManager`) |
| `CUnaryExpression` | unary op (`-x`, `!x`) |
| `CBinaryExpression` | binary op (`a+b`, `a<b`, …); its `Evaluate` (`fcn.00536780`) **recursively evaluates both operands** and applies the operator |
| `CBooleanExpression` | boolean combination (`&&`/`||`) |
| `CArrayExpression` | indexed array access |

Evaluation is the textbook recursive tree-walk: an operator node evaluates
its children and folds them.

## Statements — `statment.cpp` (`CStatement`)

`CStatement` (base vtable `0x61af08`) has the virtual **`Execute`** (slot 0,
base `fcn.0053eba0`). The statement node types give the language full
structured control flow:

| Class | Statement |
|---|---|
| `CSAssignment` | `var = expr` |
| `CSArrayAssignment` | `arr[i] = expr` |
| `CSCompound` | a `{ … }` block (a list of statements run in order) |
| `CSIfThenElse` | `if (expr) … else …` |
| `CSWhile` | `while (expr) …` |
| `CSFunction` | a function call / definition (the bridge to the engine verbs) |

So a script is a tree of statements; `CSCompound` holds the body,
`CSIfThenElse`/`CSWhile` branch on a `CExpression`, and `CSFunction` is how
a script reaches the native command vocabulary (the `mgcscrpt` verbs in
[`script-commands.md`](script-commands.md) / [`skills-magic.md`](skills-magic.md)).

## Variables — `CVariableManager`

`CVariableManager` holds the script's named variables; a
`CVariableExpression` reads through it and `CSAssignment` writes through
it. This is the scratch state a spell/script accumulates as it runs.

## How it fits

- **Magic scripts** — `mgcscrpt.cpp` compiles each `.mgc` spell into this
  AST; `CMagicInterpreter` ([`skills-magic.md`](skills-magic.md)) runs it
  by `Execute`-ing the statement tree, so a spell's per-level scaling,
  conditionals, and multi-step effects are ordinary `if`/`while`/assignment
  statements over `props.000`-sourced variables.
- **Command vocabulary** — the leaf `CSFunction` calls are the documented
  `mgcscrpt` / script verbs; this doc is the *grammar* those verbs live
  inside, [`script-commands.md`](script-commands.md) is the *vocabulary*.
- **Compilation** — `.\GAME\compilestart.cpp` is the compile entry that
  turns script text into the AST the interpreter walks.

This is the fourth script system of the engine, and the most general: where
[Osiris](osiris.md) is a RETE rule engine and [agentscript](npc-ai.md) is a
flat 125-command line language, the `.\script\` VM is a full expression/
statement language with variables and control flow.

A second `CExpression` hierarchy reuses this framework: the **`CBE*`
boost-expression** nodes (vtables immediately after `CExpression`'s,
sharing `Evaluate` `fcn.00536670`) compute **magic-item affix magnitudes**
— `CBERandom` is the dice roll (`rand ()`/`signrand ()`),
`CBEBasicArithmetic`/`CBEBasicUnary` combine, `CBEPrefix` wraps the named
affix ([`formats/itemgen.md`](formats/itemgen.md)). So item generation and
spell scripting share one expression evaluator.

## Status

- Expression AST ✅ — `CExpression` (base `Evaluate` `fcn.00536670`) +
  Const/Variable/Unary/Binary/Boolean/Array; `CBinaryExpression::Evaluate`
  (`fcn.00536780`) is a recursive operand fold.
- Statement AST ✅ — `CStatement` (base `Execute` `fcn.0053eba0`) +
  Assignment/ArrayAssignment/Compound/IfThenElse/While/Function — full
  structured control flow.
- Variables ✅ — `CVariableManager`; reads via `CVariableExpression`,
  writes via `CSAssignment`.
- Source units ✅ — `expr.cpp` / `statment.cpp` / `interpr.cpp` /
  `mgcscrpt.cpp`; compile entry `compilestart.cpp`.
- Operator set / bytecode-vs-tree details 🟡 — the exact operator
  enumeration in `CBinaryExpression` and whether `compilestart` emits a
  tree or a flattened form are not split out (the eval is a tree-walk).

## Citations

```text
div.exe:0x00536670   CExpression::Evaluate (base, .\script\expr.cpp).
div.exe:0x00536780   CBinaryExpression::Evaluate — recursive operand fold + operator.
div.exe:0x0053eba0   CStatement::Execute (base, .\script\statment.cpp).
vtables: CExpression 0x61a49c · CStatement 0x61af08 · CBinaryExpression 0x61a530
         CSWhile 0x61af70 · CSIfThenElse 0x61af60 · CSFunction 0x61af80
classes: CVariableManager · C{Const,Variable,Unary,Binary,Boolean,Array}Expression
         CS{Assignment,ArrayAssignment,Compound,IfThenElse,While,Function}
str: .\script\expr.cpp · .\script\statment.cpp · .\script\interpr.cpp · .\GAME\compilestart.cpp
```
