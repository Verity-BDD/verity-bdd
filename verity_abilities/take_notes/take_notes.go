package take_notes

import (
	verity "github.com/verity-bdd/verity-bdd"
	internalnotes "github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
)

// TakeNotesAbility wraps the NoteBook so it can be registered as an ability.
type TakeNotesAbility = internalnotes.TakeNotesAbility

// NoteBook stores actor notes in a thread-safe map.
// It is meant to be used as an ability attached to an actor.
type NoteBook = internalnotes.NoteBook

// UsingEmptyNotepad returns a new ability instance with an empty notepad.
func UsingEmptyNotepad() verity.Ability {
	return internalnotes.UsingEmptyNotepad()
}

// Using returns an ability that stores notes in the provided notepad.
func Using(notepad *NoteBook) verity.Ability {
	return internalnotes.Using(notepad)
}

// NewNoteBook creates a new empty NoteBook.
func NewNoteBook() *NoteBook {
	return internalnotes.NewNoteBook()
}

// NotepadWith creates a notepad pre-filled with the provided values.
func NotepadWith(initial map[string]any) *NoteBook {
	return internalnotes.NotepadWith(initial)
}

// TakeNoteOf starts a TakeNote activity definition for the given value.
func TakeNoteOf(value any) interface{ As(string) verity.Activity } {
	return internalnotes.TakeNoteOf(value)
}

// NoteValue returns an untyped question that retrieves the note stored under the given key.
func NoteValue(key string) verity.Question[any] {
	return internalnotes.NoteValue(key)
}

// Note returns a typed question that retrieves the note stored under the given key as type T.
func Note[T any](key string) verity.Question[T] {
	return internalnotes.Note[T](key)
}
