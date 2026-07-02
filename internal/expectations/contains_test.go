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

func TestContainsAnswerTo_PassesWhenQuestionAnswerMatches(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("world", nil)
	mockQ.EXPECT().Description().Return("the substring").AnyTimes()

	err := expectations.ContainsAnswerTo(mockQ).Evaluate(context.Background(), nil, "hello world")
	assert.NoError(t, err)
}

func TestContainsAnswerTo_FailsWhenQuestionErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("", errors.New("q failed"))
	mockQ.EXPECT().Description().Return("the substring").AnyTimes()

	err := expectations.ContainsAnswerTo(mockQ).Evaluate(context.Background(), nil, "hello world")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "q failed")
}

func TestContainsKeyAnswerTo_PassesWhenQuestionAnswerMatches(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("name", nil)
	mockQ.EXPECT().Description().Return("the key").AnyTimes()

	err := expectations.ContainsKeyAnswerTo(mockQ).Evaluate(context.Background(), nil, map[string]interface{}{"name": "Alice"})
	assert.NoError(t, err)
}

func TestContainsKeyAnswerTo_FailsWhenQuestionErrors(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("", errors.New("q failed"))
	mockQ.EXPECT().Description().Return("the key").AnyTimes()

	err := expectations.ContainsKeyAnswerTo(mockQ).Evaluate(context.Background(), nil, map[string]interface{}{"name": "Alice"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "q failed")
}

func TestContainsAnswerTo_FailsWhenSubstringNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("missing", nil)
	mockQ.EXPECT().Description().Return("the substring").AnyTimes()

	err := expectations.ContainsAnswerTo(mockQ).Evaluate(context.Background(), nil, "hello world")
	require.Error(t, err)
}

func TestContainsKeyAnswerTo_FailsWhenKeyNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockQ := coreMocks.NewMockQuestion[string](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Return("age", nil)
	mockQ.EXPECT().Description().Return("the key").AnyTimes()

	err := expectations.ContainsKeyAnswerTo(mockQ).Evaluate(context.Background(), nil, map[string]interface{}{"name": "Alice"})
	require.Error(t, err)
}
