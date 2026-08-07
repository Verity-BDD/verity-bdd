package verity_expectations

import (
	"context"

	verity "github.com/verity-bdd/verity-bdd"
	internalexpectations "github.com/verity-bdd/verity-bdd/internal/expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// ContainsSubstring checks if a string contains the expected substring.
func ContainsSubstring(substring string) ensure.Expectation[string] {
	return internalexpectations.ContainsSubstring(substring)
}

// ContainsKey checks if a map contains the expected key.
func ContainsKey(key string) ensure.Expectation[interface{}] {
	return internalexpectations.ContainsKey(key)
}

// IsGreaterThan checks if a numeric value is greater than the expected value.
func IsGreaterThan(expected interface{}) ensure.Expectation[interface{}] {
	return internalexpectations.IsGreaterThan(expected)
}

// IsLessThan checks if a numeric value is less than the expected value.
func IsLessThan(expected interface{}) ensure.Expectation[interface{}] {
	return internalexpectations.IsLessThan(expected)
}

// IsEmpty checks if a value is empty (string, slice, array, or map).
func IsEmpty[T any]() ensure.Expectation[T] {
	return internalexpectations.IsEmpty[T]()
}

// Includes checks if a slice includes the expected element.
func Includes[T any](expected T) ensure.Expectation[[]T] {
	return internalexpectations.Includes(expected)
}

// ArrayLengthEquals checks if an array, slice, or string has the expected length.
func ArrayLengthEquals[T any](expectedLength int) ensure.Expectation[T] {
	return internalexpectations.ArrayLengthEquals[T](expectedLength)
}

// Equals checks if the actual value equals the expected value using deep equality.
func Equals[T any](expected T) ensure.Expectation[T] {
	return internalexpectations.Equals(expected)
}

// Satisfies creates a custom expectation with a description and validation function.
//
// The function should return nil if the expectation is met, or an error describing the failure.
//
// Example:
//
//	actor.AttemptsTo(
//		ensure.That(ValueOf(value), Satisfies("is positive number", func(actual int) error {
//			if actual <= 0 {
//				return fmt.Errorf("expected positive value, but got %v", actual)
//			}
//			return nil
//		})),
//	)
func Satisfies[T any](description string, fn func(T) error) ensure.Expectation[T] {
	return internalexpectations.Satisfies(description, fn)
}

// Not wraps an expectation and inverts its result.
func Not[T any](inner ensure.Expectation[T]) ensure.Expectation[T] {
	return internalexpectations.Not(inner)
}

// EqualsAnswerTo checks if the actual value equals the answer to the given question.
func EqualsAnswerTo[T any](q verity.Question[T]) ensure.Expectation[T] {
	return internalexpectations.EqualsAnswerTo(q)
}

// ContainsSubstringAnswerTo checks if a string contains the answer to the given question as a substring.
func ContainsSubstringAnswerTo(q verity.Question[string]) ensure.Expectation[string] {
	return internalexpectations.ContainsSubstringAnswerTo(q)
}

// ContainsKeyAnswerTo checks if a map contains the answer to the given question as a key.
func ContainsKeyAnswerTo(q verity.Question[string]) ensure.Expectation[interface{}] {
	return internalexpectations.ContainsKeyAnswerTo(q)
}

// ArrayLengthEqualsAnswerTo checks if an array, slice, or string has the length equal to the answer of the given question.
func ArrayLengthEqualsAnswerTo[T any](q verity.Question[int]) ensure.Expectation[T] {
	return internalexpectations.ArrayLengthEqualsAnswerTo[T](q)
}

// IsGreaterThanAnswerTo checks if a value is greater than the answer to the given question.
func IsGreaterThanAnswerTo(q verity.Question[interface{}]) ensure.Expectation[interface{}] {
	return internalexpectations.IsGreaterThanAnswerTo(q)
}

// IsLessThanAnswerTo checks if a value is less than the answer to the given question.
func IsLessThanAnswerTo(q verity.Question[interface{}]) ensure.Expectation[interface{}] {
	return internalexpectations.IsLessThanAnswerTo(q)
}

// SatisfiesAnswer creates a custom expectation where the fn receives ctx and actor at evaluation time.
func SatisfiesAnswer[T any](description string, fn func(context.Context, verity.Actor, T) error) ensure.Expectation[T] {
	return internalexpectations.SatisfiesAnswer(description, fn)
}
