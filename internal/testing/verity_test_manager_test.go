package testing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	"github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"
	reportingMocks "github.com/verity-bdd/verity-bdd/internal/reporting/mocks"
	"github.com/verity-bdd/verity-bdd/internal/testing/mocks"

	"github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
)

type sceneDefaultAbility struct {
	owner string
}

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
