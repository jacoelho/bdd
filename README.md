# bdd

`github.com/jacoelho/bdd` provides a typed, inline BDD DSL over Go's standard
`testing` package. It requires Go 1.27 or newer and has no dependencies.

```sh
go get github.com/jacoelho/bdd
```

The [tutorial](#tutorial) grows one cart example, one construct per step. The
[reference](#reference) states the rules behind it. For signatures, run
`go doc github.com/jacoelho/bdd`.

## Tutorial

### 1. Write a first scenario

A Feature holds scenarios. Each scenario receives fresh state from the setup
function, here `newWorld`, and runs its `Given`, `When`, and `Then` steps in
order. A step is a function of `testing.TB` and that state.

```go
package cart_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

var prices = map[string]int{"book": 1000, "pen": 200} // cents

// cartTotal is the code under test: three or more books save 300.
func cartTotal(items map[string]int) int {
	total := 0
	for sku, quantity := range items {
		total += quantity * prices[sku]
	}
	if items["book"] >= 3 {
		total -= 300
	}
	return total
}

// world is the state one scenario works on.
type world struct {
	items map[string]int // SKU to quantity
}

func newWorld(testing.TB) *world {
	return &world{items: make(map[string]int)}
}

func TestFirstScenario(t *testing.T) {
	bdd.Feature(t, "cart", newWorld, func(f *bdd.Suite[*world]) {
		f.Scenario("two books cost 2000", func(s *bdd.ScenarioSpec[*world]) {
			s.Given("an empty cart", func(_ testing.TB, w *world) {
				clear(w.items)
			}).
				When("the customer adds two books", func(_ testing.TB, w *world) {
					w.items["book"] += 2
				}).
				Then("the total is 2000", func(t testing.TB, w *world) {
					if got := cartTotal(w.items); got != 2000 {
						t.Fatalf("total = %d; want 2000", got)
					}
				})
		})
	})
}
```

`go test -v -run TestFirstScenario` prints the narrative. Each line ends with
the file and line that declared it:

```text
=== RUN   TestFirstScenario
=== RUN   TestFirstScenario/feature-cart
=== RUN   TestFirstScenario/feature-cart/scenario-two-books-cost-2000
    Scenario: two books cost 2000       # cart_test.go:34
      Given an empty cart               # cart_test.go:35
      When the customer adds two books  # cart_test.go:38
      Then the total is 2000            # cart_test.go:41
--- PASS: TestFirstScenario (0.00s)
    --- PASS: TestFirstScenario/feature-cart (0.00s)
        --- PASS: TestFirstScenario/feature-cart/scenario-two-books-cost-2000 (0.00s)
```

The generated path is exactly
`TestFirstScenario/feature-cart/scenario-two-books-cost-2000`. The Feature is
the only grouping subtest. Every Scenario or Outline example is one direct child, called
a leaf below. Steps do not create subtests, so a selected scenario is atomic
and uses one `testing.T` for setup, hooks, background, steps, cleanup, and
context.

### 2. Reuse steps

Move the mechanics into functions that return a `bdd.Step`, so the scenario
reads as narrative. `And` and `But` continue the chain:

```go
func emptyCart() bdd.Step[*world] {
	return func(_ testing.TB, w *world) {
		clear(w.items)
	}
}

func customerAdds(sku string, quantity int) bdd.Step[*world] {
	return func(_ testing.TB, w *world) {
		w.items[sku] += quantity
	}
}

func totalIs(want int) bdd.Step[*world] {
	return func(t testing.TB, w *world) {
		if got := cartTotal(w.items); got != want {
			t.Fatalf("total = %d; want %d", got, want)
		}
	}
}

func TestReusableSteps(t *testing.T) {
	bdd.Feature(t, "cart totals", newWorld, func(f *bdd.Suite[*world]) {
		f.Scenario("books and pens add up", func(s *bdd.ScenarioSpec[*world]) {
			s.Given("an empty cart", emptyCart()).
				When("the customer adds two books", customerAdds("book", 2)).
				And("the customer adds a pen", customerAdds("pen", 1)).
				Then("the total is 2200", totalIs(2200)).
				But("no discount applies", totalIs(2*prices["book"]+prices["pen"]))
		})
	})
}
```

`Compose` joins several steps into one clause:

```go
func cartWithStationery() bdd.Step[*world] {
	return bdd.Compose(
		emptyCart(),
		customerAdds("pen", 2),
	)
}
```

`Compose` executes its components in order as one logged clause. It clones the
component slice at construction, fails an empty or nil component list when it
runs (naming the `Compose` call site), and stops after a failure or skip.
Fluent clauses remain individually logged operations.

Structured data stays ordinary Go data: pass rows, structs, or JSON to a step
function as arguments. There is no reflection registry or magic parameter
convention.

### 3. Share setup with `Background` and `Before`

`Background` declares visible steps that run before every scenario in the
Suite. `Before` registers a hidden hook for setup the narrative should not
mention:

```go
func logFinalCart() bdd.Step[*world] {
	return func(t testing.TB, w *world) {
		t.Cleanup(func() { t.Logf("final cart: %v", w.items) })
	}
}

func TestSharedSetup(t *testing.T) {
	bdd.Feature(t, "cart with shared setup", newWorld, func(f *bdd.Suite[*world]) {
		f.Before(logFinalCart())
		f.Background("an empty cart", emptyCart()).
			And("the customer adds a pen", customerAdds("pen", 1))

		f.Scenario("the pen alone costs 200", func(s *bdd.ScenarioSpec[*world]) {
			s.Then("the total is 200", totalIs(200))
		})
		f.Scenario("a book raises the total", func(s *bdd.ScenarioSpec[*world]) {
			s.When("the customer adds a book", customerAdds("book", 1)).
				Then("the total is 1200", totalIs(1200))
		})
	})
}
```

`Background` returns a chain that exposes only `And` and `But`. A scenario can
register its own hook with `s.Before`. Hooks run before Background; the full
order is under [Setup and lifecycle](#setup-and-lifecycle).

### 4. Group scenarios with `Rule`

`Rule` groups scenarios and nests recursively. It adds a `rule-<slug>`
breadcrumb to the generated test name and inherits outer hooks, background
operations, and tags, but does not create a test node:

```go
func TestRules(t *testing.T) {
	bdd.Feature(t, "cart discounts", newWorld, func(f *bdd.Suite[*world]) {
		f.Background("an empty cart", emptyCart())

		f.Rule("three or more books save 300", func(r *bdd.Suite[*world]) {
			r.Scenario("two books pay full price", func(s *bdd.ScenarioSpec[*world]) {
				s.When("the customer adds two books", customerAdds("book", 2)).
					Then("the total is 2000", totalIs(2000))
			})
			r.Rule("the discount applies once", func(r *bdd.Suite[*world]) {
				r.Scenario("six books save 300", func(s *bdd.ScenarioSpec[*world]) {
					s.When("the customer adds six books", customerAdds("book", 6)).
						Then("the total is 5700", totalIs(5700))
				})
			})
		})
	})
}
```

The two scenarios are still direct children of the Feature:

```text
TestRules/feature-cart-discounts/rule-three-or-more-books-save-300__scenario-two-books-pay-full-price
TestRules/feature-cart-discounts/rule-three-or-more-books-save-300__rule-the-discount-applies-once__scenario-six-books-save-300
```

A Rule builder receives its own `*bdd.Suite`, which can declare `Before` and
`Background` for that Rule. Declare on that parameter, not on the enclosing
Suite; see [Declaration order](#declaration-order). Empty active Features,
Rules, Scenarios, and Outlines are structural errors. Builders declare
behavior only; they must not perform application effects.

### 5. Run one scenario per row with `Outline`

`Outline` runs its builder once per `Case`. Each example is its own leaf:

```go
func TestOutline(t *testing.T) {
	type row struct {
		books int
		want  int
	}
	examples := []bdd.Example[row]{
		bdd.Case("one book", row{books: 1, want: 1000}),
		bdd.Case("two books", row{books: 2, want: 2000}),
		bdd.Case("three books earn the discount", row{books: 3, want: 2700}),
	}

	bdd.Feature(t, "book pricing", newWorld, func(f *bdd.Suite[*world]) {
		f.Outline("books are priced by quantity", examples, func(tc row, s *bdd.ScenarioSpec[*world]) {
			s.Given("an empty cart", emptyCart()).
				When("the customer adds the books", customerAdds("book", tc.books)).
				Then("the total matches", totalIs(tc.want))
		})
	})
}
```

```text
TestOutline/feature-book-pricing/outline-books-are-priced-by-quantity__example-one-book
TestOutline/feature-book-pricing/outline-books-are-priced-by-quantity__example-two-books
TestOutline/feature-book-pricing/outline-books-are-priced-by-quantity__example-three-books-earn-the-discount
```

The generic row type is inferred by Go 1.27 from `examples` and the callback.
Use `Case`; `Example` is opaque and has no exported fields, and a zero
`Example` is invalid. The option slice is copied, while the row value is
shallow-copied, so reference fields remain caller-owned.

### 6. Tag and select scenarios

`bdd.Tags` is an option for Features, Rules, Scenarios, Outlines, and Cases.
Tags are inherited and appear in the generated name, so `go test -run` can
select them:

```go
func TestTags(t *testing.T) {
	bdd.Feature(t, "tagged cart", newWorld, func(f *bdd.Suite[*world]) {
		f.Scenario("a pen costs 200", func(s *bdd.ScenarioSpec[*world]) {
			s.When("the customer adds a pen", customerAdds("pen", 1)).
				Then("the total is 200", totalIs(200))
		}, bdd.Tags("smoke"))

		f.Rule("discounts", func(r *bdd.Suite[*world]) {
			r.Scenario("three books save 300", func(s *bdd.ScenarioSpec[*world]) {
				s.When("the customer adds three books", customerAdds("book", 3)).
					Then("the total is 2700", totalIs(2700))
			})
		}, bdd.Tags("pricing"))
	}, bdd.Tags("unit"))
}
```

```text
TestTags/feature-tagged-cart/@smoke_@unit__scenario-a-pen-costs-200
TestTags/feature-tagged-cart/@pricing_@unit__rule-discounts__scenario-three-books-save-300
```

```sh
go test ./... -run='//(^|_)@smoke(_|$)'
go test ./... -run='^TestTags$/^feature-tagged-cart$/^@smoke_@unit__scenario-a-pen-costs-200$'
```

The first command runs every leaf tagged `smoke`; the second selects one leaf
by its exact path. A selector that starts with `//` matches every top-level
test, so each top-level test body still runs to discover its descendants. If
nothing matches, `go test` still exits successfully. For isolation, use
BDD-only packages or select the exact top-level test as in the second command.
[Names and tags](#names-and-tags) has the naming rules.

### 7. Skip scenarios or mark them pending

`Skip(reason)` and `Pending(reason)` are node options for Features, Rules,
Scenarios, Outlines, and Cases; a node with either option is disposed. A
disposed Feature, Rule, or Scenario may use a nil builder:

```go
func TestSkipAndPending(t *testing.T) {
	bdd.Feature(t, "checkout", newWorld, func(f *bdd.Suite[*world]) {
		f.Scenario("the cart total is shown", func(s *bdd.ScenarioSpec[*world]) {
			s.When("the customer adds a pen", customerAdds("pen", 1)).
				Then("the total is 200", totalIs(200))
		})
		f.Scenario("gift cards reduce the total", nil, bdd.Pending("gift cards are not designed yet"))
		f.Rule("refunds", nil, bdd.Skip("blocked by the payment sandbox"))
	})
}
```

```text
    cart_test.go:167: bdd: Scenario: gift cards reduce the total pending: gift cards are not designed yet
    cart_test.go:167: bdd: Rule: refunds skipped: blocked by the payment sandbox
--- PASS: TestSkipAndPending (0.00s)
    --- PASS: TestSkipAndPending/feature-checkout (0.00s)
        --- PASS: TestSkipAndPending/feature-checkout/scenario-the-cart-total-is-shown (0.00s)
        --- SKIP: TestSkipAndPending/feature-checkout/scenario-gift-cards-reduce-the-total (0.00s)
        --- SKIP: TestSkipAndPending/feature-checkout/rule-refunds (0.00s)
```

Reasons must be nonblank. The name and options of a disposed node are still
validated, and no user builder or setup code runs. A node option disposes the
whole leaf: no setup, prior operation, or builder operation runs for it. Split
any independently valuable setup or behavior into another active Scenario.

A disposed node is registered as a skipped direct child of the Feature
subtest, not as a skipped Feature, so its tags remain selectable. That child
reports `SKIP`; the Feature subtest and the top-level test report `PASS`. A
disposed Feature reports, for example:

```text
bdd: Feature "cart pricing" skipped: blocked by inventory service
```

A disposed Outline still registers every Case, and a Case option can override
its Outline: a Case disposition overrides the Outline disposition for that
row. An Outline with a nil builder is allowed only when every effective Case
is disposed; an active row always requires a builder. A nil Rule builder is
legal only when a Skip or Pending option makes it non-executable.

There is no `ScenarioSpec.Pending` operation.
[Migration from the previous API](#migration-from-the-previous-api) shows the
v0.0.3 equivalent.

### 8. Run scenarios in parallel

`s.Parallel()` marks a scenario as safe to run in parallel with other tests.
Declare it before the first step:

```go
func TestParallel(t *testing.T) {
	bdd.Feature(t, "parallel carts", newWorld, func(f *bdd.Suite[*world]) {
		for sku, price := range prices {
			f.Scenario("one "+sku, func(s *bdd.ScenarioSpec[*world]) {
				s.Parallel()
				s.Given("an empty cart", emptyCart()).
					When("the customer adds one "+sku, customerAdds(sku, 1)).
					Then("the total is its price", totalIs(price))
			})
		}
	})
}
```

Each scenario still receives its own state from setup, which runs after the
scenario resumes. Callers remain responsible for state isolation and
concurrency safety. `Setenv` and `Chdir` retain standard testing restrictions
below parallel ancestors.

## Reference

### Declaration order

Each builder accepts its declarations in one order. The numbered comments
give that order:

```go
func TestDeclarationOrder(t *testing.T) {
	bdd.Feature(t, "declaration order", newWorld, func(f *bdd.Suite[*world]) {
		// 1. Before hooks: any number.
		f.Before(logFinalCart())

		// 2. Background: optional, at most one chain, continued with And or But.
		f.Background("an empty cart", emptyCart()).
			And("the customer adds a pen", customerAdds("pen", 1))

		// 3. Children: at least one Scenario, Outline, or Rule, in any order.
		f.Scenario("the pen costs 200", func(s *bdd.ScenarioSpec[*world]) {
			// 1. Parallel (at most once) and Before hooks, in any order.
			s.Parallel()
			s.Before(logFinalCart())

			// 2. Exactly one chain: a Given, When, or Then starter, continued
			//    with Given, When, Then, And, or But.
			s.Then("the total is 200", totalIs(200))
		})
	})
}
```

A declaration out of order fails the owning test with its call site and the
legal order:

```text
bdd: /src/cart_test.go:31: Before must be declared before Background; a Suite declares Before hooks, then one Background chain, then Scenario, Outline, and Rule children
```

While a Rule builder runs, its enclosing Suites are suspended, so a scenario
cannot be filed outside the Rule it is written in. Declare on the builder's
own parameter:

```go
f.Rule("bulk discounts", func(r *bdd.Suite[*world]) {
	r.Scenario("the discount applies", discountApplies) // not f.Scenario
})
```

Every scope is sealed when its builder returns. Using a Suite, ScenarioSpec,
or chain afterwards panics, naming the call site, the call, and the fix. The
usual cause is declaring on the Suite from inside a Scenario builder, which
runs later:

```text
bdd: /src/cart_test.go:40: Suite.Before called on a sealed Suite: its Feature or Rule builder has returned; declare inside that builder, and inside a Scenario builder declare on its *ScenarioSpec
```

### Chains

A scenario has one linear narrative chain. `Given`, `When`, and `Then` are the
three starters; every later keyword (`Given`, `When`, `Then`, `And`, or `But`)
is called on the returned chain. Keyword order and repetition are
intentionally flexible, following the
[Gherkin reference](https://cucumber.io/docs/gherkin/reference).

`ScenarioChain` and `BackgroundChain` are exported so helpers can accept and
return a chain; obtain one only from a starter or `Background`, because the
zero value panics:

```go
func thenTotalIs(chain bdd.ScenarioChain[*world], want int) bdd.ScenarioChain[*world] {
	return chain.Then("the total is right", totalIs(want))
}

func TestChainInVariable(t *testing.T) {
	bdd.Feature(t, "chains", newWorld, func(f *bdd.Suite[*world]) {
		f.Scenario("a chain is stored and passed on", func(s *bdd.ScenarioSpec[*world]) {
			chain := s.When("the customer adds a pen", customerAdds("pen", 1))
			chain = chain.Given("the cart held nothing else", totalIs(200))
			chain = thenTotalIs(chain, 200)
			_ = chain.But("no discount applies", totalIs(prices["pen"]))
		})
	})
}
```

Reassign every returned chain when using a variable. Copying a chain is
harmless. While its builder is active, using a chain that is not the latest
fails the owning test at that call; the message names where the stale chain
was issued and says to assign every returned value. Using any chain after its
builder returns panics with the sealed-scope diagnostic described under
[Declaration order](#declaration-order). Ignoring the result of a single
starter is valid. A second direct starter is rejected; use the returned chain
instead. `Background` exposes only `And` and `But`, and the same reassignment
rule applies.

### Setup and lifecycle

`setup` runs exactly once for each selected active Scenario or Outline example,
after that leaf's `Parallel` has resumed and before any hook or operation. It
receives the leaf's `testing.TB`, may use `Context`, `Cleanup`, and `TempDir`,
and may return the zero value of `S`, including a nil pointer. Setup-created
resources live until scenario completion. `Error`, `Fatal`, `Skip`, or panic
uses standard Go testing behavior: later hooks, background, and steps do not
run, while registered cleanup runs once. The scenario context is canceled
before cleanup.

For an active leaf the order is:

```text
scenario builder → Parallel → setup → outer-to-inner Suite Before
→ Scenario Before → outer-to-inner Background → visible operations
→ context cancellation → Cleanup
```

Setup, hooks, and steps must not type-assert `testing.TB` to create subtests or
call `Parallel`; doing so breaks the atomicity contract.

Scenario and Outline-example builders are lazy: they run only after their
leaf is selected. A filtered or disposed leaf invokes no builder, setup, hook,
background, step, or disposition callback. Because an unselected builder does
not run, its declaration errors stay hidden: after a migration, run
`go test ./...` without a selector to find them.

### Failure output

With `-v`, or when it fails, a scenario prints its narrative: its Rules or
Outline on one line when it has any, its title, then each hook and operation
as it starts. Every line ends with the file and line that declared it. A
scenario stopped by `Error`, `Fatal`, `Skip`, or panic during setup, a hook,
or an operation then reports the operation it stopped in, where that operation
was declared, and the operations that did not run:

```text
--- FAIL: TestFirstScenario (0.00s)
    --- FAIL: TestFirstScenario/feature-cart (0.00s)
        --- FAIL: TestFirstScenario/feature-cart/scenario-two-books-cost-2000 (0.00s)
            Scenario: two books cost 2000       # cart_test.go:34
              Given an empty cart               # cart_test.go:35
              When the customer adds two books  # cart_test.go:38
              Then the total is 2000            # cart_test.go:41
            cart_test.go:43: total = 2700; want 2000
            bdd: scenario stopped in Then "the total is 2000" declared at cart_test.go:41; not run: And "the cart holds only books"
```

Lines logged by setup, hooks, and steps keep the standard `file:line:` prefix
and appear under the operation that logged them.

### Names and tags

Every generated path has exactly three components:

```text
<TestFunction>/feature-<slug>/<scenario-or-outline-breadcrumbs>
```

Rules are logical breadcrumbs. Tags appear only in the third component:

```text
TestCartPricing/feature-café-pricing/@unit__rule-bulk__scenario-remise-appliquée
```

Names trim Unicode whitespace, lowercase rune by rune, preserve Unicode
letters, numbers, and combining marks inside tokens, and collapse whitespace,
`/`, punctuation, symbols, and controls to one ASCII `-`. `/` is collapsed
because Go uses it to separate subtest path components. Names and tags are not
canonically normalized: precomposed and decomposed forms remain distinct.

Pass tags without `@`; each hyphen-separated segment starts with a Unicode
letter or number and may contain Unicode letters, numbers, or combining marks.
Tags are lowercased, inherited, deduplicated, and sorted by their normalized
UTF-8 strings (without NFC/NFD normalization). UTF-8 tags match as written:

```sh
go test ./... -run='//(^|_)@café(_|$)'
```

A Feature named `Café pricing` generates `feature-café-pricing`.

## Migration from the previous API

The first example migrates from the unreleased API at commit `7a3233f`, not
from the released v0.0.3 API:

Before (`7a3233f` migration branch):

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Background("an empty cart", emptyCart())
	f.And("inventory is available", inventoryAvailable())
	f.Scenario("discount applies", func(s *bdd.ScenarioSpec[*cartWorld]) {
		s.Given("a premium customer", premiumCustomer())
		s.When("three books are added", addBooks(3))
		s.Then("the discount is applied", discountIsApplied())
	})
})
```

After:

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Background("an empty cart", emptyCart()).
		And("inventory is available", inventoryIsAvailable())
	f.Scenario("discount applies", func(s *bdd.ScenarioSpec[*cartWorld]) {
		s.Given("a premium customer", premiumCustomer()).
			When("three books are added", addBooks(3)).
			Then("the discount is applied", discountIsApplied())
	})
})
```

For released v0.0.3 users, the relevant shapes were different:

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Background("an empty cart", emptyCart()) // returns nothing
	f.Context("premium customer", func(c *bdd.Suite[*cartWorld]) {
		c.Scenario("pricing policy", func(s *bdd.ScenarioSpec[*cartWorld]) {
			s.Given("the cart is ready", cartIsReady()).
				And("the customer is premium", premiumCustomer())
			s.Pending("the discount is applied", "waiting for policy")
		})
	})
})
```

In that release, `ScenarioSpec` fluent `And`/`But` continued the returned
scenario chain, while `Suite.Background` returned nothing and `Context` was a
public grouping method. The current API uses `Background(...).And(...)`,
nested `Rule`, and a node-level `Pending` option.

In v0.0.3, setup and all operations declared before `s.Pending` ran before the
pending step. A node option now disposes the whole leaf:

Before (v0.0.3):

```go
f.Scenario("pricing policy", func(s *bdd.ScenarioSpec[*cartWorld]) {
	s.Given("the cart is ready", cartIsReady()).
		And("pricing is undecided", pricingIsUndecided())
	s.Pending("the policy is applied", "waiting for policy")
})
```

After:

```go
f.Scenario("pricing policy is pending", nil, bdd.Pending("waiting for policy"))
```

The breaking migration also removes:

- `StateFactory` (present in v0.0.3); pass an ordinary unnamed
  `func(testing.TB) S` setup callback. No `Setup` or `ScenarioSetup`
  compatibility aliases are introduced.
- The top-level `Outline`; use `f.Outline(...)`.
- `Noop`; use an ordinary typed `Step` closure.
- `Suite.Tags`, `ScenarioSpec.Tags`, `ScenarioSpec.Background`, direct Suite or
  ScenarioSpec `And`/`But`, and `ScenarioSpec.Pending`.
- `Suite.Context`; there is no compatibility path. Use nested `Rule`, update
  `context-` selectors to `rule-`, and rename the Rule if its normalized slug
  collides with an existing sibling Rule.
- `Example.Opts` and all other exported Example fields; construct rows with
  `Case` and do not mutate them through fields.

Declaration methods no longer return a chain, except `Background`, `Given`,
`When`, and `Then`. Split multiple Features into separate top-level `Test...`
functions because one direct top-level test owns one Feature.

Step-created `Context`, `Cleanup`, and `TempDir` resources now live until
scenario completion rather than until a step subtest completes. The framework
does not add compatibility aliases or deferred Gherkin/Table APIs.

## Architecture

[ARCHITECTURE.md](ARCHITECTURE.md) records the state machines, their owners,
and the design decisions behind them.
