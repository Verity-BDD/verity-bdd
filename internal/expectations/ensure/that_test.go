package ensure_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	"github.com/verity-bdd/verity-bdd/internal/expectations"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
	"github.com/verity-bdd/verity-bdd/internal/expectations/testhelpers"
)

func TestEnsureActivity_PerformAs_ForwardsCtxAndActor(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "sentinel")

	mockActor := coreMocks.NewMockActor(ctrl)
	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(ctx, mockActor).Return(42, nil)
	mockQ.EXPECT().Description().Return("the value").AnyTimes()

	inner := &testhelpers.CapturingExpectation[int]{}
	activity := ensure.That[int](mockQ, inner)

	err := activity.PerformAs(ctx, mockActor)
	assert.NoError(t, err)

	assert.Equal(t, ctx, inner.CapturedCtx, "ctx should be forwarded unchanged to Evaluate")
	assert.Equal(t, mockActor, inner.CapturedActor, "actor should be forwarded unchanged to Evaluate")
}

func TestEnsureActivity_PerformAs_QuestionResolutionError_ReturnedUnwrapped(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	dbErr := errors.New("db error")
	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(42, nil)
	mockQ.EXPECT().Description().Return("outer question").AnyTimes()

	// Expectation that returns a questionResolutionError (simulating an AnswerTo failure).
	inner := &testhelpers.CapturingExpectation[int]{
		ReturnErr: ensure.NewQuestionResolutionError("inner question", dbErr),
	}

	err := ensure.That(mockQ, inner).PerformAs(context.Background(), nil)

	require.Error(t, err)
	assert.True(t, ensure.IsQuestionResolutionError(err), "questionResolutionError must pass through unwrapped")
	assert.NotContains(t, err.Error(), "expectation failed", "questionResolutionError must not be wrapped in 'expectation failed for'")
	assert.Contains(t, err.Error(), "inner question")
	assert.True(t, errors.Is(err, dbErr), "original cause must be reachable via errors.Is")
}

func TestEnsureActivity_PerformAs_NormalEvaluationError_WrappedWithContext(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(42, nil)
	mockQ.EXPECT().Description().Return("outer question").AnyTimes()

	inner := &testhelpers.CapturingExpectation[int]{ReturnErr: errors.New("value mismatch")}

	err := ensure.That(mockQ, inner).PerformAs(context.Background(), nil)

	require.Error(t, err)
	assert.False(t, ensure.IsQuestionResolutionError(err))
	assert.Contains(t, err.Error(), "expectation failed")
	assert.Contains(t, err.Error(), "outer question")
}

func TestEnsureActivity_PerformAs_EqualsAnswerTo_InnerQuestionError_ReturnsQuestionResolutionError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	ctx := context.Background()

	outerQ := coreMocks.NewMockQuestion[string](ctrl)
	outerQ.EXPECT().AnsweredBy(ctx, nil).Return("x", nil)
	outerQ.EXPECT().Description().Return("outer question").AnyTimes()

	innerQ := coreMocks.NewMockQuestion[string](ctrl)
	innerQ.EXPECT().AnsweredBy(ctx, nil).Return("", errors.New("db error"))
	innerQ.EXPECT().Description().Return("inner question").AnyTimes()

	err := ensure.That[string](outerQ, expectations.EqualsAnswerTo(innerQ)).PerformAs(ctx, nil)

	require.Error(t, err)
	assert.True(t, ensure.IsQuestionResolutionError(err), "expected questionResolutionError, got: %v", err)
}

func TestEnsureActivity_PerformAs_NotEqualsAnswerTo_InnerQuestionError_PropagatesQuestionResolutionError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	ctx := context.Background()

	outerQ := coreMocks.NewMockQuestion[string](ctrl)
	outerQ.EXPECT().AnsweredBy(ctx, nil).Return("x", nil)
	outerQ.EXPECT().Description().Return("outer question").AnyTimes()

	innerQ := coreMocks.NewMockQuestion[string](ctrl)
	innerQ.EXPECT().AnsweredBy(ctx, nil).Return("", errors.New("db error"))
	innerQ.EXPECT().Description().Return("inner question").AnyTimes()

	err := ensure.That[string](outerQ, expectations.Not(expectations.EqualsAnswerTo(innerQ))).PerformAs(ctx, nil)

	require.Error(t, err)
	assert.True(t, ensure.IsQuestionResolutionError(err), "expected questionResolutionError, got: %v", err)
}
