package core

// FailureMode defines how an Activity implementation asks the actor pipeline to
// handle an error. Built-in tasks and interactions return FailFast. The
// Interaction returned by Do has no WithFailureMode method; use a custom
// Activity implementation to select another mode.
type FailureMode int

const (
	// FailFast stops execution after the activity fails. Built-in activities use
	// this mode.
	FailFast FailureMode = iota

	// ErrorButContinue marks the test failed and allows later activities to run.
	ErrorButContinue

	// Ignore logs the error without failing the test and allows later activities to run.
	Ignore
)

// Critical returns FailFast for use by a custom Activity's FailureMode method.
func Critical() FailureMode { return FailFast }

// NonCritical returns ErrorButContinue for use by a custom Activity's
// FailureMode method.
func NonCritical() FailureMode { return ErrorButContinue }

// Optional returns Ignore for use by a custom Activity's FailureMode method.
func Optional() FailureMode { return Ignore }
