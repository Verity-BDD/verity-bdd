package testing

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	"github.com/verity-bdd/verity-bdd/internal/reporting/allure_reporter"
	testingMocks "github.com/verity-bdd/verity-bdd/internal/testing/mocks"
)

func TestVerityTest_WithAllureReporter_WritesResults(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)

	test := NewVerityTest(t, Scene{
		Context:  context.Background(),
		Reporter: reporter,
	})

	actor := test.ActorCalled("Sam").WhoCan(take_notes.UsingEmptyNotepad())
	ability, err := actor.AbilityTo(&take_notes.TakeNotesAbility{})
	require.NoError(t, err)
	notebook := ability.(*take_notes.TakeNotesAbility)
	notebook.Set("token", "secret")

	actor.AttemptsTo(core.Do("records an action", func(ctx context.Context, actor core.Actor) error {
		return nil
	}))

	test.Shutdown()

	resultPath := readResultFilePath(t, resultsDir)
	payload, err := os.ReadFile(resultPath)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(payload, &result))

	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)

	attachments, ok := result["attachments"].([]any)
	require.True(t, ok)
	require.Len(t, attachments, 1)

	attachment := attachments[0].(map[string]any)
	require.Equal(t, "notes", attachment["name"])

	source := attachment["source"].(string)
	attachmentPayload, err := os.ReadFile(filepath.Join(resultsDir, source))
	require.NoError(t, err)

	var notesJSON map[string]map[string]any
	require.NoError(t, json.Unmarshal(attachmentPayload, &notesJSON))
	require.Equal(t, "secret", notesJSON["Sam"]["token"])
}

func TestVerityTest_WithAllureReporter_WritesNestedTaskResults(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)

	test := NewVerityTest(t, Scene{
		Context:  context.Background(),
		Reporter: reporter,
	})

	actor := test.ActorCalled("Sam")
	actor.AttemptsTo(
		core.TaskWhere("#actor creates an order",
			core.Do("#actor opens order page", func(ctx context.Context, actor core.Actor) error {
				return nil
			}),
			core.Do("#actor saves order", func(ctx context.Context, actor core.Actor) error {
				return nil
			}),
		),
	)

	test.Shutdown()

	resultPath := readResultFilePath(t, resultsDir)
	payload, err := os.ReadFile(resultPath)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(payload, &result))

	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)

	taskStep := steps[0].(map[string]any)
	require.Equal(t, "Sam creates an order", taskStep["name"])

	childSteps, ok := taskStep["steps"].([]any)
	require.True(t, ok)
	require.Len(t, childSteps, 2)
	require.Equal(t, "Sam opens order page", childSteps[0].(map[string]any)["name"])
	require.Equal(t, "Sam saves order", childSteps[1].(map[string]any)["name"])
}

func TestTestActor_WithAllureReporter_ReportsFactSetupFailureAsStep(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)
	reporter.OnTestStart("FactSetupFailure")

	ctrl := gomock.NewController(t)
	testContext := testingMocks.NewMockTestContext(ctrl)
	setupErr := errors.New("boom")
	testContext.EXPECT().Errorf("Fact %q setup failed for actor %q: %v", "registered account", "Sam", setupErr)
	testContext.EXPECT().FailNow()

	actor := &testActor{
		name:        "Sam",
		testContext: testContext,
		reporter:    reporting.NewTestRunnerAdapter(reporter),
		ctx:         context.Background(),
	}
	actor.Has(&callbackFact{
		description: "registered account",
		setup: func(context.Context, core.Actor) error {
			return setupErr
		},
		teardown: func(context.Context, core.Actor) error {
			return nil
		},
	})
	reporter.OnTestFinish(&testResult{
		name:   "FactSetupFailure",
		status: reporting.StatusFailed,
		err:    setupErr,
	})

	resultPath := readResultFilePath(t, resultsDir)
	payload, err := os.ReadFile(resultPath)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(payload, &result))
	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)
	setupStep := steps[0].(map[string]any)
	require.Equal(t, "Sam has registered account", setupStep["name"])
	require.Equal(t, "failed", setupStep["status"])
	statusDetails := setupStep["statusDetails"].(map[string]any)
	require.Equal(t, "boom", statusDetails["message"])
}

func TestTestActor_WithAllureReporter_ReportsFactTeardownFailureAsStep(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)
	reporter.OnTestStart("FactTeardownFailure")

	ctrl := gomock.NewController(t)
	testContext := testingMocks.NewMockTestContext(ctrl)
	teardownErr := errors.New("boom")
	testContext.EXPECT().Errorf("Fact %q teardown failed for actor %q: %v", "registered account", "Sam", teardownErr)

	actor := &testActor{
		name:        "Sam",
		testContext: testContext,
		reporter:    reporting.NewTestRunnerAdapter(reporter),
		ctx:         context.Background(),
	}
	actor.Has(&callbackFact{
		description: "registered account",
		setup: func(context.Context, core.Actor) error {
			return nil
		},
		teardown: func(context.Context, core.Actor) error {
			return teardownErr
		},
	})
	actor.teardownFacts()
	reporter.OnTestFinish(&testResult{
		name:   "FactTeardownFailure",
		status: reporting.StatusFailed,
		err:    teardownErr,
	})

	resultPath := readResultFilePath(t, resultsDir)
	payload, err := os.ReadFile(resultPath)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(payload, &result))
	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 2)
	teardownStep := steps[1].(map[string]any)
	require.Equal(t, "Sam tears down registered account", teardownStep["name"])
	require.Equal(t, "failed", teardownStep["status"])
	statusDetails := teardownStep["statusDetails"].(map[string]any)
	require.Equal(t, "boom", statusDetails["message"])
}

func readResultFilePath(t *testing.T, dir string) string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	results := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		results = append(results, filepath.Join(dir, entry.Name()))
	}

	require.Len(t, results, 1)
	return results[0]
}
