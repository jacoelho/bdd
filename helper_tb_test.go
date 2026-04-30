package bdd

import (
	"os"
	"os/exec"
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

	cmd := exec.Command(os.Args[0], "-test.run=^TestStepFailureReportsStepLineWithoutHelper$")
	cmd.Env = append(os.Environ(), "BDD_HELPER_TB_SUBPROCESS=1")

	out, err := cmd.CombinedOutput()
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
	if strings.Contains(got, "scenario.go:") {
		t.Fatalf("output includes framework helper frame:\n%s", got)
	}
}
