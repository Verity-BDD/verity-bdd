package verity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	verity "github.com/verity-bdd/verity-bdd"
)

type factContextKey struct{}

func TestActorHasNoFactsIsNoOp(t *testing.T) {
	t.Parallel()

	test := verity.NewVerityTest(t, verity.Scene{})
	test.ActorCalled("Sam").Has()
}

func TestActorFactTearsDownOnShutdown(t *testing.T) {
	t.Parallel()

	test := verity.NewVerityTest(t, verity.Scene{})
	actor := test.ActorCalled("Sam")
	teardownCalled := false

	actor.Has(verity.FactAboutWithTeardown(
		"has a customer account",
		func(context.Context, verity.Actor) error { return nil },
		func(_ context.Context, gotActor verity.Actor) error {
			require.Same(t, actor, gotActor)
			teardownCalled = true
			return nil
		},
	))

	require.False(t, teardownCalled)
	test.Shutdown()
	require.True(t, teardownCalled)
}

func TestActorHasSetsUpFactSynchronously(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), factContextKey{}, "test context")
	test := verity.NewVerityTest(t, verity.Scene{Context: ctx})
	actor := test.ActorCalled("Sam")
	setupCalled := false

	fact := verity.FactAbout("has a customer account", func(gotCtx context.Context, gotActor verity.Actor) error {
		require.Same(t, ctx, gotCtx)
		require.Same(t, actor, gotActor)
		setupCalled = true
		return nil
	})

	require.Equal(t, "has a customer account", fact.Description())
	actor.Has(fact)
	require.True(t, setupCalled)
}
