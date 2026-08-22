# bdd

Go-native BDD helpers over `testing`. Features are declared with typed Go
closures, while Go's test runner remains the execution and selection engine.

Requires Go 1.27 or newer:

```sh
go get github.com/jacoelho/bdd
```

## A feature

`Feature` takes a per-scenario setup callback. The callback receives the
scenario's `testing.TB`, so resources registered with `Cleanup`, `TempDir`, or
`Context` belong to that scenario.

```go
type cartWorld struct {
	items map[string]int
	total int
}

func newCartWorld(t testing.TB) *cartWorld {
	return &cartWorld{items: make(map[string]int)}
}

func TestCartPricing(t *testing.T) {
	bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
		f.Scenario("item total is calculated", func(s *bdd.ScenarioSpec[*cartWorld]) {
			s.When("the customer adds two books", customerAdds("book", 2))
			s.Then("the total is updated", totalIs(200))
		})
	})
}
```

The generated hierarchy is exactly:

```text
TestCartPricing/feature-cart-pricing/scenario-item-total-is-calculated
```

The Feature is the only grouping subtest. Each scenario or outline example is
one direct child; steps do not create subtests. This makes a scenario atomic
for `go test -run`, and lets every step use the same scenario context and
cleanup lifetime.

## Steps and shared setup

Steps are ordinary typed functions:

```go
func customerAdds(sku string, qty int) bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		w.items[sku] += qty
		w.total = qty * 100
	}
}

func totalIs(want int) bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		if w.total != want {
			t.Fatalf("total = %d; want %d", w.total, want)
		}
	}
}
```

Use `Before` for hidden setup and `Background` for setup that belongs in the
behavior narrative. `Background` continuations are direct `And` and `But`
methods on the Suite:

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Before(loadCatalog())
	f.Background("an empty cart", emptyCart())
	f.And("inventory is available", inventoryAvailable())
	f.Scenario("item total is calculated", func(s *bdd.ScenarioSpec[*cartWorld]) {
		s.When("the customer adds two books", customerAdds("book", 2))
		s.Then("the total is updated", totalIs(200))
	})
})
```

Scenario `Before` hooks are hidden. Scenario `Given`, `When`, `Then`, `And`,
and `But` operations are visible and run in declaration order. `And` and `But`
require a preceding visible clause. A `Pending(text, reason)` operation is
terminal and requires a reason.

## Rules and contexts

Rules and Contexts are logical breadcrumbs. They inherit hooks, Background
operations, and tags but do not add testing subtests:

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Rule("bulk discounts are applied", func(r *bdd.Suite[*cartWorld]) {
		r.Context("premium customer", func(c *bdd.Suite[*cartWorld]) {
			c.Scenario("third book receives a discount", func(s *bdd.ScenarioSpec[*cartWorld]) {
				s.Given("a premium customer", premiumCustomer())
				s.Then("the discount is applied", totalIs(2700))
			})
		})
	})
})
```

The leaf name includes the breadcrumbs:

```text
TestCartPricing/feature-cart-pricing/rule-bulk-discounts-are-applied__context-premium-customer__scenario-third-book-receives-a-discount
```

## Outlines

`Suite.Outline[E]` keeps the example row type at the call site:

```go
type row struct {
	password string
	want     string
}

cases := []bdd.Example[row]{
	bdd.Case("short passwords are weak", row{"abc", "weak"}),
	bdd.Case("long passwords are strong", row{"correct-horse", "strong"}),
}

bdd.Feature(t, "password strength", newPasswordWorld, func(f *bdd.Suite[*passwordWorld]) {
	f.Outline("length-based strength", cases, func(tc row, s *bdd.ScenarioSpec[*passwordWorld]) {
		s.Given("the password is known", passwordIs(tc.password))
		s.Then("the expected strength is returned", strengthIs(tc.want))
	})
})
```

Each example is one direct child named:

```text
feature-password-strength/outline-length-based-strength__example-short-passwords-are-weak
```

`Case` copies its option slice. The row value is shallow-copied, so reference
fields remain owned by the caller. An example's `Skip` or `Pending` option
overrides the outline disposition.

Structured data stays ordinary typed Go data. Use a typed slice for table-like
inputs and a raw string when the behavior under test is a JSON document:

```go
type priceRow struct {
	SKU   string
	Cents int
}

rows := []priceRow{{SKU: "book", Cents: 1200}}
s.Given("these prices exist", seedPrices(rows))
s.When("the JSON request is submitted", submitJSON(`{"sku":"book","quantity":2}`))
```

## Tags, skip, and pending

Tags omit `@`, accept Unicode letters, numbers, combining marks, and hyphenated
segments, and are inherited by descendants:

```go
bdd.Feature(t, "Café pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Scenario("remise appliquée", scenarioBody, bdd.Tags("café", "unit"))
}, bdd.Tags("checkout"))
```

The generated leaf preserves printable UTF-8:

```text
@café_@checkout__scenario-remise-appliquée
```

Tags are present only in the third (scenario/example) path component; Feature
and logical Rule/Context components never contain tag prefixes.

Names and tags are lowercased by rune, but are not Unicode-normalized. A slash
is replaced because Go treats slash as a subtest path separator. Whitespace,
punctuation, symbols, and controls become one `-` separator; separator-only
names are invalid.

Select a tag through standard Go filtering:

```sh
go test ./... -run='//(^|_)@unit(_|$)'
go test ./... -run='//(^|_)@café(_|$)'
```

`-run` still executes matching top-level Go test bodies while they discover
descendants. An unselected BDD leaf does not run its scenario builder, setup,
hooks, Background, or steps. Standard Go behavior applies when nothing
matches: the command exits successfully with status 0. For isolation, keep
BDD features in BDD-only packages or select the exact top-level test as well:

```sh
go test ./cart -run='^TestCartPricing$/^feature-cart-pricing$/^@unit__rule-bulk-discounts-are-applied__scenario-third-book-receives-a-discount$'
```

Use `Skip(reason)` and `Pending(reason)` on Features, Rules, Contexts,
Scenarios, Outlines, or Cases. Reasons are required. Disposed leaves are
registered as skipped terminal children, so no setup, builder, hook,
Background, or step runs. A disposed Outline still registers every example.
Skipped and pending leaves are terminal children of a passing Feature barrier.
The parent therefore reports `PASS` while the selected terminal child reports
`SKIP`; the reason is printed on that child.

## Lifecycle and parallel scenarios

For an active scenario the order is:

```text
scenario builder → Parallel → setup → Suite Before → Scenario Before
→ Background → visible operations → context cancellation → Cleanup
```

Setup runs once per selected active scenario or example, after `Parallel`
resumes. It is never called for a filtered or disposed leaf. It may return any
value, including a nil pointer or the zero value of `NoState`. `Error`,
`Fatal`, `Skip`, or a panic from setup gates hooks, Background, and visible
operations using the standard `testing` behavior; setup-registered cleanup
still runs once. The scenario context is canceled before those cleanups run.

Feature, Rule, Context, and Scenario builders are declaration-only. They are
compiled before executable children are registered, except that active
Scenario and Outline-example builders are lazy and run only in their selected
leaf. Setup, hooks, and Steps execute directly in that atomic leaf: they must
not type-assert `testing.TB` to create subtests or call `Parallel`.

Call `s.Parallel()` only when setup, hooks, steps, and captured data are safe to
run concurrently. The library does not clone or roll back state. `Setenv` and
`Chdir` remain subject to the standard testing restrictions for parallel
tests.

## Migration from v0.0.x

Before:

```go
bdd.Feature(t, "pricing", newWorld, func(f *bdd.Suite[*world]) {
	f.Scenario("discount applies", func(s *bdd.ScenarioSpec[*world]) {
		s.Then("the total is discounted", discountIsApplied()).And("the receipt is sent", receiptIsSent())
	})
})
```

After:

```go
bdd.Feature(t, "pricing", newWorld, func(f *bdd.Suite[*world]) {
	f.Scenario("discount applies", func(s *bdd.ScenarioSpec[*world]) {
		s.Then("the total is discounted", discountIsApplied())
		s.And("the receipt is sent", receiptIsSent())
	})
})
```

- `StateFactory` is removed. Pass a normal `func(testing.TB) S` callback.
- The top-level `Outline` function is replaced by `f.Outline(...)`.
- `Noop` is removed; use a small ordinary `Step` closure where needed.
- `Suite.Tags`, `ScenarioSpec.Tags`, and `ScenarioSpec.Background` are removed.
- `Example.Opts` is renamed to `Example.Options`.
- Declaration methods return nothing; replace chained calls with direct `And`
  and `But` calls.
- Scenario-specific setup formerly expressed as `ScenarioSpec.Background` is
  either a `ScenarioSpec.Before` hook or a visible `Given` operation.
- Step-created contexts and cleanups now live until scenario completion rather
  than until an individual step subtest completes.
- Scenario builders are lazy and do not run for filtered or disposed leaves.
- A direct top-level `Test...` owns one Feature; split tests that declared
  multiple Features.
