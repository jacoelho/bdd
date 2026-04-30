package bdd

import "testing"

// Example is one row in a scenario outline.
type Example[E any] struct {
	Name  string
	Value E
	Opts  []Option
}

// Case constructs one scenario outline example.
func Case[E any](name string, value E, opts ...Option) Example[E] {
	return Example[E]{Name: name, Value: value, Opts: opts}
}

// Outline expands table-driven examples into scenarios under a typed Suite.
// This is a top-level function because Go methods cannot introduce an
// additional type parameter for the example row type.
func Outline[S any, E any](
	suite *Suite[S],
	name string,
	examples []Example[E],
	build func(example E, spec *ScenarioSpec[S]),
	opts ...Option,
) {
	suite.scope.t.Helper()
	m := suite.scope.meta.withTags(suite.tags...).with(opts...)
	suite.scope.t.Run(outlineName(name, m), func(t *testing.T) {
		if msg := m.skipMessage(); msg != "" {
			t.Skip(msg)
		}
		_ = examples[0]

		outlineScope := &scope{t: t, meta: m}
		for _, example := range examples {
			spec := suite.newSpec()
			build(example.Value, spec)
			runScenario(outlineScope, example.Name, suite.newState, spec, example.Opts...)
		}
	})
}
