package expectations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	expectations "github.com/verity-bdd/verity-bdd/internal/expectations"
)

func TestIsGreaterThanAnswerTo_PassesWhenActualExceedsQuestionAnswer(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(int(5), nil)
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsGreaterThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(10))
	assert.NoError(t, err)
}

func TestIsGreaterThanAnswerTo_FailsWhenQuestionErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(nil, errors.New("q failed"))
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsGreaterThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(10))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "q failed")
}

func TestIsLessThanAnswerTo_PassesWhenActualIsBelowQuestionAnswer(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(int(10), nil)
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsLessThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(5))
	assert.NoError(t, err)
}

func TestIsLessThanAnswerTo_FailsWhenQuestionErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(nil, errors.New("q failed"))
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsLessThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(5))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "q failed")
}

func TestIsGreaterThanAnswerTo_FailsWhenActualIsNotGreater(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(int(10), nil)
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsGreaterThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(5))
	require.Error(t, err)
}

func TestIsLessThanAnswerTo_FailsWhenActualIsNotLess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[interface{}](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(int(5), nil)
	mockQ.EXPECT().Description().Return("the threshold").AnyTimes()

	err := expectations.IsLessThanAnswerTo(mockQ).Evaluate(context.Background(), nil, int(10))
	require.Error(t, err)
}
