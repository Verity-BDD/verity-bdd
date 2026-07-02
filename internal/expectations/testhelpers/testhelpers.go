package testhelpers

import (
	"context"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// CapturingExpectation records the ctx and actor passed to Evaluate.
// ReturnErr controls the result: nil means pass, non-nil means fail.
type CapturingExpectation[T any] struct {
	CapturedCtx   context.Context
	CapturedActor core.Actor
	ReturnErr     error
}

func (c *CapturingExpectation[T]) Evaluate(ctx context.Context, actor core.Actor, _ T) error {
	c.CapturedCtx = ctx
	c.CapturedActor = actor
	return c.ReturnErr
}

func (c *CapturingExpectation[T]) Description() string {
	return "capturing"
}

var _ ensure.Expectation[int] = (*CapturingExpectation[int])(nil)
