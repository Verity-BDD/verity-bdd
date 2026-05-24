package verity_wait_test

import (
	"testing"
	"time"

	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	vw "github.com/verity-bdd/verity-bdd/verity_wait"
)

func TestChannelReceiverAPIContractCompiles(t *testing.T) {
	ch := make(chan string, 1)
	_ = vw.UntilReceived(ch)
}

func TestChannelReceiverChainingCompiles(t *testing.T) {
	ch := make(chan int, 1)
	_ = vw.UntilReceived(ch).For(10 * time.Second)
}

func TestWaitAPIContractCompiles(t *testing.T) {
	q := answerable.ValueOf("ready")
	_ = vw.Until(q, ve.Equals("ready"))
}

func TestWaitAPIContractChainingCompiles(t *testing.T) {
	q := answerable.ValueOf("ready")
	_ = vw.Until(q, ve.Equals("ready")).
		For(30 * time.Second).
		CheckingEvery(1 * time.Second)
}
