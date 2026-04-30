package bdd

import "testing"

type step[S any] struct {
	kind stepKind
	text string
	fn   Step[S]
}

// ScenarioSpec collects steps and hooks for one scenario.
type ScenarioSpec[S any] struct {
	background []step[S]
	before     []Step[S]
	steps      []step[S]
	tags       []string
	parallel   bool
}

// Parallel marks the scenario as safe to run in parallel with other tests.
func (s *ScenarioSpec[S]) Parallel() {
	s.parallel = true
}

// Tags adds scenario-level tags.
func (s *ScenarioSpec[S]) Tags(tags ...string) {
	s.tags = appendTags(s.tags, tags...)
}

// Background adds a visible narrative setup step.
func (s *ScenarioSpec[S]) Background(text string, fn Step[S]) *stepChain[S] {
	s.background = append(s.background, newStep(kindBackground, text, fn))
	return &stepChain[S]{steps: &s.background}
}

// Before registers a hidden setup hook. Prefer Background when the setup is
// part of the behavior's story.
func (s *ScenarioSpec[S]) Before(fn Step[S]) {
	s.before = append(s.before, fn)
}

func (s *ScenarioSpec[S]) Given(text string, fn Step[S]) *stepChain[S] {
	s.steps = append(s.steps, newStep(kindGiven, text, fn))
	return &stepChain[S]{steps: &s.steps}
}

func (s *ScenarioSpec[S]) When(text string, fn Step[S]) *stepChain[S] {
	s.steps = append(s.steps, newStep(kindWhen, text, fn))
	return &stepChain[S]{steps: &s.steps}
}

func (s *ScenarioSpec[S]) Then(text string, fn Step[S]) *stepChain[S] {
	s.steps = append(s.steps, newStep(kindThen, text, fn))
	return &stepChain[S]{steps: &s.steps}
}

// Pending adds a pending visible step. The scenario is reported as skipped and
// following steps do not run.
func (s *ScenarioSpec[S]) Pending(text, reason string) *stepChain[S] {
	s.steps = append(s.steps, newStep(kindPending, text, func(t testing.TB, _ S) {
		t.Helper()
		if reason == "" {
			t.Skip("pending")
		}
		t.Skip("pending: " + reason)
	}))
	return &stepChain[S]{steps: &s.steps}
}

type stepChain[S any] struct {
	steps *[]step[S]
}

func (c *stepChain[S]) And(text string, fn Step[S]) *stepChain[S] {
	c.append(kindAnd, text, fn)
	return c
}

func (c *stepChain[S]) But(text string, fn Step[S]) *stepChain[S] {
	c.append(kindBut, text, fn)
	return c
}

func (c *stepChain[S]) append(kind stepKind, text string, fn Step[S]) {
	*c.steps = append(*c.steps, newStep(kind, text, fn))
}

func runScenario[S any](scope *scope, name string, newState StateFactory[S], spec *ScenarioSpec[S], opts ...Option) {
	m := scope.meta.withTags(spec.tags...).with(opts...)
	scope.t.Run(scenarioName(name, m), func(t *testing.T) {
		if msg := m.skipMessage(); msg != "" {
			t.Skip(msg)
		}
		if spec.parallel {
			t.Parallel()
		}

		state := newState(t)

		for i := 0; !t.Failed() && !t.Skipped() && i < len(spec.before); i++ {
			spec.before[i](t, state)
		}
		if !t.Failed() && !t.Skipped() {
			runSteps(t, state, spec.background, spec.steps)
		}
	})
}

type stepResult struct {
	ok      bool
	skipped bool
}

func runVisibleStep[S any](parent *testing.T, st step[S], state S) stepResult {
	parent.Helper()
	var skipped bool
	ok := parent.Run(stepName(st.kind, st.text), func(t *testing.T) {
		t.Helper()
		defer func() {
			skipped = t.Skipped()
		}()
		st.fn(t, state)
	})

	return stepResult{ok: ok, skipped: skipped}
}

func runSteps[S any](t *testing.T, state S, groups ...[]step[S]) {
	t.Helper()
	for _, steps := range groups {
		for _, st := range steps {
			result := runVisibleStep(t, st, state)
			if result.skipped {
				t.SkipNow()
			}
			if !result.ok {
				return
			}
		}
	}
}

func newStep[S any](kind stepKind, text string, fn Step[S]) step[S] {
	return step[S]{kind: kind, text: text, fn: fn}
}
