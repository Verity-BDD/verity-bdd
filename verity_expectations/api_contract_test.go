package verity_expectations_test

import (
	"context"
	"errors"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

func TestContainsSubstringMatchesSubstring(t *testing.T) {
	t.Parallel()

	err := ve.ContainsSubstring("world").Evaluate(context.Background(), nil, "hello world")
	if err != nil {
		t.Fatalf("expected substring to match: %v", err)
	}
}

func TestContainsSubstringAnswerToMatchesQuestionAnswer(t *testing.T) {
	t.Parallel()

	question := answerable.ValueOf("world")
	err := ve.ContainsSubstringAnswerTo(question).Evaluate(context.Background(), nil, "hello world")
	if err != nil {
		t.Fatalf("expected question answer substring to match: %v", err)
	}
}

func TestIncludesFindsOnlyElement(t *testing.T) {
	t.Parallel()

	err := ve.Includes("item").Evaluate(context.Background(), nil, []string{"item"})
	if err != nil {
		t.Fatalf("expected slice to include item: %v", err)
	}
}

func TestIncludesFindsElementAfterFirst(t *testing.T) {
	t.Parallel()

	err := ve.Includes(2).Evaluate(context.Background(), nil, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("expected slice to include element after first: %v", err)
	}
}

func TestIncludesFailsWhenElementIsMissing(t *testing.T) {
	t.Parallel()

	err := ve.Includes(4).Evaluate(context.Background(), nil, []int{1, 2, 3})
	if err == nil {
		t.Fatal("expected missing element to fail")
	}
	const want = "expected slice to include 4, but got [1 2 3]"
	if err.Error() != want {
		t.Fatalf("unexpected error: got %q, want %q", err, want)
	}
}

func TestIncludesDescribesExpectedElement(t *testing.T) {
	t.Parallel()

	got := ve.Includes(2).Description()
	const want = "includes 2"
	if got != want {
		t.Fatalf("unexpected description: got %q, want %q", got, want)
	}
}

func TestIncludesUsesDeepEqualityForNonComparableElements(t *testing.T) {
	t.Parallel()

	err := ve.Includes([]int{1, 2}).Evaluate(context.Background(), nil, [][]int{{0}, {1, 2}})
	if err != nil {
		t.Fatalf("expected slice to include deeply equal element: %v", err)
	}
}

func TestIncludesFailsForNilAndEmptySlices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		actual []int
	}{
		{name: "nil", actual: nil},
		{name: "empty", actual: []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ve.Includes(1).Evaluate(context.Background(), nil, tt.actual)
			if err == nil {
				t.Fatal("expected missing element to fail")
			}
		})
	}
}

func TestIncludesDistinguishesNilAndEmptySliceElements(t *testing.T) {
	t.Parallel()

	expectation := ve.Includes[[]int](nil)
	if err := expectation.Evaluate(context.Background(), nil, [][]int{{}}); err == nil {
		t.Fatal("expected empty slice element not to match nil")
	}
	if err := expectation.Evaluate(context.Background(), nil, [][]int{nil}); err != nil {
		t.Fatalf("expected nil slice element to match nil: %v", err)
	}
}

func TestIncludesComposesWithEnsureThat(t *testing.T) {
	t.Parallel()

	_ = ensure.That(answerable.ValueOf([]int{1, 2}), ve.Includes(2))
	_ = ensure.That(answerable.ValueOf([][]int{{1, 2}}), ve.Includes([]int{1, 2}))
}

func TestExpectationsAPIContractCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("hello")
	_ = ensure.That(q, ve.ContainsSubstring("ell"))
}

func TestNotExpectationAPIContractCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("hello")
	_ = ensure.That(q, ve.Not(ve.ContainsSubstring("xyz")))
}

func TestAnswerToFactoriesAPIContractCompiles(t *testing.T) {
	t.Parallel()
	strQ := answerable.ValueOf("world")
	intQ := answerable.ValueOf(42)
	ifaceQ := answerable.ValueOf[interface{}](int(5))

	_ = ensure.That(answerable.ValueOf("hello world"), ve.ContainsSubstringAnswerTo(strQ))
	_ = ensure.That(answerable.ValueOf[interface{}](map[string]interface{}{"name": "Alice"}), ve.ContainsKeyAnswerTo(strQ))
	_ = ensure.That(answerable.ValueOf(42), ve.EqualsAnswerTo(intQ))
	_ = ensure.That(answerable.ValueOf([]string{"a", "b"}), ve.ArrayLengthEqualsAnswerTo[[]string](answerable.ValueOf(2)))
	_ = ensure.That(answerable.ValueOf[interface{}](int(10)), ve.IsGreaterThanAnswerTo(ifaceQ))
	_ = ensure.That(answerable.ValueOf[interface{}](int(2)), ve.IsLessThanAnswerTo(ifaceQ))
	_ = ensure.That(answerable.ValueOf(42), ve.SatisfiesAnswer[int]("is positive", func(_ context.Context, _ verity.Actor, actual int) error {
		if actual > 0 {
			return nil
		}
		return errors.New("not positive")
	}))
}
