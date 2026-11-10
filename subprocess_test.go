package bdd

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// runSelf re-executes the test binary with env set, so a test can observe the
// output and exit status of a failing, skipped, or filtered run of itself.
// Unless args carry their own -test.run, only the calling top-level test runs.
func runSelf(t *testing.T, env string, args ...string) ([]byte, error) {
	t.Helper()
	if !slices.ContainsFunc(args, func(arg string) bool { return strings.HasPrefix(arg, "-test.run=") }) {
		name, _, _ := strings.Cut(t.Name(), "/")
		args = append(args, "-test.run=^"+name+"$")
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), env)
	return cmd.CombinedOutput()
}
