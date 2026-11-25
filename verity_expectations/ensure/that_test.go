package ensure_test

import (
	"testing"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

func TestThatWithFailureModeCompiles(t *testing.T) {
	t.Parallel()

	_ = ensure.That(answerable.ValueOf("ready"), ve.Equals("ready")).
		WithFailureMode(verity.Optional()).
		After(time.Second)
}

func TestThatWithFailureModePreservesModeAfterDelay(t *testing.T) {
	t.Parallel()

	activity := ensure.That(answerable.ValueOf("not ready"), ve.Equals("ready")).
		WithFailureMode(verity.Optional()).
		After(time.Second)

	if got := activity.FailureMode(); got != verity.Ignore {
		t.Fatalf("expected Ignore, got %v", got)
	}
}
