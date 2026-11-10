package bdd

import (
	"os"
	"strings"
	"testing"
)

func TestExactLeafSelectionRunsOnlySelectedLifecycle(t *testing.T) {
	if os.Getenv("BDD_SELECTION_SUBPROCESS") == "1" {
		Feature(t, "selection", func(t testing.TB) NoState {
			t.Log("SETUP")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Before(func(t testing.TB, _ NoState) { t.Log("SUITE_BEFORE") })
			f.Background("shared background", func(t testing.TB, _ NoState) { t.Log("BACKGROUND") })
			f.Scenario("selected", func(s *ScenarioSpec[NoState]) {
				t.Log("SELECTED_BUILDER")
				s.Before(func(t testing.TB, _ NoState) { t.Log("SCENARIO_BEFORE") })
				s.Then("the selected operation runs", func(t testing.TB, _ NoState) { t.Log("STEP") })
			})
			f.Scenario("unselected", func(*ScenarioSpec[NoState]) {
				t.Fatal("UNSELECTED_BUILDER")
			})
			f.Scenario("disposed", func(*ScenarioSpec[NoState]) {
				t.Fatal("DISPOSED_BUILDER")
			}, Skip("disposed reason"))
		}, Tags("unit"))
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_SELECTION_SUBPROCESS=1", "-test.run=^TestExactLeafSelectionRunsOnlySelectedLifecycle$/^feature-selection$/^@unit__scenario-selected$", "-test.v")
	text := string(out)
	if err != nil || strings.Contains(text, "UNSELECTED_BUILDER") || strings.Contains(text, "DISPOSED_BUILDER") || strings.Contains(text, "disposed reason") {
		t.Fatalf("selected leaf result = %v, output:\n%s", err, out)
	}
	for _, marker := range []string{"SELECTED_BUILDER", "SETUP", "SUITE_BEFORE", "SCENARIO_BEFORE", "BACKGROUND", "STEP"} {
		if got := strings.Count(text, marker); got != 1 {
			t.Errorf("selected marker %q count = %d; want 1\n%s", marker, got, text)
		}
	}

	out, err = runSelf(t, "BDD_SELECTION_SUBPROCESS=1", "-test.run=^TestExactLeafSelectionRunsOnlySelectedLifecycle$/^feature-selection$/^@other__scenario-selected$", "-test.v")
	text = string(out)
	if err != nil || strings.Contains(text, "SETUP") || strings.Contains(text, "SUITE_BEFORE") || strings.Contains(text, "SCENARIO_BEFORE") || strings.Contains(text, "BACKGROUND") || strings.Contains(text, "STEP") || strings.Contains(text, "UNSELECTED_BUILDER") || strings.Contains(text, "DISPOSED_BUILDER") || strings.Contains(text, "disposed reason") {
		t.Fatalf("nonmatching selector result = %v, output:\n%s", err, out)
	}
}

func TestStepTextSelectorDoesNotSelectScenario(t *testing.T) {
	if os.Getenv("BDD_STEP_SELECTOR_SUBPROCESS") == "1" {
		Feature(t, "step selector", func(t testing.TB) NoState {
			t.Fatal("setup ran for a step-text selector")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("needle step", func(t testing.TB, _ NoState) { t.Fatal("step selector executed a step") }).Then("later step", func(t testing.TB, _ NoState) { t.Fatal("later step ran") })
			})
		})
		return
	}

	t.Parallel()
	if out, err := runSelf(t, "BDD_STEP_SELECTOR_SUBPROCESS=1", "-test.run=^TestStepTextSelectorDoesNotSelectScenario$/.*needle.*$"); err != nil {
		t.Fatalf("step-text selector result = %v, output:\n%s", err, out)
	}
}
