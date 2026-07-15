package expectations

import (
	"context"
	"fmt"
	"reflect"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// IsEmptyExpectation checks if a value is empty (string, slice, array, or map)
type IsEmptyExpectation[T any] struct{}

// Evaluate evaluates the is empty expectation
func (ie IsEmptyExpectation[T]) Evaluate(_ context.Context, _ core.Actor, actual T) error {
	val := reflect.ValueOf(actual)

	switch val.Kind() {
	case reflect.String:
		if val.String() != "" {
			return fmt.Errorf("expected string to be empty, but got '%s'", val.String())
		}
	case reflect.Slice, reflect.Array:
		if val.Len() != 0 {
			return fmt.Errorf("expected slice/array to be empty, but got %d elements", val.Len())
		}
	case reflect.Map:
		if val.Len() != 0 {
			return fmt.Errorf("expected map to be empty, but got %d elements", val.Len())
		}
	default:
		return fmt.Errorf("IsEmpty expectation only works with strings, slices, arrays, and maps, but got %T", actual)
	}

	return nil
}

// Description returns the expectation description
func (ie IsEmptyExpectation[T]) Description() string {
	return "is empty"
}

// IsEmpty creates an IsEmpty expectation for the given type
func IsEmpty[T any]() ensure.Expectation[T] {
	return IsEmptyExpectation[T]{}
}

// ArrayLengthEqualsExpectation checks if an array/slice/string has the expected length
type ArrayLengthEqualsExpectation[T any] struct {
	expectedLength int
}

// Evaluate evaluates the array length expectation
func (ale ArrayLengthEqualsExpectation[T]) Evaluate(_ context.Context, _ core.Actor, actual T) error {
	val := reflect.ValueOf(actual)

	var length int
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		length = val.Len()
	case reflect.String:
		length = len(val.String())
	default:
		return fmt.Errorf("ArrayLengthEquals expectation only works with arrays, slices, and strings, but got %T", actual)
	}

	if length != ale.expectedLength {
		return fmt.Errorf("expected length to be %d, but got %d", ale.expectedLength, length)
	}
	return nil
}

// Description returns the expectation description
func (ale ArrayLengthEqualsExpectation[T]) Description() string {
	return fmt.Sprintf("has length %d", ale.expectedLength)
}

// ArrayLengthEquals creates an ArrayLengthEquals expectation for the given type
func ArrayLengthEquals[T any](expectedLength int) ensure.Expectation[T] {
	return ArrayLengthEqualsExpectation[T]{expectedLength: expectedLength}
}

// ArrayLengthEqualsAnswerToExpectation checks if an array/slice/string has the length equal to the answer of a question
type ArrayLengthEqualsAnswerToExpectation[T any] struct {
	question core.Question[int]
}

// Evaluate answers the question, then delegates to ArrayLengthEqualsExpectation
func (a ArrayLengthEqualsAnswerToExpectation[T]) Evaluate(ctx context.Context, actor core.Actor, actual T) error {
	expected, err := a.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(a.question.Description(), err)
	}
	return ArrayLengthEqualsExpectation[T]{expectedLength: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (a ArrayLengthEqualsAnswerToExpectation[T]) Description() string {
	return fmt.Sprintf("has length equal to the answer to '%s'", a.question.Description())
}

// ArrayLengthEqualsAnswerTo checks if an array/slice/string has the length equal to the answer of the given question.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to ArrayLengthEquals instead.
func ArrayLengthEqualsAnswerTo[T any](q core.Question[int]) ensure.Expectation[T] {
	return ArrayLengthEqualsAnswerToExpectation[T]{question: q}
}
