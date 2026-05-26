package take_notes_test

import (
	"context"
	"testing"

	take_notes "github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
)

func TestNoteQuestionReturnsTypedValue(t *testing.T) {
	t.Parallel()
	ability := take_notes.UsingEmptyNotepad()
	ability.(*take_notes.TakeNotesAbility).Set("name", "verity")
	actor := newStubActor("reader", context.Background(), ability)

	answer, err := take_notes.Note[string]("name").AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected note to be answered, got error: %v", err)
	}
	if answer != "verity" {
		t.Fatalf("expected 'verity', got %v", answer)
	}
}

func TestNoteQuestionErrorsWhenMissing(t *testing.T) {
	t.Parallel()
	actor := newStubActor("reader", context.Background(), take_notes.UsingEmptyNotepad())

	_, err := take_notes.Note[string]("missing").AnsweredBy(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when note missing")
	}
}

func TestNoteQuestionErrorsWhenNoAbility(t *testing.T) {
	t.Parallel()
	actor := newStubActor("reader", context.Background())

	_, err := take_notes.Note[string]("missing").AnsweredBy(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when actor lacks notes ability")
	}
}

func TestNoteQuestionErrorsOnAbilityTypeMismatch(t *testing.T) {
	t.Parallel()
	actor := newStubActor("reader", context.Background(), &dummyAbility{})

	_, err := take_notes.Note[string]("missing").AnsweredBy(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when ability type mismatched")
	}
}

func TestNoteQuestionErrorsOnTypeMismatch(t *testing.T) {
	t.Parallel()
	ability := take_notes.UsingEmptyNotepad()
	ability.(*take_notes.TakeNotesAbility).Set("count", 123)
	actor := newStubActor("reader", context.Background(), ability)

	_, err := take_notes.Note[string]("count").AnsweredBy(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error on type mismatch")
	}
}

func TestNoteValueReturnsUntyped(t *testing.T) {
	t.Parallel()
	ability := take_notes.UsingEmptyNotepad()
	ability.(*take_notes.TakeNotesAbility).Set("count", 321)
	actor := newStubActor("reader", context.Background(), ability)

	value, err := take_notes.NoteValue("count").AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected note to be answered, got error: %v", err)
	}
	if value != 321 {
		t.Fatalf("expected 321, got %v", value)
	}
}
