package bdd

// A chain is a capability to append the next clause of one narrative. It
// records its owner's operation count when it was issued; the owner accepts a
// chain only while that count is unchanged, which rejects stale and ignored
// chains. The owner, not the chain, stores the operations.

// BackgroundChain continues the Background of one Suite with And and But. It
// is returned by Suite.Background; the zero value panics. Reassign every
// returned value: a chain that is not the latest fails its Suite's test.
type BackgroundChain[S any] struct {
	suite      *Suite[S]
	generation int
	issued     string
}

// And appends a Background continuation and returns the next chain.
func (c BackgroundChain[S]) And(text string, fn Step[S]) BackgroundChain[S] {
	return c.append(declarationSource(), kindAnd, text, fn)
}

// But appends a Background continuation and returns the next chain.
func (c BackgroundChain[S]) But(text string, fn Step[S]) BackgroundChain[S] {
	return c.append(declarationSource(), kindBut, text, fn)
}

func (c BackgroundChain[S]) append(where string, kind stepKind, text string, fn Step[S]) BackgroundChain[S] {
	if c.suite == nil {
		panic("bdd: chain token is zero")
	}
	return c.suite.continueBackground(c, where, kind, text, fn)
}

// ScenarioChain continues the narrative of one scenario. It is returned by
// ScenarioSpec.Given, When, or Then; the zero value panics. Given, When, Then,
// And, and But are valid in flexible order. Reassign every returned value: a
// chain that is not the latest fails the scenario.
type ScenarioChain[S any] struct {
	spec       *ScenarioSpec[S]
	generation int
	issued     string
}

// Given appends a Given operation and returns the next chain.
func (c ScenarioChain[S]) Given(text string, fn Step[S]) ScenarioChain[S] {
	return c.append(declarationSource(), kindGiven, text, fn)
}

// When appends a When operation and returns the next chain.
func (c ScenarioChain[S]) When(text string, fn Step[S]) ScenarioChain[S] {
	return c.append(declarationSource(), kindWhen, text, fn)
}

// Then appends a Then operation and returns the next chain.
func (c ScenarioChain[S]) Then(text string, fn Step[S]) ScenarioChain[S] {
	return c.append(declarationSource(), kindThen, text, fn)
}

// And appends an And operation and returns the next chain.
func (c ScenarioChain[S]) And(text string, fn Step[S]) ScenarioChain[S] {
	return c.append(declarationSource(), kindAnd, text, fn)
}

// But appends a But operation and returns the next chain.
func (c ScenarioChain[S]) But(text string, fn Step[S]) ScenarioChain[S] {
	return c.append(declarationSource(), kindBut, text, fn)
}

func (c ScenarioChain[S]) append(where string, kind stepKind, text string, fn Step[S]) ScenarioChain[S] {
	if c.spec == nil {
		panic("bdd: chain token is zero")
	}
	return c.spec.continueNarrative(c, where, kind, text, fn)
}
