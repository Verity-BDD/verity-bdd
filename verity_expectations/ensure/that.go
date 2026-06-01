package ensure

import (
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	internalensure "github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

type Expectation[T any] interface {
	Evaluate(actual T) error
	Description() string
}

type EnsureThat[T any] interface {
	verity.Activity
	After(duration time.Duration) verity.Activity
}

func That[T any](question verity.Question[T], expectation Expectation[T]) EnsureThat[T] {
	return internalensure.That(question, expectation)
}
