package verity_answerable

import (
	verity "github.com/verity-bdd/verity-bdd"
	internalanswerable "github.com/verity-bdd/verity-bdd/internal/answerable"
)

// ValueOf creates a Question[T] that returns the provided static value
// when answered by any actor.
//
// The value is treated as-is, even if it's an error type. This means that
// error values are passed through as the answer rather than being treated
// as failure conditions.
//
// Parameters:
//   - value: The static value to be wrapped as a Question
//
// Returns:
//   - Question[T]: A question that always returns the provided value
//
// Example:
//
//	q := ValueOf(42)
//	result, err := q.AnsweredBy(actor) // result = 42, err = nil
func ValueOf[T any](value T) verity.Question[T] {
	return internalanswerable.ValueOf(value)
}
