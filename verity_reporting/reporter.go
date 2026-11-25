// Package verity_reporting exposes reporter callback contracts and adapters.
// Its TestResult and Status types are reporter-facing interfaces and are
// distinct from the root verity package's core TestResult and Status types.
package verity_reporting

import internalreporting "github.com/verity-bdd/verity-bdd/internal/reporting"

// Reporter handles test execution reporting.
type Reporter = internalreporting.Reporter

// LogEntry contains the actor name and ordered values emitted by a Log interaction.
type LogEntry = internalreporting.LogEntry

// TestResult represents the result passed to a test or step callback. Duration
// is measured in seconds. The contract supports attachments, although normal
// actor execution currently supplies none to step-finish callbacks.
type TestResult = internalreporting.TestResult

// Attachment represents additional data to include in a report entry.
// Content contains the payload bytes and ContentType identifies their media type.
type Attachment = internalreporting.Attachment

// Status represents the status of a test or step.
type Status = internalreporting.Status

const (
	// StatusPassed indicates a test or step passed.
	StatusPassed = internalreporting.StatusPassed
	// StatusFailed indicates a test or step failed.
	StatusFailed = internalreporting.StatusFailed
	// StatusSkipped indicates a test or step was skipped.
	StatusSkipped = internalreporting.StatusSkipped
)

// TestRunnerAdapter provides integration with test runners.
type TestRunnerAdapter = internalreporting.TestRunnerAdapter

// ActivityTracker tracks activity execution for reporting.
type ActivityTracker = internalreporting.ActivityTracker

// NewTestRunnerAdapter creates a new test runner adapter.
func NewTestRunnerAdapter(reporter Reporter) *TestRunnerAdapter {
	return internalreporting.NewTestRunnerAdapter(reporter)
}

// NewActivityTracker creates a tracker that emits reporter step callbacks for
// activity. Call Start before execution and Finish afterward.
func NewActivityTracker(reporter Reporter, activity string) *ActivityTracker {
	return internalreporting.NewActivityTracker(reporter, activity)
}

// NewActivityTrackerWithActor creates a tracker and replaces a leading
// "#actor " activity placeholder with actorName in callback descriptions.
func NewActivityTrackerWithActor(reporter Reporter, activity, actorName string) *ActivityTracker {
	return internalreporting.NewActivityTrackerWithActor(reporter, activity, actorName)
}
