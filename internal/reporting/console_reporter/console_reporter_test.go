package console_reporter

import (
	"bytes"
	"errors"
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

func TestConsoleReporterRendersLogAtCurrentNestingLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reporter := NewConsoleReporter()
	reporter.SetOutput(&buf)

	reporter.OnStepStart("Sam creates checkout")
	reporter.OnStepStart("Sam logs")
	reporter.OnLog(reporting.LogEntry{ActorName: "Sam", Values: []string{"Current page", "Checkout"}})

	require.Equal(t, "    ℹ️ Sam: Current page Checkout\n", buf.String())
}

func TestConsoleReporterPrintsAttachmentsOnFailure(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reporter := NewConsoleReporter()
	reporter.SetOutput(&buf)

	result := &stubResult{
		name:     "NotesFailed",
		status:   reporting.StatusFailed,
		duration: 0.05,
		err:      errors.New("boom"),
		attachments: []reporting.Attachment{
			{
				Name:        "notes",
				ContentType: "application/json",
				Content:     []byte(`{"token":"secret"}`),
			},
		},
	}

	reporter.OnTestFinish(result)

	output := buf.String()
	require.Contains(t, output, "FAILED")
	require.Contains(t, output, "Error: boom")
	require.Contains(t, output, "Attachments:")
	require.Contains(t, output, "- notes (application/json): {\"token\":\"secret\"}")
}
