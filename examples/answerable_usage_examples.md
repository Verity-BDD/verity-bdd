# Answerable API Usage Examples

The `answerable.ValueOf()` function converts static values into `core.Question[T]` objects that can be used with `ensure.That()` assertions.

## Basic Usage

```go
package main

import (
    "testing"
    "github.com/verity-bdd/verity-bdd"
    "github.com/verity-bdd/verity-bdd/verity_expectations"
    "github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
    verity "github.com/verity-bdd/verity-bdd"
)

func TestStaticValues(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("Tester")

    // Static scalar values
    actor.AttemptsTo(
        ensure.That(answerable.ValueOf(42), expectations.Equals(42)),
        ensure.That(answerable.ValueOf("hello"), expectations.ContainsSubstring("ell")),
        ensure.That(answerable.ValueOf(true), expectations.Equals(true)),
    )

    // Complex types
    type User struct { Name string; Age int }
    user := User{Name: "John", Age: 30}
    actor.AttemptsTo(
        ensure.That(answerable.ValueOf(user), expectations.Equals(User{Name: "John", Age: 30})),
    )

    // Error values (treated as values, not failures)
    err := errors.New("connection failed")
    actor.AttemptsTo(
        ensure.That(answerable.ValueOf(err.Error()), expectations.ContainsSubstring("connection")),
    )
}
```

## Mixed with Dynamic Questions

```go
func TestMixedQuestions(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    apiTester := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt("https://api.example.com"))

    // Mix static value questions with dynamic API questions
    apiTester.AttemptsTo(
        // Dynamic: Get actual data from API
        api.SendGetRequest("/users/1"),

        // Static: Compare against expected status code
        ensure.That(answerable.ValueOf(200), expectations.Equals(200)),

        // Dynamic: Check actual API response
        ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
        ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("name")),
    )
}
```

## Description Format

Static value questions generate clear descriptions in test output:
- `answerable.ValueOf(42)` → `"42 (int)"`
- `answerable.ValueOf("hello")` → `"hello (string)"`
- `answerable.ValueOf(errors.New("test"))` → `"error test (error)"`

## Key Benefits

1. **Static Value Assertions**: Use any static value in ensure.That() assertions
2. **Type Safety**: Generic type inference ensures compile-time type checking
3. **Error Handling**: Error values are treated as values, not failures
4. **Clear Descriptions**: Generated descriptions show value and type
5. **Backward Compatibility**: Works with existing ensure.That() API

## When to Use

- **Expected Values**: When testing against known static values
- **Constant Comparisons**: When comparing against constants
- **Error Messages**: When asserting on specific error message content
- **Mixed Scenarios**: When combining static values with dynamic questions
