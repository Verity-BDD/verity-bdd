package core

import (
	"context"
	"errors"
	"testing"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
)

type testAbility struct{ id string }
type otherAbility struct{}

type stubActor struct {
	name          string
	abilityToResp abilities.Ability
	abilityErr    error
}

func (s *stubActor) Context() context.Context { return context.Background() }
func (s *stubActor) Name() string             { return s.name }
func (s *stubActor) WhoCan(_ ...abilities.Ability) Actor {
	return s
}
func (s *stubActor) AbilityTo(_ abilities.Ability) (abilities.Ability, error) {
	if s.abilityErr != nil {
		return nil, s.abilityErr
	}
	return s.abilityToResp, nil
}
func (s *stubActor) AttemptsTo(_ ...Activity) {}

func TestAbilityNameStripsPointer(t *testing.T) {
	t.Parallel()
	ab := &testAbility{}
	if got := AbilityName(ab); got != "core.testAbility" {
		t.Fatalf("expected core.testAbility, got %s", got)
	}
}

func TestAbilityNameHandlesNil(t *testing.T) {
	t.Parallel()
	if got := AbilityName(nil); got != "<nil>" {
		t.Fatalf("expected <nil>, got %s", got)
	}
}

func TestAbilityOfSuccess(t *testing.T) {
	t.Parallel()
	wanted := &testAbility{id: "ok"}
	actor := &stubActor{name: "A", abilityToResp: wanted}

	got, err := AbilityOf[*testAbility](actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != wanted {
		t.Fatalf("expected %#v, got %#v", wanted, got)
	}
}

func TestAbilityOfNilActor(t *testing.T) {
	t.Parallel()
	_, err := AbilityOf[*testAbility](nil)
	expected := "actor is nil; cannot get core.testAbility ability"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}

func TestAbilityOfPropagatesFriendlyMessageOnMissing(t *testing.T) {
	t.Parallel()
	actor := &stubActor{name: "A", abilityErr: errors.New("missing")}
	_, err := AbilityOf[*testAbility](actor)
	expected := "actor 'A' can't core.testAbility. Did you give them the ability?"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}

func TestAbilityOfDetectsTypeMismatch(t *testing.T) {
	t.Parallel()
	actor := &stubActor{name: "A", abilityToResp: &otherAbility{}}
	_, err := AbilityOf[*testAbility](actor)
	expected := "actor 'A' returned ability of wrong type (got core.otherAbility, want core.testAbility)"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected %q, got %v", expected, err)
	}
}
