package bdd

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestZeroChainTokensFailImmediately(t *testing.T) {
	assertPanic := func(want string, call func()) {
		t.Helper()
		defer func() {
			if got := recover(); got != want {
				t.Fatalf("panic = %v; want %q", got, want)
			}
		}()
		call()
	}

	assertPanic("bdd: chain token is zero", func() {
		var chain ScenarioChain[NoState]
		chain.And("and", func(testing.TB, NoState) {})
	})
	assertPanic("bdd: chain token is zero", func() {
		var chain BackgroundChain[NoState]
		chain.But("but", func(testing.TB, NoState) {})
	})
}

func TestCopiedChainTokenFailsBeforeAppending(t *testing.T) {
	if os.Getenv("BDD_STALE_CHAIN_SUBPROCESS") == "1" {
		Feature(t, "stale chain", func(testing.TB) NoState {
			t.Log("STALE_SETUP_RAN")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("stale token", func(s *ScenarioSpec[NoState]) {
				chain := s.Given("the first operation", func(testing.TB, NoState) {})
				_ = chain.When("the second operation", func(testing.TB, NoState) {})
				chain.Then("the stale operation", func(testing.TB, NoState) {})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_STALE_CHAIN_SUBPROCESS=1")
	if err == nil || !strings.Contains(string(out), "stale scenario chain token") || !strings.Contains(string(out), "chains_test.go:") || strings.Contains(string(out), "STALE_SETUP_RAN") {
		t.Fatalf("stale chain result = %v; output:\n%s", err, out)
	}
}

func TestCopiedBackgroundTokenFailsBeforeAppending(t *testing.T) {
	if os.Getenv("BDD_STALE_BACKGROUND_SUBPROCESS") == "1" {
		Feature(t, "stale background", func(testing.TB) NoState {
			t.Log("STALE_BACKGROUND_SETUP_RAN")
			return NoState{}
		}, func(f *Suite[NoState]) {
			chain := f.Background("the first background", func(testing.TB, NoState) {})
			_ = chain.And("the second background", func(testing.TB, NoState) {})
			chain.But("the stale background", func(testing.TB, NoState) {})
			f.Scenario("never runs", func(s *ScenarioSpec[NoState]) {
				s.Then("the step", func(testing.TB, NoState) {})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_STALE_BACKGROUND_SUBPROCESS=1")
	text := string(out)
	if err == nil || !strings.Contains(text, "stale Background chain token") || strings.Count(text, "chains_test.go:") < 2 || !strings.Contains(text, "assign every returned chain value") || strings.Contains(text, "STALE_BACKGROUND_SETUP_RAN") {
		t.Fatalf("stale Background result = %v; output:\n%s", err, out)
	}
}

func TestSecondScenarioStarterReportsBothDeclarationKinds(t *testing.T) {
	if os.Getenv("BDD_SECOND_STARTER_SUBPROCESS") == "1" {
		Feature(t, "second starter", func(testing.TB) NoState {
			t.Log("SECOND_ROOT_SETUP_RAN")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("one root", func(s *ScenarioSpec[NoState]) {
				s.Given("the first root", func(testing.TB, NoState) {})
				s.When("the second root", func(testing.TB, NoState) {})
			})
		})
		return
	}

	t.Parallel()
	out, err := runSelf(t, "BDD_SECOND_STARTER_SUBPROCESS=1")
	text := string(out)
	if err == nil || !strings.Contains(text, "scenario already started with Given") || !strings.Contains(text, "attempted second root When") || !strings.Contains(text, "continue it with the returned chain") || strings.Count(text, "chains_test.go:") < 2 || strings.Contains(text, "SECOND_ROOT_SETUP_RAN") {
		t.Fatalf("second starter result = %v; output:\n%s", err, out)
	}
}

func TestChainCannotEscapeScenarioBuilder(t *testing.T) {
	var current, stale ScenarioChain[NoState]
	Feature(t, "chain lifetime", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("captures a chain", func(s *ScenarioSpec[NoState]) {
			stale = s.Given("the operation", func(testing.TB, NoState) {})
			current = stale.When("the second operation", func(testing.TB, NoState) {})
		})
	}, Tags("unit", "café"))
	for name, chain := range map[string]ScenarioChain[NoState]{"current": current, "stale": stale} {
		t.Run(name, func(t *testing.T) {
			assertSealedPanic(t, "ScenarioChain.And", func() {
				chain.And("after the builder", func(testing.TB, NoState) {})
			})
		})
	}
}

func TestSuiteChainCannotEscapeBuilder(t *testing.T) {
	var suite *Suite[NoState]
	var group *Suite[NoState]
	var currentBackground, staleBackground BackgroundChain[NoState]
	Feature(t, "suite lifetime", NewNoState, func(f *Suite[NoState]) {
		suite = f
		staleBackground = f.Background("the background", func(testing.TB, NoState) {})
		currentBackground = staleBackground.And("the continuation", func(testing.TB, NoState) {})
		f.Rule("group", func(r *Suite[NoState]) {
			group = r
			r.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
		})
	}, Tags("unit", "café"))
	for name, tc := range map[string]struct {
		operation string
		call      func()
	}{
		"root":               {"Suite.Before", func() { suite.Before(func(testing.TB, NoState) {}) }},
		"group":              {"Suite.Before", func() { group.Before(func(testing.TB, NoState) {}) }},
		"current Background": {"BackgroundChain.And", func() { currentBackground.And("after the builder", func(testing.TB, NoState) {}) }},
		"stale Background":   {"BackgroundChain.But", func() { staleBackground.But("after the builder", func(testing.TB, NoState) {}) }},
	} {
		t.Run(name, func(t *testing.T) { assertSealedPanic(t, tc.operation, tc.call) })
	}
}

// assertSealedPanic requires call to panic with the sealed-scope diagnostic:
// the call site, the operation that was attempted, and the scope that owns it.
func assertSealedPanic(t testing.TB, operation string, call func()) {
	t.Helper()
	scope := "ScenarioSpec"
	if strings.HasPrefix(operation, "Suite.") || strings.HasPrefix(operation, "BackgroundChain.") {
		scope = "Suite"
	}
	want := regexp.MustCompile(`^bdd: \S*_test\.go:\d+: ` + regexp.QuoteMeta(operation) + ` called on a sealed ` + scope + `: `)
	defer func() {
		t.Helper()
		if got, _ := recover().(string); !want.MatchString(got) {
			t.Fatalf("panic = %q; want match for %s", got, want)
		}
	}()
	call()
}

func TestAbruptBuilderTerminationSealsChainOwners(t *testing.T) {
	assertPanic := func(name, operation string, call func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) { assertSealedPanic(t, operation, call) })
	}

	t.Run("panic", func(t *testing.T) {
		suite := newSuite(newFeaturePlan(NewNoState), t, nil)
		var stale, current BackgroundChain[NoState]
		func() {
			defer suite.seal()
			defer func() {
				if got := recover(); got != "builder panic" {
					t.Fatalf("builder panic = %v; want builder panic", got)
				}
			}()
			stale = suite.Background("the background", func(testing.TB, NoState) {})
			current = stale.And("the continuation", func(testing.TB, NoState) {})
			panic("builder panic")
		}()
		assertPanic("suite current token", "BackgroundChain.And", func() {
			current.And("after the panic", func(testing.TB, NoState) {})
		})
		assertPanic("suite stale token", "BackgroundChain.But", func() {
			stale.But("after the panic", func(testing.TB, NoState) {})
		})

		spec := newScenarioSpec[NoState](t)
		var staleScenario, currentScenario ScenarioChain[NoState]
		func() {
			defer spec.seal()
			defer func() {
				if got := recover(); got != "scenario builder panic" {
					t.Fatalf("scenario builder panic = %v; want scenario builder panic", got)
				}
			}()
			staleScenario = spec.Given("the first operation", func(testing.TB, NoState) {})
			currentScenario = staleScenario.When("the second operation", func(testing.TB, NoState) {})
			panic("scenario builder panic")
		}()
		assertPanic("scenario current token", "ScenarioChain.And", func() {
			currentScenario.And("after the panic", func(testing.TB, NoState) {})
		})
		assertPanic("scenario stale token", "ScenarioChain.But", func() {
			staleScenario.But("after the panic", func(testing.TB, NoState) {})
		})
	})

	t.Run("Goexit seals Suite", func(t *testing.T) {
		suite := newSuite(newFeaturePlan(NewNoState), t, nil)
		var stale, current BackgroundChain[NoState]
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer suite.seal()
			stale = suite.Background("the background", func(testing.TB, NoState) {})
			current = stale.And("the continuation", func(testing.TB, NoState) {})
			runtime.Goexit()
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for Suite builder Goexit")
		}
		assertPanic("Goexit current Background token", "BackgroundChain.And", func() {
			current.And("after Goexit", func(testing.TB, NoState) {})
		})
		assertPanic("Goexit stale Background token", "BackgroundChain.But", func() {
			stale.But("after Goexit", func(testing.TB, NoState) {})
		})
	})

	t.Run("Goexit seals ScenarioSpec", func(t *testing.T) {
		spec := newScenarioSpec[NoState](t)
		var stale, current ScenarioChain[NoState]
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer spec.seal()
			stale = spec.Given("the first operation", func(testing.TB, NoState) {})
			current = stale.When("the second operation", func(testing.TB, NoState) {})
			runtime.Goexit()
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for ScenarioSpec builder Goexit")
		}
		assertPanic("Goexit current ScenarioSpec token", "ScenarioChain.And", func() {
			current.And("after Goexit", func(testing.TB, NoState) {})
		})
		assertPanic("Goexit stale ScenarioSpec token", "ScenarioChain.But", func() {
			stale.But("after Goexit", func(testing.TB, NoState) {})
		})
	})
}
