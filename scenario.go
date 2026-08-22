package bdd

import (
	"strings"
	"testing"
)

type step[S any] struct {
	kind stepKind
	text string
	fn   Step[S]
}

// ScenarioSpec collects hooks and visible operations for one scenario.
// ScenarioSpec is sealed when its builder returns.
type ScenarioSpec[S any] struct {
	owner      *testing.T
	background []step[S]
	before     []Step[S]
	steps      []step[S]
	parallel   bool
	sealed     bool
	lastClause bool
	terminal   bool
}

// Parallel marks the scenario as safe to run in parallel with other tests.
func (s *ScenarioSpec[S]) Parallel() {
	where := declarationSource()
	s.ensureOpen()
	s.ensureActive(where)
	if s.parallel {
		s.failfAt(where, "Parallel may be declared only once")
	}
	if len(s.steps) > 0 {
		s.failfAt(where, "Parallel must be declared before visible operations")
	}
	s.parallel = true
}

// Before registers a hidden setup hook. It must precede visible operations.
func (s *ScenarioSpec[S]) Before(fn Step[S]) {
	where := declarationSource()
	s.ensureOpen()
	s.ensureActive(where)
	if len(s.steps) > 0 {
		s.failfAt(where, "Before must be declared before visible operations")
	}
	if fn == nil {
		s.failfAt(where, "Scenario Before has a nil step")
	}
	s.before = append(s.before, fn)
}

// Given adds a visible precondition operation.
func (s *ScenarioSpec[S]) Given(text string, fn Step[S]) {
	s.visible(kindGiven, text, fn, declarationSource())
}

// When adds a visible action operation.
func (s *ScenarioSpec[S]) When(text string, fn Step[S]) {
	s.visible(kindWhen, text, fn, declarationSource())
}

// Then adds a visible assertion operation.
func (s *ScenarioSpec[S]) Then(text string, fn Step[S]) {
	s.visible(kindThen, text, fn, declarationSource())
}

// And continues the preceding visible clause.
func (s *ScenarioSpec[S]) And(text string, fn Step[S]) {
	s.continuation(kindAnd, text, fn, declarationSource())
}

// But continues the preceding visible clause.
func (s *ScenarioSpec[S]) But(text string, fn Step[S]) {
	s.continuation(kindBut, text, fn, declarationSource())
}

// Pending adds a terminal pending operation. The reason is required so a
// skipped scenario explains why it is not complete.
func (s *ScenarioSpec[S]) Pending(text, reason string) {
	where := declarationSource()
	s.ensureOpen()
	s.ensureActive(where)
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "Pending text is blank")
	}
	if strings.TrimSpace(reason) == "" {
		s.failfAt(where, "Pending %q requires a nonblank reason", text)
	}
	s.steps = append(s.steps, newStep(kindPending, text, pendingStep[S](reason)))
	s.lastClause = false
	s.terminal = true
}

func (s *ScenarioSpec[S]) visible(kind stepKind, text string, fn Step[S], where string) {
	s.ensureOpen()
	s.ensureActive(where)
	if fn == nil {
		s.failfAt(where, "%s %q has a nil step", kind, text)
	}
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "%s text is blank", kind)
	}
	s.steps = append(s.steps, newStep(kind, text, fn))
	s.lastClause = true
}

func (s *ScenarioSpec[S]) continuation(kind stepKind, text string, fn Step[S], where string) {
	s.ensureOpen()
	if !s.lastClause || len(s.steps) == 0 || s.steps[len(s.steps)-1].kind == kindPending {
		s.failfAt(where, "%s requires a preceding visible clause", kind)
	}
	s.visible(kind, text, fn, where)
}

func pendingStep[S any](reason string) Step[S] {
	return func(t testing.TB, _ S) {
		t.Skip("pending: " + strings.TrimSpace(reason))
	}
}

func newStep[S any](kind stepKind, text string, fn Step[S]) step[S] {
	return step[S]{kind: kind, text: text, fn: fn}
}

func (s *ScenarioSpec[S]) ensureOpen() {
	if s == nil {
		panic("bdd: ScenarioSpec receiver is nil")
	}
	if s.owner == nil {
		panic("bdd: ScenarioSpec receiver is zero")
	}
	if s.sealed {
		panic("bdd: ScenarioSpec builder has returned")
	}
}

func (s *ScenarioSpec[S]) ensureActive(where string) {
	if s.terminal {
		s.failfAt(where, "no operations may follow Pending")
	}
}

func (s *ScenarioSpec[S]) seal() {
	s.sealed = true
}

func (s *ScenarioSpec[S]) failfAt(where, format string, args ...any) {
	s.ensureOpen()
	if where != "" {
		format = where + ": " + format
	}
	s.owner.Fatalf("bdd: "+format, args...)
}

func runStep[S any](t *testing.T, st step[S], state S) {
	t.Helper()
	t.Logf("%s: %s", st.kind, st.text)
	st.fn(t, state)
}
