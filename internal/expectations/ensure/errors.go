package ensure

import (
	"errors"
	"fmt"
)

// questionResolutionError is returned by AnswerTo expectations when the expected-value
// question cannot be answered. Composite expectations (Not, etc.) and polling activities
// check for this type to distinguish infrastructure failures from value-mismatch failures.
type questionResolutionError struct {
	question string
	err      error
}

// NewQuestionResolutionError creates an error representing a failure to answer an
// expected-value question inside an AnswerTo expectation.
func NewQuestionResolutionError(description string, err error) error {
	return &questionResolutionError{question: description, err: err}
}

func (e *questionResolutionError) Error() string {
	return fmt.Sprintf("failed to answer question '%s': %v", e.question, e.err)
}

func (e *questionResolutionError) Unwrap() error {
	return e.err
}

// IsQuestionResolutionError reports whether err or any error it wraps is a
// question resolution failure (the expected-value question could not be answered).
func IsQuestionResolutionError(err error) bool {
	var qErr *questionResolutionError
	return errors.As(err, &qErr)
}
