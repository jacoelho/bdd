package bdd

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestNarrativeNamesEachLineWithItsDeclarationSite(t *testing.T) {
	if os.Getenv("BDD_NARRATIVE_SUBPROCESS") == "1" {
		// Every declaration logs a marker from its own line, so testing's
		// attribution of the marker is the expected declaration site.
		Feature(t, "narrative", NewNoState, func(f *Suite[NoState]) {
			f.Before(func(t testing.TB, _ NoState) { t.Log("HOOK") })
			f.Background("the shared fixture", func(t testing.TB, _ NoState) { t.Log("BACKGROUND") })
			f.Rule("outer", func(r *Suite[NoState]) {
				r.Rule("inner", func(r *Suite[NoState]) {
					r.Scenario("narrated", func(s *ScenarioSpec[NoState]) { t.Log("TITLE"); narrate(s) })
				})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_NARRATIVE_SUBPROCESS=1", "-test.v")
	if err != nil {
		t.Fatalf("narrative subprocess failed: %v\n%s", err, out)
	}
	lines := strings.Split(string(out), "\n")
	indexOf := func(pattern string) int {
		t.Helper()
		re, found := regexp.MustCompile(pattern), -1
		for i, line := range lines {
			if re.MatchString(line) {
				if found >= 0 {
					t.Fatalf("more than one line matches %s\n%s", pattern, out)
				}
				found = i
			}
		}
		if found < 0 {
			t.Fatalf("no line matches %s\n%s", pattern, out)
		}
		return found
	}

	// The ancestors line carries no declaration site.
	previous := indexOf(`^\s*Rule: outer > Rule: inner$`)
	column := -1
	for _, want := range []struct{ text, marker string }{
		{"Scenario: narrated", "TITLE"},
		{"  Before hook", "HOOK"},
		{"  Background: the shared fixture", "BACKGROUND"},
		{"  Given the first clause", "GIVEN"},
		{"  But a clause long enough to set the column", "BUT"},
	} {
		markerAt := indexOf(`narrative_test\.go:\d+: ` + want.marker + `$`)
		site := regexp.MustCompile(`narrative_test\.go:\d+`).FindString(lines[markerAt])
		at := indexOf(`^    ` + regexp.QuoteMeta(want.text) + ` +# ` + regexp.QuoteMeta(site) + `$`)
		if at <= previous {
			t.Fatalf("line %q is out of order\n%s", want.text, out)
		}
		if want.marker != "TITLE" && markerAt <= at {
			t.Fatalf("operation %q logged before its narrative line\n%s", want.text, out)
		}
		if here := strings.Index(lines[at], "#"); column >= 0 && here != column {
			t.Fatalf("declaration site of %q is at column %d; want %d\n%s", want.text, here, column, out)
		} else {
			column = here
		}
		previous = at
	}
}

func narrate(s *ScenarioSpec[NoState]) {
	chain := s.Given("the first clause", func(t testing.TB, _ NoState) { t.Log("GIVEN") })
	chain.But("a clause long enough to set the column", func(t testing.TB, _ NoState) { t.Log("BUT") })
}
