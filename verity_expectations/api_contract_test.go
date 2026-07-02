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

func TestExpectationsAPIContractCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("hello")
	_ = ensure.That(q, ve.Contains("ell"))
}

func TestNotExpectationAPIContractCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("hello")
	_ = ensure.That(q, ve.Not(ve.Contains("xyz")))
}

func TestAnswerToFactoriesAPIContractCompiles(t *testing.T) {
	t.Parallel()
	strQ := answerable.ValueOf("world")
	intQ := answerable.ValueOf(42)
	ifaceQ := answerable.ValueOf[interface{}](int(5))

	_ = ensure.That(answerable.ValueOf("hello world"), ve.ContainsAnswerTo(strQ))
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
