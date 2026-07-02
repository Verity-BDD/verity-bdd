package expectations

import (
	"context"
	"fmt"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// NotExpectation inverts any other expectation
type NotExpectation[T any] struct {
	inner ensure.Expectation[T]
}

// Evaluate returns nil if the inner expectation fails, and an error if it passes.
// Question-resolution errors from AnswerTo expectations are propagated rather than
// treated as "expectation not met".
func (n NotExpectation[T]) Evaluate(ctx context.Context, actor core.Actor, actual T) error {
	err := n.inner.Evaluate(ctx, actor, actual)
	if err == nil {
		return fmt.Errorf("not %s: got %v", n.inner.Description(), actual)
	}
	if IsQuestionResolutionError(err) {
		return err
	}
	return nil
}

// Description returns the expectation description
func (n NotExpectation[T]) Description() string {
	return fmt.Sprintf("not %s", n.inner.Description())
}

// Not wraps an expectation and inverts its result
func Not[T any](inner ensure.Expectation[T]) ensure.Expectation[T] {
	return NotExpectation[T]{inner: inner}
}
