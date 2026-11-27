package verity_test

import (
	"context"
	"sync"
	"sync/atomic"
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

func TestCheckWhetherDerivedConfigurationsDoNotMutateEarlierVariants(t *testing.T) {
	t.Parallel()

	question := verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
		return true, nil
	})
	base := verity.CheckWhether(question)
	var firstPerformed, secondPerformed []string
	first := base.AndIfSo(recordingActivity{description: "first activity", run: &firstPerformed})
	second := base.AndIfSo(recordingActivity{description: "second activity", run: &secondPerformed})

	if err := first.PerformAs(context.Background(), nil); err != nil {
		t.Fatalf("perform first configured check: %v", err)
	}
	if err := second.PerformAs(context.Background(), nil); err != nil {
		t.Fatalf("perform second configured check: %v", err)
	}

	if got, want := firstPerformed, []string{"first activity"}; !equalStrings(got, want) {
		t.Fatalf("first configured check performed %v, want %v", got, want)
	}
	if got, want := secondPerformed, []string{"second activity"}; !equalStrings(got, want) {
		t.Fatalf("second configured check performed %v, want %v", got, want)
	}
}

func TestCheckWhetherCopiesConfiguredActivityInput(t *testing.T) {
	t.Parallel()

	question := verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
		return true, nil
	})
	var configuredPerformed, replacementPerformed []string
	activities := []verity.Activity{recordingActivity{description: "configured activity", run: &configuredPerformed}}
	condition := verity.CheckWhether(question).AndIfSo(activities...)
	activities[0] = recordingActivity{description: "replacement activity", run: &replacementPerformed}

	if err := condition.PerformAs(context.Background(), nil); err != nil {
		t.Fatalf("perform configured check: %v", err)
	}

	if got, want := configuredPerformed, []string{"configured activity"}; !equalStrings(got, want) {
		t.Fatalf("configured activities performed %v, want %v", got, want)
	}
	if got := replacementPerformed; len(got) != 0 {
		t.Fatalf("replacement activities performed %v, want none", got)
	}
}

func TestCheckWhetherCopiesOtherwiseActivityInput(t *testing.T) {
	t.Parallel()

	question := verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
		return false, nil
	})
	var configuredPerformed, replacementPerformed []string
	activities := []verity.Activity{recordingActivity{description: "configured activity", run: &configuredPerformed}}
	condition := verity.CheckWhether(question).Otherwise(activities...)
	activities[0] = recordingActivity{description: "replacement activity", run: &replacementPerformed}

	if err := condition.PerformAs(context.Background(), nil); err != nil {
		t.Fatalf("perform configured check: %v", err)
	}

	if got, want := configuredPerformed, []string{"configured activity"}; !equalStrings(got, want) {
		t.Fatalf("configured activities performed %v, want %v", got, want)
	}
	if got := replacementPerformed; len(got) != 0 {
		t.Fatalf("replacement activities performed %v, want none", got)
	}
}

type countingActivity struct {
	description string
	calls       *atomic.Int64
}

func (a countingActivity) Description() string           { return a.description }
func (countingActivity) FailureMode() verity.FailureMode { return verity.FailFast }
func (a countingActivity) PerformAs(context.Context, verity.Actor) error {
	a.calls.Add(1)
	return nil
}

func TestCheckWhetherConfiguredCheckCanBeReusedConcurrently(t *testing.T) {
	t.Parallel()

	const workers = 16
	const iterations = 100

	var answers, calls atomic.Int64
	condition := verity.CheckWhether(verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
		answers.Add(1)
		return true, nil
	})).AndIfSo(countingActivity{description: "matching activity", calls: &calls})

	performConcurrently(t, workers, iterations, func() error {
		return condition.PerformAs(context.Background(), nil)
	})

	if got, want := answers.Load(), int64(workers*iterations); got != want {
		t.Fatalf("question answered %d times, want %d", got, want)
	}
	if got, want := calls.Load(), int64(workers*iterations); got != want {
		t.Fatalf("matching activities performed %d times, want %d", got, want)
	}
}

func TestCheckWhetherCanDeriveAndPerformConfigurationsConcurrently(t *testing.T) {
	t.Parallel()

	const workers = 16
	const iterations = 100

	var calls atomic.Int64
	base := verity.CheckWhether(verity.QuestionAbout("whether Ava can proceed", func(context.Context, verity.Actor) (bool, error) {
		return true, nil
	}))

	performConcurrently(t, workers, iterations, func() error {
		condition := base.AndIfSo(countingActivity{description: "matching activity", calls: &calls})
		return condition.PerformAs(context.Background(), nil)
	})

	if got, want := calls.Load(), int64(workers*iterations); got != want {
		t.Fatalf("matching activities performed %d times, want %d", got, want)
	}
}

func performConcurrently(t *testing.T, workers, iterations int, perform func() error) {
	t.Helper()

	start := make(chan struct{})
	errs := make(chan error, workers)
	var ready, done sync.WaitGroup
	ready.Add(workers)
	done.Add(workers)
	for range workers {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			for range iterations {
				if err := perform(); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("perform check concurrently: %v", err)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
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
