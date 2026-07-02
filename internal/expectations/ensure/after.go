package ensure

import (
	"context"
	"fmt"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/core"
)

// After creates a timeout for Ensure assertion with the new API
func (e *EnsureActivity[T]) After(duration time.Duration) core.Activity {
	return &AfterActivity[T]{
		duration:    duration,
		question:    e.question,
		expectation: e.expectation,
	}
}

type AfterActivity[T any] struct {
	duration    time.Duration
	question    core.Question[T]
	expectation Expectation[T]
}

func (e *AfterActivity[T]) Description() string {
	return fmt.Sprintf("#actor waits for %v, then ensures that %s %s",
		e.duration, e.question.Description(), e.expectation.Description())
}

func (e *AfterActivity[T]) FailureMode() core.FailureMode {
	return core.FailFast
}

func (e *AfterActivity[T]) PerformAs(ctx context.Context, actor core.Actor) error {
	timer := time.NewTimer(e.duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("while waiting for %v: %w", e.duration, ctx.Err())
	case <-timer.C:
	}

	actual, err := e.question.AnsweredBy(ctx, actor)
	if err != nil {
		return fmt.Errorf("after waiting %v: question error: %w", e.duration, err)
	}
	if evalErr := e.expectation.Evaluate(ctx, actor, actual); evalErr != nil {
		return fmt.Errorf("after waiting %v: expectation failed for '%s': %w",
			e.duration, e.question.Description(), evalErr)
	}
	return nil
}
