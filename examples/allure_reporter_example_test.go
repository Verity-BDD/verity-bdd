package examples

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/take_notes"
	"github.com/verity-bdd/verity-bdd/verity_reporting/allure_reporter"
)

// TestAllureReporterExample_GeneratesReportFiles is a documentation-style example:
// create an Allure reporter, run actor steps, then assert generated result files.
func TestAllureReporterExample_GeneratesReportFiles(t *testing.T) {
	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)

	test := verity.NewVerityTest(t, verity.Scene{
		Context:     context.Background(),
		Reporter:    reporter,
		Description: "Demonstrates Allure requirement metadata",
		Links: []verity.Link{
			{Name: "Example requirement", URL: "https://requirements.example/allure", Type: "requirement"},
		},
	})

	actor := test.ActorCalled("Sam").WhoCan(take_notes.UsingEmptyNotepad())
	actor.AttemptsTo(
		verity.Do("#actor does something", func(ctx context.Context, actor verity.Actor) error {
			return nil
		}),
		verity.Do("#actor records an Allure-friendly step", func(ctx context.Context, actor verity.Actor) error {
			ability, err := actor.AbilityTo(&take_notes.TakeNotesAbility{})
			if err != nil {
				return err
			}

			notebook := ability.(*take_notes.TakeNotesAbility)
			notebook.Set("token", "secret")
			return nil
		}),
		verity.Do("#actor does something else", func(ctx context.Context, actor verity.Actor) error {
			return nil
		}),
	)

	test.Shutdown()

	resultFile := findSingleAllureResultFile(t, resultsDir)
	resultPayload, err := os.ReadFile(resultFile)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(resultPayload, &result))

	require.Equal(t, "TestAllureReporterExample_GeneratesReportFiles", result["name"])
	require.Equal(t, "passed", result["status"])
	require.Equal(t, "Demonstrates Allure requirement metadata", result["description"])

	links, ok := result["links"].([]any)
	require.True(t, ok)
	require.Len(t, links, 1)
	require.Equal(t, "https://requirements.example/allure", links[0].(map[string]any)["url"])

	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 3)

	attachments, ok := result["attachments"].([]any)
	require.True(t, ok)
	require.Len(t, attachments, 1)

	notesAttachment, ok := attachments[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "notes", notesAttachment["name"])

	source, ok := notesAttachment["source"].(string)
	require.True(t, ok)
	require.NotEmpty(t, source)

	attachmentPayload, err := os.ReadFile(filepath.Join(resultsDir, source))
	require.NoError(t, err)

	var notes map[string]map[string]any
	require.NoError(t, json.Unmarshal(attachmentPayload, &notes))
	require.Equal(t, "secret", notes["Sam"]["token"])
}

func TestAllureReporterExample_GeneratesNestedTaskSteps(t *testing.T) {
	resultsDir := t.TempDir()
	reporter := allure_reporter.NewAllureReporterWithDir(resultsDir)

	test := verity.NewVerityTest(t, verity.Scene{
		Context:  context.Background(),
		Reporter: reporter,
	})

	actor := test.ActorCalled("Sam")
	actor.AttemptsTo(
		verity.TaskWhere("#actor creates an order",
			verity.TaskWhere("#actor submits order details",
				verity.Do("#actor opens order page", func(ctx context.Context, actor verity.Actor) error {
					return nil
				}),
			),
		),
	)

	test.Shutdown()

	resultFile := findSingleAllureResultFile(t, resultsDir)
	resultPayload, err := os.ReadFile(resultFile)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(resultPayload, &result))

	steps, ok := result["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)

	rootStep := steps[0].(map[string]any)
	require.Equal(t, "Sam creates an order", rootStep["name"])

	nestedSteps, ok := rootStep["steps"].([]any)
	require.True(t, ok)
	require.Len(t, nestedSteps, 1)
	require.Equal(t, "Sam submits order details", nestedSteps[0].(map[string]any)["name"])

	leafSteps, ok := nestedSteps[0].(map[string]any)["steps"].([]any)
	require.True(t, ok)
	require.Len(t, leafSteps, 1)
	require.Equal(t, "Sam opens order page", leafSteps[0].(map[string]any)["name"])
}

func findSingleAllureResultFile(t *testing.T, resultsDir string) string {
	t.Helper()

	entries, err := os.ReadDir(resultsDir)
	require.NoError(t, err)

	resultFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		resultFiles = append(resultFiles, filepath.Join(resultsDir, entry.Name()))
	}

	require.Len(t, resultFiles, 1)
	return resultFiles[0]
}
