package bdd

import (
	"slices"
)

// Example is one opaque row in a scenario outline. Construct it with Case;
// zero Examples and struct literals without Case source metadata are rejected
// by Outline.
type Example[E any] struct {
	name    string
	value   E
	options []Option
	source  string
}

// Case constructs one scenario outline example. Case is the required
// constructor because it captures the row declaration source. The option
// slice is copied; the row value is intentionally shallow-copied.
func Case[E any](name string, value E, opts ...Option) Example[E] {
	return Example[E]{name: name, value: value, options: slices.Clone(opts), source: declarationSource()}
}

func (e Example[E]) clone() Example[E] {
	e.options = slices.Clone(e.options)
	return e
}

// Outline expands table-driven examples into atomic direct Feature children.
// Go 1.27 permits the row type parameter on this generic method, allowing the
// row type to be inferred from examples at the call site. A nil builder is
// legal only when every effective Case is disposed with Skip or Pending; an
// Outline containing an active Case requires a builder.
func (s *Suite[S]) Outline[E any](name string, examples []Example[E], build func(E, *ScenarioSpec[S]), opts ...Option) {
	where := declarationSource()
	s.declare(where, "Suite.Outline", declareChild)
	if len(examples) == 0 {
		s.failfAt(where, "Outline %q requires at least one example", name)
	}
	slug, err := slugName(name)
	if err != nil {
		s.failfAt(where, "invalid Outline name %q: %v", name, err)
	}
	outlineMeta, err := s.nodeMeta(opts...)
	if err != nil {
		s.failfAt(where, "invalid Outline %q options: %v", name, err)
	}
	outlineCrumbs := s.crumbs("outline", slug, name)

	// Every example is validated before any leaf is registered.
	type preparedExample struct {
		example Example[E]
		meta    meta
		crumbs  []crumb
	}
	prepared := make([]preparedExample, 0, len(examples))
	for _, original := range examples {
		if original.source == "" {
			s.failfAt(where, "Outline %q contains a zero Example; use bdd.Case", name)
		}
		example := original.clone()
		exampleSlug, err := slugName(example.name)
		if err != nil {
			s.failfAt(example.source, "invalid Outline example name %q: %v", example.name, err)
		}
		// An explicit example disposition overrides the outline disposition;
		// otherwise the inherited outline disposition remains in force.
		inherited := outlineMeta
		if hasDispositionOption(example.options) {
			inherited = meta{tags: outlineMeta.tags}
		}
		exampleMeta, err := inherited.with(example.options...)
		if err != nil {
			s.failfAt(example.source, "invalid Outline example %q options: %v", example.name, err)
		}
		crumbs := append(slices.Clone(outlineCrumbs), crumb{kind: "example", slug: exampleSlug, raw: example.name})
		prepared = append(prepared, preparedExample{example: example, meta: exampleMeta, crumbs: crumbs})
	}
	if build == nil {
		for _, item := range prepared {
			if !item.meta.hasDisposition() {
				s.failfAt(where, "Outline %q has a nil builder for active example %q", name, item.example.name)
			}
		}
	}
	for _, item := range prepared {
		s.addLeaf(item.example.source, item.crumbs, item.meta, func(spec *ScenarioSpec[S]) {
			build(item.example.value, spec)
		})
	}
}

func hasDispositionOption(opts []Option) bool {
	for _, opt := range opts {
		if opt.kind == optionSkip || opt.kind == optionPending {
			return true
		}
	}
	return false
}
