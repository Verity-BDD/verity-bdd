package wait

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/core"
)

// ChannelActivity blocks until a value arrives on the channel, or timeout/context cancellation.
type ChannelActivity[T any] struct {
	timeout time.Duration
	ch      <-chan T
}

// UntilReceived creates a ChannelActivity with the default timeout (5s).
func UntilReceived[T any](ch <-chan T) *ChannelActivity[T] {
	return &ChannelActivity[T]{timeout: defaultTimeout, ch: ch}
}

func (c *ChannelActivity[T]) For(d time.Duration) *ChannelActivity[T] {
	c.timeout = d
	return c
}

func (c *ChannelActivity[T]) Description() string {
	return fmt.Sprintf("wait up to %v to receive from channel", c.timeout)
}

// FailureMode always returns FailFast: a wait that silently continues after failing
// would mask flaky conditions and make test output misleading.
func (c *ChannelActivity[T]) FailureMode() core.FailureMode {
	return core.FailFast
}

func (c *ChannelActivity[T]) PerformAs(ctx context.Context, _ core.Actor) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	select {
	case _, ok := <-c.ch:
		if !ok {
			return fmt.Errorf("channel closed while waiting to receive")
		}
		return nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timed out after %v waiting to receive from channel", c.timeout)
		}
		return fmt.Errorf("context cancelled while waiting to receive from channel")
	}
}
