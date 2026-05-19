package verity_wait

import (
	"context"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	internalensure "github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
	internalwait "github.com/verity-bdd/verity-bdd/internal/wait"
)

// Waiter polls a question until an expectation is met or timeout expires.
// Created by Until(). Configure with For() and CheckingEvery().
type Waiter[T any] interface {
	For(time.Duration) Waiter[T]
	CheckingEvery(time.Duration) Waiter[T]
	PerformAs(context.Context, verity.Actor) error
	Description() string
	FailureMode() verity.FailureMode
}

type waiterAdapter[T any] struct {
	inner *internalwait.ConditionActivity[T]
}

func (a *waiterAdapter[T]) For(d time.Duration) Waiter[T] {
	a.inner.For(d)
	return a
}

func (a *waiterAdapter[T]) CheckingEvery(d time.Duration) Waiter[T] {
	a.inner.CheckingEvery(d)
	return a
}

func (a *waiterAdapter[T]) PerformAs(ctx context.Context, actor verity.Actor) error {
	return a.inner.PerformAs(ctx, actor)
}

func (a *waiterAdapter[T]) Description() string {
	return a.inner.Description()
}

func (a *waiterAdapter[T]) FailureMode() verity.FailureMode {
	return a.inner.FailureMode()
}

// Until creates a wait condition that polls question until expectation is met.
// Default timeout: 5s. Default polling interval: 500ms.
// Configure with .For() and .CheckingEvery().
//
// Example:
//
//	actor.AttemptsTo(
//	    wait.Until(someQuestion, expectations.Equals("ready")),
//	    wait.Until(someQuestion, expectations.Equals("ready")).For(30*time.Second),
//	    wait.Until(someQuestion, expectations.Equals("ready")).For(30*time.Second).CheckingEvery(1*time.Second),
//	)
func Until[T any](question verity.Question[T], expectation internalensure.Expectation[T]) Waiter[T] {
	return &waiterAdapter[T]{inner: internalwait.Until(question, expectation)}
}
