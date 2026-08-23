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
// testing behavior. Recursively nested Rules organize the generated test name
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
// Given, When, and Then return a private chain capability. Its And, But,
// Given, When, and Then methods are callable by consumers even though the
// concrete return type is private. Reassign every returned token. Copying a
// token is harmless; using a stale token while its builder is active fails at
// the continuation call through the owner. Any token used after its builder
// returns panics with the owner-sealed diagnostic. Background returns the
// corresponding And/But-only capability.
package bdd
