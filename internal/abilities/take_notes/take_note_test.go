package take_notes_test

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	take_notes "github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	"github.com/verity-bdd/verity-bdd/internal/reporting/mocks"
	veritytesting "github.com/verity-bdd/verity-bdd/internal/testing"
)

type stubActor struct {
	name      string
	ctx       context.Context
	abilities []abilities.Ability
}

func newStubActor(name string, ctx context.Context, abilities ...abilities.Ability) *stubActor {
	return &stubActor{name: name, ctx: ctx, abilities: abilities}
}

func (a *stubActor) Name() string { return a.name }

func (a *stubActor) Context() context.Context { return a.ctx }

func (a *stubActor) WhoCan(abilities ...abilities.Ability) core.Actor {
	a.abilities = append(a.abilities, abilities...)
	return a
}

func (a *stubActor) AbilityTo(target abilities.Ability) (abilities.Ability, error) {
	for _, ability := range a.abilities {
		if fmt.Sprintf("%T", ability) == fmt.Sprintf("%T", target) {
			return ability, nil
		}
	}
	return nil, fmt.Errorf("actor '%s' can't %s. Did you give them the ability?", a.name, core.AbilityName(target))
}

func (a *stubActor) AttemptsTo(activities ...core.Activity) {}

func TestTakeNoteStoresValue(t *testing.T) {
	t.Parallel()
	ability := take_notes.UsingEmptyNotepad()
	actor := newStubActor("alice", context.Background(), ability)

	activity := take_notes.TakeNoteOf("secret-token").As("auth")

	if activity.Description() != "#actor takes note \"auth\"" {
		t.Fatalf("unexpected description: %s", activity.Description())
	}

	if activity.FailureMode() != core.FailFast {
		t.Fatalf("expected FailFast failure mode")
	}

	err := activity.PerformAs(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected no error performing take note, got %v", err)
	}

	value, err := ability.(*take_notes.TakeNotesAbility).Get("auth")
	if err != nil {
		t.Fatalf("expected stored note, got error: %v", err)
	}
	if value != "secret-token" {
		t.Fatalf("expected value to be stored, got %v", value)
	}
}

func TestTakeNoteRequiresAbility(t *testing.T) {
	t.Parallel()
	actor := newStubActor("bob", context.Background())

	activity := take_notes.TakeNoteOf("value").As("missing")
	err := activity.PerformAs(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when actor lacks notebook ability")
	}
	if err.Error() == "" {
		t.Fatalf("expected error message, got empty string")
	}
}

type dummyAbility struct{}

func (d *dummyAbility) String() string { return "dummy" }

func TestTakeNoteReportsStep(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	reporter := mocks.NewMockReporter(ctrl)

	reporter.EXPECT().OnTestStart(t.Name())
	reporter.EXPECT().OnStepStart("Sam takes note \"remember\"")
	reporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
		if result.Name() != "Sam takes note \"remember\"" {
			t.Fatalf("unexpected step name: %s", result.Name())
		}
		if result.Status() != reporting.StatusPassed {
			t.Fatalf("expected status passed, got %v", result.Status())
		}
	})
	reporter.EXPECT().OnTestFinish(gomock.Any())

	verityTest := veritytesting.NewVerityTestWithReporter(context.Background(), t, reporter)
	actor := verityTest.ActorCalled("Sam").WhoCan(take_notes.UsingEmptyNotepad())

	actor.AttemptsTo(take_notes.TakeNoteOf("secret").As("remember"))
}

func TestTakeNoteErrorsWhenAbilityTypeMismatch(t *testing.T) {
	t.Parallel()
	actor := newStubActor("mike", context.Background(), &dummyAbility{})

	activity := take_notes.TakeNoteOf("value").As("k")
	err := activity.PerformAs(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when ability type is not TakeNotesAbility")
	}
}
