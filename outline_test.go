package bdd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestOutlineRequiresExamples(t *testing.T) {
	if os.Getenv("BDD_EMPTY_OUTLINE_SUBPROCESS") == "1" {
		Feature(t, "outline validation", NewNoState, func(f *Suite[NoState]) {
			Outline(f, "empty table", []Example[int]{}, func(int, *ScenarioSpec[NoState]) {})
		})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestOutlineRequiresExamples$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_OUTLINE_SUBPROCESS=1")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("empty outline unexpectedly passed:\n%s", out)
	}
	if got := string(out); !strings.Contains(got, "index out of range") {
		t.Fatalf("output missing empty outline failure:\n%s", got)
	}
}

func TestOutlineNameIncludesKindTagsAndCleanName(t *testing.T) {
	name := outlineName(" auth / login \n attempts ", meta{tags: []string{"fast", "login-flow"}})
	want := "Scenario Outline: @fast @login-flow auth / login attempts"

	if name != want {
		t.Fatalf("outlineName() = %q; want %q", name, want)
	}
}
