package bdd

// chainCore is the private, owner-bound capability shared by the two fluent
// declaration tokens. It contains no operation slice: the owner callback owns
// validation and storage, while the expected generation detects stale or
// ignored tokens.
type chainCore[S any] struct {
	expected int
	issued   string
	advance  func(int, string, string, stepKind, string, Step[S]) int
}

func (c chainCore[S]) append(where string, kind stepKind, text string, fn Step[S]) chainCore[S] {
	if c.advance == nil {
		panic("bdd: chain token is zero")
	}
	next := c.advance(c.expected, c.issued, where, kind, text, fn)
	return chainCore[S]{expected: next, issued: where, advance: c.advance}
}

// backgroundChain is the private fluent capability returned by Suite.Background.
// It exposes only And and But.
type backgroundChain[S any] struct {
	core chainCore[S]
}

// And appends a Background continuation and returns the next token.
func (c backgroundChain[S]) And(text string, fn Step[S]) backgroundChain[S] {
	where := declarationSource()
	return backgroundChain[S]{core: c.core.append(where, kindAnd, text, fn)}
}

// But appends a Background continuation and returns the next token.
func (c backgroundChain[S]) But(text string, fn Step[S]) backgroundChain[S] {
	where := declarationSource()
	return backgroundChain[S]{core: c.core.append(where, kindBut, text, fn)}
}

// scenarioChain is the private fluent capability returned by a scenario's
// Given, When, or Then starter. Given, When, Then, And, and But are valid on
// this one token in flexible order.
type scenarioChain[S any] struct {
	core chainCore[S]
}

// Given appends a Given operation and returns the next token.
func (c scenarioChain[S]) Given(text string, fn Step[S]) scenarioChain[S] {
	where := declarationSource()
	return scenarioChain[S]{core: c.core.append(where, kindGiven, text, fn)}
}

// When appends a When operation and returns the next token.
func (c scenarioChain[S]) When(text string, fn Step[S]) scenarioChain[S] {
	where := declarationSource()
	return scenarioChain[S]{core: c.core.append(where, kindWhen, text, fn)}
}

// Then appends a Then operation and returns the next token.
func (c scenarioChain[S]) Then(text string, fn Step[S]) scenarioChain[S] {
	where := declarationSource()
	return scenarioChain[S]{core: c.core.append(where, kindThen, text, fn)}
}

// And appends an And operation and returns the next token.
func (c scenarioChain[S]) And(text string, fn Step[S]) scenarioChain[S] {
	where := declarationSource()
	return scenarioChain[S]{core: c.core.append(where, kindAnd, text, fn)}
}

// But appends a But operation and returns the next token.
func (c scenarioChain[S]) But(text string, fn Step[S]) scenarioChain[S] {
	where := declarationSource()
	return scenarioChain[S]{core: c.core.append(where, kindBut, text, fn)}
}
