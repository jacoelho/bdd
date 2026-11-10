package bdd

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestStepFailureReportsStepLineWithoutHelper(t *testing.T) {
	if os.Getenv("BDD_HELPER_TB_SUBPROCESS") == "1" {
		Feature(t, "helper reporting", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("reports user step line", func(s *ScenarioSpec[NoState]) {
				s.Then("the step fails", func(t testing.TB, _ NoState) {
					t.Fatalf("failure without explicit helper")
				})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_HELPER_TB_SUBPROCESS=1")
	if err == nil {
		t.Fatalf("failing step unexpectedly passed:\n%s", out)
	}

	got := string(out)
	if !strings.Contains(got, "failure without explicit helper") {
		t.Fatalf("output missing step failure:\n%s", got)
	}
	if !strings.Contains(got, "helper_tb_test.go") {
		t.Fatalf("output missing user step file:\n%s", got)
	}
	// The narrative carries no framework source: its line names the step's
	// declaration site instead.
	if !regexp.MustCompile(`(?m)^\s*Then the step fails\s+# helper_tb_test\.go:\d+$`).MatchString(got) {
		t.Fatalf("narrative line does not name the step's declaration site:\n%s", got)
	}
}
