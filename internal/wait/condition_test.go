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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

func TestUntil_ForOverridesDefaultTimeout(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(1)).For(30 * time.Second)

	const expected = "#actor waits up to 30s to ensure test question equals 1"
	if actual := activity.Description(); actual != expected {
		t.Fatalf("expected description %q, got %q", expected, actual)
	}
}

func TestUntil_ContextAlreadyCanceled(t *testing.T) {
	t.Parallel()
	q := &sequenceQuestion[int]{values: []int{0}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	activity := wait.Until(q, expectations.Equals(1)).
		For(10 * time.Second)

	err := activity.PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected error for pre-canceled context, got nil")
	}
	if strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected cancellation error, not timeout: %v", err)
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

func TestUntil_IntervalExceedsTimeout(t *testing.T) {
	t.Parallel()
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

func TestUntil_AnswerToExpectation_SucceedsAndReevaluatesPerTick(t *testing.T) {
	t.Parallel()
	// outerQ returns "bar" on the first poll, then "foo" on subsequent polls.
	outerQ := &sequenceQuestion[string]{values: []string{"bar", "foo"}}
	// innerQ (the expected value) always returns "foo".
	innerQ := &sequenceQuestion[string]{values: []string{"foo"}}

	activity := wait.Until(outerQ, expectations.EqualsAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if outerQ.idx != 2 {
		t.Fatalf("expected 2 outer polls (1 miss + 1 hit), got %d", outerQ.idx)
	}
	// innerQ is re-answered on every poll tick — once per outer poll.
	if innerQ.idx != 2 {
		t.Fatalf("expected 2 inner polls (re-evaluated per tick), got %d", innerQ.idx)
	}
}

// ctxCheckingQuestion returns ctx.Err() if the context is already done, value otherwise.
// Used to simulate a well-behaved question that respects context cancellation.
type ctxCheckingQuestion[T any] struct {
	value T
}

func (q *ctxCheckingQuestion[T]) AnsweredBy(ctx context.Context, _ core.Actor) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	return q.value, nil
}

func (q *ctxCheckingQuestion[T]) Description() string { return "ctx-checking question" }

func TestUntil_AnswerToExpectation_ExpiredCtxProducesCanceledError(t *testing.T) {
	t.Parallel()
	// outerQ ignores ctx and always returns a value — simulates a synchronous question
	// that can succeed even after the context deadline has passed.
	outerQ := &sequenceQuestion[string]{values: []string{"anything"}}
	// innerQ is ctx-aware: returns ctx.Err() when the context is already done.
	innerQ := &ctxCheckingQuestion[string]{value: "target"}

	// Pre-cancel the context so the deadline race is deterministic: outerQ succeeds
	// (ignores ctx), then innerQ returns context.Canceled. That error flows to the
	// select branch in the wait loop, which surfaces it as the "context canceled"
	// message rather than a transient poll failure.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := wait.Until(outerQ, expectations.EqualsAnswerTo(innerQ)).
		For(5*time.Second).
		PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected 'canceled' in error message, got: %v", err)
	}
}

func TestUntil_FailureMode(t *testing.T) {
	t.Parallel()
	q := &sequenceQuestion[int]{values: []int{0}}
	activity := wait.Until(q, expectations.Equals(0))

	if mode := activity.FailureMode(); mode != core.FailFast {
		t.Fatalf("expected FailFast, got %v", mode)
	}
}

func TestUntil_AnswerToExpectation_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"target"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 3, value: "target"}

	activity := wait.Until(outerQ, expectations.EqualsAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_AnswerToExpectation_TimesOutWhenInnerQuestionAlwaysErrors(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"anything"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 999, value: "never"}

	start := time.Now()
	err := wait.Until(outerQ, expectations.EqualsAnswerTo(innerQ)).
		For(50*time.Millisecond).
		CheckingEvery(1*time.Millisecond).
		PerformAs(context.Background(), &stubActor{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected error to contain \"timed out\", got: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected elapsed time >= 50ms (did not fast-fail), got: %v", elapsed)
	}
}

func TestUntil_AnswerToExpectation_TimeoutErrorContainsInnerQuestionErrorText(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"anything"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 999, value: "never"}

	err := wait.Until(outerQ, expectations.EqualsAnswerTo(innerQ)).
		For(50*time.Millisecond).
		CheckingEvery(1*time.Millisecond).
		PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "not ready yet") {
		t.Fatalf("expected timeout error to contain inner question error text \"not ready yet\", got: %v", err)
	}
}

func TestUntil_NotEqualsAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"target"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 3, value: "other"}

	activity := wait.Until(outerQ, expectations.Not(expectations.EqualsAnswerTo(innerQ))).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_NotEqualsAnswerTo_TimesOutWhenInnerQuestionAlwaysErrors(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"anything"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 999, value: "never"}

	start := time.Now()
	err := wait.Until(outerQ, expectations.Not(expectations.EqualsAnswerTo(innerQ))).
		For(50*time.Millisecond).
		CheckingEvery(1*time.Millisecond).
		PerformAs(context.Background(), &stubActor{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected error to contain \"timed out\", got: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected elapsed time >= 50ms (did not fast-fail), got: %v", elapsed)
	}
}

func TestUntil_ContainsSubstringAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[string]{values: []string{"hello world"}}
	innerQ := &errorThenValueQuestion[string]{errCount: 2, value: "world"}

	activity := wait.Until(outerQ, expectations.ContainsSubstringAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_IsGreaterThanAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[interface{}]{values: []interface{}{int(10)}}
	innerQ := &errorThenValueQuestion[interface{}]{errCount: 2, value: int(5)}

	activity := wait.Until(outerQ, expectations.IsGreaterThanAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_ArrayLengthEqualsAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[[]string]{values: [][]string{{"a", "b", "c"}}}
	innerQ := &errorThenValueQuestion[int]{errCount: 2, value: 3}

	activity := wait.Until(outerQ, expectations.ArrayLengthEqualsAnswerTo[[]string](innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_IsLessThanAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[interface{}]{values: []interface{}{int(3)}}
	innerQ := &errorThenValueQuestion[interface{}]{errCount: 2, value: int(10)}

	activity := wait.Until(outerQ, expectations.IsLessThanAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}

func TestUntil_ContainsKeyAnswerTo_RetriesOnTransientInnerQuestionError(t *testing.T) {
	t.Parallel()
	outerQ := &sequenceQuestion[interface{}]{values: []interface{}{map[string]interface{}{"foo": 1}}}
	innerQ := &errorThenValueQuestion[string]{errCount: 2, value: "foo"}

	activity := wait.Until(outerQ, expectations.ContainsKeyAnswerTo(innerQ)).
		CheckingEvery(1 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error after transient inner question errors, got %v", err)
	}
}
