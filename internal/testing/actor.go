package testing

import (
	"context"
	"fmt"
	"sync"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
)

// testActor implements the Actor interface with TestContext integration.
// This actor automatically handles errors through the embedded TestContext,
// providing a seamless testing experience without manual error checking.
//
// Key Features:
//   - Automatic error propagation to test framework
//   - Thread-safe operations
//   - Integrated reporting capabilities
//   - Support for all standard Actor methods
type testActor struct {
	name        string                       // Actor name for reporting
	abilities   []abilities.Ability          // Actor abilities
	testContext TestContext                  // Embedded test context for error handling
	reporter    *reporting.TestRunnerAdapter // Integrated reporter for activity tracking
	ctx         context.Context              // Context for cancellation and timeout
	facts       []core.Fact                  // Successfully set up actor facts
	factsClosed bool                         // Whether fact registration is terminal
	factsMutex  sync.Mutex                   // Serializes fact setup and terminal teardown
	mutex       sync.RWMutex                 // Mutex for thread-safe operations
}

// Name returns the actor's name
func (ta *testActor) Name() string {
	return ta.name
}

// Context returns the actor's context for cancellation and timeout
func (ta *testActor) Context() context.Context {
	return ta.ctx
}

// WhoCan adds abilities to the actor and returns the same actor instance for chaining.
// This method is thread-safe and can be called multiple times.
//
// Example:
//
//	actor := test.ActorCalled("APIUser").
//		WhoCan(api.CallAnApiAt("https://api.example.com")).
//		WhoCan(db.ConnectToDatabase("postgres://localhost/test"))
//
// Parameters:
//
//	abilities - List of abilities to add to the actor
//
// Returns:
//
//	The same actor instance with added abilities for method chaining
func (ta *testActor) WhoCan(abilities ...abilities.Ability) core.Actor {
	ta.mutex.Lock()
	defer ta.mutex.Unlock()

	ta.abilities = append(ta.abilities, abilities...)
	return ta
}

// AbilityTo returns the specified ability
func (ta *testActor) AbilityTo(abilityType abilities.Ability) (abilities.Ability, error) {
	ta.mutex.RLock()
	defer ta.mutex.RUnlock()

	for _, ability := range ta.abilities {
		if abilityMatchesType(ability, abilityType) {
			return ability, nil
		}
	}

	abName := core.AbilityName(abilityType)
	return nil, fmt.Errorf("actor '%s' can't %s. Did you give them the ability?", ta.name, abName)
}

// Has declares facts about the actor for the current test.
func (ta *testActor) Has(facts ...core.Fact) {
	ta.factsMutex.Lock()
	defer ta.factsMutex.Unlock()

	if ta.factsClosed {
		panic("verity: Actor.Has called after Shutdown")
	}

	for _, fact := range facts {
		if err := fact.Setup(ta.ctx, ta); err != nil {
			ta.testContext.Errorf("Fact %q setup failed for actor %q: %v", fact.Description(), ta.name, err)
			ta.testContext.FailNow()
			return
		}
		ta.facts = append(ta.facts, fact)
	}
}

func (ta *testActor) teardownFacts() {
	ta.factsMutex.Lock()
	ta.factsClosed = true
	facts := ta.facts
	ta.facts = nil
	ta.factsMutex.Unlock()

	for i := len(facts) - 1; i >= 0; i-- {
		fact := facts[i]
		if err := fact.Teardown(ta.ctx, ta); err != nil {
			ta.testContext.Errorf("Fact %q teardown failed for actor %q: %v", fact.Description(), ta.name, err)
		}
	}
}

// AttemptsTo executes activities and automatically handles errors through TestContext.
// No manual error checking is required: failures are recorded on TestContext, and
// FailFast activities also call FailNow.
//
// Example:
//
//	actor.AttemptsTo(
//		api.SendGetRequest("/users"),
//		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
//	)
//
// Parameters:
//
//	activities - List of activities to execute in order
//
// This method automatically handles different failure modes:
//   - FailFast: Marks the test failed and stops execution immediately
//   - ErrorButContinue: Marks the test failed via Errorf and continues
//   - Ignore: Logs the error via Logf and continues without failing the test
func (ta *testActor) AttemptsTo(activities ...core.Activity) {
	for _, activity := range activities {
		err := ta.PerformActivity(ta.ctx, activity)

		if err != nil {
			failureMode := activity.FailureMode()
			switch failureMode {
			case core.FailFast:
				ta.testContext.Errorf("Critical activity error '%s' failed: %v", activity.Description(), err)
				ta.testContext.FailNow()
				return
			case core.ErrorButContinue:
				ta.testContext.Errorf("Non-critical activity error '%s' failed: %v", activity.Description(), err)
			case core.Ignore:
				ta.testContext.Logf("Ignore activity error '%s' failed: %v", activity.Description(), err)
			}
		}
	}
}

// PerformActivity executes a single activity through the actor reporting pipeline
// and returns the underlying execution error to the caller.
func (ta *testActor) PerformActivity(ctx context.Context, activity core.Activity) error {
	var tracker *reporting.ActivityTracker
	if ta.reporter != nil {
		tracker = reporting.NewActivityTrackerWithActor(ta.reporter.GetReporter(), activity.Description(), ta.name)
		tracker.Start()
	}

	err := activity.PerformAs(ctx, ta)

	if tracker != nil {
		tracker.Finish(err)
	}

	return err
}
