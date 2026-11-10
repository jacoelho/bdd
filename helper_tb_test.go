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
	// Log lines of a running scenario are attributed to the Feature call site.
	if !regexp.MustCompile(`helper_tb_test\.go:\d+: Then: the step fails`).MatchString(got) {
		t.Fatalf("framework log line is not attributed to the user's test file:\n%s", got)
	}
}
