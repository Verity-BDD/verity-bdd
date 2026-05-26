package take_notes_test

import (
	"strconv"
	"sync"
	"testing"

	take_notes "github.com/verity-bdd/verity-bdd/internal/abilities/take_notes"
)

func TestNoteBookStoresAndRetrievesValues(t *testing.T) {
	t.Parallel()
	noteBook := take_notes.NewNoteBook()

	noteBook.Set("greeting", "hello")
	noteBook.Set("count", 42)

	value, err := noteBook.Get("greeting")
	if err != nil {
		t.Fatalf("expected greeting note, got error: %v", err)
	}
	if value != "hello" {
		t.Fatalf("expected 'hello', got %v", value)
	}

	value, err = noteBook.Get("count")
	if err != nil {
		t.Fatalf("expected count note, got error: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %v", value)
	}
}

func TestNoteBookOverwritesValues(t *testing.T) {
	t.Parallel()
	noteBook := take_notes.NewNoteBook()

	noteBook.Set("key", "first")
	noteBook.Set("key", "second")

	value, err := noteBook.Get("key")
	if err != nil {
		t.Fatalf("expected note, got error: %v", err)
	}
	if value != "second" {
		t.Fatalf("expected overwritten value 'second', got %v", value)
	}
}

func TestNoteBookReturnsErrorWhenMissing(t *testing.T) {
	t.Parallel()
	noteBook := take_notes.NewNoteBook()

	_, err := noteBook.Get("missing")
	if err == nil {
		t.Fatalf("expected error for missing note")
	}
}

func TestNoteBookAllReturnsCopy(t *testing.T) {
	t.Parallel()
	noteBook := take_notes.NewNoteBook()
	noteBook.Set("k1", "v1")
	noteBook.Set("k2", "v2")

	all := noteBook.All()
	all["k1"] = "changed"

	value, _ := noteBook.Get("k1")
	if value != "v1" {
		t.Fatalf("expected original map to stay unchanged, got %v", value)
	}
}

func TestNoteBookIsConcurrentSafe(t *testing.T) {
	t.Parallel()
	noteBook := take_notes.NewNoteBook()
	count := 200
	wg := sync.WaitGroup{}
	wg.Add(count)

	for i := 0; i < count; i++ {
		i := i
		go func() {
			defer wg.Done()
			noteBook.Set("key"+strconv.Itoa(i), i)
		}()
	}

	wg.Wait()

	for i := 0; i < count; i++ {
		value, err := noteBook.Get("key" + strconv.Itoa(i))
		if err != nil {
			t.Fatalf("expected key %d, got error: %v", i, err)
		}
		if value != i {
			t.Fatalf("expected %d, got %v", i, value)
		}
	}
}

func TestUsingEmptyNotepadCreatesEmptyAbility(t *testing.T) {
	t.Parallel()
	ability := take_notes.UsingEmptyNotepad()

	takeNotesAbility, ok := ability.(*take_notes.TakeNotesAbility)
	if !ok {
		t.Fatalf("expected *TakeNotesAbility, got %T", ability)
	}

	all := takeNotesAbility.All()
	if len(all) != 0 {
		t.Fatalf("expected empty notepad, got %v", all)
	}
}

func TestUsingReusesProvidedNotepad(t *testing.T) {
	t.Parallel()
	notepad := take_notes.NewNoteBook()
	notepad.Set("token", "abc")

	ability := take_notes.Using(notepad)
	takeNotesAbility, ok := ability.(*take_notes.TakeNotesAbility)
	if !ok {
		t.Fatalf("expected *TakeNotesAbility, got %T", ability)
	}

	if takeNotesAbility.NoteBook != notepad {
		t.Fatalf("expected ability to reuse provided notepad instance")
	}

	value, err := takeNotesAbility.Get("token")
	if err != nil {
		t.Fatalf("expected token to be available, got error: %v", err)
	}
	if value != "abc" {
		t.Fatalf("expected abc, got %v", value)
	}
}

func TestNotepadWithCopiesInitialValues(t *testing.T) {
	t.Parallel()
	initial := map[string]any{
		"firstName": "Sam",
		"count":     1,
	}

	notepad := take_notes.NotepadWith(initial)
	initial["firstName"] = "Changed"
	initial["new"] = "value"

	firstName, err := notepad.Get("firstName")
	if err != nil {
		t.Fatalf("expected firstName note, got error: %v", err)
	}
	if firstName != "Sam" {
		t.Fatalf("expected Sam, got %v", firstName)
	}

	_, err = notepad.Get("new")
	if err == nil {
		t.Fatalf("expected missing note for key added after creation")
	}
}
