# bdd

`github.com/jacoelho/bdd` provides a typed, inline BDD DSL over Go's standard
`testing` package. It requires Go 1.27 or newer and has no dependencies.

## Quick start

```go
type cartWorld struct {
	items map[string]int
	total int
}

func newCartWorld(testing.TB) *cartWorld {
	return &cartWorld{items: make(map[string]int)}
}

func TestCartPricing(t *testing.T) {
	bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
		f.Scenario("item total is calculated", func(s *bdd.ScenarioSpec[*cartWorld]) {
			s.Given("an empty cart", func(_ testing.TB, w *cartWorld) {
				w.items = make(map[string]int)
				w.total = 0
			}).
				When("the customer adds two books", func(_ testing.TB, w *cartWorld) {
					w.items["book"] = 2
					w.total = 200
				}).
				Then("the total is updated", func(t testing.TB, w *cartWorld) {
					if w.total != 200 {
						t.Fatalf("total = %d; want 200", w.total)
					}
				})
		})
	})
}
```

The generated hierarchy is exactly:

```text
TestCartPricing/feature-cart-pricing/scenario-item-total-is-calculated
```

The Feature is the only grouping subtest. Every Scenario or Outline example
is one direct child. Steps do not create subtests, so a selected scenario is
atomic and uses one `testing.T` for setup, hooks, background, steps, cleanup,
and context.

## Public API

The following is a conceptual signature block. The two chain result types are
private; use the executable call patterns below rather than naming them.

```go
func Feature[S any](t *testing.T, name string, setup func(testing.TB) S,
	build func(*Suite[S]), opts ...Option)

type Step[S any] func(testing.TB, S)
type Suite[S any] struct{}
type ScenarioSpec[S any] struct{}

func (s *Suite[S]) Before(Step[S])
func (s *Suite[S]) Background(string, Step[S]) backgroundChain[S]
func (s *Suite[S]) Rule(string, func(*Suite[S]), ...Option)
func (s *Suite[S]) Scenario(string, func(*ScenarioSpec[S]), ...Option)
func (s *Suite[S]) Outline[E any](string, []Example[E], func(E, *ScenarioSpec[S]), ...Option)

func (s *ScenarioSpec[S]) Parallel()
func (s *ScenarioSpec[S]) Before(Step[S])
func (s *ScenarioSpec[S]) Given(string, Step[S]) scenarioChain[S]
func (s *ScenarioSpec[S]) When(string, Step[S]) scenarioChain[S]
func (s *ScenarioSpec[S]) Then(string, Step[S]) scenarioChain[S]

func Compose[S any](...Step[S]) Step[S]
func Case[E any](string, E, ...Option) Example[E]
func Tags(...string) Option
func Skip(string) Option
func Pending(string) Option
```

The chain result types are private, but their exported methods are available at
the call site. A scenario has one linear narrative chain. `Given`, `When`, and
`Then` are the three starters; every later keyword (`Given`, `When`, `Then`,
`And`, or `But`) is called on the returned token. Keyword order and repetition
are intentionally flexible, following the [Gherkin reference](https://cucumber.io/docs/gherkin/reference).

```go
chain := s.When("the request is received", receiveRequest())
chain = chain.Given("the account is active", accountIsActive())
chain = chain.Then("the response is successful", responseIsSuccessful())
_ = chain.But("no error is reported", noError())
```

Reassign every returned token when using a variable. Copying a token is
harmless. Using a stale token while its builder is active fails through the
owning declaration/test at the continuation call, with the issuing source and
guidance to reassign the returned value. Using any token after its builder
returns panics with the owner-sealed `bdd:` diagnostic. Ignoring the result of
a single starter is valid. A second direct starter is rejected; use the
returned chain instead. `Background` exposes only `And` and `But`, and the
same reassignment rule applies:

```go
f.Background("the catalog is loaded", loadCatalog()).
	And("inventory is available", inventoryIsAvailable()).
	But("the maintenance window is closed", maintenanceIsClosed())
```

`Compose` executes its components in order as one logged clause. It clones the
component slice at construction, validates empty or nil components when it
runs, and stops after a failure or skip. Fluent clauses remain individually
logged operations.

## Rules and scenarios

`Rule` is a logical, recursively nestable breadcrumb. It inherits outer hooks,
background operations, and tags but does not create a test node. A nil Rule
builder is legal only when a Skip or Pending option makes it non-executable:

```go
f.Rule("bulk discounts", func(r *bdd.Suite[*cartWorld]) {
	r.Rule("premium customers", func(r *bdd.Suite[*cartWorld]) {
		r.Scenario("the discount applies", func(s *bdd.ScenarioSpec[*cartWorld]) {
			s.Given("a premium customer", premiumCustomer()).
				Then("the discount is applied", discountIsApplied())
		})
	})
})
```

There is no `Context` compatibility path. Migrate a former Context to a
nested Rule, then audit selectors and rename if its normalized slug collides
with an existing sibling Rule. Empty active Features, Rules, Scenarios, and
Outlines are structural errors. Builders declare behavior only; they must not
perform application effects.

## Outlines and typed data

Use `Case`; `Example` is opaque and has no exported fields. The option slice is
copied, while the row value is shallow-copied, so reference fields remain
caller-owned:

```go
type passwordCase struct {
	password string
	want     string
}

type passwordWorld struct {
	password string
	strength string
}

func newPasswordWorld(testing.TB) *passwordWorld {
	return &passwordWorld{}
}

func TestPasswordStrength(t *testing.T) {
	cases := []bdd.Example[passwordCase]{
		bdd.Case("short password", passwordCase{"abc", "weak"}),
		bdd.Case("long password", passwordCase{"correct-horse", "strong"}, bdd.Tags("strong")),
	}

	bdd.Feature(t, "password strength", newPasswordWorld, func(f *bdd.Suite[*passwordWorld]) {
		f.Outline("length-based strength", cases, func(tc passwordCase, s *bdd.ScenarioSpec[*passwordWorld]) {
			s.Given("the password is known", func(_ testing.TB, w *passwordWorld) {
				w.password = tc.password
			}).
				When("the password is evaluated", func(_ testing.TB, w *passwordWorld) {
					if len(w.password) >= 12 {
						w.strength = "strong"
					} else {
						w.strength = "weak"
					}
				}).
				Then("the expected strength is returned", func(t testing.TB, w *passwordWorld) {
					if w.strength != tc.want {
						t.Fatalf("strength = %s; want %s", w.strength, tc.want)
					}
				})
		})
	})
}
```

The generic row type is inferred by Go 1.27 from `cases` and the callback. A
zero `Example` is invalid; use `bdd.Case`. A Case disposition overrides the
Outline disposition for that row. An Outline with a nil builder is allowed
only when every effective Case is disposed (`Skip` or `Pending`); an active
row always requires a builder.

Structured data stays ordinary Go data—there is no reflection registry or
magic parameter convention:

```go
type priceRow struct {
	SKU   string
	Cents int
}

rows := []priceRow{{SKU: "book", Cents: 1200}}
s.Given("these prices exist", seedPrices(rows)).
	When("the JSON request is submitted", submitJSON(`{"sku":"book","quantity":2}`))
```

## Setup and lifecycle

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
call `Parallel`; doing so breaks the atomicity contract. Callers remain
responsible for state isolation and concurrency safety. `Setenv` and `Chdir`
retain standard testing restrictions below parallel ancestors.

Scenario and Outline-example builders are lazy: they run only after their
leaf is selected. A filtered or disposed leaf invokes no builder, setup, hook,
background, step, or disposition callback. Therefore run exactly
`go test ./...` after migration to discover declarations hidden by a selector.

## Names, tags, and selection

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
`/`, punctuation, symbols, and controls to one ASCII `-`. `/` must be treated
as a separator because Go uses it for subtest paths. Names and tags are not
canonically normalized: precomposed and decomposed forms remain distinct.

Tags omit `@`; each hyphen-separated segment starts with a Unicode letter or
number and may contain Unicode letters, numbers, or combining marks. Tags are
lowercased, inherited, deduplicated, and sorted by their normalized UTF-8
strings (without NFC/NFD normalization).

```sh
go test ./... -run='//(^|_)@unit(_|$)'
go test ./... -run='//(^|_)@café(_|$)'
go test ./cart -run='^TestCartPricing$/^feature-cart-pricing$/^scenario-item-total-is-calculated$'
```

The last command selects the direct Scenario shown in Quick Start. The
`@café` command demonstrates UTF-8 tag matching; a Feature named `Café
pricing` would generate `feature-café-pricing`.

`//tag` still executes matching top-level Go test bodies while they discover
descendants. A standard Go zero-match run exits successfully. For isolation,
use BDD-only packages or select the exact top-level test as in the last
example.

## Skip and pending

`Skip(reason)` and `Pending(reason)` are node options for Features, Rules,
Scenarios, Outlines, and Cases. Reasons must be nonblank. A disposed Feature,
Rule, or Scenario may use a nil builder after names/options have been
validated; no user builder or setup code runs. A disposed Outline still
registers every Case, and a Case option can override its Outline.

Dispositions are terminal direct children rather than skipped Feature
barriers, so tags remain selectable. The terminal child reports `SKIP`; its
Feature and top-level parents report `PASS`:

```text
bdd: Feature "cart pricing" skipped: blocked by inventory service
```

There is no `ScenarioSpec.Pending` operation. In the released v0.0.3 API,
setup and all operations declared before `s.Pending` ran before the pending
step. In the new API, a node option disposes the whole leaf, so split any
independently valuable setup or behavior into another active Scenario:

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

No setup, prior operation, or builder operation runs for the disposed node.

## Migration from the previous API

The first example is source truth for the immediate `7a3233f` migration branch
API, not a released v0.0.3 API:

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

The breaking migration also removes:

- `StateFactory` (removed from v0.0.3); pass an ordinary unnamed
  `func(testing.TB) S` setup callback. No `Setup` or `ScenarioSetup`
  compatibility aliases are introduced.
- The top-level `Outline`; use `f.Outline(...)`.
- `Noop`; use an ordinary typed `Step` closure.
- `Suite.Tags`, `ScenarioSpec.Tags`, `ScenarioSpec.Background`, direct Suite or
  ScenarioSpec `And`/`But`, and `ScenarioSpec.Pending`.
- `Suite.Context`; use nested `Rule` and update `context-` selectors to
  `rule-`.
- `Example.Opts` and all other exported Example fields; construct rows with
  `Case` and do not mutate them through fields.

Declaration methods are no longer chain-returning methods except for the two
private fluent capabilities described above. Split multiple Features into
separate top-level `Test...` functions because one direct top-level test owns
one Feature.

Step-created `Context`, `Cleanup`, and `TempDir` resources now live until
scenario completion rather than until a step subtest completes. The framework
does not add compatibility aliases or deferred Gherkin/Table APIs.
