// Package answerable provides utilities for converting static values into core.Question[T] objects.
//
// The primary use case is to enable the use of static values in ensure.That() assertions:
//
//	// Static values
//	ensure.That(answerable.ValueOf(4), expectations.Equals(4))
//	ensure.That(answerable.ValueOf("hello"), expectations.Contains("ell"))
//	ensure.That(answerable.ValueOf(user), expectations.HasField("Name", "John"))
//
// For dynamic calculations backed by a function, use verity.QuestionAbout instead.
//
// ValueOf creates a Question[T] that returns the static value when asked by any actor.
// This is particularly useful when you want to assert against static values rather than
// dynamic system state.
//
// Static value examples:
//
//	// Basic types
//	ensure.That(answerable.ValueOf(42), expectations.Equals(42))
//	ensure.That(answerable.ValueOf("test"), expectations.Contains("es"))
//	ensure.That(answerable.ValueOf(true), expectations.Equals(true))
//
//	// Complex types
//	user := User{Name: "John", Age: 30}
//	ensure.That(answerable.ValueOf(user), expectations.HasField("Name", "John"))
//
//	// Error values (errors are treated as values, not as failures)
//	err := fmt.Errorf("something went wrong")
//	ensure.That(answerable.ValueOf(err), expectations.Equals(err))
//
//	// Pointers and nil handling
//	var ptr *string
//	ensure.That(answerable.ValueOf(ptr), expectations.IsNil())
//
// The created Question[T] from ValueOf is independent of any actor context - it will always
// return the same static value regardless of which actor asks the question.
package answerable

import (
	"github.com/verity-bdd/verity-bdd/internal/core"
)

// ValueOf creates a core.Question[T] that returns the provided static value
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
//   - core.Question[T]: A question that always returns the provided value
//
// Example:
//
//	q := answerable.ValueOf(42)
//	result, err := q.AnsweredBy(ctx, actor) // result = 42, err = nil
func ValueOf[T any](value T) core.Question[T] {
	return &valueQuestion[T]{value: value}
}
