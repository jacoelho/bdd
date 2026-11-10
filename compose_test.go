package bdd

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestComposeRejectsEmptyStepList(t *testing.T) {
	if os.Getenv("BDD_COMPOSE_EMPTY_SUBPROCESS") == "1" {
		step := Compose[NoState]()
		if step == nil {
			t.Fatal("empty Compose returned nil")
		}
		step(t, NoState{})
		return
	}
	t.Parallel()
	out, err := runSelf(t, "BDD_COMPOSE_EMPTY_SUBPROCESS=1")
	if err == nil || !regexp.MustCompile(`bdd: \S*compose_test\.go:\d+: Compose requires at least one step`).Match(out) {
		t.Fatalf("empty Compose result = %v\n%s", err, out)
	}
}

func TestComposeRejectsNilComponentWithoutRunningLaterComponents(t *testing.T) {
	if os.Getenv("BDD_COMPOSE_NIL_SUBPROCESS") == "1" {
		step := Compose[NoState](nil, func(t testing.TB, _ NoState) { t.Fatal("later component ran") })
		if step == nil {
			t.Fatal("nil-component Compose returned nil")
		}
		step(t, NoState{})
		return
	}
	t.Parallel()
	out, err := runSelf(t, "BDD_COMPOSE_NIL_SUBPROCESS=1")
	text := string(out)
	if err == nil || !regexp.MustCompile(`bdd: \S*compose_test\.go:\d+: Compose received a nil step`).MatchString(text) || strings.Contains(text, "later component ran") {
		t.Fatalf("nil Compose result = %v\n%s", err, out)
	}
}

func TestComposeRunsComponentsInOrder(t *testing.T) {
	Feature(t, "compose order", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("ordered components", func(s *ScenarioSpec[NoState]) {
			s.Then("components run in order", func(t testing.TB, state NoState) {
				var got []int
				step := Compose(
					func(testing.TB, NoState) { got = append(got, 1) },
					func(testing.TB, NoState) { got = append(got, 2) },
				)
				if step == nil {
					t.Fatal("Compose returned nil")
				}
				step(t, state)
				if len(got) != 2 || got[0] != 1 || got[1] != 2 {
					t.Fatalf("component order = %#v; want [1 2]", got)
				}
			})
		})
	})
}

func TestComposeSnapshotsComponentSlice(t *testing.T) {
	var ranOriginal, ranReplacement bool
	parts := []Step[NoState]{func(testing.TB, NoState) { ranOriginal = true }}
	composed := Compose(parts...)
	parts[0] = func(testing.TB, NoState) { ranReplacement = true }
	composed(t, NoState{})
	if !ranOriginal || ranReplacement {
		t.Fatalf("Compose did not snapshot components: original=%v replacement=%v", ranOriginal, ranReplacement)
	}
}

func TestOptionsAndExamplesAreClonedAtBoundaries(t *testing.T) {
	tags := []string{"Café"}
	tagOption := Tags(tags...)
	tags[0] = "changed"
	m, err := (meta{}).with(tagOption)
	if err != nil || len(m.tags) != 1 || m.tags[0] != "café" {
		t.Fatalf("Tags did not clone/normalize input: %#v, %v", m.tags, err)
	}

	options := []Option{Tags("unit")}
	example := Case("example", 1, options...)
	options[0] = Option{}
	if _, err := (meta{}).with(example.options...); err != nil {
		t.Fatalf("Case retained caller option slice: %v", err)
	}

	examples := []Example[int]{Case("original", 1, Tags("unit"))}
	Feature(t, "outline cloning", NewNoState, func(f *Suite[NoState]) {
		f.Outline("rows", examples, func(value int, s *ScenarioSpec[NoState]) {
			s.Then("the cloned row runs", func(t testing.TB, _ NoState) {
				if value != 1 {
					t.Fatalf("outline value = %d; want 1", value)
				}
			})
		})
		// Outline copies the example and its options before its lazy leaf runs.
		examples[0].name = "mutated after declaration"
		examples[0].options[0] = Option{}
	})
	owned := []int{1}
	shallow := Case("shallow", owned)
	owned[0] = 2
	if shallow.value[0] != 2 {
		t.Fatalf("Case unexpectedly deep-copied reference data: %#v", shallow.value)
	}
}
