package expectations

import "github.com/verity-bdd/verity-bdd/internal/expectations/ensure"

func newQuestionResolutionError(description string, err error) error {
	return ensure.NewQuestionResolutionError(description, err)
}

// IsQuestionResolutionError reports whether err or any error it wraps is a
// question resolution failure (the expected-value question could not be answered).
func IsQuestionResolutionError(err error) bool {
	return ensure.IsQuestionResolutionError(err)
}
