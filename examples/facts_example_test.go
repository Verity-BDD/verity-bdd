package examples

import (
	"context"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
)

type accountFact struct {
	number   string
	accounts map[string]string
}

func (f *accountFact) Description() string { return "account " + f.number }

func (f *accountFact) Setup(_ context.Context, actor verity.Actor) error {
	f.accounts[f.number] = actor.Name()
	return nil
}

func (f *accountFact) Teardown(_ context.Context, _ verity.Actor) error {
	delete(f.accounts, f.number)
	return nil
}

type accountFacts struct {
	accounts map[string]string
}

func (f accountFacts) Numbered(number string) verity.Fact {
	return &accountFact{number: number, accounts: f.accounts}
}

func TestActorFacts(t *testing.T) {
	accounts := make(map[string]string)
	profileReady := false

	// Registered first so this check runs after Verity's automatic shutdown.
	t.Cleanup(func() {
		if _, exists := accounts["1234"]; exists {
			t.Error("account teardown did not run at test shutdown")
		}
	})

	test := verity.NewVerityTest(t, verity.Scene{})
	sam := test.ActorCalled("Sam")
	anAccount := accountFacts{accounts: accounts}

	sam.Has(
		verity.FactAbout("ready profile", func(_ context.Context, _ verity.Actor) error {
			profileReady = true
			return nil
		}),
		anAccount.Numbered("1234"),
	)

	if !profileReady {
		t.Error("setup-only fact was not established by Actor.Has")
	}
	if owner := accounts["1234"]; owner != "Sam" {
		t.Errorf("account setup owner = %q, want %q", owner, "Sam")
	}
}
