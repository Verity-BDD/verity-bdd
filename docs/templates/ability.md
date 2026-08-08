# Custom ability template

This compact template uses the current root API. Replace the example operations with your domain behavior.

## Plan the ability

- [ ] Define the single external capability or domain responsibility.
- [ ] Choose the smallest contract needed by activities and questions.
- [ ] Decide whether state is shared and how concurrent access is protected.
- [ ] List the activities, questions, and error cases the package must expose.

## Suggested file layout

```text
yourability/
├── ability.go       # contract, implementation, and constructors
├── activities.go    # actor actions
├── questions.go     # observable state
└── ability_test.go  # direct and actor-level tests
```

Small abilities can keep these parts in one file; split them only when it improves navigation.

```go
package yourability

import (
    "context"
    "fmt"

    verity "github.com/verity-bdd/verity-bdd"
)

// Ability is the domain contract stored on an actor.
type Ability interface {
    Execute(context.Context, string) error
    Result() string
}

type ability struct {
    result string
}

func New() Ability {
    return &ability{}
}

func (a *ability) Execute(ctx context.Context, input string) error {
    if err := ctx.Err(); err != nil {
        return err
    }
    a.result = input
    return nil
}

func (a *ability) Result() string { return a.result }

// Execute creates a fail-fast interaction.
func Execute(input string) verity.Activity {
    return verity.Do("#actor executes an operation", func(ctx context.Context, actor verity.Actor) error {
        a, err := verity.AbilityOf[Ability](actor)
        if err != nil {
            return fmt.Errorf("your ability: %w", err)
        }
        return a.Execute(ctx, input)
    })
}

// Result asks the actor's ability for its current result.
func Result() verity.Question[string] {
    return verity.QuestionAbout("the operation result", func(ctx context.Context, actor verity.Actor) (string, error) {
        if err := ctx.Err(); err != nil {
            return "", err
        }
        a, err := verity.AbilityOf[Ability](actor)
        if err != nil {
            return "", fmt.Errorf("your ability: %w", err)
        }
        return a.Result(), nil
    })
}
```

Example test:

```go
func TestAbility(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("Operator").WhoCan(yourability.New())

    actor.AttemptsTo(
        yourability.Execute("done"),
        ensure.That(yourability.Result(), expectations.Equals("done")),
    )
}
```

## Testing skeleton

Keep a fast unit test for the ability's domain behavior and one actor-level test for each public activity/question path:

```go
func TestAbilityDomainBehavior(t *testing.T) {
    ability := yourability.New()
    require.NoError(t, ability.Execute(context.Background(), "done"))
    require.Equal(t, "done", ability.Result())
}

func TestAbilityMissingFromActor(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("Operator")

    _, err := verity.AbilityOf[yourability.Ability](actor)
    require.Error(t, err)
}
```

For networked abilities, inject a narrow client interface and test it with a fake or `httptest.Server`; avoid live services in unit tests.

## Customization checklist

- [ ] Rename the package, contract, constructors, descriptions, and errors for the domain.
- [ ] Replace `Execute` and `Result` with focused operations and questions.
- [ ] Add configuration only when callers need it; validate it at construction time.
- [ ] Add synchronization or resource cleanup when the client or stored state requires it.
- [ ] Cover success, missing ability, cancellation, and domain failures.
- [ ] Keep public examples compilable and use the current callback signatures.

Checklist:

- Activities implement `PerformAs(context.Context, verity.Actor) error`, `Description() string`, and `FailureMode() verity.FailureMode`, or are created with `verity.Do`.
- Questions implement `AnsweredBy(context.Context, verity.Actor) (T, error)` and `Description() string`, or are created with `verity.QuestionAbout`.
- Retrieve the stored contract with `verity.AbilityOf[Ability](actor)`, not a pointer to that interface.
- Do not assign the result of `actor.AttemptsTo`; it reports failures through the test context and returns no value.
- Do not call `WithFailureMode` on `verity.Interaction`; that method does not exist.
