package ensure

import (
	"context"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	internalensure "github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// Expectation[T] is a condition that can be evaluated against an actual value of type T.
type Expectation[T any] interface {
	// Evaluate checks whether actual satisfies the expectation.
	Evaluate(ctx context.Context, actor verity.Actor, actual T) error
	Description() string
}

// EnsureThat[T] is the activity returned by That; it supports an optional wait via After.
type EnsureThat[T any] interface {
	verity.Activity
	After(duration time.Duration) verity.Activity
}

// That creates an activity that asserts the question's answer meets the expectation.
func That[T any](question verity.Question[T], expectation Expectation[T]) EnsureThat[T] {
	return internalensure.That(question, expectation)
}
