package testing

import (
	"testing"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
)

type helperIfaceAbility interface {
	abilities.Ability
	Foo() string
}

type helperIfaceImpl struct{ id string }

func (i helperIfaceImpl) Foo() string { return i.id }

type helperOtherAbility struct{ abilities.Ability }

func TestAbilityMatchesTypeNilInputs(t *testing.T) {
	t.Parallel()
	if abilityMatchesType(nil, nil) {
		t.Fatalf("expected false for nil inputs")
	}

	if abilityMatchesType(&helperIfaceImpl{id: "x"}, nil) {
		t.Fatalf("expected false for nil target type")
	}

	if abilityMatchesType(nil, (*helperIfaceAbility)(nil)) {
		t.Fatalf("expected false for nil ability")
	}
}

func TestAbilityMatchesTypeConcreteToConcrete(t *testing.T) {
	t.Parallel()
	ab := &helperOtherAbility{}
	target := &helperOtherAbility{}
	if !abilityMatchesType(ab, target) {
		t.Fatalf("expected concrete types to match")
	}
}

func TestAbilityMatchesTypePointerImplementsInterface(t *testing.T) {
	t.Parallel()
	ab := &helperIfaceImpl{id: "ok"}
	var target helperIfaceAbility
	if !abilityMatchesType(ab, &target) {
		t.Fatalf("expected pointer receiver to satisfy interface")
	}
}

func TestAbilityMatchesTypeValueImplementsInterface(t *testing.T) {
	t.Parallel()
	ab := helperIfaceImpl{id: "ok"}
	var target helperIfaceAbility
	if !abilityMatchesType(ab, &target) {
		t.Fatalf("expected value receiver to satisfy interface")
	}
}

func TestAbilityMatchesTypePointerInterfaceTarget(t *testing.T) {
	t.Parallel()
	ab := &helperIfaceImpl{id: "ok"}
	var target helperIfaceAbility
	if !abilityMatchesType(ab, &target) {
		t.Fatalf("expected pointer interface target to match")
	}
}

func TestAbilityMatchesTypeNonMatching(t *testing.T) {
	t.Parallel()
	ab := &helperIfaceImpl{id: "ok"}
	nonMatching := &helperOtherAbility{}
	if abilityMatchesType(ab, nonMatching) {
		t.Fatalf("expected non-matching types to be false")
	}
}
