package verity_answerable_test

import (
	"testing"

	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
)

func TestAnswerableAPIContractCompiles(t *testing.T) {
	t.Parallel()
	_ = answerable.ValueOf(42)
}
