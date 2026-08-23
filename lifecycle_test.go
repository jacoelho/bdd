package bdd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSetupFailureModesGateLaterWorkAndRunCleanup(t *testing.T) {
	modes := []struct {
		name string
		want string
		skip bool
	}{
		{name: "error", want: "setup error"},
		{name: "fatal", want: "setup fatal"},
		{name: "skip", want: "setup skip", skip: true},
		{name: "panic", want: "setup panic"},
	}
	if mode := os.Getenv("BDD_SETUP_FAILURE_MODE"); mode != "" {
		runSetupFailureMode(t, mode)
		return
	}

	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSetupFailureModesGateLaterWorkAndRunCleanup$", "-test.v")
			cmd.Env = append(os.Environ(), "BDD_SETUP_FAILURE_MODE="+tc.name)
			out, err := cmd.CombinedOutput()
			text := string(out)
			if tc.skip {
				if err != nil {
					t.Fatalf("skip result = %v\n%s", err, out)
				}
				if !strings.Contains(text, "SKIP") {
					t.Fatalf("skip output lacks SKIP:\n%s", out)
				}
			} else if err == nil {
				t.Fatalf("%s unexpectedly passed:\n%s", tc.name, out)
			}
			if !strings.Contains(text, tc.want) {
				t.Fatalf("output lacks %q:\n%s", tc.want, out)
			}
			if got := strings.Count(text, "SETUP_CLEANUP"); got != 1 {
				t.Fatalf("setup cleanup count = %d; want 1\n%s", got, out)
			}
			for _, forbidden := range []string{"SUITE_HOOK_RAN", "SCENARIO_HOOK_RAN", "BACKGROUND_RAN", "STEP_RAN"} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("later operation %q ran:\n%s", forbidden, out)
				}
			}
		})
	}
}

func runSetupFailureMode(t *testing.T, mode string) {
	t.Helper()
	Feature(t, "setup failure modes", func(t testing.TB) NoState {
		t.Cleanup(func() { t.Log("SETUP_CLEANUP") })
		switch mode {
		case "error":
			t.Error("setup error")
		case "fatal":
			t.Fatal("setup fatal")
		case "skip":
			t.Skip("setup skip")
		case "panic":
			panic("setup panic")
		default:
			t.Fatalf("unknown setup failure mode %q", mode)
		}
		return NoState{}
	}, func(f *Suite[NoState]) {
		f.Before(func(t testing.TB, _ NoState) { t.Fatal("SUITE_HOOK_RAN") })
		f.Background("background", func(t testing.TB, _ NoState) { t.Fatal("BACKGROUND_RAN") })
		f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
			s.Before(func(t testing.TB, _ NoState) { t.Fatal("SCENARIO_HOOK_RAN") })
			s.Then("step", func(t testing.TB, _ NoState) { t.Fatal("STEP_RAN") })
		})
	})
}

func TestLifecycleOrderIncludesParallelAndCleanup(t *testing.T) {
	if os.Getenv("BDD_LIFECYCLE_ORDER_SUBPROCESS") == "1" {
		Feature(t, "lifecycle order", func(t testing.TB) NoState {
			t.Log("SETUP")
			ctx := t.Context()
			t.Cleanup(func() { t.Logf("CLEANUP canceled=%t", ctx.Err() != nil) })
			return NoState{}
		}, func(f *Suite[NoState]) {
			t.Log("FEATURE_BUILDER")
			f.Before(func(t testing.TB, _ NoState) { t.Log("OUTER_SUITE_BEFORE") })
			f.Background("outer background", func(t testing.TB, _ NoState) { t.Log("OUTER_BACKGROUND") })
			f.Rule("rule", func(r *Suite[NoState]) {
				r.Before(func(t testing.TB, _ NoState) { t.Log("INNER_SUITE_BEFORE") })
				r.Background("inner background", func(t testing.TB, _ NoState) { t.Log("INNER_BACKGROUND") })
				r.Rule("context", func(c *Suite[NoState]) {
					c.Before(func(t testing.TB, _ NoState) { t.Log("CONTEXT_SUITE_BEFORE") })
					c.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
						t.Log("SCENARIO_BUILDER")
						s.Parallel()
						s.Before(func(t testing.TB, _ NoState) { t.Log("SCENARIO_BEFORE") })
						s.Then("visible", func(t testing.TB, _ NoState) { t.Log("VISIBLE") })
					})
				})
			})
		})
		return
	}

	// The scenario name is not tagged and is nested logically, so select it by
	// its generated direct-child name after inspecting the expected breadcrumb.
	cmd := exec.Command(os.Args[0], "-test.run=^TestLifecycleOrderIncludesParallelAndCleanup$/^feature-lifecycle-order$/^rule-rule__rule-context__scenario-scenario$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_LIFECYCLE_ORDER_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lifecycle subprocess result = %v\n%s", err, out)
	}
	text := string(out)
	path := "TestLifecycleOrderIncludesParallelAndCleanup/feature-lifecycle-order/rule-rule__rule-context__scenario-scenario"
	featureBuilder := strings.Index(text, "FEATURE_BUILDER")
	scenarioBuilder := strings.Index(text, "SCENARIO_BUILDER")
	pause := strings.Index(text, "=== PAUSE "+path)
	cont := strings.Index(text, "=== CONT  "+path)
	setup := strings.Index(text, "SETUP")
	if featureBuilder < 0 || scenarioBuilder < 0 || pause < 0 || cont < 0 || setup < 0 || !(featureBuilder < scenarioBuilder && scenarioBuilder < pause && pause < cont && cont < setup) {
		t.Fatalf("parallel lifecycle markers not ordered feature builder -> scenario builder -> PAUSE -> CONT -> setup:\n%s", out)
	}
	markers := []string{"FEATURE_BUILDER", "SCENARIO_BUILDER", "SETUP", "OUTER_SUITE_BEFORE", "INNER_SUITE_BEFORE", "CONTEXT_SUITE_BEFORE", "SCENARIO_BEFORE", "OUTER_BACKGROUND", "INNER_BACKGROUND", "VISIBLE", "CLEANUP canceled=true"}
	previous := -1
	for _, marker := range markers {
		at := strings.Index(text, marker)
		if at < 0 {
			t.Fatalf("lifecycle output lacks %q:\n%s", marker, out)
		}
		if at <= previous {
			t.Fatalf("lifecycle marker %q out of order:\n%s", marker, out)
		}
		previous = at
	}
}

func TestSetupMayReturnNilPointer(t *testing.T) {
	type world struct{}
	Feature(t, "nil pointer setup", func(testing.TB) *world { return nil }, func(f *Suite[*world]) {
		f.Scenario("zero pointer is legal", func(s *ScenarioSpec[*world]) {
			s.Then("the nil pointer reaches the step", func(t testing.TB, w *world) {
				if w != nil {
					t.Fatalf("setup returned %p; want nil", w)
				}
			})
		})
	})
}
