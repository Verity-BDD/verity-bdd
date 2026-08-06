# Verity-BDD: Screenplay Pattern Testing Framework for Go

![CI](https://github.com/verity-bdd/verity-bdd/workflows/CI/badge.svg) ![codecov](https://codecov.io/gh/nchursin/verity-bdd/graph/badge.svg) ![Version](https://img.shields.io/github/v/release/nchursin/verity-bdd)

> [!WARNING]
> This project is still at version 0.x.x. It means I do not guarantee ANY backwards compatibility for ANY changes. I use this project daily and adjust the API to real world usage.
> The plan is to go v1.x.x in Summer 2026.

A Go implementation of the Screenplay Pattern for acceptance testing, focused on API testing capabilities.

![Verity-BDD](https://raw.githubusercontent.com/nchursin/resources/refs/heads/master/verity-bdd/dark.png)

## Overview

Verity-BDD brings the power of the Screenplay Pattern to Go testing, providing:

- **Actor-centric testing** - Tests describe what actors do, not how they do it
- **Reusable components** - Build a library of reusable tasks and interactions
- **Clear domain language** - Tests that read like business requirements
- **Modular design** - Use only what you need for your testing scenarios
- **Framework agnostic** - Works with any Go test runner

## Quick Start

### Installation

```bash
go get github.com/verity-bdd/verity-bdd
```

### Basic Example

```go
package main

import (
    "testing"

    "github.com/verity-bdd/verity-bdd/verity_abilities/api"
    expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
    "github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
    verity "github.com/verity-bdd/verity-bdd"
)

func TestAPI(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    // Create an actor with API ability
    actor := test.ActorCalled("APITester").WhoCan(
        api.CallAnApiAt("https://jsonplaceholder.typicode.com"),
    )

    // Define test data
    newPost := map[string]interface{}{
        "title":  "Test Post",
        "body":   "This is a test post",
        "userId": 1,
    }

    // Test the API
    actor.AttemptsTo(
        api.SendPostRequest("/posts").
            WithBody(newPost),
        ensure.That(api.LastResponseStatus{}, expectations.Equals(201)),
        ensure.That(api.LastResponseBody{}, expectations.Contains("Test Post")),
    )
}
```

## Core Concepts

### Actors

Actors represent people or systems interacting with your application:

```go
test := verity.NewVerityTest(t, verity.Scene{})

// Create an actor
actor := test.ActorCalled("John Doe")

// Give the actor abilities to interact with your system
actor = actor.WhoCan(api.CallAnApiAt("https://api.example.com"))
```

### Abilities

Abilities enable actors to interact with different interfaces:

```go
test := verity.NewVerityTest(t, verity.Scene{})

// HTTP API ability
apiAbility := api.CallAnApiAt("https://api.example.com")

// Actor with multiple abilities
actor := test.ActorCalled("TestUser").WhoCan(
    apiAbility,
    // ... other abilities
)
```

### Activities

Activities represent actions that actors perform:

#### Interactions (low-level actions)
```go
// Send HTTP requests with fluent interface
api.SendGetRequest("/users")
api.SendPostRequest("/posts").WithBody(postData)
api.SendPutRequest("/users/1").WithBody(updatedUser)
api.SendPatchRequest("/users/1").WithBody(partialUpdate)
api.SendDeleteRequest("/posts/123")
```

#### Tasks (high-level business actions)
```go
// Define reusable task
createUserTask := core.Where(
    "creates a new user",
    core.Do("creates a new user", func(a core.Actor) error {
        req, err := api.NewRequestBuilder("POST", "/users").
            WithJSONBody(userData).
            Build()
        if err != nil {
            return err
        }
        return api.SendRequest(req).PerformAs(a)
    }),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(201)),
)

// Use the task
actor.AttemptsTo(createUserTask)
```

### Questions

Questions retrieve information from the system:

```go
// Basic built-in questions
ensure.That(api.LastResponseStatus{}, expectations.Equals(200))
ensure.That(api.LastResponseBody{}, expectations.Contains("success"))
ensure.That(api.NewResponseHeader("content-type"), expectations.Contains("json"))

// Advanced questions with JSON parsing
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

err := actor.AttemptsTo(
    api.SendGetRequest("/users/1"),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),

    // Parse response as JSON struct
    ensure.That(api.NewResponseBodyAsJSON[User](), expectations.Satisfies("has valid user", func(actual User) error {
        if actual.Name == "" {
            return fmt.Errorf("user name is empty")
        }
        if !strings.Contains(actual.Email, "@") {
            return fmt.Errorf("invalid email format")
        }
        return nil
    })),

    // JSONPath queries
    ensure.That(api.NewJSONPath("name"), expectations.Contains("John")),
    ensure.That(api.NewJSONPath("data.users.*.email"), expectations.Contains("@")),

    // Response time validation
    ensure.That(api.ResponseTime{}, expectations.IsLessThan(1000)), // milliseconds
)
```

### Assertions

Verify that expectations are met:

```go
ensure.That(question, expectations.Equals(expected))
ensure.That(question, expectations.Contains(substring))
ensure.That(question, expectations.IsEmpty())
ensure.That(question, expectations.ArrayLengthEquals(5))
ensure.That(question, expectations.IsGreaterThan(10))
ensure.That(question, expectations.ContainsKey("id"))

// Custom validation with Satisfies
ensure.That(answerable.ValueOf(value), expectations.Satisfies("custom description", func(actual T) error {
    // Your validation logic here
    return nil // or error with description
}))
```

## API Testing

### HTTP Requests

```go
// GET request
err := actor.AttemptsTo(
    api.SendGetRequest("/posts"),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
)

// POST request with JSON data
newPost := map[string]interface{}{
    "title":  "New Post",
    "body":   "Post content",
    "userId": 1,
}

err = actor.AttemptsTo(
    api.SendPostRequest("/posts").
        WithBody(newPost),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(201)),
)

// PUT request with single header
err = actor.AttemptsTo(
    api.SendPutRequest("/posts/1").
        WithHeader("Authorization", "Bearer token").
        WithBody(updatedData),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
)

// PUT request with multiple headers
err = actor.AttemptsTo(
    api.SendPutRequest("/posts/1").
        WithHeaders(map[string]string{
            "Content-Type": "application/json",
            "Authorization": "Bearer token",
            "X-Custom-Header": "custom-value",
        }).
        WithBody(updatedData),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
)

// DELETE request
err = actor.AttemptsTo(
    api.SendDeleteRequest("/posts/1"),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
)
```

### Request Building

```go
// Fluent request building
err = actor.AttemptsTo(
    api.SendPostRequest("/posts").
        WithHeader("Content-Type", "application/json").
        WithHeader("Authorization", "Bearer token").
        WithBody(postData),
)
```

### Response Validation

```go
err := actor.AttemptsTo(
    api.SendGetRequest("/posts/1").
        WithHeader("Accept", "application/json"),
    ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
    ensure.That(api.LastResponseBody{}, expectations.Contains("title")),
    ensure.That(api.NewResponseHeader("content-type"), expectations.Contains("json")),
)
```

## Console Reporting

Verity-BDD provides automatic console reporting for test results with emoji indicators, timing information, and detailed error messages.

### Automatic Integration

The TestContext API automatically provides console reporting:

```go
func TestAPITesting(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})

    actor := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt("https://jsonplaceholder.typicode.com"))

    actor.AttemptsTo(
        api.SendGetRequest("/posts"),
        ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
    )
}
```

Console output:
```
🚀 Starting: TestAPITesting
  🔄 Sends GET request to /posts
  ✅ Sends GET request to /posts (0.21s)
  🔄 Ensures that the last response status code equals 200
  ✅ Ensures that the last response status code equals 200 (0.00s)
✅ TestAPITesting: PASSED (0.26s)
```

### Output Format

| Status | Emoji | Description |
|--------|-------|-------------|
| ✅ | ✅ | Test passed |
| ❌ | ❌ | Test failed |
| ⚠️ | ⚠️ | Warning (unused actor) |

Example output:
```
🚀 Starting: TestAPITesting
  🔄 Sends GET request to /posts
  ✅ Sends GET request to /posts (0.21s)
  🔄 Ensures that the last response status code equals 200
  ✅ Ensures that the last response status code equals 200 (0.00s)
✅ TestAPITesting: PASSED (0.26s)

🚀 Starting: TestFailedExpectation
  🔄 Sends GET request to /posts
  ❌ Sends GET request to /posts (0.15s)
     Error: Expected status code to equal 200, but got 404
❌ TestFailedExpectation: FAILED (0.15s)
```

### Custom Configuration

```go
import (
    "os"
    "github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
    verity "github.com/verity-bdd/verity-bdd"
)

// Create custom console reporter
reporter := console_reporter.NewConsoleReporter()

test := verity.NewVerityTestWithReporter(t, reporter)
```

For detailed documentation on console reporting, see [docs/reporting.md](docs/reporting.md).

### File Output

```go
import (
    "os"
    "github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
    verity "github.com/verity-bdd/verity-bdd"
)

// Create file for output
file, err := os.Create("test-results.txt")
if err != nil {
    t.Fatalf("Failed to create file: %v", err)
}
defer file.Close()

// Create reporter with file output
reporter := console_reporter.NewConsoleReporter()
reporter.SetOutput(file)

test := verity.NewVerityTestWithReporter(t, reporter)
```

For detailed documentation on console reporting, see [docs/reporting.md](docs/reporting.md).

## Allure Reporting

Verity-BDD includes a native Allure reporter that writes Allure 2 result files, step data, and attachments.

```go
import (
    "context"

    "github.com/verity-bdd/verity-bdd/verity_reporting/allure_reporter"
    verity "github.com/verity-bdd/verity-bdd"
)

func TestWithAllure(t *testing.T) {
    reporter := allure_reporter.NewAllureReporterWithDir("allure-results")

    test := verity.NewVerityTest(t, verity.Scene{
        Context:  context.Background(),
        Reporter: reporter,
    })

    actor := test.ActorCalled("Tester")
    actor.AttemptsTo(
        // your activities
    )
}
```

Generate a local HTML report after test run:

```bash
allure serve allure-results
```

## Working Examples

The `examples/` directory contains working examples with real APIs:

- `basic_test.go` - JSONPlaceholder API testing examples including basic operations, error scenarios, and multiple actors
- `jsonplaceholder_test.go` - Additional JSONPlaceholder API examples with CRUD operations
- `satisfies_demo_test.go` - Comprehensive examples of custom `Satisfies` expectations including go-cmp integration

Run examples:

```bash
go test ./examples -v
```

For detailed `Satisfies` examples, see [docs/SATISFIES_EXAMPLES.md](docs/SATISFIES_EXAMPLES.md).

## Architecture

### Core Components

- **github.com/verity-bdd/verity-bdd** - Core Screenplay API, testing API, and answerable helpers
- **github.com/verity-bdd/verity-bdd/verity_abilities** - Default ability contracts and ability packages
- **github.com/verity-bdd/verity-bdd/verity_expectations** - Expectations and assertion helpers
- **github.com/verity-bdd/verity-bdd/verity_expectations/ensure** - Ensure activities
- **github.com/verity-bdd/verity-bdd/verity_reporting** - Reporting contracts and adapters
- **github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter** - Console reporter
- **github.com/verity-bdd/verity-bdd/verity_reporting/allure_reporter** - Allure reporter

### Design Principles

1. **Composable** - Build complex behaviors from simple components
2. **Reusable** - Create libraries of tasks and interactions
3. **Readable** - Tests that read like business specifications
4. **Extensible** - Add new abilities and integrations
5. **Type-safe** - Leverage Go's type system for safety

## Advanced Usage

### Custom Interactions

```go
customInteraction := core.Do("performs custom action", func(actor core.Actor) error {
    // Your custom logic here
    return nil
})

actor.AttemptsTo(customInteraction)
```

### Custom Questions

```go
customQuestion := core.QuestionAbout[int]("custom value", func(actor core.Actor, ctx context.Context) (int, error) {
    // Your custom logic here
    return 42, nil
})

ensure.That(customQuestion, expectations.Equals(42))
```

### Custom Expectations with Satisfies

Create custom expectations using the `Satisfies` function for complex validation logic:

```go
// Simple custom validation
actor.AttemptsTo(
    ensure.That(answerable.ValueOf(age), expectations.Satisfies("is positive number", func(actual int) error {
        if actual <= 0 {
            return fmt.Errorf("expected positive value, but got %d", actual)
        }
        return nil
    })),
)

// Advanced struct comparison with go-cmp
actor.AttemptsTo(
    ensure.That(answerable.ValueOf(actualUser), expectations.Satisfies("matches expected user", func(actual User) error {
        if diff := cmp.Diff(expectedUser, actual); diff != "" {
            return fmt.Errorf("user mismatch:\n%s", diff)
        }
        return nil
    })),
)

// Complex business logic validation
actor.AttemptsTo(
    ensure.That(answerable.ValueOf(order), expectations.Satisfies("has valid order data", func(actual Order) error {
        if !strings.HasPrefix(actual.ID, "ORD-") {
            return fmt.Errorf("order ID must start with ORD-, got %s", actual.ID)
        }
        if actual.Amount <= 0 {
            return fmt.Errorf("order amount must be positive, got %f", actual.Amount)
        }
        // Add more validation logic...
        return nil
    })),
)
```

The `Satisfies` function takes:
- A description string that appears in test failure messages
- A validation function that returns `nil` for success or an error for failure

This enables powerful, type-safe custom validations while maintaining the Screenplay Pattern's readable test structure.

### Task Composition

```go
// Build complex workflows from simple tasks
setupTask := core.Where("setup test data", setupDataAction)
testTask := core.Where("run test scenario", testAction)
cleanupTask := core.Where("cleanup test data", cleanupAction)

actor.AttemptsTo(
    setupTask,
    testTask,
    cleanupTask,
)
```

### Multiple Actors

```go
test := verity.NewVerityTest(t, verity.Scene{})

admin := test.ActorCalled("Admin").WhoCan(api.CallAnApiAt(baseURL))
user := test.ActorCalled("RegularUser").WhoCan(api.CallAnApiAt(baseURL))

// Admin creates resources
admin.AttemptsTo(createResourceTask)

// User interacts with resources
user.AttemptsTo(accessResourceTask)
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License
Apache 2.0 - see LICENSE file for details.
