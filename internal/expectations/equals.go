package expectations

import (
	"context"
	"fmt"
	"reflect"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// EqualsExpectation checks if the actual value equals the expected value
type EqualsExpectation[T any] struct {
	expected T
}

// NewEquals creates a new Equals expectation
func NewEquals[T any](expected T) ensure.Expectation[T] {
	return EqualsExpectation[T]{expected: expected}
}

// Evaluate evaluates the equals expectation
func (eq EqualsExpectation[T]) Evaluate(_ context.Context, _ core.Actor, actual T) error {
	if !reflect.DeepEqual(actual, eq.expected) {
		return fmt.Errorf("expected %v, but got %v", eq.expected, actual)
	}
	return nil
}

// Description returns the expectation description
func (eq EqualsExpectation[T]) Description() string {
	return fmt.Sprintf("equals %v", eq.expected)
}

// Convenience function for creating Equals expectations
func Equals[T any](expected T) ensure.Expectation[T] {
	return NewEquals(expected)
}

// EqualsAnswerToExpectation checks if the actual value equals the answer to a question
type EqualsAnswerToExpectation[T any] struct {
	question core.Question[T]
}

// Evaluate answers the question, then delegates to EqualsExpectation
func (e EqualsAnswerToExpectation[T]) Evaluate(ctx context.Context, actor core.Actor, actual T) error {
	expected, err := e.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(e.question.Description(), err)
	}
	return EqualsExpectation[T]{expected: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (e EqualsAnswerToExpectation[T]) Description() string {
	return fmt.Sprintf("equals the answer to '%s'", e.question.Description())
}

// EqualsAnswerTo checks if the actual value equals the answer to the given question.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to Equals instead.
// If the expected-value question returns an error on any tick, the poll exits immediately
// rather than retrying — it does not count as a transient failure.
func EqualsAnswerTo[T any](q core.Question[T]) ensure.Expectation[T] {
	return EqualsAnswerToExpectation[T]{question: q}
}
