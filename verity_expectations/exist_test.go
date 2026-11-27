package verity_expectations_test

import (
	"context"
	"testing"
	"unsafe"

	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
)

func TestExistFailsForNilPointer(t *testing.T) {
	t.Parallel()

	var actual *string
	err := ve.Exist[*string]().Evaluate(context.Background(), nil, actual)
	if err == nil {
		t.Fatal("expected nil pointer not to exist")
	}

	const want = "expected value to exist, but got nil"
	if err.Error() != want {
		t.Fatalf("unexpected error: got %q, want %q", err, want)
	}
}

func TestExistFailsForNilableValues(t *testing.T) {
	t.Parallel()

	var nilMap map[string]string
	var nilSlice []string
	var nilChannel chan string
	var nilFunction func()
	var nilPointer *string

	tests := []struct {
		name   string
		actual any
	}{
		{name: "map", actual: nilMap},
		{name: "slice", actual: nilSlice},
		{name: "channel", actual: nilChannel},
		{name: "function", actual: nilFunction},
		{name: "unsafe pointer", actual: unsafe.Pointer(nil)},
		{name: "typed nil interface", actual: nilPointer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ve.Exist[any]().Evaluate(context.Background(), nil, tt.actual)
			if err == nil {
				t.Fatal("expected nil value not to exist")
			}
		})
	}
}

func TestExistPassesForValuesThatExist(t *testing.T) {
	t.Parallel()

	value := "value"
	for _, actual := range []any{&value, map[string]string{}, []string{}, 0, ""} {
		if err := ve.Exist[any]().Evaluate(context.Background(), nil, actual); err != nil {
			t.Fatalf("expected %T value to exist: %v", actual, err)
		}
	}
}

func TestExistDescribesExpectation(t *testing.T) {
	t.Parallel()

	if got := ve.Exist[*string]().Description(); got != "exists" {
		t.Fatalf("unexpected description: got %q, want %q", got, "exists")
	}
}
