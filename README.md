# bdd

Go-native BDD helpers over `testing`.

Tests still run with `go test`. Features, rules, contexts, scenarios, and steps
are subtests. Step functions receive `testing.TB`, so normal Go assertions,
helpers, cleanup, temp dirs, and contexts keep working.

## Install

```sh
go get github.com/jacoelho/bdd
```

Requires Go 1.26.2 or newer.

## 1. Write a Feature

A feature owns a typed scenario state. Each scenario gets a fresh state from the
state factory.

```go
package checkout_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

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

Run it:

```sh
go test ./...
```

## 2. Move Mechanics into Steps

Steps are plain Go functions returning `bdd.Step[S]`.

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

Keep scenario text readable. Put setup, actions, and assertions in reusable
steps.

## 3. Share Setup with Background

Use `Background` when setup is part of the story.

```go
func emptyCart() bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		w.items = make(map[string]int)
	}
}

func TestCartPricing(t *testing.T) {
	bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
		f.Background("an empty cart", emptyCart())

		f.Scenario("item total is calculated", func(s *bdd.ScenarioSpec[*cartWorld]) {
			s.When("the customer adds two books", customerAdds("book", 2))
			s.Then("the total is updated", totalIs(200))
		})
	})
}
```

Use `Before` for hidden setup that is not part of the behavior description.

## 4. Group Scenarios

Use `Rule` and `Context` to organize scenarios. Backgrounds, `Before` hooks, and
tags are inherited by nested groups.

```go
func TestCartPricing(t *testing.T) {
	bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
		f.Background("an empty cart", emptyCart())

		f.Rule("bulk discounts are applied", func(r *bdd.Suite[*cartWorld]) {
			r.Scenario("third book receives a discount", func(s *bdd.ScenarioSpec[*cartWorld]) {
				s.Given("a logged-in premium customer", premiumCustomer("ada@example.com"))
				s.When("the customer adds three books", customerAdds("book", 3))
				s.Then("the total includes the discount", totalIs(2700))
			})
		})
	})
}
```

## 5. Compose Steps

Use `Compose` when one domain phrase should run several smaller steps.

```go
func premiumCustomer(email string) bdd.Step[*cartWorld] {
	return bdd.Compose(
		loggedInCustomer(email),
		emptyCart(),
	)
}
```

## 6. Use And and But

`And` and `But` are labels. They run the same kind of step function as `Given`,
`When`, and `Then`.

```go
s.When("checkout starts", startCheckout()).
	And("payment is authorized", authorizePayment())

s.Then("the order is created", orderIsCreated()).
	And("a receipt is sent", receiptIsSent()).
	But("no duplicate order is created", noDuplicateOrder())
```

## 7. Use Outlines

Use `Outline` for table-driven scenarios. `Outline` is a function, not a method,
because Go methods cannot add an extra type parameter for the row type.

```go
func TestPasswordStrength(t *testing.T) {
	type world struct {
		password string
		got      string
	}

	type row struct {
		password string
		want     string
	}

	cases := []bdd.Example[row]{
		bdd.Case("short passwords are weak", row{"abc", "weak"}),
		bdd.Case("long passwords are strong", row{"correct-horse", "strong"}),
	}

	bdd.Feature(t, "password strength", func(t testing.TB) *world {
		return &world{}
	}, func(f *bdd.Suite[*world]) {
		bdd.Outline(f, "length-based strength", cases, func(tc row, s *bdd.ScenarioSpec[*world]) {
			s.Given("a password", func(t testing.TB, w *world) {
				w.password = tc.password
			})
			s.When("the password is evaluated", func(t testing.TB, w *world) {
				if len(w.password) >= 12 {
					w.got = "strong"
				} else {
					w.got = "weak"
				}
			})
			s.Then("the expected strength is returned", func(t testing.TB, w *world) {
				if w.got != tc.want {
					t.Fatalf("strength = %s; want %s", w.got, tc.want)
				}
			})
		})
	})
}
```

Each case becomes one scenario. Case options apply to that scenario:

```go
bdd.Case("not implemented yet", row{"", "weak"}, bdd.Pending("policy undecided"))
```

## 8. Tag, Skip, and Mark Pending

Tags are added to generated test names as `@tag`, so `go test -run` works.

```go
bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
	f.Tags("unit")
	f.Scenario("discounts apply", func(s *bdd.ScenarioSpec[*cartWorld]) {
		// ...
	}, bdd.Tags("pricing"))
})
```

```sh
go test ./... -run '@unit'
go test ./... -run '@pricing'
```

Use `bdd.Skip(reason)` or `bdd.Pending(reason)` on features, rules, contexts,
scenarios, outlines, or cases. Pending nodes are reported as skipped tests.

Use `s.Pending(text, reason)` inside a scenario when a specific step is pending
and following steps should not run.

## 9. Use Cleanup and Context

State factories and steps receive `testing.TB`.

Create scenario-lifetime resources in the state factory:

```go
type world struct {
	ctx       context.Context
	serverURL string
}

func newWorld(t testing.TB) *world {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	return &world{
		ctx:       t.Context(),
		serverURL: server.URL,
	}
}
```

Use step `t.Context()` for work scoped to that visible step.

Visible steps run as subtests. A context from a step is canceled when that step
ends. A context from the state factory lives until the scenario ends and is
canceled just before scenario cleanup functions run.

## 10. No State

Use `bdd.NoState` when a feature does not need shared scenario state.

```go
func TestAPIHealth(t *testing.T) {
	bdd.Feature(t, "API health", bdd.NewNoState, func(f *bdd.Suite[bdd.NoState]) {
		f.Scenario("service responds", func(s *bdd.ScenarioSpec[bdd.NoState]) {
			s.Given("the service is running", bdd.Noop[bdd.NoState]())
			s.When("the health endpoint is requested", bdd.Noop[bdd.NoState]())
			s.Then("the response is healthy", bdd.Noop[bdd.NoState]())
		})
	})
}
```

## Notes

- Nil step functions fail when the step runs.
- Empty outline example tables fail the outline test.
- Call `s.Parallel()` only when scenario state and dependencies are isolated.
