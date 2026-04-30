package bdd

import "testing"

// StateFactory creates a fresh scenario state. In most suites S is a pointer to
// a domain-specific World type, for example *checkoutbdd.World.
type StateFactory[S any] func(t testing.TB) S

// Step is a reusable operation or assertion against a scenario state.
type Step[S any] func(t testing.TB, state S)

const (
	kindBackground stepKind = "Background"
	kindGiven      stepKind = "Given"
	kindWhen       stepKind = "When"
	kindThen       stepKind = "Then"
	kindAnd        stepKind = "And"
	kindBut        stepKind = "But"
	kindPending    stepKind = "Pending"
)

type stepKind string

// NoState is useful for scenarios that do not need shared state.
type NoState struct{}

// NewNoState returns an empty scenario state.
func NewNoState(t testing.TB) NoState {
	t.Helper()
	return NoState{}
}

// Noop is a step that deliberately does nothing.
func Noop[S any]() Step[S] {
	return func(t testing.TB, _ S) {
		t.Helper()
	}
}

// Compose returns one step from many smaller steps. It is useful for reusable
// domain fixtures such as LoggedInPremiumCustomer().
func Compose[S any](steps ...Step[S]) Step[S] {
	return func(t testing.TB, state S) {
		t.Helper()
		for _, step := range steps {
			step(t, state)
			if t.Failed() || t.Skipped() {
				return
			}
		}
	}
}
