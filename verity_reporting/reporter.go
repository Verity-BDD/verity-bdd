package verity_reporting

import internalreporting "github.com/verity-bdd/verity-bdd/internal/reporting"

// Reporter handles test execution reporting.
type Reporter = internalreporting.Reporter

// TestResult represents the result of a test or step execution.
type TestResult = internalreporting.TestResult

// Attachment represents additional data to include in a report entry.
// Content should be a serialized payload (for example, JSON).
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

// NewActivityTracker creates a new activity tracker.
func NewActivityTracker(reporter Reporter, activity string) *ActivityTracker {
	return internalreporting.NewActivityTracker(reporter, activity)
}

// NewActivityTrackerWithActor creates a new activity tracker with an actor name.
func NewActivityTrackerWithActor(reporter Reporter, activity, actorName string) *ActivityTracker {
	return internalreporting.NewActivityTrackerWithActor(reporter, activity, actorName)
}
