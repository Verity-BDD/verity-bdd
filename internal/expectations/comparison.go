package expectations

import (
	"context"
	"fmt"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
	"github.com/verity-bdd/verity-bdd/internal/expectations/utils"
)

// IsGreaterThanExpectation checks if a numeric value is greater than expected
type IsGreaterThanExpectation struct {
	expected interface{}
}

// NewIsGreaterThan creates a new IsGreaterThan expectation
func NewIsGreaterThan(expected interface{}) ensure.Expectation[interface{}] {
	return IsGreaterThanExpectation{expected: expected}
}

// Evaluate evaluates the greater than expectation
func (igt IsGreaterThanExpectation) Evaluate(_ context.Context, _ core.Actor, actual interface{}) error {
	return utils.CompareValues(actual, igt.expected, ">")
}

// Description returns the expectation description
func (igt IsGreaterThanExpectation) Description() string {
	return fmt.Sprintf("is greater than %v", igt.expected)
}

// Convenience function for creating IsGreaterThan expectations
func IsGreaterThan(expected interface{}) ensure.Expectation[interface{}] {
	return NewIsGreaterThan(expected)
}

// IsLessThanExpectation checks if a numeric value is less than expected
type IsLessThanExpectation struct {
	expected interface{}
}

// NewIsLessThan creates a new IsLessThan expectation
func NewIsLessThan(expected interface{}) ensure.Expectation[interface{}] {
	return IsLessThanExpectation{expected: expected}
}

// Evaluate evaluates the less than expectation
func (ilt IsLessThanExpectation) Evaluate(_ context.Context, _ core.Actor, actual interface{}) error {
	return utils.CompareValues(actual, ilt.expected, "<")
}

// Description returns the expectation description
func (ilt IsLessThanExpectation) Description() string {
	return fmt.Sprintf("is less than %v", ilt.expected)
}

// Convenience function for creating IsLessThan expectations
func IsLessThan(expected interface{}) ensure.Expectation[interface{}] {
	return NewIsLessThan(expected)
}

// IsGreaterThanAnswerToExpectation checks if a value is greater than the answer to a question
type IsGreaterThanAnswerToExpectation struct {
	question core.Question[interface{}]
}

// Evaluate answers the question, then delegates to IsGreaterThanExpectation
func (ig IsGreaterThanAnswerToExpectation) Evaluate(ctx context.Context, actor core.Actor, actual interface{}) error {
	expected, err := ig.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(ig.question.Description(), err)
	}
	return IsGreaterThanExpectation{expected: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (ig IsGreaterThanAnswerToExpectation) Description() string {
	return fmt.Sprintf("is greater than the answer to '%s'", ig.question.Description())
}

// IsGreaterThanAnswerTo checks if a value is greater than the answer to the given question.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to IsGreaterThan instead.
// If the expected-value question returns an error on any tick, the poll exits immediately
// rather than retrying — it does not count as a transient failure.
func IsGreaterThanAnswerTo(q core.Question[interface{}]) ensure.Expectation[interface{}] {
	return IsGreaterThanAnswerToExpectation{question: q}
}

// IsLessThanAnswerToExpectation checks if a value is less than the answer to a question
type IsLessThanAnswerToExpectation struct {
	question core.Question[interface{}]
}

// Evaluate answers the question, then delegates to IsLessThanExpectation
func (il IsLessThanAnswerToExpectation) Evaluate(ctx context.Context, actor core.Actor, actual interface{}) error {
	expected, err := il.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(il.question.Description(), err)
	}
	return IsLessThanExpectation{expected: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (il IsLessThanAnswerToExpectation) Description() string {
	return fmt.Sprintf("is less than the answer to '%s'", il.question.Description())
}

// IsLessThanAnswerTo checks if a value is less than the answer to the given question.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to IsLessThan instead.
// If the expected-value question returns an error on any tick, the poll exits immediately
// rather than retrying — it does not count as a transient failure.
func IsLessThanAnswerTo(q core.Question[interface{}]) ensure.Expectation[interface{}] {
	return IsLessThanAnswerToExpectation{question: q}
}
