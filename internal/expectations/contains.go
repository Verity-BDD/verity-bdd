package expectations

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// ContainsExpectation checks if a string contains the expected substring
type ContainsExpectation struct {
	substring string
}

// NewContains creates a new Contains expectation
func NewContains(substring string) ensure.Expectation[string] {
	return ContainsExpectation{substring: substring}
}

// Evaluate evaluates the contains expectation
func (c ContainsExpectation) Evaluate(_ context.Context, _ core.Actor, actual string) error {
	if actual == "" {
		return fmt.Errorf("expected string to contain '%s', but got empty string", c.substring)
	}
	if !strings.Contains(actual, c.substring) {
		return fmt.Errorf("expected string to contain '%s', but got '%s'", c.substring, actual)
	}
	return nil
}

// Description returns the expectation description
func (c ContainsExpectation) Description() string {
	return fmt.Sprintf("contains '%s'", c.substring)
}

// Convenience function for creating Contains expectations
func Contains(substring string) ensure.Expectation[string] {
	return NewContains(substring)
}

// ContainsKeyExpectation checks if a map contains the expected key
type ContainsKeyExpectation struct {
	key string
}

// NewContainsKey creates a new ContainsKey expectation
func NewContainsKey(key string) ensure.Expectation[interface{}] {
	return ContainsKeyExpectation{key: key}
}

// Evaluate evaluates the contains key expectation
func (ck ContainsKeyExpectation) Evaluate(_ context.Context, _ core.Actor, actual interface{}) error {
	val := reflect.ValueOf(actual)
	if val.Kind() != reflect.Map {
		return fmt.Errorf("expected a map, but got %T", actual)
	}

	// Try to convert to map[string]interface{} for string keys
	if mapStr, ok := actual.(map[string]interface{}); ok {
		if _, exists := mapStr[ck.key]; !exists {
			return fmt.Errorf("expected map to contain key '%s'", ck.key)
		}
		return nil
	}

	// Fallback to reflection for any map type
	mapKey := reflect.ValueOf(ck.key)
	if !val.MapIndex(mapKey).IsValid() {
		return fmt.Errorf("expected map to contain key '%s'", ck.key)
	}
	return nil
}

// Description returns the expectation description
func (ck ContainsKeyExpectation) Description() string {
	return fmt.Sprintf("contains key '%s'", ck.key)
}

// Convenience function for creating ContainsKey expectations
func ContainsKey(key string) ensure.Expectation[interface{}] {
	return NewContainsKey(key)
}

// ContainsAnswerToExpectation checks if a string contains the answer to a question as a substring
type ContainsAnswerToExpectation struct {
	question core.Question[string]
}

// Evaluate answers the question, then delegates to ContainsExpectation
func (c ContainsAnswerToExpectation) Evaluate(ctx context.Context, actor core.Actor, actual string) error {
	expected, err := c.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(c.question.Description(), err)
	}
	return ContainsExpectation{substring: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (c ContainsAnswerToExpectation) Description() string {
	return fmt.Sprintf("contains the answer to '%s'", c.question.Description())
}

// ContainsAnswerTo checks if a string contains the answer to the given question as a substring.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to Contains instead.
// If the expected-value question returns an error on any tick, the poll exits immediately
// rather than retrying — it does not count as a transient failure.
func ContainsAnswerTo(q core.Question[string]) ensure.Expectation[string] {
	return ContainsAnswerToExpectation{question: q}
}

// ContainsKeyAnswerToExpectation checks if a map contains the answer to a question as a key
type ContainsKeyAnswerToExpectation struct {
	question core.Question[string]
}

// Evaluate answers the question, then delegates to ContainsKeyExpectation
func (c ContainsKeyAnswerToExpectation) Evaluate(ctx context.Context, actor core.Actor, actual interface{}) error {
	expected, err := c.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(c.question.Description(), err)
	}
	return ContainsKeyExpectation{key: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (c ContainsKeyAnswerToExpectation) Description() string {
	return fmt.Sprintf("contains key from the answer to '%s'", c.question.Description())
}

// ContainsKeyAnswerTo checks if a map contains the answer to the given question as a key.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to ContainsKey instead.
// If the expected-value question returns an error on any tick, the poll exits immediately
// rather than retrying — it does not count as a transient failure.
func ContainsKeyAnswerTo(q core.Question[string]) ensure.Expectation[interface{}] {
	return ContainsKeyAnswerToExpectation{question: q}
}
