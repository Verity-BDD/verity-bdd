# Custom ability examples

These examples focus on the Verity integration points. Database, filesystem, WebSocket, and Redis client setup is domain-specific; the same current API pattern applies to each.

## Database example

```go
package database

import (
    "context"
    "database/sql"
    "fmt"

    verity "github.com/verity-bdd/verity-bdd"
)

type Ability interface {
    ExecContext(context.Context, string, ...any) (sql.Result, error)
    QueryRowContext(context.Context, string, ...any) *sql.Row
}

func Execute(statement string, args ...any) verity.Activity {
    return verity.Do("#actor executes a database statement", func(ctx context.Context, actor verity.Actor) error {
        db, err := verity.AbilityOf[Ability](actor)
        if err != nil {
            return fmt.Errorf("database ability: %w", err)
        }
        _, err = db.ExecContext(ctx, statement, args...)
        return err
    })
}

func Count(query string, args ...any) verity.Question[int] {
    return verity.QuestionAbout("database row count", func(ctx context.Context, actor verity.Actor) (int, error) {
        db, err := verity.AbilityOf[Ability](actor)
        if err != nil {
            return 0, fmt.Errorf("database ability: %w", err)
        }
        var count int
        err = db.QueryRowContext(ctx, query, args...).Scan(&count)
        return count, err
    })
}
```

```go
func TestDatabase(t *testing.T) {
    db := openTestDatabase(t)
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("DB admin").WhoCan(db)

    actor.AttemptsTo(
        database.Execute("INSERT INTO users(name) VALUES ($1)", "Alice"),
        ensure.That(database.Count("SELECT COUNT(*) FROM users"), expectations.Equals(1)),
    )
}
```

## Filesystem example

```go
package filesystem

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    verity "github.com/verity-bdd/verity-bdd"
)

type Ability struct {
    Root string
}

func WriteFile(path string, content []byte) verity.Activity {
    return verity.Do("#actor writes "+path, func(ctx context.Context, actor verity.Actor) error {
        if err := ctx.Err(); err != nil {
            return err
        }
        fs, err := verity.AbilityOf[*Ability](actor)
        if err != nil {
            return fmt.Errorf("filesystem ability: %w", err)
        }
        return os.WriteFile(filepath.Join(fs.Root, path), content, 0o600)
    })
}

func FileContent(path string) verity.Question[string] {
    return verity.QuestionAbout("content of "+path, func(ctx context.Context, actor verity.Actor) (string, error) {
        if err := ctx.Err(); err != nil {
            return "", err
        }
        fs, err := verity.AbilityOf[*Ability](actor)
        if err != nil {
            return "", fmt.Errorf("filesystem ability: %w", err)
        }
        data, err := os.ReadFile(filepath.Join(fs.Root, path))
        return string(data), err
    })
}
```

```go
func TestFilesystem(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("File editor").WhoCan(
        &filesystem.Ability{Root: t.TempDir()},
    )

    actor.AttemptsTo(
        filesystem.WriteFile("message.txt", []byte("hello")),
        ensure.That(filesystem.FileContent("message.txt"), expectations.Equals("hello")),
    )
}
```

## Protocol client pattern

For WebSocket, Redis, message-bus, or other clients:

1. Store the connected client or a narrow client interface as the actor's ability.
2. Create interactions with `verity.Do(description, func(context.Context, verity.Actor) error)`.
3. Retrieve the client with `verity.AbilityOf[ClientContract](actor)`.
4. Create state questions with `verity.QuestionAbout`.
5. Compose operations with `verity.TaskWhere` when they represent one business task.

For asynchronous behavior, combine the custom question with `wait.Until(...)`, or wait for a channel with `wait.UntilReceived(...)`.

`actor.AttemptsTo` returns no value. Activity failures are reported through the `TestContext`, so examples should not use `err := actor.AttemptsTo(...)` or `require.NoError` around it.
