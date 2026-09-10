package verity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	verity "github.com/verity-bdd/verity-bdd"
)

type factContextKey struct{}

type customerAccountFact struct {
	setupActor    verity.Actor
	teardownActor verity.Actor
}

func (f *customerAccountFact) Description() string { return "customer account" }

func (f *customerAccountFact) Setup(_ context.Context, actor verity.Actor) error {
	f.setupActor = actor
	return nil
}

func (f *customerAccountFact) Teardown(_ context.Context, actor verity.Actor) error {
	f.teardownActor = actor
	return nil
}

func TestActorHasNoFactsIsNoOp(t *testing.T) {
	t.Parallel()

	test := verity.NewVerityTest(t, verity.Scene{})
	test.ActorCalled("Sam").Has()
}

func TestActorFactTearsDownOnShutdown(t *testing.T) {
	t.Parallel()

	test := verity.NewVerityTest(t, verity.Scene{})
	actor := test.ActorCalled("Sam")
	fact := &customerAccountFact{}

	actor.Has(fact)

	require.Same(t, actor, fact.setupActor)
	require.Nil(t, fact.teardownActor)
	test.Shutdown()
	require.Same(t, actor, fact.teardownActor)
}

func TestActorHasSetsUpFactSynchronously(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), factContextKey{}, "test context")
	test := verity.NewVerityTest(t, verity.Scene{Context: ctx})
	actor := test.ActorCalled("Sam")
	setupCalled := false

	fact := verity.FactAbout("customer account", func(gotCtx context.Context, gotActor verity.Actor) error {
		require.Same(t, ctx, gotCtx)
		require.Same(t, actor, gotActor)
		setupCalled = true
		return nil
	})

	require.Equal(t, "customer account", fact.Description())
	actor.Has(fact)
	require.True(t, setupCalled)
}
