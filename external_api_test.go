package bdd_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

func TestExternalTypedAPI(t *testing.T) {
	type world struct{ value int }
	var option bdd.Option = bdd.Tags("external")
	bdd.Feature(t, "external typed API", func(testing.TB) *world {
		return &world{}
	}, func(f *bdd.Suite[*world]) {
		f.Background("the world is ready", func(testing.TB, *world) {}).
			And("the fixture is loaded", func(testing.TB, *world) {})
		f.Scenario("direct methods compile", func(s *bdd.ScenarioSpec[*world]) {
			chain := s.Given("a typed world", func(testing.TB, *world) {})
			chain = chain.And("an additional condition", func(testing.TB, *world) {})
			chain = chain.When("the value changes", func(_ testing.TB, w *world) { w.value++ })
			chain = chain.Then("the value is one", func(t testing.TB, w *world) {
				if w.value != 1 {
					t.Fatalf("value = %d; want 1", w.value)
				}
			})
			_ = chain.But("no error is reported", func(testing.TB, *world) {})
		}, option)
		f.Scenario("all starters are callable", func(s *bdd.ScenarioSpec[*world]) {
			s.When("a request arrives", func(testing.TB, *world) {}).
				Given("the world is typed", func(testing.TB, *world) {}).
				Then("the response is checked", func(testing.TB, *world) {}).
				But("the result is stable", func(testing.TB, *world) {})
		})
		f.Scenario("repeated flexible keywords", func(s *bdd.ScenarioSpec[*world]) {
			chain := s.Then("the first assertion", func(testing.TB, *world) {})
			chain = chain.Then("the repeated assertion", func(testing.TB, *world) {})
			chain = chain.Given("a later precondition", func(testing.TB, *world) {})
			_ = chain.When("the final action", func(testing.TB, *world) {})
		})
	}, option)
}

func TestExternalInferredOutlineAndNoState(t *testing.T) {
	type row struct{ want int }
	examples := []bdd.Example[row]{
		bdd.Case("one", row{want: 1}),
		bdd.Case("two", row{want: 2}),
	}
	bdd.Feature(t, "external outline", bdd.NewNoState, func(f *bdd.Suite[bdd.NoState]) {
		f.Outline("inferred rows", examples, func(tc row, s *bdd.ScenarioSpec[bdd.NoState]) {
			s.Then("the row is available", func(t testing.TB, _ bdd.NoState) {
				if tc.want != 1 && tc.want != 2 {
					t.Fatalf("unexpected row %d", tc.want)
				}
			})
		})
	})
}
