package ensure_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
	"github.com/verity-bdd/verity-bdd/internal/expectations/testhelpers"
)

func TestAfterActivity_PerformAs_StopsWhenParentContextIsCanceled(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockQ := coreMocks.NewMockQuestion[int](ctrl)
	mockQ.EXPECT().AnsweredBy(gomock.Any(), gomock.Any()).Times(0)

	activity := ensure.That(mockQ, &testhelpers.CapturingExpectation[int]{}).
		After(10 * time.Millisecond)

	err := activity.PerformAs(ctx, nil)

	assert.ErrorIs(t, err, context.Canceled)
}
