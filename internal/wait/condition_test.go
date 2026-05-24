package wait_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations"
	"github.com/verity-bdd/verity-bdd/internal/wait"
)

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

// signalOnFirstPollQuestion signals via a channel on the first AnsweredBy call, then returns value every time
type signalOnFirstPollQuestion[T any] struct {
	value  T
	polled chan struct{}
	once   sync.Once
}

func (q *signalOnFirstPollQuestion[T]) AnsweredBy(_ context.Context, _ core.Actor) (T, error) {
	q.once.Do(func() { close(q.polled) })
	return q.value, nil
}

func (q *signalOnFirstPollQuestion[T]) Description() string { return "signal on first poll" }

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

func TestUntil_TimeoutWithPersistentQuestionError(t *testing.T) {
	q := &errorThenValueQuestion[int]{errCount: 1000, value: 0}
	activity := wait.Until(q, expectations.Equals(0)).
		For(50 * time.Millisecond).
		CheckingEvery(5 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	// error must wrap the underlying question error
	if !strings.Contains(err.Error(), "not ready yet") {
		t.Fatalf("expected wrapped question error in message, got: %v", err)
	}
}

func TestUntil_ExternalContextCancellation(t *testing.T) {
	polled := make(chan struct{})
	q := &signalOnFirstPollQuestion[int]{value: 0, polled: polled}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	activity := wait.Until(q, expectations.Equals(1)).
		For(10 * time.Second).
		CheckingEvery(5 * time.Millisecond)

	go func() {
		<-polled // wait until at least one poll happened
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

func TestUntil_ContextAlreadyCancelled(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	activity := wait.Until(q, expectations.Equals(1)).
		For(10 * time.Second)

	err := activity.PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected error for pre-cancelled context, got nil")
	}
	if strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected cancellation error, not timeout: %v", err)
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

func TestUntil_IntervalExceedsTimeout(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	// interval (100ms) > timeout (10ms): only one poll should happen before timeout
	activity := wait.Until(q, expectations.Equals(1)).
		For(10 * time.Millisecond).
		CheckingEvery(100 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if q.idx != 1 {
		t.Fatalf("expected exactly 1 poll when interval > timeout, got %d", q.idx)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestUntil_FailureMode(t *testing.T) {
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(0))

	if mode := activity.FailureMode(); mode != core.FailFast {
		t.Fatalf("expected FailFast, got %v", mode)
	}
}
