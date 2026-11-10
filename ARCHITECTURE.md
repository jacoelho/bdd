# Architecture

`bdd` is one package with two stages. **Declaration** runs user builders and
records a plan; **execution** runs each selected leaf as one `testing` subtest.
Builders declare only: no setup, hook, or step runs during declaration.

| File | Owns |
| --- | --- |
| `suite.go` | `Feature`, `Suite` declaration grammar, leaf registration, leaf execution |
| `scenario.go` | `ScenarioSpec` declaration grammar, the `step` operation |
| `chains.go` | `ScenarioChain`, `BackgroundChain`: tokens returned by `Background`, `Given`, `When`, `Then` |
| `outline.go` | `Example`, `Case`, expansion of an Outline into leaves |
| `options.go` | `Option` constructors, `meta` (tags and disposition) |
| `names.go` | slug and tag normalisation |
| `steps.go` | `Step`, `Compose`, `NoState` |

## State machines

Each machine has one owner. Change the grammar in the owner, then update
`doc.go` and the README "Declaration order" section.

### Suite declaration: `suitePhase.next` in `suite.go`

```text
hooks      --Before-->                 hooks
hooks      --Background-->             background
background --And, But (current token)--> background
hooks, background, children --Scenario, Outline, Rule--> children
children   --Rule builder starts-->    suspended
suspended  --Rule builder returns-->   children
any        --builder returns-->        sealed
```

Every other declaration is rejected. `suitePhase.next` is a pure function
from (phase, declaration) to the next phase or the diagnostic; `Suite.declare`
applies it. `Rule` suspends and resumes its Suite around the nested builder,
and `seal` ends it. A suspended Suite rejects everything, so a declaration made on an
enclosing Suite inside a Rule builder cannot be filed outside that Rule.

### ScenarioSpec declaration: `specPhase` in `scenario.go`

```text
preamble  --Parallel (once), Before-->   preamble
preamble  --Given, When, Then-->         narrative
narrative --chain method (current token)--> narrative
any       --builder returns-->           sealed
```

### Chain token: `chains.go`, validated by the owning scope

A token is `{owner, generation, issued}`. The owner accepts it only while
`generation` equals its operation count, so a stale or ignored token fails
before anything is appended. The owner checks its own phase first: a token
used after the builder returned always reports the sealed-owner panic.

### Leaf execution: `featurePlan.runLeaf` in `suite.go`

```text
disposed:  Skip
active:    builder → Parallel → setup → Suite Before (outer to inner)
           → Scenario Before → Background (outer to inner) → visible operations
```

Hooks, Background clauses, and visible clauses are one `step` type run as one
ordered sequence. `Error`, `Fatal`, `Skip`, or panic stops the stages after
it; a `Cleanup` registered before setup then logs the stop report.

### Disposition: `meta` in `options.go`

A node is active, skipped, or pending. It is resolved at declaration and
inherited, except that a Case disposition replaces its Outline's. A disposed
node registers a leaf with no builder, hooks, or Background.

### Feature claim: `featureClaims` in `suite.go`

One top-level test owns at most one Feature: unclaimed → claimed → released
by `Cleanup`.

## Failure modes

- A declaration error fails the owning test through `Fatalf` with
  `bdd: <file>:<line>: <problem>`. Ordering errors append the legal order.
- A nil, zero, or sealed scope panics. Its owning test may already have
  finished, and `Fatalf` on a finished test is itself an error. The sealed
  panic is `bdd: <file>:<line>: <call> called on a sealed <scope>: ...` with
  the fix; nil and zero receivers keep fixed messages.
- `Option`, `Example`, and `Compose` constructors have no test to fail, so
  they capture their call site and the declaration or step that uses the value
  reports the problem.

## Decisions

- **One phase value per scope, not independent flags.** Flags admitted
  combinations no declaration sequence produces and spread the grammar over
  every method.
- **Enclosing Suites are suspended during a Rule builder.** Accepting
  `f.Scenario` inside `f.Rule(..., func(r) { ... })` filed the scenario outside
  the Rule, without its tags, hooks, or Background, and let an empty Rule pass
  its non-empty check.
- **Runtime ordering checks, not typestate.** Encoding the order in distinct
  builder types would change the exported builder signatures and still could
  not order separate statements on one receiver.
- **Stop report through `Cleanup`, not `defer`.** A deferred log during
  `Goexit` is attributed to a runtime frame; a cleanup is attributed to the
  frame that registered it, and it runs after `Fatal`, `Skip`, and panic.
- **Declaration errors are not helper-attributed.** They carry the call site
  in the message; marking every declaration frame as a helper would only
  repeat it.
- **Sealed-scope panics name the call site, the call, and the fix.** The
  usual cause, declaring on a Suite from inside a lazily run Scenario builder,
  was not diagnosable from "builder has returned". The text is a diagnostic,
  not a contract; tests match its structure.
- **Chain types are exported.** Methods of unexported types are absent from
  `go doc`, and callers could not name a chain to pass it to a helper. The
  fields stay private and the zero value panics, so a chain still comes only
  from its owner.
- **`featureClaims` is package state.** `testing.T` offers no per-test
  storage, so the registry is keyed by `*testing.T` and released by `Cleanup`.
