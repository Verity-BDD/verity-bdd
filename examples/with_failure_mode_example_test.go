package examples

import (
	"context"
	"fmt"
	"io"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/wait"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
	"github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
)

type failureModeExampleContext struct {
	cleanups []func()
	errors   int
	logs     int
}

func (c *failureModeExampleContext) Cleanup(cleanup func()) { c.cleanups = append(c.cleanups, cleanup) }
func (c *failureModeExampleContext) Errorf(string, ...interface{}) {
	c.errors++
}
func (c *failureModeExampleContext) Failed() bool                { return c.errors > 0 }
func (c *failureModeExampleContext) FailNow()                    { panic("unexpected FailNow") }
func (c *failureModeExampleContext) Helper()                     {}
func (c *failureModeExampleContext) Logf(string, ...interface{}) { c.logs++ }
func (c *failureModeExampleContext) Name() string                { return "ExampleWithFailureMode" }

func (c *failureModeExampleContext) RunCleanups() {
	for i := len(c.cleanups) - 1; i >= 0; i-- {
		c.cleanups[i]()
	}
}

// Example_withFailureMode shows failed wait and ensure activities that continue.
func Example_withFailureMode() {
	testContext := &failureModeExampleContext{}
	reporter := console_reporter.NewConsoleReporter()
	reporter.SetOutput(io.Discard)
	test := verity.NewVerityTest(testContext, verity.Scene{Reporter: reporter})
	defer testContext.RunCleanups()

	continued := 0
	actor := test.ActorCalled("Ava")
	actor.AttemptsTo(
		wait.Until(answerable.ValueOf("not ready"), expectations.Equals("ready")).
			WithFailureMode(verity.NonCritical()).
			For(time.Millisecond).
			CheckingEvery(time.Millisecond),
		verity.Do("records that the wait continued", func(context.Context, verity.Actor) error {
			continued++
			return nil
		}),
	)
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf("not ready"), expectations.Equals("ready")).
			WithFailureMode(verity.Optional()),
		verity.Do("records that the assertion continued", func(context.Context, verity.Actor) error {
			continued++
			return nil
		}),
	)

	fmt.Printf("continued=%d errors=%d logs=%d\n", continued, testContext.errors, testContext.logs)
	// Output:
	// continued=2 errors=1 logs=1
}
