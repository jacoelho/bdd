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
// Background operations, and steps run in that order. Error, Fatal, Skip, and
// panic from setup gate later operations while its cleanup remains standard
// testing behavior. Rules and Contexts organize the generated test name but
// do not add test nodes.
//
// Builders declare behavior only. Steps, setup, and hooks execute directly in
// the atomic scenario child; they must not create subtests or call Parallel by
// type-asserting testing.TB.
//
// The intended style is:
//
//   - narrative text stays in the scenario
//   - mechanics live in reusable typed Step functions
//   - each scenario receives fresh typed state
//   - ordinary Go assertions remain valid
package bdd
