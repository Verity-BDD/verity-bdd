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
	WithFailureMode(verity.FailureMode) EnsureThat[T]
}

// That creates an activity that asserts the question's answer meets the expectation.
func That[T any](question verity.Question[T], expectation Expectation[T]) EnsureThat[T] {
	return &ensureThatAdapter[T]{inner: internalensure.That(question, expectation)}
}

type ensureThatAdapter[T any] struct {
	inner *internalensure.EnsureActivity[T]
}

func (a *ensureThatAdapter[T]) PerformAs(ctx context.Context, actor verity.Actor) error {
	return a.inner.PerformAs(ctx, actor)
}

func (a *ensureThatAdapter[T]) Description() string {
	return a.inner.Description()
}

func (a *ensureThatAdapter[T]) FailureMode() verity.FailureMode {
	return a.inner.FailureMode()
}

func (a *ensureThatAdapter[T]) After(duration time.Duration) verity.Activity {
	return a.inner.After(duration)
}

func (a *ensureThatAdapter[T]) WithFailureMode(mode verity.FailureMode) EnsureThat[T] {
	a.inner.WithFailureMode(mode)
	return a
}
