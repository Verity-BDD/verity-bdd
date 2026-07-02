package expectations

import (
	"context"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// SatisfiesExpectation represents a custom expectation that evaluates using a provided function
type SatisfiesExpectation[T any] struct {
	description string
	fn          func(T) error
}

// Satisfies creates a new custom expectation with a description and validation function
//
// The function should return nil if the expectation is met, or an error describing the failure.
//
// Example:
//
//	actor.AttemptsTo(
//		ensure.That(answerable.ValueOf(value), expectations.Satisfies("is positive number", func(actual int) error {
//			if actual <= 0 {
//				return fmt.Errorf("expected positive value, but got %v", actual)
//			}
//			return nil
//		})),
//	)
func Satisfies[T any](description string, fn func(T) error) ensure.Expectation[T] {
	return &SatisfiesExpectation[T]{
		description: description,
		fn:          fn,
	}
}

// Evaluate evaluates the expectation by calling the provided function
func (s *SatisfiesExpectation[T]) Evaluate(_ context.Context, _ core.Actor, actual T) error {
	return s.fn(actual)
}

// Description returns the custom description of the expectation
func (s *SatisfiesExpectation[T]) Description() string {
	return s.description
}

// SatisfiesAnswerExpectation represents a custom expectation whose fn receives ctx and actor
type SatisfiesAnswerExpectation[T any] struct {
	description string
	fn          func(context.Context, core.Actor, T) error
}

// SatisfiesAnswer creates a custom expectation where the fn receives ctx and actor at evaluation time
func SatisfiesAnswer[T any](description string, fn func(context.Context, core.Actor, T) error) ensure.Expectation[T] {
	return &SatisfiesAnswerExpectation[T]{
		description: description,
		fn:          fn,
	}
}

// Evaluate calls the provided function with ctx, actor, and actual
func (s *SatisfiesAnswerExpectation[T]) Evaluate(ctx context.Context, actor core.Actor, actual T) error {
	return s.fn(ctx, actor, actual)
}

// Description returns the custom description of the expectation
func (s *SatisfiesAnswerExpectation[T]) Description() string {
	return s.description
}
