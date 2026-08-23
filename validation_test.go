package bdd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDeclarationValidationMatrix(t *testing.T) {
	cases := map[string]string{
		"nil-feature-builder":             "nil builder",
		"nil-rule-builder":                `Rule "rule" has a nil builder`,
		"nil-scenario-builder":            `Scenario "scenario" has a nil builder`,
		"nil-outline-builder":             `Outline "outline" has a nil builder`,
		"nil-suite-before":                "Before has a nil step",
		"nil-suite-background":            `Background "background" has a nil step`,
		"nil-suite-and":                   `And "continuation" has a nil step`,
		"nil-suite-but":                   `But "continuation" has a nil step`,
		"nil-scenario-before":             "Scenario Before has a nil step",
		"nil-given":                       `Given "given" has a nil step`,
		"nil-when":                        `When "when" has a nil step`,
		"nil-then":                        `Then "then" has a nil step`,
		"nil-scenario-and":                `And "and" has a nil step`,
		"nil-scenario-but":                `But "but" has a nil step`,
		"blank-feature":                   "invalid Feature name",
		"blank-rule":                      "invalid Rule name",
		"blank-scenario":                  "invalid Scenario name",
		"blank-outline":                   "invalid Outline name",
		"blank-example":                   "invalid Outline example name",
		"blank-given":                     "Given text is blank",
		"blank-when":                      "When text is blank",
		"blank-then":                      "Then text is blank",
		"blank-and":                       "And text is blank",
		"blank-but":                       "But text is blank",
		"blank-background":                "Background text is blank",
		"blank-suite-and":                 "And text is blank",
		"blank-suite-but":                 "But text is blank",
		"blank-pending-reason":            "requires a nonblank reason",
		"blank-skip-reason":               "skip requires a nonblank reason",
		"blank-option-pending":            "pending requires a nonblank reason",
		"zero-option":                     "zero option",
		"conflicting-disposition":         "multiple disposition options",
		"suite-before-after-background":   "Before must be declared before Background",
		"suite-second-background":         "only one local Background",
		"suite-background-after-child":    "Background must be declared before child scenarios",
		"suite-before-after-child":        "Before must be declared before child scenarios",
		"suite-and-after-child":           "Background must be declared before child scenarios",
		"suite-but-after-child":           "Background must be declared before child scenarios",
		"scenario-parallel-twice":         "Parallel may be declared only once",
		"scenario-parallel-after-visible": "Parallel must be declared before visible operations",
		"scenario-before-after-visible":   "Before must be declared before visible operations",
	}

	if os.Getenv("BDD_VALIDATION_CASE") != "" {
		runValidationCase(t, os.Getenv("BDD_VALIDATION_CASE"))
		return
	}

	for name, want := range cases {
		name, want := name, want
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestDeclarationValidationMatrix$")
			cmd.Env = append(os.Environ(), "BDD_VALIDATION_CASE="+name)
			out, err := cmd.CombinedOutput()
			text := string(out)
			if err == nil || !strings.Contains(text, want) || !strings.Contains(text, "validation_test.go:") {
				t.Fatalf("validation result = %v; want %q and caller source\n%s", err, want, out)
			}
		})
	}
}

func runValidationCase(t *testing.T, name string) {
	t.Helper()
	validStep := func(testing.TB, NoState) {}
	validScenario := func(s *ScenarioSpec[NoState]) {
		s.Then("visible", validStep)
	}
	validExamples := []Example[int]{Case("example", 1)}
	feature := func(build func(*Suite[NoState]), opts ...Option) {
		Feature(t, "feature", NewNoState, build, opts...)
	}

	switch name {
	case "nil-feature-builder":
		Feature(t, "feature", NewNoState, nil)
	case "nil-rule-builder":
		feature(func(f *Suite[NoState]) { f.Rule("rule", nil) })
	case "nil-scenario-builder":
		feature(func(f *Suite[NoState]) { f.Scenario("scenario", nil) })
	case "nil-outline-builder":
		feature(func(f *Suite[NoState]) { f.Outline("outline", validExamples, nil) })
	case "nil-suite-before":
		feature(func(f *Suite[NoState]) { f.Before(nil) })
	case "nil-suite-background":
		feature(func(f *Suite[NoState]) { f.Background("background", nil) })
	case "nil-suite-and":
		feature(func(f *Suite[NoState]) {
			f.Background("background", validStep).And("continuation", nil)
		})
	case "nil-suite-but":
		feature(func(f *Suite[NoState]) {
			f.Background("background", validStep).But("continuation", nil)
		})
	case "nil-scenario-before":
		feature(func(f *Suite[NoState]) { f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Before(nil) }) })
	case "nil-given":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("given", nil) })
		})
	case "nil-when":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.When("when", nil) })
		})
	case "nil-then":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Then("then", nil) })
		})
	case "nil-scenario-and":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("given", validStep).And("and", nil) })
		})
	case "nil-scenario-but":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("given", validStep).But("but", nil) })
		})
	case "blank-feature":
		Feature(t, "", NewNoState, func(*Suite[NoState]) {})
	case "blank-rule":
		feature(func(f *Suite[NoState]) { f.Rule("", func(*Suite[NoState]) {}) })
	case "blank-scenario":
		feature(func(f *Suite[NoState]) { f.Scenario("", validScenario) })
	case "blank-outline":
		feature(func(f *Suite[NoState]) { f.Outline("", validExamples, func(int, *ScenarioSpec[NoState]) {}) })
	case "blank-example":
		feature(func(f *Suite[NoState]) {
			f.Outline("outline", []Example[int]{Case("", 1)}, func(int, *ScenarioSpec[NoState]) {})
		})
	case "blank-given":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("", validStep) })
		})
	case "blank-when":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.When("", validStep) })
		})
	case "blank-then":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Then("", validStep) })
		})
	case "blank-and":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("given", validStep).And("", validStep) })
		})
	case "blank-but":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Given("given", validStep).But("", validStep) })
		})
	case "blank-background":
		feature(func(f *Suite[NoState]) { f.Background("", validStep) })
	case "blank-suite-and":
		feature(func(f *Suite[NoState]) { f.Background("background", validStep).And("", validStep) })
	case "blank-suite-but":
		feature(func(f *Suite[NoState]) { f.Background("background", validStep).But("", validStep) })
	case "blank-pending-reason":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", nil, Pending(" "))
		})
	case "blank-skip-reason":
		feature(func(f *Suite[NoState]) {}, Skip(" "))
	case "blank-option-pending":
		feature(func(f *Suite[NoState]) {}, Pending(" "))
	case "zero-option":
		feature(func(f *Suite[NoState]) {}, Option{})
	case "conflicting-disposition":
		feature(func(f *Suite[NoState]) {}, Skip("skip"), Pending("pending"))
	case "suite-before-after-background":
		feature(func(f *Suite[NoState]) { f.Background("background", validStep); f.Before(validStep) })
	case "suite-second-background":
		feature(func(f *Suite[NoState]) { f.Background("first", validStep); f.Background("second", validStep) })
	case "suite-background-after-child":
		feature(func(f *Suite[NoState]) { f.Scenario("scenario", validScenario); f.Background("background", validStep) })
	case "suite-before-after-child":
		feature(func(f *Suite[NoState]) { f.Scenario("scenario", validScenario); f.Before(validStep) })
	case "suite-and-after-child":
		feature(func(f *Suite[NoState]) {
			chain := f.Background("background", validStep)
			f.Scenario("scenario", validScenario)
			chain.And("continuation", validStep)
		})
	case "suite-but-after-child":
		feature(func(f *Suite[NoState]) {
			chain := f.Background("background", validStep)
			f.Scenario("scenario", validScenario)
			chain.But("continuation", validStep)
		})
	case "scenario-parallel-twice":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Parallel(); s.Parallel() })
		})
	case "scenario-parallel-after-visible":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Then("visible", validStep); s.Parallel() })
		})
	case "scenario-before-after-visible":
		feature(func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) { s.Then("visible", validStep); s.Before(validStep) })
		})
	default:
		t.Fatalf("unknown validation case %q", name)
	}
}

func TestMalformedTags(t *testing.T) {
	if _, err := (meta{}).with(Tags()); err == nil || !strings.Contains(err.Error(), "tags requires at least one tag") {
		t.Fatalf("Tags() error = %v; want zero-tag validation", err)
	}
	for _, tag := range []string{"", "@", "white space", "slash/name", "under_score", "punct!", "control\x01", "a--b", "́leading-mark", "a-"} {
		if _, err := (meta{}).with(Tags(tag)); err == nil {
			t.Errorf("Tags(%q) unexpectedly accepted", tag)
		}
	}
}
