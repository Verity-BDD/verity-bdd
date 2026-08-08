# Creating custom abilities

An ability is a value attached to an actor with `WhoCan`. `verity.Ability` is currently an empty interface, so every Go value satisfies it. Embedding `verity.Ability` in your own interface is optional and can be useful as documentation, but the compiler does not require it.

## A complete minimal pattern

The following custom ability, activity, and question use the current public API and signatures:

```go
package files

import (
    "context"
    "fmt"
    "sync"

    verity "github.com/verity-bdd/verity-bdd"
)

type Manager interface {
    Read(path string) (string, error)
    Write(path, content string) error
}

type memoryManager struct {
    mu    sync.RWMutex
    files map[string]string
}

func ManageInMemory() Manager {
    return &memoryManager{files: make(map[string]string)}
}

func (m *memoryManager) Read(path string) (string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    content, ok := m.files[path]
    if !ok {
        return "", fmt.Errorf("file %q not found", path)
    }
    return content, nil
}

func (m *memoryManager) Write(path, content string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.files[path] = content
    return nil
}

func Write(path, content string) verity.Activity {
    return verity.Do("#actor writes "+path, func(ctx context.Context, actor verity.Actor) error {
        manager, err := verity.AbilityOf[Manager](actor)
        if err != nil {
            return fmt.Errorf("file manager ability: %w", err)
        }
        return manager.Write(path, content)
    })
}

func Content(path string) verity.Question[string] {
    return verity.QuestionAbout("content of "+path, func(ctx context.Context, actor verity.Actor) (string, error) {
        manager, err := verity.AbilityOf[Manager](actor)
        if err != nil {
            return "", fmt.Errorf("file manager ability: %w", err)
        }
        return manager.Read(path)
    })
}
```

Use it in a test:

```go
func TestFiles(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("File user").WhoCan(files.ManageInMemory())

    actor.AttemptsTo(
        files.Write("message.txt", "hello"),
        ensure.That(files.Content("message.txt"), expectations.Equals("hello")),
    )
}
```

## Required activity and question contracts

A custom activity must implement all three methods:

```go
type Activity interface {
    PerformAs(context.Context, verity.Actor) error
    Description() string
    FailureMode() verity.FailureMode
}
```

Using `verity.Do` is the simplest way to create a fail-fast interaction. It accepts `func(context.Context, verity.Actor) error`. The returned `verity.Interaction` does not expose `WithFailureMode`; implement your own activity type when a different failure mode is needed.

A typed question must implement:

```go
type Question[T any] interface {
    Description() string
    AnsweredBy(context.Context, verity.Actor) (T, error)
}
```

`verity.QuestionAbout` supplies those methods around a callback.

## Retrieving abilities

Prefer the typed helper:

```go
manager, err := verity.AbilityOf[Manager](actor)
```

The lower-level `actor.AbilityTo(probe)` returns an untyped `verity.Ability` and uses the probe's type for lookup. Avoid requesting a pointer to an interface (for example, `AbilityOf[*Manager]`); request the interface itself.

## Design guidance

- Keep state in the ability when activities and questions need to share it.
- Make stateful abilities safe for the concurrency model of your tests.
- Return descriptive wrapped errors from activities and questions.
- Keep interactions small and compose business workflows with `verity.TaskWhere`.
- Use interfaces for replaceable external clients when it improves testing; an interface is not required merely to satisfy `verity.Ability`.

## Testing a custom ability

Test the domain implementation directly, then add an actor-level test to verify ability lookup, activity execution, and the question contract.

```go
func TestMemoryManager(t *testing.T) {
    manager := files.ManageInMemory()
    require.NoError(t, manager.Write("message.txt", "hello"))

    content, err := manager.Read("message.txt")
    require.NoError(t, err)
    require.Equal(t, "hello", content)
}

func TestFilesWithActor(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("File user").WhoCan(files.ManageInMemory())

    actor.AttemptsTo(
        files.Write("message.txt", "hello"),
        ensure.That(files.Content("message.txt"), expectations.Equals("hello")),
    )
}
```

Use fakes or local test servers for external clients, cover missing-ability and domain-error paths, and run actor-level tests through `NewVerityTest` so failure handling and automatic cleanup are exercised.

See [the compact template](templates/ability.md) and [current examples](examples/abilities.md).
