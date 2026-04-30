package bdd

import "testing"

func TestPendingStepSkipsFollowingSteps(t *testing.T) {
	var ran bool

	Feature(t, "pending scenario", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("stops after pending step", func(s *ScenarioSpec[NoState]) {
			s.Pending("implementation is missing", "not built yet")
			s.Then("following steps do not run", func(t testing.TB, _ NoState) {
				ran = true
			})
		})
	})

	if ran {
		t.Fatal("step after pending ran")
	}
}

func TestPendingStepWithoutReasonSkipsFollowingSteps(t *testing.T) {
	var ran bool

	Feature(t, "pending scenario without reason", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("stops after pending step", func(s *ScenarioSpec[NoState]) {
			s.Pending("implementation is missing", "")
			s.Then("following steps do not run", func(t testing.TB, _ NoState) {
				ran = true
			})
		})
	})

	if ran {
		t.Fatal("step after pending ran")
	}
}

func TestStateFactoryCanUseScenarioCleanupAndContext(t *testing.T) {
	type world struct {
		contextLive bool
		cleaned     *bool
	}

	var cleaned, canceledBeforeCleanup bool

	Feature(t, "scenario cleanup", func(t testing.TB) *world {
		ctx := t.Context()
		t.Cleanup(func() {
			canceledBeforeCleanup = ctx.Err() != nil
			cleaned = true
		})
		return &world{
			contextLive: ctx.Err() == nil,
			cleaned:     &cleaned,
		}
	}, func(f *Suite[*world]) {
		f.Scenario("state factory owns scenario resources", func(s *ScenarioSpec[*world]) {
			s.Then("scenario context is available", func(t testing.TB, w *world) {
				if !w.contextLive {
					t.Fatal("context was canceled before scenario ran")
				}
				if *w.cleaned {
					t.Fatal("cleanup ran before scenario completed")
				}
			})
		})
	})

	if !cleaned {
		t.Fatal("scenario cleanup did not run")
	}
	if !canceledBeforeCleanup {
		t.Fatal("scenario context was not canceled before cleanup")
	}
}
