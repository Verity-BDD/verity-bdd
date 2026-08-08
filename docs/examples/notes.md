# Notes (actor notepad)

The `take_notes` package stores values on an actor and exposes them as questions.

```go
package example

import (
    "testing"

    verity "github.com/verity-bdd/verity-bdd"
    "github.com/verity-bdd/verity-bdd/verity_abilities/take_notes"
)

func TestNotesExample(t *testing.T) {
    test := verity.NewVerityTest(t, verity.Scene{})
    actor := test.ActorCalled("Nina").WhoCan(take_notes.UsingEmptyNotepad())

    actor.AttemptsTo(
        take_notes.TakeNoteOf("Bearer abc123").As("auth token"),
    )

    token, err := take_notes.Note[string]("auth token").AnsweredBy(test.Context(), actor)
    if err != nil {
        t.Fatal(err)
    }
    if token != "Bearer abc123" {
        t.Fatalf("unexpected token: %s", token)
    }
}
```

- `UsingEmptyNotepad()` creates the ability.
- `Using(notepad)` attaches an existing `*NoteBook`.
- `TakeNoteOf(value).As(key)` creates an activity.
- `Note[T](key)` creates a typed question with `AnsweredBy(context.Context, verity.Actor) (T, error)`.
- `NoteValue(key)` creates an untyped `Question[any]`.

At test shutdown, actor notes are serialized into the built-in test-level `notes` reporting attachment. The current activity pipeline does not produce step-level attachments.
