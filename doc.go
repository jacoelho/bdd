// Package bdd is a small Go-native behavior-driven development layer over the
// standard testing package.
//
// Features are declared inline in ordinary Go tests. A Feature is one
// grouping subtest; each Scenario or outline example is one direct child of
// that group and all of its steps run in that child. Consequently, scenarios
// are atomic test units while every step still receives testing.TB and keeps
// the standard testing cleanup, context, and assertion behavior.
//
// Each selected active scenario receives fresh state exactly once from the
// setup callback. Setup runs after a declared Parallel resumes, then hooks,
// Background operations, and steps run in that order. Error, Fatal, Skip, or
// panic in setup stops the later operations; cleanup registered by setup still
// runs. Recursively nested Rules organize the generated test name
// but do not add test nodes.
//
// Builders declare behavior only. Steps, setup, and hooks execute directly in
// the atomic scenario child; they must not create subtests or call Parallel by
// type-asserting testing.TB.
//
// The intended style is:
//
//   - narrative text stays in one linear scenario chain
//   - mechanics live in reusable typed Step functions
//   - each scenario receives fresh typed state
//   - ordinary Go assertions remain valid
//
// # Declaration order
//
// Each builder accepts its declarations in one order. A declaration out of
// order fails the owning test with the call site and the legal order:
//
//	Suite (Feature or Rule builder):
//	  Before*  ->  [Background [.And | .But]*]  ->  (Scenario | Outline | Rule)+
//
//	ScenarioSpec (Scenario or Outline builder):
//	  (Parallel | Before)*  ->  (Given | When | Then) [.Given | .When | .Then | .And | .But]*
//
// A Suite declares at most one Background; a scenario declares Parallel at
// most once and has exactly one starter. While a Rule builder runs, its
// enclosing Suites are suspended: declare on the builder's own *Suite
// parameter. Every scope is sealed when its builder returns; using it
// afterwards panics, naming the call site, the call, and the fix.
//
// # Failure output
//
// Log lines of a running scenario are attributed to the Feature call site. A
// scenario stopped by Error, Fatal, Skip, or panic during setup, a hook, or an
// operation logs the operation it stopped in, where that operation was
// declared, and the operations that did not run:
//
//	bdd: scenario stopped in When "the customer adds two books" declared at /src/cart_test.go:27; not run: Then "the total is updated"
//
// Declaration errors have the form "bdd: <file>:<line>: <problem>".
//
// # Chains
//
// Given, When, and Then return a ScenarioChain with Given, When, Then, And,
// and But methods; Background returns a BackgroundChain with And and But.
// Helpers may accept and return a chain. Reassign every returned chain.
// Copying a chain is harmless. While its builder is active, using a chain that
// is not the latest fails the owning test at that call. Any chain used after
// its builder returns panics with the sealed-scope diagnostic.
package bdd
