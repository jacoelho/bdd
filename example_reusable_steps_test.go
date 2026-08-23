package bdd_test

import (
	"testing"

	"github.com/jacoelho/bdd"
)

type healthWorld struct {
	running bool
	healthy bool
	status  int
}

func TestReusableSteps(t *testing.T) {
	bdd.Feature(t, "API health", func(testing.TB) *healthWorld {
		return &healthWorld{}
	}, func(f *bdd.Suite[*healthWorld]) {
		f.Scenario("service responds", func(s *bdd.ScenarioSpec[*healthWorld]) {
			s.Given("the service is running", serviceIsRunning()).
				When("the health endpoint is requested", healthEndpointRequested()).
				Then("the response is healthy", responseIsHealthy()).
				And("the status is reported", statusIsReported())
		})
	})
}

func serviceIsRunning() bdd.Step[*healthWorld] {
	return func(_ testing.TB, w *healthWorld) {
		w.running = true
	}
}

func healthEndpointRequested() bdd.Step[*healthWorld] {
	return func(t testing.TB, w *healthWorld) {
		if !w.running {
			t.Fatal("health endpoint requested before service started")
		}
		w.healthy = true
		w.status = 200
	}
}

func responseIsHealthy() bdd.Step[*healthWorld] {
	return func(t testing.TB, w *healthWorld) {
		if !w.healthy {
			t.Fatal("response is not healthy")
		}
	}
}

func statusIsReported() bdd.Step[*healthWorld] {
	return func(t testing.TB, w *healthWorld) {
		if w.status != 200 {
			t.Fatalf("status = %d; want 200", w.status)
		}
	}
}
