package bdd

import (
	"slices"
)

// Example is one row in a scenario outline.
type Example[E any] struct {
	Name    string
	Value   E
	Options []Option
}

// Case constructs one scenario outline example. The option slice is copied;
// the row value is intentionally shallow-copied.
func Case[E any](name string, value E, opts ...Option) Example[E] {
	return Example[E]{Name: name, Value: value, Options: slices.Clone(opts)}
}

// Outline expands table-driven examples into atomic direct Feature children.
// Go 1.27 permits the row type parameter on this generic method, allowing the
// row type to be inferred from examples at the call site.
func (s *Suite[S]) Outline[E any](name string, examples []Example[E], build func(E, *ScenarioSpec[S]), opts ...Option) {
	where := declarationSource()
	s.declareChild()
	if build == nil {
		s.failfAt(where, "Outline %q has a nil builder", name)
	}
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
	outlineCrumbs := append(slices.Clone(s.breadcrumbs), crumb{kind: "outline", slug: slug, raw: name})
	for _, original := range examples {
		example := Example[E]{Name: original.Name, Value: original.Value, Options: slices.Clone(original.Options)}
		exampleSlug, err := slugName(example.Name)
		if err != nil {
			s.failfAt(where, "invalid Outline example name %q: %v", example.Name, err)
		}
		// An explicit example disposition overrides the outline disposition;
		// otherwise the inherited outline disposition remains in force.
		var exampleMeta meta
		if hasDispositionOption(example.Options) {
			exampleMeta, err = (meta{tags: slices.Clone(outlineMeta.tags)}).with(example.Options...)
			if err != nil {
				s.failfAt(where, "invalid Outline example %q options: %v", example.Name, err)
			}
		} else {
			exampleMeta, err = outlineMeta.with(example.Options...)
			if err != nil {
				s.failfAt(where, "invalid Outline example %q options: %v", example.Name, err)
			}
		}
		crumbs := append(slices.Clone(outlineCrumbs), crumb{kind: "example", slug: exampleSlug, raw: example.Name})
		exampleBuild := build
		exampleValue := example.Value
		s.addLeaf(&leaf[S]{
			name:       s.leafName(crumbs, exampleMeta.tags),
			key:        breadcrumbKey(crumbs),
			breadcrumb: rawBreadcrumb(crumbs),
			diagnostic: dispositionDiagnostic(rawBreadcrumb(crumbs), exampleMeta),
			meta:       exampleMeta,
			before:     slices.Clone(s.before),
			background: slices.Clone(s.background),
			build: func(spec *ScenarioSpec[S]) {
				exampleBuild(exampleValue, spec)
			},
			state:  s.feature,
			source: where,
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
