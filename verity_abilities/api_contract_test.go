package verity_abilities_test

import (
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
	"github.com/verity-bdd/verity-bdd/verity_abilities/take_notes"
	"github.com/verity-bdd/verity-bdd/verity_abilities/wait"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
)

func TestAbilitiesAPIContractCompiles(t *testing.T) {
	t.Parallel()
	var _ verity.Ability = api.CallAnApiAt("https://example.com")

	notebook := take_notes.NewNoteBook()
	_ = take_notes.Using(notebook)
	_ = take_notes.UsingEmptyNotepad()

	ch := make(chan string, 1)
	_ = wait.UntilReceived(ch)
	q := answerable.ValueOf("ready")
	_ = wait.Until(q, ve.Equals("ready"))
}
