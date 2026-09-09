package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	"github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"
	reportingMocks "github.com/verity-bdd/verity-bdd/internal/reporting/mocks"
	"github.com/verity-bdd/verity-bdd/internal/testing/mocks"

	"github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
)

type sceneDefaultAbility struct {
	owner string
}

type countingReporter struct {
	finishCount atomic.Int64
}

func (*countingReporter) OnTestStart(string) {}

func (r *countingReporter) OnTestFinish(reporting.TestResult) {
	r.finishCount.Add(1)
}

func (*countingReporter) OnStepStart(string) {}

func (*countingReporter) OnStepFinish(reporting.TestResult) {}
func (*countingReporter) OnLog(reporting.LogEntry)          {}

func (*countingReporter) SetOutput(io.Writer) {}

func TestNewVerityTest_ConfiguredByScene(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := mocks.NewMockTestContext(ctrl)

	sceneCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scene := Scene{
		Context:  sceneCtx,
		Reporter: mockReporter,
		DefaultAbilities: []DefaultAbilityFactory{
			func(actorName string) abilities.Ability {
				return &sceneDefaultAbility{owner: actorName}
			},
		},
	}

	mockTestContext.EXPECT().Helper()
	mockTestContext.EXPECT().Name().Return("SceneConfiguredTest")
	mockTestContext.EXPECT().Cleanup(gomock.Any())
	mockTestContext.EXPECT().Failed().Return(false)

	mockReporter.EXPECT().OnTestStart("SceneConfiguredTest")
	mockReporter.EXPECT().OnTestFinish(gomock.Any())

	test := NewVerityTest(mockTestContext, scene)
	actor := test.ActorCalled("Sam")

	require.Same(t, sceneCtx, actor.Context())

	ability, err := actor.AbilityTo(&sceneDefaultAbility{})
	require.NoError(t, err)
	require.Equal(t, "Sam", ability.(*sceneDefaultAbility).owner)

	test.Shutdown()
}

func TestSceneDefaultAbilities_AreIsolatedPerActor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	test := NewVerityTest(t, Scene{
		Context: ctx,
		DefaultAbilities: []DefaultAbilityFactory{
			func(actorName string) abilities.Ability {
				return take_notes.UsingEmptyNotepad()
			},
		},
	})

	alice := test.ActorCalled("Alice")
	bob := test.ActorCalled("Bob")

	aliceAbilityRaw, err := alice.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)
	bobAbilityRaw, err := bob.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)

	aliceNotes := aliceAbilityRaw.(*take_notes.TakeNotesAbility)
	bobNotes := bobAbilityRaw.(*take_notes.TakeNotesAbility)

	aliceNotes.Set("token", "alice-secret")

	aliceToken, err := aliceNotes.Get("token")
	require.NoError(t, err)
	require.Equal(t, "alice-secret", aliceToken)

	bobToken, err := bobNotes.Get("token")
	require.Error(t, err)
	require.Nil(t, bobToken)

	require.NotSame(t, aliceNotes, bobNotes)
}

func TestSceneDefaultAbilities_CanPreFillNotesByActorName(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{
		Context: context.Background(),
		DefaultAbilities: []DefaultAbilityFactory{
			func(actorName string) abilities.Ability {
				return take_notes.Using(take_notes.NotepadWith(map[string]any{
					"firstName": actorName,
					"role":      "Tester",
				}))
			},
		},
	})

	alice := test.ActorCalled("Alice")
	bob := test.ActorCalled("Bob")

	aliceAbilityRaw, err := alice.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)
	bobAbilityRaw, err := bob.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)

	aliceNotes := aliceAbilityRaw.(*take_notes.TakeNotesAbility)
	bobNotes := bobAbilityRaw.(*take_notes.TakeNotesAbility)

	aliceFirstName, err := aliceNotes.Get("firstName")
	require.NoError(t, err)
	bobFirstName, err := bobNotes.Get("firstName")
	require.NoError(t, err)

	require.Equal(t, "Alice", aliceFirstName)
	require.Equal(t, "Bob", bobFirstName)

	role, err := aliceNotes.Get("role")
	require.NoError(t, err)
	require.Equal(t, "Tester", role)
}

func TestVerityTestWithConsoleReporter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Create a VerityTest with console reporter
	test := NewVerityTestWithReporter(ctx, t, console_reporter.NewConsoleReporter())

	actor := test.ActorCalled("TestActor")
	require.NotNil(t, actor)

	// Verify that reporter is configured
	adapter := test.GetReporterAdapter()
	require.NotNil(t, adapter)
	require.IsType(t, &console_reporter.ConsoleReporter{}, adapter.GetReporter())
}

func TestNewVerityTestUsesConsoleReporter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	test := NewVerityTestWithContext(ctx, t)

	adapter := test.GetReporterAdapter()
	require.NotNil(t, adapter)

	// Verify it's a ConsoleReporter
	reporter := adapter.GetReporter()
	_, isConsole := reporter.(*console_reporter.ConsoleReporter)
	require.True(t, isConsole, "Expected ConsoleReporter")
}

func TestVerityTestLifecycleReporting(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := mocks.NewMockTestContext(ctrl)

	// Expect test lifecycle events
	mockReporter.EXPECT().OnTestStart("TestExample")
	mockReporter.EXPECT().OnTestFinish(gomock.Any()).Do(func(result reporting.TestResult) {
		require.Equal(t, "TestExample", result.Name())
		require.Equal(t, reporting.StatusPassed, result.Status())
		require.True(t, result.Duration() >= 0)
		require.NoError(t, result.Error())
	})

	mockTestContext.EXPECT().Name().Return("TestExample")
	mockTestContext.EXPECT().Failed().Return(false)
	mockTestContext.EXPECT().Helper()
	mockTestContext.EXPECT().Cleanup(gomock.Any())

	ctx := context.Background()
	test := NewVerityTestWithReporter(ctx, mockTestContext, mockReporter)

	// Simulate test end
	test.Shutdown()
}

func TestVerityTestLifecycleReportingFailed(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := mocks.NewMockTestContext(ctrl)

	// Expect test lifecycle events for failed test
	mockReporter.EXPECT().OnTestStart("FailedTest")
	mockReporter.EXPECT().OnTestFinish(gomock.Any()).Do(func(result reporting.TestResult) {
		require.Equal(t, "FailedTest", result.Name())
		require.Equal(t, reporting.StatusFailed, result.Status())
		require.True(t, result.Duration() >= 0)
		require.Error(t, result.Error())
		require.Equal(t, "test failed", result.Error().Error())
	})

	mockTestContext.EXPECT().Name().Return("FailedTest")
	mockTestContext.EXPECT().Failed().Return(true)
	mockTestContext.EXPECT().Helper()
	mockTestContext.EXPECT().Cleanup(gomock.Any())

	ctx := context.Background()
	test := NewVerityTestWithReporter(ctx, mockTestContext, mockReporter)

	// Simulate test end
	test.Shutdown()
}

func TestVerityTestAddsNotesAttachmentOnShutdown(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := mocks.NewMockTestContext(ctrl)

	mockReporter.EXPECT().OnTestStart("NotesTest")
	mockReporter.EXPECT().OnTestFinish(gomock.Any()).Do(func(result reporting.TestResult) {
		attachments := result.Attachments()
		require.Len(t, attachments, 1)

		attachment := attachments[0]
		require.Equal(t, "notes", attachment.Name)
		require.Equal(t, "application/json", attachment.ContentType)

		var payload map[string]map[string]any
		require.NoError(t, json.Unmarshal(attachment.Content, &payload))

		notesForActor, ok := payload["Sam"]
		require.True(t, ok, "expected notes for actor Sam")
		require.Equal(t, "secret", notesForActor["token"])
		intCount, ok := notesForActor["count"].(float64)
		require.True(t, ok, "expected numeric count")
		require.Equal(t, float64(2), intCount)
	})

	mockTestContext.EXPECT().Name().Return("NotesTest")
	mockTestContext.EXPECT().Failed().Return(false)
	mockTestContext.EXPECT().Helper()
	mockTestContext.EXPECT().Cleanup(gomock.Any())

	ctx := context.Background()
	test := NewVerityTestWithReporter(ctx, mockTestContext, mockReporter)

	actor := test.ActorCalled("Sam").WhoCan(take_notes.UsingEmptyNotepad())
	ability, err := actor.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)
	notebook := ability.(*take_notes.TakeNotesAbility)
	notebook.Set("token", "secret")
	notebook.Set("count", 2)

	test.Shutdown()
}

func TestVerityTestAddsNotesAttachmentOnFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := mocks.NewMockTestContext(ctrl)

	mockReporter.EXPECT().OnTestStart("NotesFailedTest")
	mockReporter.EXPECT().OnTestFinish(gomock.Any()).Do(func(result reporting.TestResult) {
		require.Equal(t, "NotesFailedTest", result.Name())
		require.Equal(t, reporting.StatusFailed, result.Status())
		require.Error(t, result.Error())
		require.EqualError(t, result.Error(), "test failed")

		attachments := result.Attachments()
		require.Len(t, attachments, 1)

		attachment := attachments[0]
		require.Equal(t, "notes", attachment.Name)
		require.Equal(t, "application/json", attachment.ContentType)

		var payload map[string]map[string]any
		require.NoError(t, json.Unmarshal(attachment.Content, &payload))

		notesForActor, ok := payload["Sam"]
		require.True(t, ok, "expected notes for actor Sam")
		require.Equal(t, "secret", notesForActor["token"])
		intCount, ok := notesForActor["count"].(float64)
		require.True(t, ok, "expected numeric count")
		require.Equal(t, float64(3), intCount)
	})

	mockTestContext.EXPECT().Name().Return("NotesFailedTest")
	mockTestContext.EXPECT().Failed().Return(true)
	mockTestContext.EXPECT().Helper()
	mockTestContext.EXPECT().Cleanup(gomock.Any())

	ctx := context.Background()
	test := NewVerityTestWithReporter(ctx, mockTestContext, mockReporter)

	actor := test.ActorCalled("Sam").WhoCan(take_notes.UsingEmptyNotepad())
	ability, err := actor.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)
	notebook := ability.(*take_notes.TakeNotesAbility)
	notebook.Set("token", "secret")
	notebook.Set("count", 3)

	test.Shutdown()
}

func TestNewVerityNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	test := NewVerityTest(t, Scene{})
	require.NotNil(t, test)

	withContext := NewVerityTestWithContext(ctx, t)
	require.NotNil(t, withContext)

	withReporter := NewVerityTestWithReporter(ctx, t, console_reporter.NewConsoleReporter())
	require.NotNil(t, withReporter)
}

func TestVerityTestActors_EmptySnapshotIsNonNil(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})

	actors := test.Actors()

	require.NotNil(t, actors)
	require.Empty(t, actors)
}

func TestVerityTestActors_IncludesEveryRegisteredActorInstance(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	emptyName := test.ActorCalled("")
	alice := test.ActorCalled("Alice")

	actors := test.Actors()

	require.Len(t, actors, 2)
	require.Contains(t, actors, emptyName)
	require.Contains(t, actors, alice)
	require.Same(t, emptyName, actors[0])
	require.Same(t, alice, actors[1])
}

func TestVerityTestActors_SortsUniqueActorsByCaseSensitiveName(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	bob := test.ActorCalled("bob")
	aliceUpper := test.ActorCalled("Alice")
	aliceLower := test.ActorCalled("alice")
	require.Same(t, aliceUpper, test.ActorCalled("Alice"))

	actors := test.Actors()

	require.Len(t, actors, 3)
	require.Equal(t, []string{"Alice", "alice", "bob"}, []string{actors[0].Name(), actors[1].Name(), actors[2].Name()})
	require.Same(t, aliceUpper, actors[0])
	require.Same(t, aliceLower, actors[1])
	require.Same(t, bob, actors[2])
}

func TestVerityTestActors_ReturnsFreshSliceSnapshots(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	alice := test.ActorCalled("Alice")
	bob := test.ActorCalled("Bob")

	first := test.Actors()
	first[0] = bob
	mutated := append(first, alice)

	require.Len(t, mutated, 3)
	require.Equal(t, []core.Actor{alice, bob}, test.Actors())
}

func TestVerityTestActors_ShutdownReleasesRegistryAndReturnsNonNilEmptySnapshot(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	test.ActorCalled("Alice")
	test.Shutdown()

	actors := test.Actors()

	require.Nil(t, test.(*verityTest).actors)
	require.NotNil(t, actors)
	require.Empty(t, actors)
}

func TestVerityTestActorCalled_CompletedBeforeShutdownRemainsLegal(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})

	alice := test.ActorCalled("Alice")
	test.Shutdown()

	require.Equal(t, "Alice", alice.Name())
	require.Empty(t, test.Actors())
}

func TestVerityTestActorCalled_PanicsAfterShutdown(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	test.Shutdown()

	require.PanicsWithValue(t, "verity: ActorCalled called after Shutdown", func() {
		test.ActorCalled("Alice")
	})
}

func TestVerityTestActorCalled_ConcurrentWithShutdownLinearizesAtTerminalState(t *testing.T) {
	t.Parallel()
	const attempts = 256

	type actorCallResult struct {
		actor      core.Actor
		panicValue any
	}

	for range attempts {
		reporter := &countingReporter{}
		test := NewVerityTest(t, Scene{Reporter: reporter})
		concrete := test.(*verityTest)

		var ready sync.WaitGroup
		ready.Add(2)
		start := make(chan struct{})
		actorResult := make(chan actorCallResult, 1)
		shutdownPanic := make(chan any, 1)

		go func() {
			result := actorCallResult{}
			defer func() {
				result.panicValue = recover()
				actorResult <- result
			}()
			ready.Done()
			<-start
			result.actor = test.ActorCalled("Alice")
		}()

		go func() {
			var panicValue any
			defer func() {
				panicValue = recover()
				shutdownPanic <- panicValue
			}()
			ready.Done()
			<-start
			test.Shutdown()
		}()

		ready.Wait()
		close(start)
		result := <-actorResult
		require.Nil(t, <-shutdownPanic)

		if result.panicValue == nil {
			require.NotNil(t, result.actor)
			require.Equal(t, "Alice", result.actor.Name())
		} else {
			require.Equal(t, "verity: ActorCalled called after Shutdown", result.panicValue)
			require.Nil(t, result.actor)
		}

		require.Nil(t, concrete.actors, "shutdown must not leave a hidden actor registry")
		require.Empty(t, test.Actors())
		require.PanicsWithValue(t, "verity: ActorCalled called after Shutdown", func() {
			test.ActorCalled("Bob")
		})
		require.Equal(t, int64(1), reporter.finishCount.Load())
	}
}

func TestVerityTestShutdown_RepeatedAndConcurrentCallsReportExactlyOnce(t *testing.T) {
	t.Parallel()
	const callers = 64

	reporter := &countingReporter{}
	test := NewVerityTest(t, Scene{Reporter: reporter})
	test.ActorCalled("Alice")

	var ready sync.WaitGroup
	ready.Add(callers)
	start := make(chan struct{})
	panics := make(chan any, callers)
	var callersWG sync.WaitGroup
	callersWG.Add(callers)
	for range callers {
		go func() {
			defer callersWG.Done()
			var panicValue any
			defer func() {
				panicValue = recover()
				panics <- panicValue
			}()
			ready.Done()
			<-start
			test.Shutdown()
		}()
	}

	ready.Wait()
	close(start)
	callersWG.Wait()
	close(panics)
	for panicValue := range panics {
		require.Nil(t, panicValue)
	}

	test.Shutdown()
	require.Equal(t, int64(1), reporter.finishCount.Load())
	require.Nil(t, test.(*verityTest).actors)
	require.Empty(t, test.Actors())
	require.PanicsWithValue(t, "verity: ActorCalled called after Shutdown", func() {
		test.ActorCalled("Bob")
	})
}

func TestVerityTestActors_ConcurrentSnapshotsAreConsistent(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	const actorCount = 64
	const registrationsPerActor = 4
	const readers = 8
	const snapshotsPerReader = 256

	var ready sync.WaitGroup
	ready.Add(readers + actorCount*registrationsPerActor)
	start := make(chan struct{})
	errs := make(chan error, readers)

	var readerWG sync.WaitGroup
	for range readers {
		readerWG.Add(1)
		go func() {
			defer readerWG.Done()
			ready.Done()
			<-start
			for range snapshotsPerReader {
				actors := test.Actors()
				if actors == nil {
					errs <- fmt.Errorf("Actors returned a nil snapshot")
					return
				}
				for i, actor := range actors {
					if actor == nil {
						errs <- fmt.Errorf("snapshot contains a nil actor at index %d", i)
						return
					}
					if i > 0 && actors[i-1].Name() >= actor.Name() {
						errs <- fmt.Errorf("snapshot is not strictly sorted: %q before %q", actors[i-1].Name(), actor.Name())
						return
					}
				}
			}
		}()
	}

	var writerWG sync.WaitGroup
	for i := range actorCount * registrationsPerActor {
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			ready.Done()
			<-start
			test.ActorCalled(fmt.Sprintf("Actor-%02d", i%actorCount))
		}()
	}

	ready.Wait()
	close(start)
	writerWG.Wait()
	readerWG.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	actors := test.Actors()
	require.Len(t, actors, actorCount)
	for i, actor := range actors {
		require.Equal(t, fmt.Sprintf("Actor-%02d", i), actor.Name())
	}
}

func TestVerityTestActors_ConcurrentWithShutdownReturnsAtomicSnapshots(t *testing.T) {
	t.Parallel()
	test := NewVerityTest(t, Scene{})
	const actorCount = 64
	const readers = 32
	const snapshotsPerReader = 256

	expected := make([]core.Actor, actorCount)
	for i := actorCount - 1; i >= 0; i-- {
		expected[i] = test.ActorCalled(fmt.Sprintf("Actor-%02d", i))
	}

	var ready sync.WaitGroup
	ready.Add(readers + 1)
	start := make(chan struct{})
	errCh := make(chan error, readers)

	var readersWG sync.WaitGroup
	for range readers {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			ready.Done()
			<-start

			for range snapshotsPerReader {
				actors := test.Actors()
				if actors == nil {
					errCh <- fmt.Errorf("Actors returned a nil snapshot")
					return
				}
				if len(actors) == 0 {
					continue
				}
				if len(actors) != len(expected) {
					errCh <- fmt.Errorf("snapshot contains %d actors, want either 0 or %d", len(actors), len(expected))
					return
				}
				for i, actor := range actors {
					if actor != expected[i] {
						errCh <- fmt.Errorf("snapshot actor %d is not the original sorted instance for %q", i, expected[i].Name())
						return
					}
				}
			}
		}()
	}

	shutdownDone := make(chan struct{})
	go func() {
		ready.Done()
		<-start
		test.Shutdown()
		close(shutdownDone)
	}()

	ready.Wait()
	close(start)
	readersWG.Wait()
	<-shutdownDone
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	actors := test.Actors()
	require.NotNil(t, actors)
	require.Empty(t, actors)
}
