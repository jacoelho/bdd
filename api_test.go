package bdd

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestOptionFormattingAndUnicodeNames(t *testing.T) {
	m, err := (meta{}).with(Tags("@fast"))
	if err == nil || !strings.Contains(err.Error(), "leading @") {
		t.Fatalf("Tags with @ error = %v; want leading @ validation", err)
	}
	m, err = (meta{}).with(Tags("fast", "login-flow", "café"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"café", "fast", "login-flow"}; !reflect.DeepEqual(m.tags, want) {
		t.Fatalf("tags = %#v; want %#v", m.tags, want)
	}
	if _, err := (meta{}).with(Option{}); err == nil {
		t.Fatal("zero Option unexpectedly accepted")
	}
	if _, err := (meta{}).with(Skip("blocked"), Pending("later")); err == nil {
		t.Fatal("multiple dispositions unexpectedly accepted")
	}

	cases := map[string]string{
		"Café pricing":    "café-pricing",
		"東京 pricing":      "東京-pricing",
		"Café / pricing":  "café-pricing",
		"Customer's cart": "customer-s-cart",
		" café ":         "café",
		"कः":              "कः",
	}
	for input, want := range cases {
		got, err := slugName(input)
		if err != nil || got != want {
			t.Errorf("slugName(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := slugName("---"); err == nil {
		t.Fatal("separator-only name unexpectedly accepted")
	}
	precomposed, _ := slugName("Café")
	decomposed, _ := slugName("Café")
	if precomposed == decomposed {
		t.Fatalf("precomposed and decomposed slugs collapsed: %q vs %q", precomposed, decomposed)
	}
}

func TestSuiteGroupingAndScenarioAPI(t *testing.T) {
	var order []string

	Feature(t, "typed feature", NewNoState, func(f *Suite[NoState]) {
		f.Before(func(testing.TB, NoState) { order = append(order, "suite before") })
		f.Background("background", func(testing.TB, NoState) { order = append(order, "background") })
		f.And("background continuation", func(testing.TB, NoState) { order = append(order, "background and") })
		f.Rule("rule", func(r *Suite[NoState]) {
			r.Context("context", func(c *Suite[NoState]) {
				c.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
					s.Before(func(testing.TB, NoState) { order = append(order, "scenario before") })
					s.Given("a condition", func(testing.TB, NoState) { order = append(order, "given") })
					s.And("an addition", func(testing.TB, NoState) { order = append(order, "and") })
					s.But("an exception", func(t testing.TB, _ NoState) {
						order = append(order, "but")
						want := []string{"suite before", "scenario before", "background", "background and", "given", "and", "but"}
						if !reflect.DeepEqual(order, want) {
							t.Fatalf("lifecycle order = %#v; want %#v", order, want)
						}
					})
				})
			})
		})
	}, Tags("unit"))

}

func TestScenarioSpecRequiresVisibleOperation(t *testing.T) {
	if os.Getenv("BDD_EMPTY_SCENARIO_SUBPROCESS") == "1" {
		Feature(t, "empty scenario", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("empty", func(*ScenarioSpec[NoState]) {})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestScenarioSpecRequiresVisibleOperation$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_SCENARIO_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "must contain at least one visible operation") || !strings.Contains(string(out), "api_test.go:") {
		t.Fatalf("empty scenario result = %v, output:\n%s", err, out)
	}
}

func TestEmptyFeatureIsRejected(t *testing.T) {
	if os.Getenv("BDD_EMPTY_FEATURE_SUBPROCESS") == "1" {
		Feature(t, "empty feature", NewNoState, func(*Suite[NoState]) {})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestEmptyFeatureIsRejected$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_FEATURE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "must contain at least one Scenario or Outline example") {
		t.Fatalf("empty Feature result = %v, output:\n%s", err, out)
	}
}

func TestEmptyRuleAndContextAreRejected(t *testing.T) {
	if os.Getenv("BDD_EMPTY_GROUP_SUBPROCESS") == "1" {
		Feature(t, "empty groups", NewNoState, func(f *Suite[NoState]) {
			f.Rule("empty rule", func(*Suite[NoState]) {})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestEmptyRuleAndContextAreRejected$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_GROUP_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "rule \"empty rule\" must contain") {
		t.Fatalf("empty Rule result = %v, output:\n%s", err, out)
	}
}

func TestEmptyContextIsRejected(t *testing.T) {
	if os.Getenv("BDD_EMPTY_CONTEXT_SUBPROCESS") == "1" {
		Feature(t, "empty context", NewNoState, func(f *Suite[NoState]) {
			f.Context("empty context", func(*Suite[NoState]) {})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestEmptyContextIsRejected$")
	cmd.Env = append(os.Environ(), "BDD_EMPTY_CONTEXT_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "context \"empty context\" must contain") {
		t.Fatalf("empty Context result = %v, output:\n%s", err, out)
	}
}

func TestSkippedFeatureDoesNotBuild(t *testing.T) {
	var ran bool
	Feature(t, "skipped feature", NewNoState, func(f *Suite[NoState]) {
		ran = true
	}, Skip("skip feature"))
	if ran {
		t.Fatal("skipped feature build ran")
	}
}

func TestPendingFeaturePlaceholderHasStablePath(t *testing.T) {
	if os.Getenv("BDD_PENDING_FEATURE_SUBPROCESS") == "1" {
		Feature(t, "pending feature", NewNoState, func(*Suite[NoState]) {}, Pending("waiting for policy"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPendingFeaturePlaceholderHasStablePath$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_PENDING_FEATURE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	path := "TestPendingFeaturePlaceholderHasStablePath/feature-pending-feature/feature-pending-feature"
	if err != nil || !strings.Contains(text, path) || !strings.Contains(text, "bdd: Feature \"pending feature\" pending: waiting for policy") || !strings.Contains(text, "--- SKIP: "+path) || !strings.Contains(text, "--- PASS: TestPendingFeaturePlaceholderHasStablePath/feature-pending-feature") {
		t.Fatalf("pending Feature result = %v, output:\n%s", err, out)
	}
}

func TestSkippedFeaturePlaceholderHasStableTaggedPath(t *testing.T) {
	if os.Getenv("BDD_SKIPPED_FEATURE_SUBPROCESS") == "1" {
		Feature(t, "skipped feature", func(t testing.TB) NoState {
			t.Fatal("setup ran for skipped Feature")
			return NoState{}
		}, func(*Suite[NoState]) {
			t.Fatal("builder ran for skipped Feature")
		}, Skip("blocked by inventory"), Tags("unit"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSkippedFeaturePlaceholderHasStableTaggedPath$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_SKIPPED_FEATURE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	path := "TestSkippedFeaturePlaceholderHasStableTaggedPath/feature-skipped-feature/@unit__feature-skipped-feature"
	barrier := "TestSkippedFeaturePlaceholderHasStableTaggedPath/feature-skipped-feature"
	if err != nil || !strings.Contains(text, path) || !strings.Contains(text, "bdd: Feature \"skipped feature\" skipped: blocked by inventory") || !strings.Contains(text, "--- SKIP: "+path) || !strings.Contains(text, "--- PASS: "+barrier) || strings.Contains(text, "setup ran for skipped Feature") || strings.Contains(text, "builder ran for skipped Feature") {
		t.Fatalf("skipped Feature result = %v, output:\n%s", err, out)
	}
}

func TestSetupErrorStopsHooksAndSteps(t *testing.T) {
	if os.Getenv("BDD_SETUP_ERROR_SUBPROCESS") == "1" {
		Feature(t, "setup error", func(t testing.TB) NoState {
			t.Error("setup failed")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Before(func(t testing.TB, _ NoState) { t.Fatal("suite hook ran") })
			f.Scenario("stops", func(s *ScenarioSpec[NoState]) {
				s.Then("does not run", func(t testing.TB, _ NoState) { t.Fatal("step ran") })
			})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSetupErrorStopsHooksAndSteps$")
	cmd.Env = append(os.Environ(), "BDD_SETUP_ERROR_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err == nil || !strings.Contains(text, "setup failed") || strings.Contains(text, "hook ran") || strings.Contains(text, "step ran") {
		t.Fatalf("setup error result = %v, output:\n%s", err, out)
	}
}

func TestParallelScenarioSetupResumesAndFeatureWaitsForCleanup(t *testing.T) {
	builderReturned := make(chan struct{})
	var ran, cleaned bool
	Feature(t, "parallel lifecycle", func(t testing.TB) NoState {
		<-builderReturned
		ran = true
		t.Cleanup(func() {
			if !ran {
				t.Error("parallel setup cleanup ran before setup marker")
			}
			cleaned = true
		})
		return NoState{}
	}, func(f *Suite[NoState]) {
		f.Scenario("runs in parallel", func(s *ScenarioSpec[NoState]) {
			s.Parallel()
			s.Then("runs after resume", func(t testing.TB, _ NoState) {
				if !ran {
					t.Fatal("parallel scenario ran without setup")
				}
			})
		})
		close(builderReturned)
	}, Tags("unit", "café"))
	if !ran || !cleaned {
		t.Fatalf("parallel lifecycle ran=%v cleaned=%v; want both true", ran, cleaned)
	}
}

func TestParallelScenariosReceiveDistinctState(t *testing.T) {
	type state struct{ id int32 }
	var next atomic.Int32
	var mu sync.Mutex
	var states []*state
	Feature(t, "distinct state", func(testing.TB) *state {
		state := &state{id: next.Add(1)}
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
		return state
	}, func(f *Suite[*state]) {
		for _, name := range []string{"first", "second"} {
			name := name
			f.Scenario(name, func(s *ScenarioSpec[*state]) {
				s.Parallel()
				s.Then("has an isolated state", func(t testing.TB, w *state) {
					if w.id != 1 && w.id != 2 {
						t.Fatalf("unexpected state id %d", w.id)
					}
					_ = name
				})
			})
		}
	}, Tags("unit", "café"))
	mu.Lock()
	got := append([]*state(nil), states...)
	mu.Unlock()
	if len(got) != 2 || got[0] == nil || got[1] == nil || got[0] == got[1] {
		t.Fatalf("setup states = %#v; want two distinct nonnil pointers", got)
	}
}

func TestParallelOutlineExamplesReceiveDistinctState(t *testing.T) {
	type state struct{ id int32 }
	var next atomic.Int32
	var mu sync.Mutex
	var states []*state
	Feature(t, "distinct outline state", func(testing.TB) *state {
		state := &state{id: next.Add(1)}
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
		return state
	}, func(f *Suite[*state]) {
		f.Outline("examples", []Example[int]{
			Case("first", 1),
			Case("second", 2),
		}, func(_ int, s *ScenarioSpec[*state]) {
			s.Parallel()
			s.Then("has one distinct state", func(t testing.TB, w *state) {
				if w == nil || (w.id != 1 && w.id != 2) {
					t.Fatalf("unexpected outline state %#v", w)
				}
			})
		})
	}, Tags("unit", "café"))
	mu.Lock()
	got := append([]*state(nil), states...)
	mu.Unlock()
	if len(got) != 2 || got[0] == nil || got[1] == nil || got[0] == got[1] {
		t.Fatalf("outline setup states = %#v; want two distinct nonnil pointers", got)
	}
}

func TestSkippedGroupsDoNotBuild(t *testing.T) {
	var ran bool
	Feature(t, "skipped groups", NewNoState, func(f *Suite[NoState]) {
		f.Rule("skipped rule", func(r *Suite[NoState]) {
			ran = true
		}, Skip("skip rule"))
		f.Context("skipped context", func(c *Suite[NoState]) {
			ran = true
		}, Pending("waiting for context"))
	})
	if ran {
		t.Fatal("skipped group build ran")
	}
}

func TestSkippedScenarioDoesNotBuild(t *testing.T) {
	var ran bool
	Feature(t, "skipped scenario", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("skipped", func(s *ScenarioSpec[NoState]) {
			ran = true
		}, Skip("skip scenario"))
	})
	if ran {
		t.Fatal("skipped scenario builder ran")
	}
}

func TestSetupCanUseScenarioCleanupAndContext(t *testing.T) {
	type world struct {
		contextLive bool
		cleaned     *bool
	}

	var cleaned, canceledBeforeCleanup, ran bool
	t.Cleanup(func() {
		if ran && (!cleaned || !canceledBeforeCleanup) {
			t.Fatalf("cleanup=%v canceledBeforeCleanup=%v", cleaned, canceledBeforeCleanup)
		}
	})
	Feature(t, "scenario cleanup", func(t testing.TB) *world {
		ctx := t.Context()
		t.Cleanup(func() {
			canceledBeforeCleanup = ctx.Err() != nil
			cleaned = true
		})
		return &world{contextLive: ctx.Err() == nil, cleaned: &cleaned}
	}, func(f *Suite[*world]) {
		f.Scenario("setup owns scenario resources", func(s *ScenarioSpec[*world]) {
			s.Then("scenario context is available", func(t testing.TB, w *world) {
				ran = true
				if !w.contextLive || *w.cleaned {
					t.Fatal("scenario resources were not live")
				}
			})
		})
	}, Tags("unit", "café"))
	if !ran || !cleaned || !canceledBeforeCleanup {
		t.Fatalf("scenario cleanup ran=%v cleaned=%v canceledBeforeCleanup=%v; want all true", ran, cleaned, canceledBeforeCleanup)
	}
}

func TestComposeStopsAfterFailedStep(t *testing.T) {
	if os.Getenv("BDD_COMPOSE_FAIL_SUBPROCESS") == "1" {
		step := Compose(
			func(t testing.TB, _ NoState) { t.Fatal("stop") },
			func(t testing.TB, _ NoState) { t.Fatal("second step ran") },
		)
		step(t, NoState{})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestComposeStopsAfterFailedStep$")
	cmd.Env = append(os.Environ(), "BDD_COMPOSE_FAIL_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "stop") || strings.Contains(string(out), "second step ran") {
		t.Fatalf("composed step result = %v, output:\n%s", err, out)
	}
}

func TestGeneratedNamesRemainUTF8AndAtomic(t *testing.T) {
	if os.Getenv("BDD_UTF8_PATH_SUBPROCESS") == "1" {
		Feature(t, "Café pricing", NewNoState, func(f *Suite[NoState]) {
			f.Rule("réductions", func(r *Suite[NoState]) {
				r.Scenario("prix réduit", func(s *ScenarioSpec[NoState]) {
					s.Given("the cart is eligible", func(testing.TB, NoState) {})
				})
			})
		}, Tags("café"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestGeneratedNamesRemainUTF8AndAtomic$", "-test.v")
	cmd.Env = append(os.Environ(), "BDD_UTF8_PATH_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("UTF-8 path subprocess failed: %v\n%s", err, out)
	}
	path := "TestGeneratedNamesRemainUTF8AndAtomic/feature-café-pricing/@café__rule-réductions__scenario-prix-réduit"
	if !strings.Contains(string(out), path) {
		t.Fatalf("output missing atomic UTF-8 path %q:\n%s", path, out)
	}
	if strings.Contains(string(out), "/Given:") {
		t.Fatalf("visible step became a subtest:\n%s", out)
	}
}

func TestUnicodeTagScenario(t *testing.T) {
	Feature(t, "Café tagged", NewNoState, func(f *Suite[NoState]) {
		f.Scenario("prix", func(s *ScenarioSpec[NoState]) {
			s.Then("runs when selected", func(testing.TB, NoState) {})
		})
	}, Tags("café"))
}

func TestTagFilteringDoesNotBuildUnselectedLeaves(t *testing.T) {
	if os.Getenv("BDD_FILTER_SUBPROCESS") == "1" {
		Feature(t, "filtering", func(testing.TB) NoState {
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("selected", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
			f.Scenario("unselected", func(*ScenarioSpec[NoState]) {
				t.Fatal("unselected scenario builder ran")
			})
		}, Tags("unit"))
		return
	}
	selected := exec.Command(os.Args[0], "-test.run=^TestTagFilteringDoesNotBuildUnselectedLeaves$/^feature-filtering$/^@unit__scenario-selected$")
	selected.Env = append(os.Environ(), "BDD_FILTER_SUBPROCESS=1")
	if out, err := selected.CombinedOutput(); err != nil {
		t.Fatalf("selected tag result = %v, output:\n%s", err, out)
	}
	none := exec.Command(os.Args[0], "-test.run=^TestTagFilteringDoesNotBuildUnselectedLeaves$/^feature-filtering$/^@other__scenario-selected$")
	none.Env = append(os.Environ(), "BDD_FILTER_SUBPROCESS=1")
	if out, err := none.CombinedOutput(); err != nil {
		t.Fatalf("nonmatching tag result = %v, output:\n%s", err, out)
	}
}

func TestDuplicateGeneratedNamesFailBeforeSetup(t *testing.T) {
	if os.Getenv("BDD_DUPLICATE_SUBPROCESS") == "1" {
		Feature(t, "duplicate", func(testing.TB) NoState {
			t.Fatal("setup ran before duplicate validation")
			return NoState{}
		}, func(f *Suite[NoState]) {
			f.Scenario("same!", func(s *ScenarioSpec[NoState]) {
				s.Then("a step", func(testing.TB, NoState) {})
			}, Tags("first"))
			f.Scenario("same?", func(s *ScenarioSpec[NoState]) {
				s.Then("another step", func(testing.TB, NoState) {})
			}, Tags("second"))
		})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDuplicateGeneratedNamesFailBeforeSetup$")
	cmd.Env = append(os.Environ(), "BDD_DUPLICATE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "duplicate semantic scenario identity") || strings.Count(string(out), "api_test.go:") < 2 || strings.Contains(string(out), "setup ran before") {
		t.Fatalf("duplicate validation result = %v, output:\n%s", err, out)
	}
}

func TestDeclarationScopesSealAtBuilderReturn(t *testing.T) {
	var suite *Suite[NoState]
	var group *Suite[NoState]
	mustPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s mutation did not panic", name)
			}
		}()
		fn()
	}
	Feature(t, "sealed scopes", NewNoState, func(f *Suite[NoState]) {
		suite = f
		f.Rule("group", func(r *Suite[NoState]) {
			group = r
			r.Scenario("first", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
		})
		f.Scenario("later", func(s *ScenarioSpec[NoState]) {
			// The root and group builders have returned before this lazy body
			// executes, so both declaration scopes must already be sealed.
			mustPanic("root Suite", func() { suite.Before(func(testing.TB, NoState) {}) })
			mustPanic("group Suite", func() { group.Before(func(testing.TB, NoState) {}) })
			s.Then("runs after compilation", func(testing.TB, NoState) {
				mustPanic("ScenarioSpec", func() { s.Then("late", func(testing.TB, NoState) {}) })
			})
		})
	})
}

func TestZeroReceiversPanicWithStableMessages(t *testing.T) {
	assertPanic := func(want string, fn func()) {
		t.Helper()
		defer func() {
			got := recover()
			if got != want {
				t.Fatalf("panic = %v; want %q", got, want)
			}
		}()
		fn()
	}

	var nilSuite *Suite[NoState]
	assertPanic("bdd: Feature requires a nonnil *testing.T", func() {
		Feature[NoState](nil, "feature", NewNoState, func(*Suite[NoState]) {})
	})
	assertPanic("bdd: Suite receiver is nil", func() {
		nilSuite.Scenario("x", func(*ScenarioSpec[NoState]) {})
	})
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.Before(func(testing.TB, NoState) {}) })
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.Background("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.And("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.But("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.Rule("x", func(*Suite[NoState]) {}) })
	assertPanic("bdd: Suite receiver is nil", func() { nilSuite.Context("x", func(*Suite[NoState]) {}) })
	assertPanic("bdd: Suite receiver is nil", func() {
		nilSuite.Outline("x", []Example[int]{{Name: "case", Value: 1}}, func(int, *ScenarioSpec[NoState]) {})
	})
	assertPanic("bdd: Suite receiver is zero", func() {
		(&Suite[NoState]{}).Background("x", func(testing.TB, NoState) {})
	})
	var nilSpec *ScenarioSpec[NoState]
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.Parallel() })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.Before(func(testing.TB, NoState) {}) })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.Given("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.When("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() {
		nilSpec.Then("x", func(testing.TB, NoState) {})
	})
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.And("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.But("x", func(testing.TB, NoState) {}) })
	assertPanic("bdd: ScenarioSpec receiver is nil", func() { nilSpec.Pending("x", "reason") })
	assertPanic("bdd: ScenarioSpec receiver is zero", func() {
		(&ScenarioSpec[NoState]{}).Parallel()
	})
}

func TestNestedFeatureIsRejected(t *testing.T) {
	if os.Getenv("BDD_NESTED_FEATURE_SUBPROCESS") == "1" {
		Feature(t, "outer", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(tb testing.TB, _ NoState) {
					Feature(tb.(*testing.T), "inner", NewNoState, func(*Suite[NoState]) {})
				})
			})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNestedFeatureIsRejected$")
	cmd.Env = append(os.Environ(), "BDD_NESTED_FEATURE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "direct top-level test") {
		t.Fatalf("nested Feature result = %v, output:\n%s", err, out)
	}
}

func TestSecondFeatureOnOneTestIsRejected(t *testing.T) {
	if os.Getenv("BDD_DUPLICATE_FEATURE_SUBPROCESS") == "1" {
		Feature(t, "first", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
		})
		Feature(t, "second", NewNoState, func(f *Suite[NoState]) {
			f.Scenario("scenario", func(s *ScenarioSpec[NoState]) {
				s.Then("runs", func(testing.TB, NoState) {})
			})
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSecondFeatureOnOneTestIsRejected$")
	cmd.Env = append(os.Environ(), "BDD_DUPLICATE_FEATURE_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "already registered") || strings.Count(string(out), "api_test.go:") < 2 {
		t.Fatalf("duplicate Feature result = %v, output:\n%s", err, out)
	}
}

func TestFeatureClaimCleanupReleasesChildOwner(t *testing.T) {
	var owner *testing.T
	t.Run("claim", func(child *testing.T) {
		owner = child
		if err := claimFeature(child, "ownership test"); err != nil {
			t.Fatalf("claimFeature failed: %v", err)
		}
		featureClaims.Lock()
		_, ok := featureClaims.owners[child]
		featureClaims.Unlock()
		if !ok {
			t.Fatal("claim registry did not contain child owner during subtest")
		}
	})
	featureClaims.Lock()
	_, ok := featureClaims.owners[owner]
	featureClaims.Unlock()
	if ok {
		t.Fatal("claim cleanup left child owner registered")
	}
}

func TestInvalidOptionsAreNotHiddenByDisposition(t *testing.T) {
	if os.Getenv("BDD_INVALID_OPTION_SUBPROCESS") == "1" {
		Feature[NoState](t, "invalid options", NewNoState, func(*Suite[NoState]) {}, Skip("visible reason"), Option{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestInvalidOptionsAreNotHiddenByDisposition$")
	cmd.Env = append(os.Environ(), "BDD_INVALID_OPTION_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "zero option") {
		t.Fatalf("invalid option result = %v, output:\n%s", err, out)
	}
}

func TestNilSetupIsNotHiddenBySkip(t *testing.T) {
	if os.Getenv("BDD_NIL_SETUP_SUBPROCESS") == "1" {
		Feature[NoState](t, "nil setup", nil, func(*Suite[NoState]) {}, Skip("visible reason"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNilSetupIsNotHiddenBySkip$")
	cmd.Env = append(os.Environ(), "BDD_NIL_SETUP_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "nil setup callback") {
		t.Fatalf("nil setup result = %v, output:\n%s", err, out)
	}
}

func TestBlankDispositionReasonIsRejected(t *testing.T) {
	if os.Getenv("BDD_BLANK_REASON_SUBPROCESS") == "1" {
		Feature(t, "blank reason", NewNoState, func(*Suite[NoState]) {}, Skip("  "))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBlankDispositionReasonIsRejected$")
	cmd.Env = append(os.Environ(), "BDD_BLANK_REASON_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "nonblank reason") {
		t.Fatalf("blank reason result = %v, output:\n%s", err, out)
	}
}
