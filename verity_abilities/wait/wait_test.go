package wait_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/verity_abilities/wait"
	"github.com/verity-bdd/verity-bdd/verity_expectations"
)

type testActor struct{}

func (a *testActor) Context() context.Context                 { return context.Background() }
func (a *testActor) Name() string                             { return "test" }
func (a *testActor) WhoCan(_ ...abilities.Ability) core.Actor { return a }
func (a *testActor) AbilityTo(_ abilities.Ability) (abilities.Ability, error) {
	return nil, errors.New("no ability")
}
func (a *testActor) AttemptsTo(_ ...core.Activity) {}

type staticQuestion[T any] struct{ value T }

func (q *staticQuestion[T]) AnsweredBy(_ context.Context, _ core.Actor) (T, error) {
	return q.value, nil
}
func (q *staticQuestion[T]) Description() string { return "static value" }

func TestPublicUntil_ConditionMet(t *testing.T) {
	t.Parallel()
	q := &staticQuestion[string]{value: "ready"}
	activity := wait.Until(q, verity_expectations.Equals("ready"))

	err := activity.PerformAs(context.Background(), &testActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPublicUntilReceived_ReceivesValue(t *testing.T) {
	t.Parallel()
	ch := make(chan string, 1)
	ch <- "event"

	activity := wait.UntilReceived(ch)
	err := activity.PerformAs(context.Background(), &testActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPublicUntilReceived_TimeoutWithNoSend(t *testing.T) {
	t.Parallel()
	ch := make(chan string)
	activity := wait.UntilReceived(ch).For(50 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &testActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestPublicUntilReceived_ClosedChannel(t *testing.T) {
	t.Parallel()
	ch := make(chan string)
	close(ch)

	activity := wait.UntilReceived(ch).For(10 * time.Second)
	err := activity.PerformAs(context.Background(), &testActor{})

	if err == nil {
		t.Fatal("expected error for closed channel, got nil")
	}
}

func TestPublicUntilReceived_ChainedFor(t *testing.T) {
	t.Parallel()
	ch := make(chan int)
	start := time.Now()
	activity := wait.UntilReceived(ch).For(50 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &testActor{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("For(50ms) did not shorten timeout, elapsed: %v", elapsed)
	}
}

func TestPublicUntil_ChainedForAndCheckingEvery(t *testing.T) {
	t.Parallel()
	q := &staticQuestion[int]{value: 0}
	activity := wait.Until(q, verity_expectations.Equals(1)).
		For(50 * time.Millisecond).
		CheckingEvery(5 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &testActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
