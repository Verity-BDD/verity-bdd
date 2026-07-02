package expectations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	"github.com/verity-bdd/verity-bdd/internal/expectations"
	"github.com/verity-bdd/verity-bdd/internal/expectations/testhelpers"
)

func TestNot_PassesWhenInnerFails(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Equals("foo")).Evaluate(context.Background(), nil, "bar")
	assert.NoError(t, err)
}

func TestNot_FailsWhenInnerPasses(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Equals("foo")).Evaluate(context.Background(), nil, "foo")
	require.Error(t, err)
	assert.Equal(t, "not equals foo: got foo", err.Error())
}

func TestNot_Description(t *testing.T) {
	t.Parallel()
	desc := expectations.Not(expectations.Equals("foo")).Description()
	assert.Equal(t, "not equals foo", desc)
}

func TestNot_IsEmpty_FailsOnEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.IsEmpty[string]()).Evaluate(context.Background(), nil, "")
	require.Error(t, err)
	assert.Equal(t, "not is empty: got ", err.Error())
}

func TestNot_IsEmpty_PassesOnNonEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.IsEmpty[string]()).Evaluate(context.Background(), nil, "hello")
	assert.NoError(t, err)
}

func TestNot_DoubleNegation_PassesWhenInnerPasses(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Not(expectations.Equals("foo"))).Evaluate(context.Background(), nil, "foo")
	assert.NoError(t, err)
}

func TestNot_DoubleNegation_FailsWhenInnerFails(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Not(expectations.Equals("foo"))).Evaluate(context.Background(), nil, "bar")
	require.Error(t, err)
	assert.Equal(t, "not not equals foo: got bar", err.Error())
}

func TestNot_ForwardsCtxAndActorToInner(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockActor := coreMocks.NewMockActor(ctrl)

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "sentinel")

	inner := &testhelpers.CapturingExpectation[string]{ReturnErr: assert.AnError}
	err := expectations.Not[string](inner).Evaluate(ctx, mockActor, "anything")
	assert.NoError(t, err) // Not inverts the inner failure

	assert.Equal(t, ctx, inner.CapturedCtx, "ctx should be forwarded unchanged")
	assert.Equal(t, mockActor, inner.CapturedActor, "actor should be forwarded unchanged")
}

func TestNot_PropagatesQuestionErrorFromInnerAnswerTo(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	dbErr := errors.New("db error")
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("", dbErr)
	mockQ.EXPECT().Description().Return("the value").AnyTimes()

	err := expectations.Not(expectations.EqualsAnswerTo(mockQ)).Evaluate(context.Background(), nil, "anything")
	require.Error(t, err, "Not should propagate question-resolution errors rather than swallowing them")
	assert.True(t, errors.Is(err, dbErr), "propagated error should wrap the original question error")
}
