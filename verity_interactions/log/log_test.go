package log_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	verity "github.com/verity-bdd/verity-bdd"
	log "github.com/verity-bdd/verity-bdd/verity_interactions/log"
	"github.com/verity-bdd/verity-bdd/verity_reporting"
)

type recordingActor struct {
	name    string
	ctx     context.Context
	entries []verity_reporting.LogEntry
}

func (a *recordingActor) Context() context.Context { return a.ctx }
func (a *recordingActor) Name() string             { return a.name }
func (a *recordingActor) WhoCan(...verity.Ability) verity.Actor {
	return a
}
func (a *recordingActor) AbilityTo(verity.Ability) (verity.Ability, error) {
	return nil, errors.New("not implemented")
}
func (a *recordingActor) AttemptsTo(...verity.Activity) {}
func (a *recordingActor) Log(entry verity_reporting.LogEntry) error {
	a.entries = append(a.entries, entry)
	return nil
}

type contextKey struct{}

type customQuestion[T any] struct {
	description string
	answer      func(context.Context, verity.Actor) (T, error)
}

func (q customQuestion[T]) Description() string { return q.description }
func (q customQuestion[T]) AnsweredBy(ctx context.Context, actor verity.Actor) (T, error) {
	return q.answer(ctx, actor)
}

func TestTheResolvesStaticValuesAndHeterogeneousCustomQuestionsInOrder(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), contextKey{}, "value")
	actor := &recordingActor{name: "Sam", ctx: ctx}
	order := make([]string, 0, 2)
	page := customQuestion[string]{
		description: "current page",
		answer: func(gotCtx context.Context, gotActor verity.Actor) (string, error) {
			require.Same(t, ctx, gotCtx)
			require.Same(t, actor, gotActor)
			order = append(order, "page")
			return "Checkout", nil
		},
	}
	count := customQuestion[int]{
		description: "item count",
		answer: func(context.Context, verity.Actor) (int, error) {
			order = append(order, "count")
			return 2, nil
		},
	}

	interaction := log.The("Current page", log.AnswerTo(page), nil, log.AnswerTo(count))
	require.Equal(t, "#actor logs", interaction.Description())
	require.Equal(t, verity.FailFast, interaction.FailureMode())
	require.NoError(t, interaction.PerformAs(ctx, actor))
	require.Equal(t, []string{"page", "count"}, order)
	require.Equal(t, []verity_reporting.LogEntry{{
		ActorName: "Sam",
		Values:    []string{"Current page", "Checkout", "<nil>", "2"},
	}}, actor.entries)
}

func TestTheEmitsAnEmptyEntryWithoutValues(t *testing.T) {
	t.Parallel()
	actor := &recordingActor{name: "Sam", ctx: context.Background()}

	require.NoError(t, log.The().PerformAs(context.Background(), actor))
	require.Equal(t, []verity_reporting.LogEntry{{ActorName: "Sam", Values: []string{}}}, actor.entries)
}

func TestTheEmitsNoEntryWhenQuestionFails(t *testing.T) {
	t.Parallel()
	actor := &recordingActor{name: "Sam", ctx: context.Background()}
	cause := errors.New("unavailable")
	interaction := log.The("before", log.AnswerTo(customQuestion[string]{
		description: "current page",
		answer: func(context.Context, verity.Actor) (string, error) {
			return "", cause
		},
	}), "after")

	err := interaction.PerformAs(context.Background(), actor)
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "current page")
	require.Empty(t, actor.entries)
}

func TestTheReturnsClearErrorWithoutLogSink(t *testing.T) {
	t.Parallel()
	interaction := log.The("value")
	err := interaction.PerformAs(context.Background(), actorWithoutLogSink{name: "Sam"})
	require.ErrorContains(t, err, "LogSink")
}

type actorWithoutLogSink struct{ name string }

func (a actorWithoutLogSink) Context() context.Context { return context.Background() }
func (a actorWithoutLogSink) Name() string             { return a.name }
func (a actorWithoutLogSink) WhoCan(...verity.Ability) verity.Actor {
	return a
}
func (actorWithoutLogSink) AbilityTo(verity.Ability) (verity.Ability, error) {
	return nil, errors.New("not implemented")
}
func (actorWithoutLogSink) AttemptsTo(...verity.Activity) {}

func TestTheIsSafeToReuseConcurrently(t *testing.T) {
	t.Parallel()
	interaction := log.The(log.AnswerTo(verity.QuestionAbout("name", func(_ context.Context, actor verity.Actor) (string, error) {
		return actor.Name(), nil
	})))
	var group sync.WaitGroup
	for i := range 20 {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			actor := &recordingActor{name: fmt.Sprintf("Sam-%d", i), ctx: context.Background()}
			require.NoError(t, interaction.PerformAs(context.Background(), actor))
			require.Equal(t, []string{actor.name}, actor.entries[0].Values)
		}(i)
	}
	group.Wait()
}
