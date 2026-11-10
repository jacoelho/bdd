package bdd

import (
	"fmt"
	"strings"
	"testing"
)

// step is one executable operation of a scenario: a hidden Before hook or a
// visible narrative clause. text is empty for hooks.
type step[S any] struct {
	kind   stepKind
	text   string
	source string
	fn     Step[S]
}

// narrate is the operation as a line of the scenario narrative. Background
// is a label rather than a keyword that reads as part of the sentence, so it
// takes a colon.
func (st step[S]) narrate() string {
	switch st.kind {
	case kindBefore:
		return "  Before hook"
	case kindBackground:
		return "  Background: " + st.text
	default:
		return "  " + string(st.kind) + " " + st.text
	}
}

// label names the operation in diagnostics.
func (st step[S]) label() string {
	if st.kind == kindBefore {
		return "Before hook"
	}
	return fmt.Sprintf("%s %q", st.kind, st.text)
}

// specPhase is the declaration state of a ScenarioSpec. The legal order is
// scenarioOrder:
//
//	specPreamble  --Parallel, Before-->   specPreamble
//	specPreamble  --Given, When, Then-->  specNarrative
//	specNarrative --chain continuation--> specNarrative
//	any           --builder returns-->    specSealed
type specPhase uint8

const (
	specUnbuilt   specPhase = iota // zero value: not created by the framework
	specPreamble                   // accepts Parallel, Before, and one starter
	specNarrative                  // a starter was declared; only its chain may continue
	specSealed                     // the builder returned
)

const scenarioOrder = "a scenario declares Parallel and Before first, then one Given, When, or Then chain"

// ScenarioSpec collects hooks and visible operations for one scenario. Declare
// Parallel and Before first, then one narrative chain started by Given, When,
// or Then. ScenarioSpec is sealed when its builder returns.
type ScenarioSpec[S any] struct {
	owner    *testing.T
	phase    specPhase
	parallel bool
	before   []step[S]
	steps    []step[S]
}

func newScenarioSpec[S any](owner *testing.T) *ScenarioSpec[S] {
	return &ScenarioSpec[S]{owner: owner, phase: specPreamble}
}

// Parallel marks the scenario as safe to run in parallel with other tests. It
// may be declared once and must precede the Given, When, or Then starter.
func (s *ScenarioSpec[S]) Parallel() {
	where := declarationSource()
	s.ensureOpen(where, "ScenarioSpec.Parallel")
	if s.parallel {
		s.failfAt(where, "Parallel may be declared only once")
	}
	if s.phase == specNarrative {
		s.failfAt(where, "Parallel must be declared before visible operations; %s", scenarioOrder)
	}
	s.parallel = true
}

// Before registers a hidden setup hook that runs after inherited Suite hooks
// and before Background. It must precede the Given, When, or Then starter.
func (s *ScenarioSpec[S]) Before(fn Step[S]) {
	where := declarationSource()
	s.ensureOpen(where, "ScenarioSpec.Before")
	if s.phase == specNarrative {
		s.failfAt(where, "Before must be declared before visible operations; %s", scenarioOrder)
	}
	if fn == nil {
		s.failfAt(where, "Scenario Before has a nil step")
	}
	s.before = append(s.before, step[S]{kind: kindBefore, source: where, fn: fn})
}

// Given starts a fluent scenario narrative and returns a ScenarioChain
// with Given, When, Then, And, and But methods. Cucumber/Gherkin keyword order
// is flexible; reassign every returned value when storing the chain. A
// scenario has exactly one starter: continue with the returned chain. See
// the Gherkin reference for the corresponding narrative keywords:
// https://cucumber.io/docs/gherkin/reference.
func (s *ScenarioSpec[S]) Given(text string, fn Step[S]) ScenarioChain[S] {
	return s.start(declarationSource(), kindGiven, text, fn)
}

// When starts a fluent scenario narrative and returns the same ScenarioChain
// as Given. It supports Given, When, Then, And, and But in flexible
// Cucumber/Gherkin order; reassign every returned value.
func (s *ScenarioSpec[S]) When(text string, fn Step[S]) ScenarioChain[S] {
	return s.start(declarationSource(), kindWhen, text, fn)
}

// Then starts a fluent scenario narrative and returns the same ScenarioChain
// as Given. It supports Given, When, Then, And, and But in flexible
// Cucumber/Gherkin order; reassign every returned value.
func (s *ScenarioSpec[S]) Then(text string, fn Step[S]) ScenarioChain[S] {
	return s.start(declarationSource(), kindThen, text, fn)
}

func (s *ScenarioSpec[S]) start(where string, kind stepKind, text string, fn Step[S]) ScenarioChain[S] {
	s.ensureOpen(where, "ScenarioSpec."+string(kind))
	if s.phase == specNarrative {
		root := s.steps[0]
		s.failfAt(where, "scenario already started with %s at %s; attempted second root %s at %s; continue it with the returned chain", root.kind, root.source, kind, where)
	}
	s.appendVisible(where, kind, text, fn)
	s.phase = specNarrative
	return s.chain(where)
}

func (s *ScenarioSpec[S]) continueNarrative(token ScenarioChain[S], where string, kind stepKind, text string, fn Step[S]) ScenarioChain[S] {
	// Owner state is checked before the generation so a chain used after the
	// builder returned always reports the sealed-scope diagnostic.
	s.ensureOpen(where, "ScenarioChain."+string(kind))
	if token.generation != len(s.steps) {
		s.failfAt(where, "stale scenario chain token issued at %s; expected operation generation %d, found %d; assign every returned chain value", token.issued, token.generation, len(s.steps))
	}
	s.appendVisible(where, kind, text, fn)
	return s.chain(where)
}

func (s *ScenarioSpec[S]) chain(issued string) ScenarioChain[S] {
	return ScenarioChain[S]{spec: s, generation: len(s.steps), issued: issued}
}

func (s *ScenarioSpec[S]) appendVisible(where string, kind stepKind, text string, fn Step[S]) {
	if fn == nil {
		s.failfAt(where, "%s %q has a nil step", kind, text)
	}
	if strings.TrimSpace(text) == "" {
		s.failfAt(where, "%s text is blank", kind)
	}
	s.steps = append(s.steps, step[S]{kind: kind, text: text, source: where, fn: fn})
}

// ensureOpen panics unless s accepts declarations. A sealed scope panics
// rather than failing its owner because that test may already have finished.
func (s *ScenarioSpec[S]) ensureOpen(where, operation string) {
	if s == nil {
		panic("bdd: ScenarioSpec receiver is nil")
	}
	switch s.phase {
	case specUnbuilt:
		panic("bdd: ScenarioSpec receiver is zero")
	case specSealed:
		panic(fmt.Sprintf("bdd: %s: %s called on a sealed ScenarioSpec: its Scenario or Outline builder has returned; declare every operation inside that builder, not inside a step", where, operation))
	}
}

func (s *ScenarioSpec[S]) seal() {
	s.phase = specSealed
}

func (s *ScenarioSpec[S]) failfAt(where, format string, args ...any) {
	s.owner.Fatalf("bdd: "+where+": "+format, args...)
}
