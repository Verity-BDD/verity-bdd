package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	"github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"
)

// ReporterProvider provides access to reporter adapter
type ReporterProvider interface {
	// GetReporterAdapter returns the test runner adapter for reporting
	GetReporterAdapter() *reporting.TestRunnerAdapter
}

// DefaultAbilityFactory creates a default ability for the actor name.
type DefaultAbilityFactory func(actorName string) abilities.Ability

// Scene configures VerityTest runtime behavior.
type Scene struct {
	Context          context.Context
	Reporter         reporting.Reporter
	DefaultAbilities []DefaultAbilityFactory
}

// VerityTest manages the lifecycle of test actors and provides the TestContext API.
// This interface serves as the main entry point for using the simplified testing approach.
//
// Lifecycle Management:
//  1. Create test instance with NewVerityTest() or NewVerityTestWithReporter()
//  2. Create actors using ActorCalled()
//  3. Execute test activities
//  4. Call Shutdown() to clean up resources (typically via defer)
//
// Thread Safety:
//
//	All VerityTest methods are thread-safe. Multiple goroutines can safely
//	create and use actors from the same test instance.
type VerityTest interface {
	// TestContext returns the embedded testing.TB interface.
	// This method provides access to the underlying testing framework.
	TestContext() TestContext

	// Context returns the context associated with this test.
	// The context is passed to all activities and questions.
	Context() context.Context

	// ActorCalled creates a new test-aware actor with the specified name.
	// The actor is automatically configured with TestContext error handling.
	//
	// Parameters:
	//	name - Human-readable name for the actor (used in reporting)
	//
	// Returns:
	//	An Actor instance configured for automatic error handling
	ActorCalled(name string) core.Actor

	// Actors returns a fresh, non-nil snapshot of the registered actors, sorted
	// by Name in ascending case-sensitive lexical order. The snapshot contains
	// the original actor instances; modifying the slice does not change the
	// registry. After Shutdown, Actors returns an empty snapshot.
	Actors() []core.Actor

	// Shutdown cleans up resources and finalizes the test.
	// This method should be called via defer after creating the test instance.
	// Failure to call Shutdown() may result in resource leaks.
	//
	// Example:
	//	test := verity.NewVerityTest(t, verity.Scene{})
	//
	// Side effects:
	//	- Flushes any pending reports
	//	- Cleans up actor resources
	//	- Finalizes test metrics
	Shutdown()

	// GetReporterAdapter returns the test runner adapter for reporting
	GetReporterAdapter() *reporting.TestRunnerAdapter
}

// Test Lifecycle Examples:
//
// Basic Test Structure:
//
//	func TestAPIEndpoints(t *testing.T) {
//		test := verity.NewVerityTest(t, verity.Scene{})
//
//		apiBaseURL := "http://127.0.0.1:8080"
//		actor := test.ActorCalled("APITester").WhoCan(
//			api.CallAnApiAt(apiBaseURL),
//		)
//
//		actor.AttemptsTo(
//			api.SendGetRequest("/posts"),
//			ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
//			ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("title")),
//		)
//	}
//
// Test with Custom Reporter:
//
//	func TestWithCustomReporting(t *testing.T) {
//		reporter := custom.NewJSONReporter()
//		test := verity.NewVerityTestWithReporter(t, reporter)
//
//		actor := test.ActorCalled("ReportedUser").WhoCan(api.CallAnApiAt(apiURL))
//		actor.AttemptsTo(api.SendGetRequest("/health"))
//	}

// testResult implements the TestResult interface
type testResult struct {
	name        string
	status      reporting.Status
	duration    time.Duration
	err         error
	attachments []reporting.Attachment
}

// Name returns the test name
func (tr *testResult) Name() string {
	return tr.name
}

// Status returns the test status
func (tr *testResult) Status() reporting.Status {
	return tr.status
}

// Duration returns the test duration in seconds
func (tr *testResult) Duration() float64 {
	return tr.duration.Seconds()
}

// Error returns the test error, if any
func (tr *testResult) Error() error {
	return tr.err
}

// Attachments returns any attachments associated with the result
func (tr *testResult) Attachments() []reporting.Attachment {
	return tr.attachments
}

// verityTest implements VerityTest
type verityTest struct {
	testCtx                 TestContext
	ctx                     context.Context
	actors                  map[string]core.Actor
	mutex                   sync.RWMutex
	adapter                 *reporting.TestRunnerAdapter
	startTime               time.Time
	testName                string
	shutdown                bool
	defaultAbilityFactories []DefaultAbilityFactory
}

// NewVerityTest creates a new VerityTest instance
func NewVerityTest(t TestContext, scene Scene) VerityTest {
	t.Helper()

	resolved := Scene{
		Context:  context.Background(),
		Reporter: console_reporter.NewConsoleReporter(),
	}

	if scene.Context != nil {
		resolved.Context = scene.Context
	}
	if scene.Reporter != nil {
		resolved.Reporter = scene.Reporter
	}
	resolved.DefaultAbilities = append(resolved.DefaultAbilities, scene.DefaultAbilities...)

	var adapter *reporting.TestRunnerAdapter
	if resolved.Reporter != nil {
		adapter = reporting.NewTestRunnerAdapter(resolved.Reporter)
	}

	testName := t.Name()

	// Notify reporter that test is starting
	if resolved.Reporter != nil {
		resolved.Reporter.OnTestStart(testName)
	}

	st := &verityTest{
		testCtx:                 t,
		ctx:                     resolved.Context,
		actors:                  make(map[string]core.Actor),
		adapter:                 adapter,
		startTime:               time.Now(),
		testName:                testName,
		defaultAbilityFactories: append([]DefaultAbilityFactory(nil), resolved.DefaultAbilities...),
	}

	t.Cleanup(func() { t.Helper(); st.Shutdown() })
	return st
}

// NewVerityTest creates a new VerityTest instance
func NewVerityTestWithContext(ctx context.Context, t TestContext) VerityTest {
	return NewVerityTest(t, Scene{
		Context:  ctx,
		Reporter: console_reporter.NewConsoleReporter(),
	})
}

// NewVerityTestWithReporter creates a new VerityTest instance with a reporter
func NewVerityTestWithReporter(ctx context.Context, t TestContext, reporter reporting.Reporter) VerityTest {
	return NewVerityTest(t, Scene{
		Context:  ctx,
		Reporter: reporter,
	})
}

// ActorCalled returns an actor with the given name
func (st *verityTest) ActorCalled(name string) core.Actor {
	st.mutex.RLock()
	actor, exists := st.actors[name]
	st.mutex.RUnlock()

	if exists {
		return actor
	}

	st.mutex.Lock()
	defer st.mutex.Unlock()

	// Double-check after acquiring write lock
	if actor, exists := st.actors[name]; exists {
		return actor
	}

	createdActor := &testActor{
		name:        name,
		abilities:   make([]abilities.Ability, 0),
		testContext: st.testCtx,
		reporter:    st.adapter,
		ctx:         st.ctx,
	}

	for _, factory := range st.defaultAbilityFactories {
		if factory == nil {
			continue
		}

		ability := factory(name)
		if ability == nil {
			continue
		}

		createdActor.WhoCan(ability)
	}

	actor = createdActor

	st.actors[name] = actor
	return actor
}

// Actors returns a snapshot of the actors registered with this test.
func (st *verityTest) Actors() []core.Actor {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	if st.shutdown {
		return make([]core.Actor, 0)
	}

	actors := make([]core.Actor, 0, len(st.actors))
	for _, actor := range st.actors {
		actors = append(actors, actor)
	}
	sort.Slice(actors, func(i, j int) bool {
		return actors[i].Name() < actors[j].Name()
	})
	return actors
}

// TestContext returns the embedded testing.TB interface.
// This method provides access to the underlying testing framework.
func (st *verityTest) TestContext() TestContext {
	return st.testCtx
}

// Context returns the context associated with this test
func (st *verityTest) Context() context.Context {
	return st.ctx
}

// GetReporterAdapter returns the test runner adapter for reporting
func (st *verityTest) GetReporterAdapter() *reporting.TestRunnerAdapter {
	return st.adapter
}

// Shutdown cleans up resources
func (st *verityTest) Shutdown() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	if st.shutdown {
		return
	}

	// Create test result
	duration := time.Since(st.startTime)
	status := reporting.StatusPassed
	var testErr error
	attachments := make([]reporting.Attachment, 0)

	if st.testCtx.Failed() {
		status = reporting.StatusFailed
		testErr = fmt.Errorf("test failed")
	}

	noteDump := st.collectNotes()
	if noteDump != nil {
		content, err := json.Marshal(noteDump)
		if err == nil {
			attachments = append(attachments, reporting.Attachment{
				Name:        "notes",
				ContentType: "application/json",
				Content:     content,
			})
		}
	}

	result := &testResult{
		name:        st.testName,
		status:      status,
		duration:    duration,
		err:         testErr,
		attachments: attachments,
	}

	// Notify reporter that test is finished
	if st.adapter != nil && st.adapter.GetReporter() != nil {
		st.adapter.GetReporter().OnTestFinish(result)
	}

	// Clear actors map
	st.actors = make(map[string]core.Actor)
	st.shutdown = true
}

type notesCollector interface {
	All() map[string]any
}

func (st *verityTest) collectNotes() map[string]map[string]any {
	if len(st.actors) == 0 {
		return nil
	}

	collected := make(map[string]map[string]any)
	for name, actor := range st.actors {
		internalActor, ok := actor.(*testActor)
		if !ok {
			continue
		}

		for _, ability := range internalActor.abilities {
			if _, ok := ability.(*take_notes.TakeNotesAbility); !ok {
				continue
			}

			noteAbility, ok := ability.(notesCollector)
			if !ok {
				continue
			}

			notesCopy := noteAbility.All()
			if len(notesCopy) == 0 {
				continue
			}

			collected[name] = notesCopy
			break
		}
	}

	if len(collected) == 0 {
		return nil
	}

	return collected
}
