package bdd_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

type shippingWorld struct {
	stock   map[string]int
	sku     string
	allowed bool
}

func newShippingWorld(t testing.TB) *shippingWorld {
	return &shippingWorld{}
}

func stockedWarehouse() bdd.Step[*shippingWorld] {
	return func(t testing.TB, w *shippingWorld) {
		w.stock = map[string]int{
			"book": 4,
			"desk": 0,
		}
	}
}

func skuIs(sku string) bdd.Step[*shippingWorld] {
	return func(t testing.TB, w *shippingWorld) {
		w.sku = sku
	}
}

func availabilityIsChecked() bdd.Step[*shippingWorld] {
	return func(t testing.TB, w *shippingWorld) {
		w.allowed = w.stock[w.sku] > 0
	}
}

func shippingAllowed(want bool) bdd.Step[*shippingWorld] {
	return func(t testing.TB, w *shippingWorld) {
		if w.allowed != want {
			t.Fatalf("allowed = %v; want %v", w.allowed, want)
		}
	}
}

func TestOutline(t *testing.T) {
	type row struct {
		sku  string
		want bool
	}

	examples := []bdd.Example[row]{
		bdd.Case("in-stock items can ship", row{sku: "book", want: true}),
		bdd.Case("out-of-stock items cannot ship", row{sku: "desk", want: false}),
	}

	bdd.Feature(t, "shipping availability", newShippingWorld, func(f *bdd.Suite[*shippingWorld]) {
		f.Background("a stocked warehouse", stockedWarehouse())

		f.Outline("stock controls shipping", examples, func(tc row, s *bdd.ScenarioSpec[*shippingWorld]) {
			s.Given("a customer selects the item", skuIs(tc.sku)).
				When("shipping availability is checked", availabilityIsChecked()).
				Then("the shipping decision is returned", shippingAllowed(tc.want))
		})
	}, bdd.Tags("unit"))
}
