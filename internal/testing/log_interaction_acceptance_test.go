package testing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/internal/reporting/allure_reporter"
	"github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"
	log "github.com/verity-bdd/verity-bdd/verity_interactions/log"
	"github.com/verity-bdd/verity-bdd/verity_reporting"
)

func TestVerityActorPipelineWritesNestedLogAttachmentToAllure(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	test := verity.NewVerityTest(t, verity.Scene{
		Context:  context.Background(),
		Reporter: allure_reporter.NewAllureReporterWithDir(resultsDir),
	})

	test.ActorCalled("Sam").AttemptsTo(
		verity.TaskWhere("#actor completes checkout",
			verity.TaskWhere("#actor records checkout state",
				log.The("Current page", "Checkout"),
			),
		),
	)
	test.Shutdown()

	resultPayload, err := os.ReadFile(readSingleAllureResult(t, resultsDir))
	require.NoError(t, err)

	var result struct {
		Steps []struct {
			Name  string `json:"name"`
			Steps []struct {
				Name  string `json:"name"`
				Steps []struct {
					Name        string `json:"name"`
					Attachments []struct {
						Name   string `json:"name"`
						Source string `json:"source"`
						Type   string `json:"type"`
					} `json:"attachments"`
				} `json:"steps"`
			} `json:"steps"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(resultPayload, &result))
	require.Len(t, result.Steps, 1)
	require.Equal(t, "Sam completes checkout", result.Steps[0].Name)
	require.Len(t, result.Steps[0].Steps, 1)

	nestedTask := result.Steps[0].Steps[0]
	require.Equal(t, "Sam records checkout state", nestedTask.Name)
	require.Len(t, nestedTask.Steps, 1)

	logStep := nestedTask.Steps[0]
	require.Equal(t, "Sam logs", logStep.Name)
	require.Len(t, logStep.Attachments, 1)
	require.Equal(t, "log", logStep.Attachments[0].Name)
	require.Equal(t, "text/plain", logStep.Attachments[0].Type)

	attachmentPayload, err := os.ReadFile(filepath.Join(resultsDir, logStep.Attachments[0].Source))
	require.NoError(t, err)
	require.Equal(t, "Current page Checkout", string(attachmentPayload))
}

func TestVerityActorPipelineFailsFastWhenLogQuestionFails(t *testing.T) {
	t.Parallel()

	cause := errors.New("page lookup failed")
	testContext := &controllableTestContext{name: "failed log question"}
	reporter := &failedQuestionReporter{}
	test := verity.NewVerityTestWithReporter(context.Background(), testContext, reporter)

	test.ActorCalled("Sam").AttemptsTo(
		log.The(log.AnswerTo(verity.QuestionAbout("current page", func(context.Context, verity.Actor) (string, error) {
			return "", cause
		}))),
		log.The("following activity"),
	)

	require.Equal(t, []string{"Sam logs"}, reporter.stepStarts)
	require.Empty(t, reporter.logs)
	require.Len(t, reporter.stepFinishes, 1)
	require.Equal(t, "Sam logs", reporter.stepFinishes[0].Name())
	require.Equal(t, verity_reporting.StatusFailed, reporter.stepFinishes[0].Status())
	require.ErrorIs(t, reporter.stepFinishes[0].Error(), cause)
	require.EqualError(t, reporter.stepFinishes[0].Error(), "answer log question \"current page\": page lookup failed")
	require.Equal(t, []string{"Critical activity error '#actor logs' failed: answer log question \"current page\": page lookup failed"}, testContext.errors)
	require.Equal(t, 1, testContext.failNowCalls)

	test.Shutdown()
}

func TestVerityActorPipelineRendersNestedLogLifecycleToConsole(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	console := console_reporter.NewConsoleReporter()
	console.SetOutput(&output)
	reporter := &lifecycleReporter{Reporter: console}
	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)

	test.ActorCalled("Sam").AttemptsTo(
		verity.TaskWhere("#actor completes checkout",
			verity.TaskWhere("#actor records checkout state",
				log.The("Current page", "Checkout"),
			),
		),
	)
	test.Shutdown()

	require.Equal(t, []string{
		"step start: Sam completes checkout",
		"step start: Sam records checkout state",
		"step start: Sam logs",
		"log: Sam: Current page Checkout",
		"step finish: Sam logs",
		"step finish: Sam records checkout state",
		"step finish: Sam completes checkout",
	}, reporter.events)

	captured := output.String()
	logLine := "      ℹ️ Sam: Current page Checkout\n"
	logFinish := "      ✅ Sam logs ("
	nestedTaskFinish := "    ✅ Sam records checkout state ("
	taskFinish := "  ✅ Sam completes checkout ("
	testFinish := "✅ TestVerityActorPipelineRendersNestedLogLifecycleToConsole: PASSED ("

	logLineIndex := strings.Index(captured, logLine)
	logFinishIndex := strings.Index(captured, logFinish)
	nestedTaskFinishIndex := strings.Index(captured, nestedTaskFinish)
	taskFinishIndex := strings.Index(captured, taskFinish)
	testFinishIndex := strings.Index(captured, testFinish)
	require.GreaterOrEqual(t, logLineIndex, 0)
	require.Greater(t, logFinishIndex, logLineIndex)
	require.Greater(t, nestedTaskFinishIndex, logFinishIndex)
	require.Greater(t, taskFinishIndex, nestedTaskFinishIndex)
	require.Greater(t, testFinishIndex, taskFinishIndex)
}

type controllableTestContext struct {
	name         string
	failed       bool
	errors       []string
	failNowCalls int
	cleanups     []func()
}

func (c *controllableTestContext) Name() string                { return c.name }
func (c *controllableTestContext) Logf(string, ...interface{}) {}
func (c *controllableTestContext) Errorf(format string, args ...interface{}) {
	c.failed = true
	c.errors = append(c.errors, fmt.Sprintf(format, args...))
}
func (c *controllableTestContext) FailNow() {
	c.failed = true
	c.failNowCalls++
}
func (c *controllableTestContext) Failed() bool { return c.failed }
func (c *controllableTestContext) Cleanup(cleanup func()) {
	c.cleanups = append(c.cleanups, cleanup)
}
func (*controllableTestContext) Helper() {}

type failedQuestionReporter struct {
	stepStarts   []string
	stepFinishes []verity_reporting.TestResult
	logs         []verity_reporting.LogEntry
}

func (*failedQuestionReporter) OnTestStart(string)                       {}
func (*failedQuestionReporter) OnTestFinish(verity_reporting.TestResult) {}
func (r *failedQuestionReporter) OnStepStart(name string) {
	r.stepStarts = append(r.stepStarts, name)
}
func (r *failedQuestionReporter) OnStepFinish(result verity_reporting.TestResult) {
	r.stepFinishes = append(r.stepFinishes, result)
}
func (r *failedQuestionReporter) OnLog(entry verity_reporting.LogEntry) {
	r.logs = append(r.logs, entry)
}
func (*failedQuestionReporter) SetOutput(io.Writer) {}

type lifecycleReporter struct {
	verity_reporting.Reporter
	events []string
}

func (r *lifecycleReporter) OnTestStart(name string) {
	r.Reporter.OnTestStart(name)
}

func (r *lifecycleReporter) OnTestFinish(result verity_reporting.TestResult) {
	r.Reporter.OnTestFinish(result)
}

func (r *lifecycleReporter) OnStepStart(name string) {
	r.events = append(r.events, "step start: "+name)
	r.Reporter.OnStepStart(name)
}

func (r *lifecycleReporter) OnStepFinish(result verity_reporting.TestResult) {
	r.events = append(r.events, "step finish: "+result.Name())
	r.Reporter.OnStepFinish(result)
}

func (r *lifecycleReporter) OnLog(entry verity_reporting.LogEntry) {
	r.events = append(r.events, "log: "+entry.ActorName+": "+strings.Join(entry.Values, " "))
	r.Reporter.OnLog(entry)
}

func (r *lifecycleReporter) SetOutput(output io.Writer) {
	r.Reporter.SetOutput(output)
}

func readSingleAllureResult(t *testing.T, dir string) string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var resultPath string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			require.Empty(t, resultPath, "expected exactly one Allure result file")
			resultPath = filepath.Join(dir, entry.Name())
		}
	}
	require.NotEmpty(t, resultPath)

	return resultPath
}
