package bdd

import (
	"slices"
	"testing"
)

// Step is a reusable operation or assertion against a scenario state.
type Step[S any] func(t testing.TB, state S)

const (
	kindBackground stepKind = "Background"
	kindGiven      stepKind = "Given"
	kindWhen       stepKind = "When"
	kindThen       stepKind = "Then"
	kindAnd        stepKind = "And"
	kindBut        stepKind = "But"
)

type stepKind string

// NoState is useful for scenarios that do not need shared state.
type NoState struct{}

// NewNoState returns an empty scenario state.
func NewNoState(_ testing.TB) NoState {
	return NoState{}
}

// Compose returns one step from many smaller steps. It is useful for reusable
// domain fixtures such as LoggedInPremiumCustomer().
func Compose[S any](steps ...Step[S]) Step[S] {
	steps = slices.Clone(steps)
	return func(t testing.TB, state S) {
		t.Helper()
		if len(steps) == 0 {
			t.Fatal("bdd: Compose requires at least one step")
		}
		for _, step := range steps {
			if step == nil {
				t.Fatal("bdd: Compose received a nil step")
			}
			step(t, state)
			if t.Failed() || t.Skipped() {
				return
			}
		}
	}
}
