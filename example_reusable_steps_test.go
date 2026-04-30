package bdd_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

func TestReusableSteps(t *testing.T) {
	bdd.Feature(t, "API health", bdd.NewNoState, func(f *bdd.Suite[bdd.NoState]) {
		f.Scenario("service responds", func(s *bdd.ScenarioSpec[bdd.NoState]) {
			s.Given("the service is running", serviceIsRunning())
			s.When("the health endpoint is requested", healthEndpointRequested())
			s.Then("the response is healthy", responseIsHealthy())
			s.And("the status is reported", statusIsReported())
		})
	})
}

func serviceIsRunning() bdd.Step[bdd.NoState] {
	return bdd.Noop[bdd.NoState]()
}

func healthEndpointRequested() bdd.Step[bdd.NoState] {
	return bdd.Noop[bdd.NoState]()
}

func responseIsHealthy() bdd.Step[bdd.NoState] {
	return bdd.Noop[bdd.NoState]()
}

func statusIsReported() bdd.Step[bdd.NoState] {
	return bdd.Noop[bdd.NoState]()
}
