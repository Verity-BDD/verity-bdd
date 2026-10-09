package verity_expectations_test

import (
	"context"
	"fmt"

	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

func ExampleExist() {
	value := "available"
	expectation := ve.Exist[*string]()

	// Use the typed expectation with a question in an actor scenario.
	_ = ensure.That(answerable.ValueOf(&value), expectation)

	fmt.Println(expectation.Evaluate(context.Background(), nil, &value) == nil)
	// Output: true
}
