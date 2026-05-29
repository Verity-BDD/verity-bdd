package wait

import (
	"context"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	internalwait "github.com/verity-bdd/verity-bdd/internal/wait"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// Waiter polls a question until an expectation is met or timeout expires.
// Created by Until(). Configure with For() and CheckingEvery().
type Waiter[T any] interface {
	verity.Activity
	// For sets the maximum time to wait before the condition is considered unmet.
	// Defaults to 5s.
	For(time.Duration) Waiter[T]
	// CheckingEvery sets how frequently the condition is re-evaluated.
	// Defaults to 500ms.
	CheckingEvery(time.Duration) Waiter[T]
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

// ChannelReceiver waits for a value to arrive on a channel or for timeout.
// Created by UntilReceived(). Configure timeout with For().
type ChannelReceiver[T any] interface {
	verity.Activity
	// For sets the maximum time to wait for a value before timing out.
	// Defaults to 5s.
	For(time.Duration) ChannelReceiver[T]
}

type channelReceiverAdapter[T any] struct {
	inner *internalwait.ChannelActivity[T]
}

func (a *channelReceiverAdapter[T]) For(d time.Duration) ChannelReceiver[T] {
	a.inner.For(d)
	return a
}

func (a *channelReceiverAdapter[T]) PerformAs(ctx context.Context, actor verity.Actor) error {
	return a.inner.PerformAs(ctx, actor)
}

func (a *channelReceiverAdapter[T]) Description() string {
	return a.inner.Description()
}

func (a *channelReceiverAdapter[T]) FailureMode() verity.FailureMode {
	return a.inner.FailureMode()
}

// UntilReceived creates a wait activity that blocks until a value arrives on ch.
// Default timeout: 5s. Configure with .For().
// Returns an error if: channel closes before value arrives, or timeout/context expires.
//
// Example:
//
//	ch := make(chan string)
//	actor.AttemptsTo(
//	    wait.UntilReceived(ch),
//	    wait.UntilReceived(ch).For(30*time.Second),
//	)
func UntilReceived[T any](ch <-chan T) ChannelReceiver[T] {
	return &channelReceiverAdapter[T]{inner: internalwait.UntilReceived(ch)}
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
func Until[T any](question verity.Question[T], expectation ensure.Expectation[T]) Waiter[T] {
	return &waiterAdapter[T]{inner: internalwait.Until(question, expectation)}
}
