package reporting

import "io"

// Reporter handles test execution reporting
type Reporter interface {
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

// Link identifies a requirement, ticket, or other related resource.
type Link struct {
	Name string
	URL  string
	Type string
}

// TestMetadata is optional descriptive context for a test result.
type TestMetadata struct {
	Description string
	Links       []Link
}

// TestMetadataProvider is implemented by test results that carry metadata.
// Reporters can type-assert this interface without changing the TestResult contract.
type TestMetadataProvider interface {
	Metadata() TestMetadata
}

// Status represents the status of a test or step
type Status int

const (
	StatusPassed Status = iota
	StatusFailed
	StatusSkipped
)
