package reporting

import "io"

// LogEntry is data emitted by a Log interaction.
type LogEntry struct {
	ActorName string
	Values    []string
}

// Reporter handles test execution reporting
//go:generate go run go.uber.org/mock/mockgen@latest -source=reporter.go -destination=mocks/mock_reporter.go -package=mocks

type Reporter interface {
	// OnLog is called when an actor emits a Log interaction.
	OnLog(entry LogEntry)

	// OnTestStart is called when a test begins
	OnTestStart(testName string)

	// OnTestFinish is called when a test completes
	OnTestFinish(result TestResult)

	// OnStepStart is called when a step/activity begins
	OnStepStart(stepDescription string)

	// OnStepFinish is called when a step/activity completes
	OnStepFinish(stepResult TestResult)

	// SetOutput sets the output destination
	SetOutput(w io.Writer)
}

// TestResult represents the result of a test or step execution
type TestResult interface {
	Name() string
	Status() Status
	Duration() float64
	Error() error
	Attachments() []Attachment
}

// Attachment represents additional data to include in a report entry.
// Content should be a serialized payload (for example, JSON).
type Attachment struct {
	Name        string
	ContentType string
	Content     []byte
}

// Status represents the status of a test or step
type Status int

const (
	StatusPassed Status = iota
	StatusFailed
	StatusSkipped
)
