package bdd

import "testing"

func TestOutlineRequiresExamples(t *testing.T) {
	var tbb fatalRecorder

	if hasOutlineExamples[int](&tbb, nil) {
		t.Fatal("empty examples unexpectedly passed")
	}
	if !tbb.failed {
		t.Fatal("empty examples did not fail")
	}
	if got := tbb.args[0]; got != "bdd: outline requires at least one example" {
		t.Fatalf("fatal message = %q; want outline requires examples", got)
	}
}

func TestOutlineNameIncludesKindTagsAndCleanName(t *testing.T) {
	name := outlineName(" auth / login \n attempts ", meta{tags: []string{"fast", "login-flow"}})
	want := "Scenario Outline: @fast @login-flow auth / login attempts"

	if name != want {
		t.Fatalf("outlineName() = %q; want %q", name, want)
	}
}

type fatalRecorder struct {
	failed bool
	args   []any
}

func (f *fatalRecorder) Helper() {}

func (f *fatalRecorder) Fatal(args ...any) {
	f.failed = true
	f.args = args
}
