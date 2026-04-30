package bdd

import (
	"slices"
	"testing"
)

type scope struct {
	t    *testing.T
	meta meta
}

// Suite is a typed BDD scope. It owns a state factory plus inherited
// backgrounds, hooks, and tags. Use Suite when several scenarios share the same
// World type.
type Suite[S any] struct {
	scope      *scope
	newState   StateFactory[S]
	background []step[S]
	before     []Step[S]
	tags       []string
}

// Feature creates a top-level BDD feature with a typed scenario state.
func Feature[S any](t *testing.T, name string, newState StateFactory[S], build func(*Suite[S]), opts ...Option) {
	t.Helper()
	m := meta{}.with(opts...)
	t.Run(groupName("Feature", name, m), func(t *testing.T) {
		if msg := m.skipMessage(); msg != "" {
			t.Skip(msg)
		}
		build(&Suite[S]{
			scope:    &scope{t: t, meta: m},
			newState: newState,
		})
	})
}

// Rule creates a nested typed rule group and carries inherited backgrounds,
// hooks, and tags into the group.
func (s *Suite[S]) Rule(name string, build func(*Suite[S]), opts ...Option) {
	s.group("Rule", name, build, opts...)
}

// Context creates a nested typed context group and carries inherited
// backgrounds, hooks, and tags into the group.
func (s *Suite[S]) Context(name string, build func(*Suite[S]), opts ...Option) {
	s.group("Context", name, build, opts...)
}

// Scenario creates a scenario using the suite's state factory and inherited
// backgrounds, hooks, and tags.
func (s *Suite[S]) Scenario(name string, build func(*ScenarioSpec[S]), opts ...Option) {
	s.scope.t.Helper()
	spec := s.newSpec()
	build(spec)
	runScenario(s.scope, name, s.newState, spec, opts...)
}

// Background adds a visible setup step inherited by scenarios in this suite and
// nested typed groups.
func (s *Suite[S]) Background(text string, fn Step[S]) {
	s.background = append(s.background, newStep(kindBackground, text, fn))
}

// Before adds a hidden setup hook inherited by scenarios in this suite and
// nested typed groups.
func (s *Suite[S]) Before(fn Step[S]) {
	s.before = append(s.before, fn)
}

// Tags adds tags inherited by scenarios in this suite and nested typed groups.
func (s *Suite[S]) Tags(tags ...string) {
	s.tags = appendTags(s.tags, tags...)
}

func (s *Suite[S]) newSpec() *ScenarioSpec[S] {
	return &ScenarioSpec[S]{
		background: slices.Clone(s.background),
		before:     slices.Clone(s.before),
		tags:       slices.Clone(s.tags),
	}
}

func (s *Suite[S]) group(kind, name string, build func(*Suite[S]), opts ...Option) {
	s.scope.t.Helper()
	m := s.scope.meta.with(opts...)
	s.scope.t.Run(groupName(kind, name, m), func(t *testing.T) {
		if msg := m.skipMessage(); msg != "" {
			t.Skip(msg)
		}
		child := &Suite[S]{
			scope:      &scope{t: t, meta: m},
			newState:   s.newState,
			background: slices.Clone(s.background),
			before:     slices.Clone(s.before),
			tags:       slices.Clone(s.tags),
		}
		build(child)
	})
}
