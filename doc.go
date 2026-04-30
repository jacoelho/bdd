// Package bdd is a small Go-native behavior-driven development layer over
// the standard testing package.
//
// It keeps Go's execution model: tests still run with `go test`, features,
// rules, contexts, scenarios, and steps are native subtests, and every step
// receives testing.TB. The BDD layer adds structure and vocabulary: Feature,
// Rule, Context, Scenario, Background, Given, When, Then, And, But, reusable
// typed steps, outlines, and typed state factories.
//
// Step bodies usually do not need to call t.Helper before reporting failures.
// Use t.Helper in reusable assertion helpers when you want Go to report the
// helper's caller instead of the helper itself.
//
// The intended style is:
//
//   - narrative text stays in the scenario
//   - mechanics live in reusable typed Step functions
//   - each scenario receives a fresh typed World object
//   - ordinary Go assertions remain valid
package bdd
