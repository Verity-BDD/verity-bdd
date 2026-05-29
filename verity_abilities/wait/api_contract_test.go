package wait_test

import (
	"testing"
	"time"

	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"

	"github.com/verity-bdd/verity-bdd/verity_abilities/wait"
)

func TestChannelReceiverAPIContractCompiles(t *testing.T) {
	t.Parallel()
	ch := make(chan string, 1)
	_ = wait.UntilReceived(ch)
}

func TestChannelReceiverChainingCompiles(t *testing.T) {
	t.Parallel()
	ch := make(chan int, 1)
	_ = wait.UntilReceived(ch).For(10 * time.Second)
}

func TestWaitAPIContractCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("ready")
	_ = wait.Until(q, ve.Equals("ready"))
}

func TestWaitAPIContractChainingCompiles(t *testing.T) {
	t.Parallel()
	q := answerable.ValueOf("ready")
	_ = wait.Until(q, ve.Equals("ready")).
		For(30 * time.Second).
		CheckingEvery(1 * time.Second)
}
