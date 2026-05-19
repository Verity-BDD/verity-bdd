package wait_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations"
	"github.com/verity-bdd/verity-bdd/internal/wait"
)

// stubActor реализует core.Actor для тестов
type stubActor struct{}

func (s *stubActor) Context() context.Context                                 { return context.Background() }
func (s *stubActor) Name() string                                             { return "test" }
func (s *stubActor) WhoCan(_ ...abilities.Ability) core.Actor                 { return s }
func (s *stubActor) AbilityTo(_ abilities.Ability) (abilities.Ability, error) { return nil, errors.New("no ability") }
func (s *stubActor) AttemptsTo(_ ...core.Activity)                            {}

// sequenceQuestion returns values in order; stays on the last value when exhausted
type sequenceQuestion[T any] struct {
	values []T
	idx    int
}

func (q *sequenceQuestion[T]) AnsweredBy(_ context.Context, _ core.Actor) (T, error) {
	v := q.values[min(q.idx, len(q.values)-1)]
	q.idx++
	return v, nil
}

func (q *sequenceQuestion[T]) Description() string { return "test question" }

// errorThenValueQuestion returns errors for the first errCount calls, then the value
type errorThenValueQuestion[T any] struct {
	errCount int
	value    T
	calls    int
}

func (q *errorThenValueQuestion[T]) AnsweredBy(_ context.Context, _ core.Actor) (T, error) {
	q.calls++
	if q.calls <= q.errCount {
		var zero T
		return zero, errors.New("not ready yet")
	}
	return q.value, nil
}

func (q *errorThenValueQuestion[T]) Description() string { return "error then value question" }

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestUntil_ConditionMetImmediately(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{42}}
	activity := wait.Until(q, expectations.Equals(42))

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if q.idx != 1 {
		t.Fatalf("expected exactly 1 poll, got %d", q.idx)
	}
}

func TestUntil_ConditionMetAfterRetries(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{1, 2, 42}}
	activity := wait.Until(q, expectations.Equals(42)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if q.idx != 3 {
		t.Fatalf("expected 3 polls, got %d", q.idx)
	}
}

func TestUntil_TimeoutExceeded(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(1)).
		For(50 * time.Millisecond).
		CheckingEvery(5 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	errMsg := err.Error()
	if !containsAll(errMsg, "50ms", "test question") {
		t.Fatalf("expected error to mention timeout and question, got: %v", err)
	}
}

func TestUntil_QuestionErrorThenSuccess(t *testing.T) {
	q := &errorThenValueQuestion[int]{errCount: 2, value: 99}
	activity := wait.Until(q, expectations.Equals(99)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after retries, got %v", err)
	}
	if q.calls != 3 {
		t.Fatalf("expected 3 calls (2 errors + 1 success), got %d", q.calls)
	}
}

func TestUntil_ExternalContextCancellation(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	ctx, cancel := context.WithCancel(context.Background())

	activity := wait.Until(q, expectations.Equals(1)).
		For(10 * time.Second).
		CheckingEvery(5 * time.Millisecond)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := activity.PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}
	if strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected cancellation error, not timeout: %v", err)
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

func TestUntil_ForOverridesDefaultTimeout(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}

	start := time.Now()
	// Use 60ms — much less than the 5s default, so if For() works
	// the test completes well under 1s; if broken it takes ~5s.
	activity := wait.Until(q, expectations.Equals(1)).
		For(60 * time.Millisecond).
		CheckingEvery(5 * time.Millisecond)

	_ = activity.PerformAs(context.Background(), &stubActor{})
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Fatalf("For(60ms) did not override default 5s timeout, elapsed: %v", elapsed)
	}
}

func TestUntil_Description(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(1)).For(30 * time.Second)

	desc := activity.Description()
	if !containsAll(desc, "30s", "test question") {
		t.Fatalf("unexpected description: %q", desc)
	}
}

func TestUntil_FailureMode(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(0))

	if mode := activity.FailureMode(); mode != core.FailFast {
		t.Fatalf("expected FailFast, got %v", mode)
	}
}
