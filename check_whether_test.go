package verity_test

import (
	"context"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
)

type checkWhetherContextKey struct{}

type recordingActivity struct {
	description string
	run         *[]string
}

func (a recordingActivity) Description() string { return a.description }
func (a recordingActivity) FailureMode() verity.FailureMode {
	return verity.FailFast
}
func (a recordingActivity) PerformAs(_ context.Context, _ verity.Actor) error {
	*a.run = append(*a.run, a.description)
	return nil
}

func TestCheckWhetherEvaluatesQuestionWithActorAndContextBeforeSelectingActivities(t *testing.T) {
	t.Parallel()

	contextValue := "from scene"
	test := verity.NewVerityTestWithContext(context.WithValue(context.Background(), checkWhetherContextKey{}, contextValue), t)
	actor := test.ActorCalled("Ava")
	var performed []string

	condition := verity.QuestionAbout("whether Ava can proceed", func(ctx context.Context, gotActor verity.Actor) (bool, error) {
		if got := ctx.Value(checkWhetherContextKey{}); got != contextValue {
			t.Fatalf("question context = %v, want %q", got, contextValue)
		}
		if got := gotActor.Name(); got != "Ava" {
			t.Fatalf("question actor = %q, want Ava", got)
		}
		return true, nil
	})

	actor.AttemptsTo(
		verity.CheckWhether(condition).
			AndIfSo(recordingActivity{description: "performs the matching activity", run: &performed}).
			Otherwise(recordingActivity{description: "performs the other activity", run: &performed}),
	)

	if got, want := len(performed), 1; got != want {
		t.Fatalf("performed %d activities, want %d (%v)", got, want, performed)
	}
	if got, want := performed[0], "performs the matching activity"; got != want {
		t.Fatalf("performed %q, want %q", got, want)
	}
}

func TestCheckWhetherPerformsOtherwiseActivitiesWhenQuestionIsFalse(t *testing.T) {
	t.Parallel()

	test := verity.NewVerityTest(t, verity.Scene{})
	actor := test.ActorCalled("Ava")
	var performed []string

	actor.AttemptsTo(
		verity.CheckWhether(verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
			return false, nil
		})).
			AndIfSo(recordingActivity{description: "performs the matching activity", run: &performed}).
			Otherwise(recordingActivity{description: "performs the other activity", run: &performed}),
	)

	if got, want := len(performed), 1; got != want {
		t.Fatalf("performed %d activities, want %d (%v)", got, want, performed)
	}
	if got, want := performed[0], "performs the other activity"; got != want {
		t.Fatalf("performed %q, want %q", got, want)
	}
}
