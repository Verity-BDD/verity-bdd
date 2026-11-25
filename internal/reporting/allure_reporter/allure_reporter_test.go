package allure_reporter

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verity-bdd/verity-bdd/internal/reporting"
)

type stubResult struct {
	name        string
	status      reporting.Status
	duration    float64
	err         error
	attachments []reporting.Attachment
}

func (sr *stubResult) Name() string {
	return sr.name
}

func (sr *stubResult) Status() reporting.Status {
	return sr.status
}

func (sr *stubResult) Duration() float64 {
	return sr.duration
}

func (sr *stubResult) Error() error {
	return sr.err
}

func (sr *stubResult) Attachments() []reporting.Attachment {
	return sr.attachments
}

func TestAllureReporterAttachesLogToNestedOpenStep(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	reporter := NewAllureReporterWithDir(resultsDir)
	reporter.OnTestStart("LogTest")
	reporter.OnStepStart("Sam checks out")
	reporter.OnStepStart("Sam logs")
	reporter.OnLog(reporting.LogEntry{ActorName: "Sam", Values: []string{"Current page", "Checkout"}})
	reporter.OnStepFinish(&stubResult{name: "Sam logs", status: reporting.StatusPassed, duration: 0.01})
	reporter.OnStepFinish(&stubResult{name: "Sam checks out", status: reporting.StatusPassed, duration: 0.02})
	reporter.OnTestFinish(&stubResult{name: "LogTest", status: reporting.StatusPassed, duration: 0.03})

	result := readSingleResultFile(t, resultsDir)
	logStep := result.Steps[0].Steps[0]
	require.Equal(t, "Sam logs", logStep.Name)
	require.Len(t, logStep.Attachments, 1)
	require.Equal(t, "log", logStep.Attachments[0].Name)
	require.Equal(t, "text/plain", logStep.Attachments[0].Type)
	payload, err := os.ReadFile(filepath.Join(resultsDir, logStep.Attachments[0].Source))
	require.NoError(t, err)
	require.Equal(t, "Current page Checkout", string(payload))
}

func TestAllureReporter_WritesResultFile(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("SampleTest")
	r.OnTestFinish(&stubResult{name: "SampleTest", status: reporting.StatusPassed, duration: 0.01})

	result := readSingleResultFile(t, resultsDir)
	require.Equal(t, "SampleTest", result.Name)
	require.Equal(t, "passed", result.Status)
	require.NotEmpty(t, result.UUID)
	require.Greater(t, result.Stop, result.Start)
}

func TestAllureReporter_MapsStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		status       reporting.Status
		expected     string
		expectedName string
	}{
		{name: "passed", status: reporting.StatusPassed, expected: "passed", expectedName: "PassedStatus"},
		{name: "failed", status: reporting.StatusFailed, expected: "failed", expectedName: "FailedStatus"},
		{name: "skipped", status: reporting.StatusSkipped, expected: "skipped", expectedName: "SkippedStatus"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resultsDir := t.TempDir()
			r := NewAllureReporterWithDir(resultsDir)

			r.OnTestStart(tc.expectedName)
			r.OnTestFinish(&stubResult{name: tc.expectedName, status: tc.status, duration: 0.01})

			result := readSingleResultFile(t, resultsDir)
			require.Equal(t, tc.expected, result.Status)
		})
	}
}

func TestAllureReporter_WritesStatusDetails(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("FailedWithError")
	r.OnTestFinish(&stubResult{
		name:     "FailedWithError",
		status:   reporting.StatusFailed,
		duration: 0.01,
		err:      errors.New("boom"),
	})

	result := readSingleResultFile(t, resultsDir)
	require.Equal(t, "failed", result.Status)
	require.NotNil(t, result.StatusDetails)
	require.Contains(t, result.StatusDetails.Message, "boom")
}

func TestAllureReporter_RecordsSteps(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("StepTest")
	r.OnStepStart("does important step")
	r.OnStepFinish(&stubResult{name: "does important step", status: reporting.StatusPassed, duration: 0.02})
	r.OnTestFinish(&stubResult{name: "StepTest", status: reporting.StatusPassed, duration: 0.03})

	result := readSingleResultFile(t, resultsDir)
	require.Len(t, result.Steps, 1)
	require.Equal(t, "does important step", result.Steps[0].Name)
	require.Equal(t, "passed", result.Steps[0].Status)
	require.Greater(t, result.Steps[0].Stop, result.Steps[0].Start)
}

func TestAllureReporter_RecordsNestedSteps(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("NestedStepTest")
	r.OnStepStart("creates an order")
	r.OnStepStart("opens order page")
	r.OnStepFinish(&stubResult{name: "opens order page", status: reporting.StatusPassed, duration: 0.01})
	r.OnStepFinish(&stubResult{name: "creates an order", status: reporting.StatusPassed, duration: 0.02})
	r.OnTestFinish(&stubResult{name: "NestedStepTest", status: reporting.StatusPassed, duration: 0.03})

	result := readSingleResultFile(t, resultsDir)
	require.Len(t, result.Steps, 1)
	require.Equal(t, "creates an order", result.Steps[0].Name)
	require.Len(t, result.Steps[0].Steps, 1)
	require.Equal(t, "opens order page", result.Steps[0].Steps[0].Name)
	require.Equal(t, "passed", result.Steps[0].Steps[0].Status)
}

func TestAllureReporter_RecordsNestedStepsAtMultipleLevels(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("NestedTaskTreeTest")
	r.OnStepStart("creates an order")
	r.OnStepStart("submits order details")
	r.OnStepStart("opens order page")
	r.OnStepFinish(&stubResult{name: "opens order page", status: reporting.StatusPassed, duration: 0.01})
	r.OnStepFinish(&stubResult{name: "submits order details", status: reporting.StatusPassed, duration: 0.02})
	r.OnStepFinish(&stubResult{name: "creates an order", status: reporting.StatusPassed, duration: 0.03})
	r.OnTestFinish(&stubResult{name: "NestedTaskTreeTest", status: reporting.StatusPassed, duration: 0.04})

	result := readSingleResultFile(t, resultsDir)
	require.Len(t, result.Steps, 1)
	require.Equal(t, "creates an order", result.Steps[0].Name)
	require.Len(t, result.Steps[0].Steps, 1)
	require.Equal(t, "submits order details", result.Steps[0].Steps[0].Name)
	require.Len(t, result.Steps[0].Steps[0].Steps, 1)
	require.Equal(t, "opens order page", result.Steps[0].Steps[0].Steps[0].Name)
}

func TestAllureReporter_WritesTestAttachments(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("AttachmentTest")
	r.OnTestFinish(&stubResult{
		name:     "AttachmentTest",
		status:   reporting.StatusPassed,
		duration: 0.01,
		attachments: []reporting.Attachment{
			{
				Name:        "notes",
				ContentType: "application/json",
				Content:     []byte(`{"ok":true}`),
			},
		},
	})

	result := readSingleResultFile(t, resultsDir)
	require.Len(t, result.Attachments, 1)
	require.Equal(t, "notes", result.Attachments[0].Name)
	require.Equal(t, "application/json", result.Attachments[0].Type)
	require.NotEmpty(t, result.Attachments[0].Source)

	payload, err := os.ReadFile(filepath.Join(resultsDir, result.Attachments[0].Source))
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(payload))
}

func TestAllureReporter_WritesStepAttachments(t *testing.T) {
	t.Parallel()

	resultsDir := t.TempDir()
	r := NewAllureReporterWithDir(resultsDir)

	r.OnTestStart("StepAttachmentTest")
	r.OnStepStart("captures evidence")
	r.OnStepFinish(&stubResult{
		name:     "captures evidence",
		status:   reporting.StatusPassed,
		duration: 0.02,
		attachments: []reporting.Attachment{
			{
				Name:        "response",
				ContentType: "application/json",
				Content:     []byte(`{"status":200}`),
			},
		},
	})
	r.OnTestFinish(&stubResult{name: "StepAttachmentTest", status: reporting.StatusPassed, duration: 0.03})

	result := readSingleResultFile(t, resultsDir)
	require.Len(t, result.Steps, 1)
	require.Len(t, result.Steps[0].Attachments, 1)
	require.Equal(t, "response", result.Steps[0].Attachments[0].Name)
	require.Equal(t, "application/json", result.Steps[0].Attachments[0].Type)

	source := result.Steps[0].Attachments[0].Source
	payload, err := os.ReadFile(filepath.Join(resultsDir, source))
	require.NoError(t, err)
	require.JSONEq(t, `{"status":200}`, string(payload))
}

type expectedResultFile struct {
	UUID          string               `json:"uuid"`
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	StatusDetails *expectedStatus      `json:"statusDetails,omitempty"`
	Start         int64                `json:"start"`
	Stop          int64                `json:"stop"`
	Steps         []expectedStep       `json:"steps,omitempty"`
	Attachments   []expectedAttachment `json:"attachments,omitempty"`
}

type expectedStatus struct {
	Message string `json:"message,omitempty"`
}

type expectedStep struct {
	Name        string               `json:"name"`
	Status      string               `json:"status"`
	Start       int64                `json:"start"`
	Stop        int64                `json:"stop"`
	Steps       []expectedStep       `json:"steps,omitempty"`
	Attachments []expectedAttachment `json:"attachments,omitempty"`
}

type expectedAttachment struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

func readSingleResultFile(t *testing.T, resultsDir string) expectedResultFile {
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
		if filepath.Base(entry.Name()) == "categories.json" {
			continue
		}
		resultFiles = append(resultFiles, filepath.Join(resultsDir, entry.Name()))
	}

	require.Len(t, resultFiles, 1)

	content, err := os.ReadFile(resultFiles[0])
	require.NoError(t, err)

	var result expectedResultFile
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	return result
}
