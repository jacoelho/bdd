package bdd

import (
	"os"
	"strings"
	"testing"
)

func TestDispositionAndBreadcrumbPaths(t *testing.T) {
	if os.Getenv("BDD_PATHS_SUBPROCESS") == "1" {
		Feature(t, "paths", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("direct scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
			f.Scenario("Café", func(s *ScenarioSpec[NoState]) {
				s.Then("precomposed name runs", func(testing.TB, NoState) {})
			})
			f.Scenario("café", func(s *ScenarioSpec[NoState]) {
				s.Then("decomposed name runs", func(testing.TB, NoState) {})
			})
			f.Rule("bulk", func(r *Suite[NoState]) {
				r.Rule("premium", func(c *Suite[NoState]) {
					c.Scenario("nested", func(s *ScenarioSpec[NoState]) {
						s.Then("runs", func(testing.TB, NoState) {})
					})
				})
			})
			f.Outline("strength", []Example[string]{
				Case("short password", "short"),
				Case("long password", "long"),
			}, func(_ string, s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
			f.Rule("skipped rule", func(*Suite[NoState]) {}, Skip("rule reason"))
			f.Rule("pending rule", func(*Suite[NoState]) {}, Pending("pending rule reason"))
			f.Rule("pending rule scope", func(*Suite[NoState]) {}, Pending("context reason"))
			f.Rule("skipped rule scope", func(*Suite[NoState]) {}, Skip("context reason"))
			f.Scenario("skipped scenario", func(*ScenarioSpec[NoState]) {
				t.Fatal("skipped scenario builder ran")
			}, Skip("scenario reason"))
			f.Scenario("pending scenario", func(*ScenarioSpec[NoState]) {
				t.Fatal("pending scenario builder ran")
			}, Pending("pending scenario reason"))
			f.Outline("pending outline", []Example[int]{
				Case("first", 1),
				Case("second", 2),
			}, func(int, *ScenarioSpec[NoState]) {
				t.Fatal("skipped outline builder ran")
			}, Pending("outline reason"))
			f.Outline("skipped outline", []Example[int]{
				Case("first", 1),
				Case("second", 2),
			}, func(int, *ScenarioSpec[NoState]) {
				t.Fatal("skipped outline builder ran")
			}, Skip("skipped outline reason"))
			f.Outline("case dispositions", []Example[int]{
				Case("case skip", 1, Skip("case skip reason")),
				Case("case pending", 2, Pending("case pending reason")),
			}, func(int, *ScenarioSpec[NoState]) {
				t.Fatal("disposed case builder ran")
			})
		}, Tags("unit", "Café", "unit"))
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_PATHS_SUBPROCESS=1", "-test.run=^TestDispositionAndBreadcrumbPaths$/^feature-paths$", "-test.v")
	if err != nil {
		t.Fatalf("path subprocess result = %v\n%s", err, out)
	}
	text := string(out)
	prefix := "TestDispositionAndBreadcrumbPaths/feature-paths/"
	active := []string{
		prefix + "@café_@unit__scenario-direct-scenario",
		prefix + "@café_@unit__scenario-café",
		prefix + "@café_@unit__scenario-café",
		prefix + "@café_@unit__rule-bulk__rule-premium__scenario-nested",
		prefix + "@café_@unit__outline-strength__example-short-password",
		prefix + "@café_@unit__outline-strength__example-long-password",
	}
	disposed := []struct {
		path   string
		reason string
	}{
		{prefix + "@café_@unit__rule-skipped-rule", "rule reason"},
		{prefix + "@café_@unit__rule-pending-rule", "pending rule reason"},
		{prefix + "@café_@unit__rule-pending-rule-scope", "context reason"},
		{prefix + "@café_@unit__rule-skipped-rule-scope", "context reason"},
		{prefix + "@café_@unit__scenario-skipped-scenario", "scenario reason"},
		{prefix + "@café_@unit__scenario-pending-scenario", "pending scenario reason"},
		{prefix + "@café_@unit__outline-pending-outline__example-first", "outline reason"},
		{prefix + "@café_@unit__outline-pending-outline__example-second", "outline reason"},
		{prefix + "@café_@unit__outline-skipped-outline__example-first", "skipped outline reason"},
		{prefix + "@café_@unit__outline-skipped-outline__example-second", "skipped outline reason"},
		{prefix + "@café_@unit__outline-case-dispositions__example-case-skip", "case skip reason"},
		{prefix + "@café_@unit__outline-case-dispositions__example-case-pending", "case pending reason"},
	}
	for _, path := range active {
		if strings.Count(path, "/") != 2 || !strings.Contains(text, "--- PASS: "+path) {
			t.Errorf("active path missing or not three components %q\n%s", path, out)
		}
	}
	for _, tc := range disposed {
		if strings.Count(tc.path, "/") != 2 || !strings.Contains(text, "--- SKIP: "+tc.path) {
			t.Errorf("disposed path missing or not three components %q\n%s", tc.path, out)
		}
		if !strings.Contains(text, tc.reason) {
			t.Errorf("disposed reason %q missing for %q\n%s", tc.reason, tc.path, out)
		}
	}
	expectedLeaves := len(active) + len(disposed)
	if got := strings.Count(text, "=== RUN   "+prefix); got != expectedLeaves {
		t.Fatalf("direct leaf RUN count = %d; want %d\n%s", got, expectedLeaves, out)
	}
	if !strings.Contains(text, "--- PASS: TestDispositionAndBreadcrumbPaths/feature-paths") {
		t.Fatalf("Feature barrier did not pass:\n%s", out)
	}
	if strings.Contains(text, "#01") {
		t.Fatalf("generated a Go duplicate suffix:\n%s", out)
	}

	out, err = runSelf(t, "BDD_PATHS_SUBPROCESS=1", "-test.run=^TestDispositionAndBreadcrumbPaths$/^feature-paths$/^@other__scenario-direct-scenario$", "-test.v")
	text = string(out)
	if err != nil || strings.Contains(text, "scenario reason") || strings.Contains(text, "SKIP") || strings.Contains(text, "disposed case builder ran") {
		t.Fatalf("nonmatching selector ran disposed leaf: %v\n%s", err, out)
	}
}

func TestDuplicateRulePathsAreReservedBeforeLeaves(t *testing.T) {
	cases := []string{"active-active", "active-disposed", "disposed-disposed"}
	if os.Getenv("BDD_DUPLICATE_RULE_CASE") != "" {
		setup := func(testing.TB) NoState {
			t.Log("DUPLICATE_RULE_SETUP_RAN")
			return NoState{}
		}
		Feature(t, "duplicate rules", setup, func(f *Suite[NoState]) {
			active := func(r *Suite[NoState]) {
				r.Scenario("first", func(s *ScenarioSpec[NoState]) {
					s.Then("runs", func(testing.TB, NoState) {})
				})
			}
			switch os.Getenv("BDD_DUPLICATE_RULE_CASE") {
			case "active-active":
				f.Rule("same!", active)
				f.Rule("same?", func(r *Suite[NoState]) {
					r.Scenario("second", func(s *ScenarioSpec[NoState]) {
						s.Then("runs", func(testing.TB, NoState) {})
					})
				})
			case "active-disposed":
				f.Rule("same!", active)
				f.Rule("same?", nil, Skip("disposed duplicate"))
			case "disposed-disposed":
				f.Rule("same!", nil, Skip("first disposed"))
				f.Rule("same?", nil, Pending("second disposed"))
			}
		})
		return
	}

	for _, ruleCase := range cases {
		ruleCase := ruleCase
		t.Run(ruleCase, func(t *testing.T) {
			t.Parallel()
			out, err := runSelf(t, "BDD_DUPLICATE_RULE_CASE="+ruleCase)
			text := string(out)
			if err == nil || !strings.Contains(text, "duplicate Rule identity") || strings.Count(text, "paths_test.go:") < 2 || strings.Contains(text, "DUPLICATE_RULE_SETUP_RAN") {
				t.Fatalf("duplicate Rule case %s result = %v; output:\n%s", ruleCase, err, out)
			}
		})
	}
}

func TestNestedSameNamedRulesRemainDistinct(t *testing.T) {
	Feature(t, "nested same rules", NewNoState, func(f *Suite[NoState]) {
		f.Rule("same", func(r *Suite[NoState]) {
			r.Scenario("outer", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
			r.Rule("same", func(nested *Suite[NoState]) {
				nested.Scenario("inner", func(s *ScenarioSpec[NoState]) {
					s.Then("runs", func(testing.TB, NoState) {})
				})
			})
		})
	}, Tags("unit", "café"))
}

func TestExactPrecomposedUTF8SelectorRunsLeaf(t *testing.T) {
	if os.Getenv("BDD_EXACT_UTF8_SELECTOR") == "1" {
		Feature(t, "Café pricing", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("réduit", func(s *ScenarioSpec[NoState]) {
				s.Then("the unique UTF-8 marker runs", func(t testing.TB, _ NoState) {
					t.Log("EXACT_PRECOMPOSED_UTF8_MARKER")
				})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_EXACT_UTF8_SELECTOR=1", "-test.run=^TestExactPrecomposedUTF8SelectorRunsLeaf$/^feature-café-pricing$/^scenario-réduit$", "-test.v")
	text := string(out)
	if err != nil || strings.Count(text, "EXACT_PRECOMPOSED_UTF8_MARKER") != 1 || !strings.Contains(text, "TestExactPrecomposedUTF8SelectorRunsLeaf/feature-café-pricing/scenario-réduit") {
		t.Fatalf("exact UTF-8 selector result = %v; output:\n%s", err, out)
	}
}
