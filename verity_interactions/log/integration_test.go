package log_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	verity "github.com/verity-bdd/verity-bdd"
	log "github.com/verity-bdd/verity-bdd/verity_interactions/log"
	"github.com/verity-bdd/verity-bdd/verity_reporting"
)

type recordingReporter struct {
	entries []verity_reporting.LogEntry
}

func (*recordingReporter) OnTestStart(string)                       {}
func (*recordingReporter) OnTestFinish(verity_reporting.TestResult) {}
func (*recordingReporter) OnStepStart(string)                       {}
func (*recordingReporter) OnStepFinish(verity_reporting.TestResult) {}
func (*recordingReporter) SetOutput(io.Writer)                      {}
func (r *recordingReporter) OnLog(entry verity_reporting.LogEntry) {
	r.entries = append(r.entries, entry)
}

func TestTheDeliversEntryThroughStandardActorReporter(t *testing.T) {
	t.Parallel()
	reporter := &recordingReporter{}
	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
	actor := test.ActorCalled("Sam")

	require.NoError(t, log.The("Current page", "Checkout").PerformAs(test.Context(), actor))
	require.Equal(t, []verity_reporting.LogEntry{{
		ActorName: "Sam",
		Values:    []string{"Current page", "Checkout"},
	}}, reporter.entries)
}
