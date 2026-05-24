package wait_test

import (
	"context"
	"errors"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
)

type stubActor struct{}

func (s *stubActor) Context() context.Context                 { return context.Background() }
func (s *stubActor) Name() string                             { return "test" }
func (s *stubActor) WhoCan(_ ...abilities.Ability) core.Actor { return s }
func (s *stubActor) AbilityTo(_ abilities.Ability) (abilities.Ability, error) {
	return nil, errors.New("no ability")
}
func (s *stubActor) AttemptsTo(_ ...core.Activity) {}
