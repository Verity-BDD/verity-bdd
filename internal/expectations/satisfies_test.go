package expectations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/core"
	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	"github.com/verity-bdd/verity-bdd/internal/expectations"
)

func TestSatisfiesAnswer_PassesWhenPredicateReturnsNil(t *testing.T) {
	t.Parallel()
	err := expectations.SatisfiesAnswer[int]("is positive", func(_ context.Context, _ core.Actor, actual int) error {
		if actual > 0 {
			return nil
		}
		return errors.New("not positive")
	}).Evaluate(context.Background(), nil, 42)
	assert.NoError(t, err)
}

func TestSatisfiesAnswer_FailsWhenPredicateReturnsError(t *testing.T) {
	t.Parallel()
	err := expectations.SatisfiesAnswer[int]("is positive", func(_ context.Context, _ core.Actor, actual int) error {
		if actual > 0 {
			return nil
		}
		return errors.New("not positive")
	}).Evaluate(context.Background(), nil, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not positive")
}

func TestSatisfiesAnswer_ForwardsCtxAndActorToFn(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockActor := coreMocks.NewMockActor(ctrl)

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "sentinel")

	var capturedCtx context.Context
	var capturedActor core.Actor

	err := expectations.SatisfiesAnswer[int]("capturing", func(c context.Context, a core.Actor, _ int) error {
		capturedCtx = c
		capturedActor = a
		return nil
	}).Evaluate(ctx, mockActor, 0)

	assert.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx, "ctx should be forwarded unchanged to fn")
	assert.Equal(t, mockActor, capturedActor, "actor should be forwarded unchanged to fn")
}
