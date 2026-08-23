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
	rootSource string
}

// Parallel marks the scenario as safe to run in parallel with other tests.
func (s *ScenarioSpec[S]) Parallel() {
	where := declarationSource()
	s.ensureOpen()
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
	if len(s.steps) > 0 {
		s.failfAt(where, "Before must be declared before visible operations")
	}
	if fn == nil {
		s.failfAt(where, "Scenario Before has a nil step")
	}
	s.before = append(s.before, fn)
}

// Given starts a fluent scenario narrative and returns a private capability
// with Given, When, Then, And, and But methods. Cucumber/Gherkin keyword order
// is flexible; reassign every returned value when storing the capability. See
// the Gherkin reference for the corresponding narrative keywords:
// https://cucumber.io/docs/gherkin/reference.
func (s *ScenarioSpec[S]) Given(text string, fn Step[S]) scenarioChain[S] {
	return s.root(kindGiven, text, fn, declarationSource())
}

// When starts a fluent scenario narrative and returns the same private fluent
// capability as Given. It supports Given, When, Then, And, and But in flexible
// Cucumber/Gherkin order; reassign every returned value.
func (s *ScenarioSpec[S]) When(text string, fn Step[S]) scenarioChain[S] {
	return s.root(kindWhen, text, fn, declarationSource())
}

// Then starts a fluent scenario narrative and returns the same private fluent
// capability as Given. It supports Given, When, Then, And, and But in flexible
// Cucumber/Gherkin order; reassign every returned value.
func (s *ScenarioSpec[S]) Then(text string, fn Step[S]) scenarioChain[S] {
	return s.root(kindThen, text, fn, declarationSource())
}

func (s *ScenarioSpec[S]) root(kind stepKind, text string, fn Step[S], where string) scenarioChain[S] {
	s.ensureOpen()
	if len(s.steps) > 0 {
		s.failfAt(where, "scenario already started with %s at %s; attempted second root %s at %s; continue it with the returned chain", s.steps[0].kind, s.rootSource, kind, where)
	}
	s.appendVisible(where, kind, text, fn)
	s.rootSource = where
	return s.newChain(where)
}

func (s *ScenarioSpec[S]) newChain(issued string) scenarioChain[S] {
	return scenarioChain[S]{core: chainCore[S]{
		expected: len(s.steps),
		issued:   issued,
		advance:  s.appendVisibleFromChain,
	}}
}

func (s *ScenarioSpec[S]) appendVisibleFromChain(expected int, issued, where string, kind stepKind, text string, fn Step[S]) int {
	// Owner state must be checked before generation state so escaped tokens
	// always report the stable owner-sealed diagnostic.
	s.ensureOpen()
	if len(s.steps) != expected {
		s.failfAt(where, "stale scenario chain token issued at %s; expected operation generation %d, found %d; assign every returned chain value", issued, expected, len(s.steps))
	}
	s.appendVisible(where, kind, text, fn)
	return expected + 1
}

func (s *ScenarioSpec[S]) appendVisible(where string, kind stepKind, text string, fn Step[S]) {
	s.ensureOpen()
	if fn == nil {
		s.failfAt(where, "%s %q has a nil step", kind, text)
	}
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "%s text is blank", kind)
	}
	s.steps = append(s.steps, newStep(kind, text, fn))
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
