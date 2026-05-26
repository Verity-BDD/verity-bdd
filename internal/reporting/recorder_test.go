package reporting

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStepRecorder_CollectAndDrain(t *testing.T) {
	t.Parallel()
	r := &StepRecorder{}
	r.Collect(Attachment{Name: "a", ContentType: "text/plain", Content: []byte("hi")})
	r.Collect(Attachment{Name: "b", ContentType: "application/json", Content: []byte("{}")})

	drained := r.Drain()
	require.Len(t, drained, 2)
	require.Equal(t, "a", drained[0].Name)
	require.Equal(t, "b", drained[1].Name)

	require.Empty(t, r.Drain(), "second drain must be empty")
}

func TestStepRecorder_DrainEmpty(t *testing.T) {
	t.Parallel()
	r := &StepRecorder{}
	require.Nil(t, r.Drain())
}

func TestContextWithRecorder(t *testing.T) {
	t.Parallel()
	ctx, r := ContextWithRecorder(context.Background())
	r.Collect(Attachment{Name: "x"})

	got, ok := ctx.Value(recorderContextKey{}).(*StepRecorder)
	require.True(t, ok)
	require.Len(t, got.Drain(), 1)
}

func TestCollectAttachment_WithRecorder(t *testing.T) {
	t.Parallel()
	ctx, r := ContextWithRecorder(context.Background())
	CollectAttachment(ctx, Attachment{Name: "y", ContentType: "text/plain", Content: []byte("data")})
	require.Len(t, r.Drain(), 1)
}

func TestCollectAttachment_WithoutRecorder(t *testing.T) {
	t.Parallel()
	// must not panic
	CollectAttachment(context.Background(), Attachment{Name: "z"})
}
