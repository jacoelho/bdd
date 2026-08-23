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
			f.Outline("empty table", []Example[int]{}, func(int, *ScenarioSpec[NoState]) {})
		})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestOutlineRequiresExamples$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_OUTLINE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "requires at least one example") || !strings.Contains(string(out), "outline_test.go:") {
		t.Fatalf("empty outline result = %v, output:\n%s", err, out)
	}
}

func TestOutlineRejectsZeroExample(t *testing.T) {
	if os.Getenv("BDD_ZERO_EXAMPLE_SUBPROCESS") == "1" {
		Feature(t, "zero example", NewNoState, func(f *Suite[NoState]) {
			f.Outline("rows", []Example[int]{{}}, func(int, *ScenarioSpec[NoState]) {})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestOutlineRejectsZeroExample$")
	cmd.Env = append(os.Environ(), "BDD_ZERO_EXAMPLE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "use bdd.Case") || !strings.Contains(string(out), "outline_test.go:") {
		t.Fatalf("zero Example result = %v; output:\n%s", err, out)
	}
}

func TestOutlineMethodAndCaseOptions(t *testing.T) {
	type row struct{ value int }
	cases := []Example[row]{
		Case("one", row{1}),
		Case("two", row{2}, Tags("table")),
	}
	Feature(t, "outline", NewNoState, func(f *Suite[NoState]) {
		f.Outline("values", cases, func(tc row, s *ScenarioSpec[NoState]) {
			s.Then("the row is observed", func(t testing.TB, _ NoState) {
				if tc.value != 1 && tc.value != 2 {
					t.Fatalf("unexpected row value %d", tc.value)
				}
			})
		})
	}, Tags("unit"))
}

func TestOutlinePendingExampleDoesNotBuild(t *testing.T) {
	var ran bool
	Feature(t, "outline pending", NewNoState, func(f *Suite[NoState]) {
		f.Outline("rows", []Example[int]{
			Case("pending", 1, Pending("waiting for data")),
		}, func(int, *ScenarioSpec[NoState]) { ran = true })
	})
	if ran {
		t.Fatal("pending outline builder ran")
	}
}

func TestOutlineExampleDispositionOverridesOutline(t *testing.T) {
	var ran bool
	Feature(t, "outline override", NewNoState, func(f *Suite[NoState]) {
		f.Outline("rows", []Example[int]{
			Case("active", 1, Skip("example skip")),
		}, func(int, *ScenarioSpec[NoState]) {
			ran = true
		}, Pending("outline pending"))
	})
	if ran {
		t.Fatal("disposed example builder ran")
	}
}

func TestDisposedOutlineRegistersEveryExample(t *testing.T) {
	if os.Getenv("BDD_DISPOSED_OUTLINE_SUBPROCESS") == "1" {
		Feature(t, "disposed outline", NewNoState, func(f *Suite[NoState]) {
			f.Outline("all rows", []Example[int]{
				Case("first", 1),
				Case("second", 2),
			}, func(int, *ScenarioSpec[NoState]) {
				t.Fatal("disposed outline builder ran")
			}, Pending("waiting for specification"))
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDisposedOutlineRegistersEveryExample$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_DISPOSED_OUTLINE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "example-first") || !strings.Contains(string(out), "example-second") || strings.Contains(string(out), "disposed outline builder ran") {
		t.Fatalf("disposed outline result = %v, output:\n%s", err, out)
	}
}

func TestExampleDispositionOverridesOutlineReason(t *testing.T) {
	if os.Getenv("BDD_EXAMPLE_DISPOSITION_SUBPROCESS") == "1" {
		Feature(t, "outline reason", NewNoState, func(f *Suite[NoState]) {
			f.Outline("rows", []Example[int]{
				Case("case", 1, Pending("case reason")),
			}, func(int, *ScenarioSpec[NoState]) {
				t.Fatal("disposed example builder ran")
			}, Skip("outline reason"))
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestExampleDispositionOverridesOutlineReason$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_EXAMPLE_DISPOSITION_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err != nil || !strings.Contains(text, "pending: case reason") || strings.Contains(text, "outline reason: outline reason") || strings.Contains(text, "disposed example builder ran") {
		t.Fatalf("example disposition result = %v, output:\n%s", err, out)
	}
}
