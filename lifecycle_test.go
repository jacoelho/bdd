package bdd

import (
	"os"
	"regexp"
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
			t.Parallel()
			out, err := runSelf(t, "BDD_SETUP_FAILURE_MODE="+tc.name, "-test.v")
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

	// The scenario is untagged and nested under two Rules, so the selector is
	// its full generated direct-child name.
	t.Parallel()
	out, err := runSelf(t, "BDD_LIFECYCLE_ORDER_SUBPROCESS=1", "-test.run=^TestLifecycleOrderIncludesParallelAndCleanup$/^feature-lifecycle-order$/^rule-rule__rule-context__scenario-scenario$", "-test.v")
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

func TestStoppedScenarioReportsStoppingOperationAndRemainder(t *testing.T) {
	const (
		notRunAfterSecond = `; not run: Then "the third operation", And "the last operation"`
		everyOperation    = `Before hook, Before hook, Background "the shared fixture", Given "the first operation", When "the second operation", Then "the third operation", And "the last operation"`
	)
	cases := []struct {
		name   string
		passes bool
		// stoppedIn and notRun describe the expected report; an empty
		// stoppedIn means no report. marker is logged by the stopping
		// operation on its declaration line, so testing's own attribution of
		// the marker is the expected declaration source.
		stoppedIn string
		marker    string
		notRun    string
	}{
		{name: "fatal", stoppedIn: `When "the second operation"`, marker: "SECOND_RUNS", notRun: notRunAfterSecond},
		{name: "panic", stoppedIn: `When "the second operation"`, marker: "SECOND_RUNS", notRun: notRunAfterSecond},
		{name: "skip", passes: true, stoppedIn: `When "the second operation"`, marker: "SECOND_RUNS", notRun: notRunAfterSecond},
		{name: "error-in-last", stoppedIn: `And "the last operation"`, marker: "LAST_RUNS"},
		{name: "setup-fatal", stoppedIn: "Setup", notRun: "; not run: " + everyOperation},
		{name: "pass", passes: true},
	}
	if mode := os.Getenv("BDD_STOP_REPORT_MODE"); mode != "" {
		runStopReportMode(t, mode)
		return
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := runSelf(t, "BDD_STOP_REPORT_MODE="+tc.name, "-test.v")
			text := string(out)
			if passed := err == nil; passed != tc.passes {
				t.Fatalf("subprocess passed = %t; want %t\n%s", passed, tc.passes, out)
			}
			reports := regexp.MustCompile(`(?m)bdd: scenario stopped in .*$`).FindAllString(text, -1)
			if tc.stoppedIn == "" {
				if len(reports) != 0 {
					t.Fatalf("completed scenario reported a stop: %q\n%s", reports, out)
				}
				return
			}
			if len(reports) != 1 {
				t.Fatalf("stop reports = %q; want exactly one\n%s", reports, out)
			}
			prefix, suffix := "bdd: scenario stopped in "+tc.stoppedIn, tc.notRun
			if tc.marker != "" {
				source := regexp.MustCompile(`(lifecycle_test\.go:\d+): ` + tc.marker).FindStringSubmatch(text)
				if source == nil {
					t.Fatalf("output lacks marker %q:\n%s", tc.marker, out)
				}
				prefix, suffix = prefix+" declared at ", "/"+source[1]+tc.notRun
			}
			if got := reports[0]; !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) || (tc.marker == "" && got != prefix+suffix) {
				t.Fatalf("stop report = %q; want %q ... %q", got, prefix, suffix)
			}
		})
	}
}

func runStopReportMode(t *testing.T, mode string) {
	t.Helper()
	pass := func(testing.TB, NoState) {}
	errorInLast := func(t testing.TB) {
		t.Helper()
		if mode == "error-in-last" {
			t.Error("stopped by Error")
		}
	}
	Feature(t, "stop report", func(t testing.TB) NoState {
		if mode == "setup-fatal" {
			t.Fatal("setup failed")
		}
		return NoState{}
	}, func(f *Suite[NoState]) {
		f.Before(pass)
		f.Background("the shared fixture", pass)
		f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
			s.Before(pass)
			chain := s.Given("the first operation", pass)
			chain = chain.When("the second operation", func(t testing.TB, _ NoState) { t.Log("SECOND_RUNS"); stopAs(t, mode) })
			chain = chain.Then("the third operation", pass)
			chain.And("the last operation", func(t testing.TB, _ NoState) { t.Log("LAST_RUNS"); errorInLast(t) })
		})
	})
}

func stopAs(t testing.TB, mode string) {
	t.Helper()
	switch mode {
	case "fatal":
		t.Fatal("stopped by Fatal")
	case "panic":
		panic("stopped by panic")
	case "skip":
		t.Skip("stopped by Skip")
	}
}
