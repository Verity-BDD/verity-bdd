package wait

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

const (
	defaultTimeout  = 5 * time.Second
	defaultInterval = 500 * time.Millisecond
)

// ConditionActivity polls question until expectation is met or timeout expires.
type ConditionActivity[T any] struct {
	timeout     time.Duration
	interval    time.Duration
	question    core.Question[T]
	expectation ensure.Expectation[T]
}

// Until creates a ConditionActivity with default timeout (5s) and interval (500ms).
func Until[T any](question core.Question[T], expectation ensure.Expectation[T]) *ConditionActivity[T] {
	return &ConditionActivity[T]{
		timeout:     defaultTimeout,
		interval:    defaultInterval,
		question:    question,
		expectation: expectation,
	}
}

func (c *ConditionActivity[T]) For(timeout time.Duration) *ConditionActivity[T] {
	c.timeout = timeout
	return c
}

func (c *ConditionActivity[T]) CheckingEvery(interval time.Duration) *ConditionActivity[T] {
	c.interval = interval
	return c
}

func (c *ConditionActivity[T]) Description() string {
	return fmt.Sprintf("#actor waits up to %v to ensure %s %s",
		c.timeout, c.question.Description(), c.expectation.Description())
}

// FailureMode always returns FailFast: a wait that silently continues after failing
// would mask flaky conditions and make test output misleading.
func (c *ConditionActivity[T]) FailureMode() core.FailureMode {
	return core.FailFast
}

// PerformAs polls immediately (poll-first, to avoid delay when condition is already met),
// then re-polls after each interval until expectation is met or timeout/context expires.
// A pre-canceled ctx will still execute one poll before the cancellation is detected.
func (c *ConditionActivity[T]) PerformAs(ctx context.Context, actor core.Actor) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	var lastErr error
	for {
		actual, err := c.question.AnsweredBy(ctx, actor)
		if err != nil {
			lastErr = err
		} else if evalErr := c.expectation.Evaluate(ctx, actor, actual); evalErr != nil {
			lastErr = evalErr
		} else {
			return nil
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("timed out after %v waiting for '%s': %w",
					c.timeout, c.question.Description(), lastErr)
			}
			return fmt.Errorf("context canceled while waiting for '%s': %w",
				c.question.Description(), lastErr)
		case <-ticker.C:
		}
	}
}
