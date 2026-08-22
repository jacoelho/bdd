package bdd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPendingStepReportsTerminalSkip(t *testing.T) {
	if os.Getenv("BDD_PENDING_STEP_SUBPROCESS") == "1" {
		Feature(t, "pending scenario", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("stops after pending step", func(s *ScenarioSpec[NoState]) {
				s.Pending("implementation is missing", "not built yet")
			})
		})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPendingStepReportsTerminalSkip$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_PENDING_STEP_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	path := "TestPendingStepReportsTerminalSkip/feature-pending-scenario/scenario-stops-after-pending-step"
	if err != nil || !strings.Contains(text, "Pending: implementation is missing") || !strings.Contains(text, "pending: not built yet") || !strings.Contains(text, "--- SKIP: "+path) || strings.Contains(text, "FAIL: "+path) {
		t.Fatalf("pending step result = %v, output:\n%s", err, out)
	}
}

func TestPendingStepMutationFailsBeforeSetup(t *testing.T) {
	if os.Getenv("BDD_PENDING_MUTATION_SUBPROCESS") == "1" {
		Feature(t, "pending mutation", func(t testing.TB) NoState {
			t.Fatal("setup ran after invalid Pending mutation")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("terminal", func(s *ScenarioSpec[NoState]) {
				s.Pending("not implemented", "waiting")
				s.Then("must be rejected", func(testing.TB, NoState) {})
			})
		})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPendingStepMutationFailsBeforeSetup$")
	cmd.Env = append(os.Environ(), "BDD_PENDING_MUTATION_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err == nil || !strings.Contains(text, "no operations may follow Pending") || strings.Contains(text, "setup ran after") {
		t.Fatalf("Pending mutation result = %v, output:\n%s", err, out)
	}
}
