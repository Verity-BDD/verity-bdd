package examples

import (
	"context"
	"fmt"

	verity "github.com/verity-bdd/verity-bdd"
)

type checkExampleActor struct {
	ctx  context.Context
	name string
}

func (a checkExampleActor) Context() context.Context { return a.ctx }
func (a checkExampleActor) Name() string             { return a.name }
func (a checkExampleActor) WhoCan(...verity.Ability) verity.Actor {
	return a
}
func (checkExampleActor) AbilityTo(verity.Ability) (verity.Ability, error) { return nil, nil }
func (checkExampleActor) Has(...verity.Fact)                               {}
func (a checkExampleActor) AttemptsTo(activities ...verity.Activity) {
	for _, activity := range activities {
		if err := activity.PerformAs(a.ctx, a); err != nil {
			panic(err)
		}
	}
}

type printActivity string

func (a printActivity) Description() string           { return string(a) }
func (printActivity) FailureMode() verity.FailureMode { return verity.FailFast }
func (a printActivity) PerformAs(context.Context, verity.Actor) error {
	fmt.Println(a)
	return nil
}

func ExampleCheckWhether() {
	actor := checkExampleActor{ctx: context.Background(), name: "Ava"}

	actor.AttemptsTo(
		verity.CheckWhether(verity.QuestionAbout("whether Ava is signed in", func(_ context.Context, actor verity.Actor) (bool, error) {
			return actor.Name() == "Ava", nil
		})).
			AndIfSo(printActivity("opens the account page")).
			Otherwise(printActivity("opens the sign-in page")),
	)

	// Output:
	// opens the account page
}
