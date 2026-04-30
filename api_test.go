package bdd

import (
	"reflect"
	"testing"
)

func TestOptionFormattingAndSkipMessages(t *testing.T) {
	var m meta
	Tags("@fast", "login flow", "", "fast")(&m)
	if !reflect.DeepEqual(m.tags, []string{"fast", "login-flow"}) {
		t.Fatalf("tags = %#v; want fast/login-flow", m.tags)
	}

	withNil := m.with(nil)
	if !reflect.DeepEqual(withNil.tags, m.tags) {
		t.Fatalf("with(nil).tags = %#v; want %#v", withNil.tags, m.tags)
	}

	skipped := meta{}.with(Skip(""))
	if got := skipped.skipMessage(); got != "skipped" {
		t.Fatalf("skipMessage() = %q; want skipped", got)
	}

	skipped = meta{}.with(Skip("blocked"))
	if got := skipped.skipMessage(); got != "blocked" {
		t.Fatalf("skipMessage() = %q; want blocked", got)
	}

	pending := meta{}.with(Pending(""))
	if got := pending.skipMessage(); got != "pending" {
		t.Fatalf("skipMessage() = %q; want pending", got)
	}

	pending = meta{}.with(Pending("needs API"))
	if got := pending.skipMessage(); got != "pending: needs API" {
		t.Fatalf("skipMessage() = %q; want pending reason", got)
	}

	if got := (meta{}).skipMessage(); got != "" {
		t.Fatalf("skipMessage() = %q; want empty", got)
	}

	if got := cleanName(" \n\t "); got != "unnamed" {
		t.Fatalf("cleanName(empty) = %q; want unnamed", got)
	}
}

func TestSuiteGroupingAndScenarioAPI(t *testing.T) {
	var before, background, given, but bool

	Feature(t, "typed feature", NewNoState, func(f *Suite[NoState]) {
		f.Rule("rule", func(r *Suite[NoState]) {
			r.Context("context", func(c *Suite[NoState]) {
				c.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
					s.Tags("unit")
					s.Background("background", func(t testing.TB, _ NoState) {
						background = true
					})
					s.Before(func(t testing.TB, _ NoState) {
						before = true
					})
					s.Given("a condition", func(t testing.TB, _ NoState) {
						given = true
					})
					s.But("an exception", func(t testing.TB, _ NoState) {
						but = true
					})
				})
			})
		})
	})

	if !before || !background || !given || !but {
		t.Fatalf("steps ran: before=%v background=%v given=%v but=%v", before, background, given, but)
	}
}

func TestSkippedFeaturesGroupsAndScenarios(t *testing.T) {
	var ran bool

	Feature(t, "skipped feature", NewNoState, func(f *Suite[NoState]) {
		ran = true
	}, Skip("skip feature"))
	if ran {
		t.Fatal("skipped feature build ran")
	}

	Feature(t, "skipped groups", NewNoState, func(f *Suite[NoState]) {
		f.Rule("skipped rule", func(r *Suite[NoState]) {
			ran = true
		}, Skip(""))
		f.Context("skipped context", func(c *Suite[NoState]) {
			ran = true
		}, Pending(""))
	})
	if ran {
		t.Fatal("skipped group build ran")
	}

	Feature(t, "skipped scenario", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("skipped", func(s *ScenarioSpec[NoState]) {
			s.Then("does not run", func(t testing.TB, _ NoState) {
				ran = true
			})
		}, Skip("skip scenario"))
	})
	if ran {
		t.Fatal("skipped scenario step ran")
	}
}

func TestTypedSuiteContextBeforeAndTags(t *testing.T) {
	var before, then bool

	Feature(t, "typed feature", NewNoState, func(f *Suite[NoState]) {
		f.Context("context", func(c *Suite[NoState]) {
			c.Tags("@typed")
			c.Before(func(t testing.TB, _ NoState) {
				before = true
			})
			c.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("it runs", func(t testing.TB, _ NoState) {
					then = true
				})
			})
		})
	})

	if !before || !then {
		t.Fatalf("steps ran: before=%v then=%v", before, then)
	}
}

func TestSkippedTypedContextAndOutlines(t *testing.T) {
	var ran bool

	Feature(t, "skipped typed context", NewNoState, func(f *Suite[NoState]) {
		f.Context("context", func(c *Suite[NoState]) {
			ran = true
		}, Skip("skip typed context"))
	})
	if ran {
		t.Fatal("skipped typed context build ran")
	}

	Feature(t, "skipped outline", NewNoState, func(f *Suite[NoState]) {
		Outline(f, "outline", []Example[int]{Case("one", 1)}, func(int, *ScenarioSpec[NoState]) {
			ran = true
		}, Pending("skip outline"))
	})
	if ran {
		t.Fatal("skipped outline build ran")
	}
}

func TestParallelScenario(t *testing.T) {
	var ran bool

	Feature(t, "parallel", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
			s.Parallel()
			s.Then("it runs", func(t testing.TB, _ NoState) {
				ran = true
			})
		})
	})

	if !ran {
		t.Fatal("parallel scenario did not run")
	}
}
