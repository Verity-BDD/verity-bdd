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
)

func TestEqualsAnswerTo_PassesWhenQuestionAnswerMatches(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(42, nil)
	mockQ.EXPECT().Description().Return("the answer").AnyTimes()

	err := expectations.EqualsAnswerTo(mockQ).Evaluate(context.Background(), nil, 42)
	assert.NoError(t, err)
}

func TestEqualsAnswerTo_FailsWhenQuestionErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(0, errors.New("db error"))
	mockQ.EXPECT().Description().Return("the answer").AnyTimes()

	err := expectations.EqualsAnswerTo(mockQ).Evaluate(context.Background(), nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestEqualsAnswerTo_FailsWhenValuesDoNotMatch(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return(42, nil)
	mockQ.EXPECT().Description().Return("the answer").AnyTimes()

	err := expectations.EqualsAnswerTo(mockQ).Evaluate(context.Background(), nil, 99)
	require.Error(t, err)
}
