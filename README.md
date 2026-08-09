# Verity-BDD: Screenplay Pattern Testing Framework for Go

![CI](https://github.com/verity-bdd/verity-bdd/workflows/CI/badge.svg) ![codecov](https://codecov.io/gh/nchursin/verity-bdd/graph/badge.svg) ![Version](https://img.shields.io/github/v/release/nchursin/verity-bdd)

> [!WARNING]
> Verity-BDD is still at version 0.x. Backwards compatibility is not guaranteed before v1.

Verity-BDD is a Go implementation of the Screenplay Pattern, focused on acceptance and HTTP API testing. It requires **Go 1.23.4 or later**.

## Installation

```bash
go get github.com/verity-bdd/verity-bdd
```

## Quick start

```go
package example

import (
    "testing"

    verity "github.com/verity-bdd/verity-bdd"
    "github.com/verity-bdd/verity-bdd/verity_abilities/api"
    expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
    "github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

func TestAPI(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("API tester").WhoCan(
        api.CallAnApiAt("https://api.example.com"),
    )

    actor.AttemptsTo(
        api.SendPostRequest("/posts").WithBody(map[string]any{
            "title": "A test post",
        }),
        ensure.That(api.LastResponseStatus{}, expectations.Equals(201)),
        ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("A test post")),
    )
}
```

`AttemptsTo` reports activity failures through the test context and does not return an error. `NewVerityTest` registers cleanup automatically with `testing.T.Cleanup`; calling `Shutdown` explicitly is optional and idempotent.

## Core API

### Actors and abilities

Actors represent people or external systems. `WhoCan` attaches abilities:

```go
actor := test.ActorCalled("Customer").WhoCan(
    api.CallAnApiAt("https://api.example.com"),
)
```

`verity.Ability` is currently an empty interface, so any Go value can be an ability. Embedding it in a custom interface is optional documentation, not an enforced requirement. Retrieve a typed ability with `verity.AbilityOf[T]`:

```go
apiAbility, err := verity.AbilityOf[api.CallAnAPI](actor)
if err != nil {
    return err
}
_ = apiAbility
```

### Interactions and tasks

Use `verity.Do` for a custom interaction and `verity.TaskWhere` to compose activities:

```go
createUser := verity.TaskWhere(
    "creates a user",
    verity.Do("prepares user data", func(ctx context.Context, actor verity.Actor) error {
        return nil
    }),
    api.SendPostRequest("/users").WithBody(userData),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(201)),
)

actor.AttemptsTo(createUser)
```

The callback signature is `func(context.Context, verity.Actor) error`. Activities created by `verity.Do` are fail-fast interactions. The exported `Critical`, `NonCritical`, and `Optional` functions return `FailureMode` values for custom activity implementations: `NonCritical` marks the test failed but continues, while `Optional` logs and continues without failing the test. `Interaction` does not currently have a `WithFailureMode` method.

### Task composition

Use `TaskWhere` to name a business workflow and combine reusable activities. Nested tasks stop at the first failing child activity:

```go
checkout := verity.TaskWhere("checks out an order", addItems, submitPayment, confirmOrder)
actor.AttemptsTo(checkout)
```

### Multiple actors

Create separate actors when a scenario has distinct personas. Each actor keeps its own abilities and state:

```go
admin := test.ActorCalled("Admin").WhoCan(api.CallAnApiAt(baseURL))
customer := test.ActorCalled("Customer").WhoCan(api.CallAnApiAt(baseURL))

admin.AttemptsTo(createProduct)
customer.AttemptsTo(orderProduct)
```

### Questions

A question implements:

```go
type Question[T any] interface {
    Description() string
    AnsweredBy(context.Context, verity.Actor) (T, error)
}
```

Create dynamic questions with `verity.QuestionAbout`:

```go
name := verity.QuestionAbout("customer name", func(ctx context.Context, actor verity.Actor) (string, error) {
    return "Alice", nil
})

actor.AttemptsTo(
    ensure.That(name, expectations.Equals("Alice")),
)
```

Wrap static values with `verity_answerable.ValueOf`.

## HTTP API testing

Give an actor `api.CallAnApiAt(baseURL)` or provide a custom client with `api.Using(client)`.

Request activities support `WithBody`, `WithHeader`, and `WithHeaders`:

```go
actor.AttemptsTo(
    api.SendPutRequest("/users/1").
        WithHeader("Authorization", "Bearer token").
        WithBody(updatedUser),
)
```

For a pre-built `*http.Request`, use `api.SendRequest(req)`. `api.RequestFor(method, url)` is also available when request construction needs to be separated from execution.

Current response questions include:

- `api.LastResponseStatus{}` and `api.LastResponseStatusQ` (`int`)
- `api.LastResponseBody{}` and `api.LastResponseBodyQ` (`string`)
- `api.LastResponseHeader(name)` (`string`)
- `api.LastResponseBodyAsJSON[T]()` (`T`)
- `api.LastResponseBodyAtJSONPath(path)` (`any`)
- `api.ResponseTime{}` and `api.ResponseTimeQ` (`int64`, currently always `0`)

`LastResponseBodyAtJSONPath` decodes JSON into ordinary Go JSON values: objects become `map[string]any`, arrays become `[]any`, numbers become `float64`, and wildcard paths can return `[]any`. Because its question type is `any`, use an `ensure.Expectation[any]` such as `expectations.Equals[any](...)` or create a typed question with `LastResponseBodyAsJSON[T]`.

Response timing is not implemented. `ResponseTime` and `ResponseTimeQ` currently return `0`; do not use them to assert measured latency.

## Expectations

```go
ensure.That(question, expectations.Equals(expected))
ensure.That(stringQuestion, expectations.ContainsSubstring("text"))
ensure.That(sliceQuestion, expectations.Includes(item))
ensure.That(question, expectations.IsEmpty[T]())
ensure.That(question, expectations.ArrayLengthEquals[T](5))
ensure.That(numberQuestion, expectations.IsGreaterThan(10)) // numberQuestion is verity.Question[any]
ensure.That(mapQuestion, expectations.ContainsKey("id"))
```

`Equals` uses deep equality. `IsEmpty` supports strings, slices, arrays, and maps. `ArrayLengthEquals` supports arrays, slices, and strings.

Use `expectations.Satisfies` for custom validation. Dynamic factories such as `EqualsAnswerTo`, `ContainsSubstringAnswerTo`, `ContainsKeyAnswerTo`, `ArrayLengthEqualsAnswerTo`, the numeric `*AnswerTo` variants, and `SatisfiesAnswer` can evaluate another question or use the current context and actor.

`ensure.That(...).After(duration)` delays once before evaluating. It does **not** poll. For polling, use the wait ability:

```go
actor.AttemptsTo(
    wait.Until(statusQuestion, expectations.Equals("ready")).
        For(10 * time.Second).
        CheckingEvery(250 * time.Millisecond),
)
```

Use `wait.UntilReceived(channel).For(timeout)` to wait for a channel value.

## Notes

Attach `take_notes.UsingEmptyNotepad()` to an actor, record values with `TakeNoteOf(...).As(...)`, and ask for them with `Note[T]`:

```go
actor := test.ActorCalled("Nina").WhoCan(take_notes.UsingEmptyNotepad())
actor.AttemptsTo(take_notes.TakeNoteOf("Bearer abc123").As("auth token"))

token, err := take_notes.Note[string]("auth token").AnsweredBy(test.Context(), actor)
if err != nil {
    t.Fatal(err)
}
_ = token
```

## Reporting

The default reporter writes test start, completed-step, and test-finish lines to stdout. `OnStepStart` tracks nesting but does not print a separate “in progress” line.

Supply a reporter either in `Scene` or with the three-argument helper:

```go
reporter := console_reporter.NewConsoleReporter()
reporter.SetOutput(os.Stdout)

test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
```

The native Allure reporter writes Allure 2 result JSON:

```go
reporter := allure_reporter.NewAllureReporterWithDir("allure-results")
test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
```

The public reporting model can represent step attachments, but the actor execution pipeline currently finishes steps without producing any. Built-in test shutdown supplies only the test-level `notes` JSON attachment when actors have notes. Custom reporter implementations should therefore expect `OnStepFinish` attachments to be empty in normal Verity execution.

The root `verity.TestResult`/`verity.Status` types describe core test state. `verity_reporting.TestResult`/`verity_reporting.Status` are separate callback contracts used by reporters.

See [docs/reporting.md](docs/reporting.md) for reporter details.

## Public package architecture

Production code is exposed through these importable packages:

- `github.com/verity-bdd/verity-bdd` — actors, activities, questions, test lifecycle
- `github.com/verity-bdd/verity-bdd/verity_abilities/api` — HTTP ability, requests, response questions
- `github.com/verity-bdd/verity-bdd/verity_abilities/take_notes` — actor notes
- `github.com/verity-bdd/verity-bdd/verity_abilities/wait` — polling and channel waits
- `github.com/verity-bdd/verity-bdd/verity_answerable` — static value questions
- `github.com/verity-bdd/verity-bdd/verity_expectations` — expectation factories
- `github.com/verity-bdd/verity-bdd/verity_expectations/ensure` — assertion activities
- `github.com/verity-bdd/verity-bdd/verity_reporting` — reporter contracts and adapters
- `github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter` — console reporter
- `github.com/verity-bdd/verity-bdd/verity_reporting/allure_reporter` — Allure reporter

`verity_abilities` is an organizational directory, not an importable production package. Implementation packages under `internal` are not public API.

## More documentation

- [Documentation index](docs/index.md)
- [Creating custom abilities](docs/abilities.md)
- [Reporting](docs/reporting.md)
- [Satisfies examples](docs/SATISFIES_EXAMPLES.md)
- [Working Go examples](examples/)

Run the examples and test suite with:

```bash
go test ./...
```

## Contributing

Contributions are welcome. Open an issue to discuss defects or proposed changes, and submit focused pull requests with tests and updated documentation where applicable.

## License

Verity-BDD is licensed under the [Apache License 2.0](LICENSE).
