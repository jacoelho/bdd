package bdd_test

import (
	"fmt"
	"testing"

	bdd "github.com/jacoelho/bdd"
)

type cartWorld struct {
	cart  map[string]int
	total int
}

func newCartWorld(t testing.TB) *cartWorld {
	return &cartWorld{cart: make(map[string]int)}
}

func emptyCart() bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		w.cart = make(map[string]int)
	}
}

func customerAdds(sku string, qty int) bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		w.cart[sku] += qty
		w.total = price(sku, qty)
	}
}

func totalIs(want int) bdd.Step[*cartWorld] {
	return func(t testing.TB, w *cartWorld) {
		if w.total != want {
			t.Fatalf("total = %d; want %d", w.total, want)
		}
	}
}

func premiumCustomer(email string) bdd.Step[*cartWorld] {
	return bdd.Compose(
		loggedInCustomer(email),
		emptyCart(),
	)
}

func loggedInCustomer(email string) bdd.Step[*cartWorld] {
	return func(t testing.TB, _ *cartWorld) {
		if email == "" {
			t.Fatal("email must not be empty")
		}
	}
}

func price(sku string, qty int) int {
	base := map[string]int{"book": 1000, "pen": 200}[sku]
	if sku == "book" && qty >= 3 {
		return qty*base - 300
	}
	return qty * base
}

func TestCartPricing(t *testing.T) {
	bdd.Feature(t, "cart pricing", newCartWorld, func(f *bdd.Suite[*cartWorld]) {
		f.Tags("unit")
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

type passwordWorld struct {
	password string
	got      string
}

func newPasswordWorld(t testing.TB) *passwordWorld {
	return &passwordWorld{}
}

type passwordCase struct {
	password string
	want     string
}

func passwordIs(password string) bdd.Step[*passwordWorld] {
	return func(t testing.TB, w *passwordWorld) {
		w.password = password
	}
}

func passwordIsEvaluated() bdd.Step[*passwordWorld] {
	return func(t testing.TB, w *passwordWorld) {
		if len(w.password) >= 12 {
			w.got = "strong"
		} else {
			w.got = "weak"
		}
	}
}

func strengthIs(want string) bdd.Step[*passwordWorld] {
	return func(t testing.TB, w *passwordWorld) {
		if w.got != want {
			t.Fatalf("strength = %s; want %s", w.got, want)
		}
	}
}

func TestPasswordStrength(t *testing.T) {
	cases := []bdd.Example[passwordCase]{
		bdd.Case("short passwords are weak", passwordCase{"abc", "weak"}),
		bdd.Case("long passwords are strong", passwordCase{"correct-horse", "strong"}),
	}

	bdd.Feature(t, "password strength", newPasswordWorld, func(f *bdd.Suite[*passwordWorld]) {
		bdd.Outline(f, "length-based strength", cases, func(tc passwordCase, s *bdd.ScenarioSpec[*passwordWorld]) {
			s.Given(fmt.Sprintf("the password is %q", tc.password), passwordIs(tc.password))
			s.When("the password is evaluated", passwordIsEvaluated())
			s.Then("the expected strength is returned", strengthIs(tc.want))
		})
	})
}
