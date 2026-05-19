package verity_wait_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/verity_wait"
	"github.com/verity-bdd/verity-bdd/verity_expectations"
)

type testActor struct{}

func (a *testActor) Context() context.Context                                 { return context.Background() }
func (a *testActor) Name() string                                             { return "test" }
func (a *testActor) WhoCan(_ ...abilities.Ability) core.Actor                 { return a }
func (a *testActor) AbilityTo(_ abilities.Ability) (abilities.Ability, error) { return nil, errors.New("no ability") }
func (a *testActor) AttemptsTo(_ ...core.Activity)                            {}

type staticQuestion[T any] struct{ value T }

func (q *staticQuestion[T]) AnsweredBy(_ context.Context, _ core.Actor) (T, error) {
	return q.value, nil
}
func (q *staticQuestion[T]) Description() string { return "static value" }

func TestPublicUntil_ConditionMet(t *testing.T) {
	q := &staticQuestion[string]{value: "ready"}
	activity := verity_wait.Until(q, verity_expectations.Equals("ready"))

	err := activity.PerformAs(context.Background(), &testActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPublicUntil_ChainedForAndCheckingEvery(t *testing.T) {
	q := &staticQuestion[int]{value: 0}
	activity := verity_wait.Until(q, verity_expectations.Equals(1)).
		For(50 * time.Millisecond).
		CheckingEvery(5 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &testActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
