package verity_expectations_test

import (
	"testing"

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
