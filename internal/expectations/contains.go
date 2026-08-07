package expectations

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// ContainsSubstringExpectation checks if a string contains the expected substring
type ContainsSubstringExpectation struct {
	substring string
}

// Evaluate evaluates the contains expectation
func (c ContainsSubstringExpectation) Evaluate(_ context.Context, _ core.Actor, actual string) error {
	if actual == "" {
		return fmt.Errorf("expected string to contain '%s', but got empty string", c.substring)
	}
	if !strings.Contains(actual, c.substring) {
		return fmt.Errorf("expected string to contain '%s', but got '%s'", c.substring, actual)
	}
	return nil
}

// Description returns the expectation description
func (c ContainsSubstringExpectation) Description() string {
	return fmt.Sprintf("contains '%s'", c.substring)
}

// ContainsSubstring checks if a string contains the expected substring.
func ContainsSubstring(substring string) ensure.Expectation[string] {
	return ContainsSubstringExpectation{substring: substring}
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

// ContainsSubstringAnswerToExpectation checks if a string contains the answer to a question as a substring
type ContainsSubstringAnswerToExpectation struct {
	question core.Question[string]
}

// Evaluate answers the question, then delegates to ContainsSubstringExpectation
func (c ContainsSubstringAnswerToExpectation) Evaluate(ctx context.Context, actor core.Actor, actual string) error {
	expected, err := c.question.AnsweredBy(ctx, actor)
	if err != nil {
		return newQuestionResolutionError(c.question.Description(), err)
	}
	return ContainsSubstringExpectation{substring: expected}.Evaluate(ctx, actor, actual)
}

// Description returns the expectation description
func (c ContainsSubstringAnswerToExpectation) Description() string {
	return fmt.Sprintf("contains the answer to '%s'", c.question.Description())
}

// ContainsSubstringAnswerTo checks if a string contains the answer to the given question as a substring.
// When used in a polling activity (e.g. wait.Until), the expected-value question is
// re-answered on every poll tick. To use a fixed expected value, resolve the question
// once and pass the result to ContainsSubstring instead.
func ContainsSubstringAnswerTo(q core.Question[string]) ensure.Expectation[string] {
	return ContainsSubstringAnswerToExpectation{question: q}
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
func ContainsKeyAnswerTo(q core.Question[string]) ensure.Expectation[interface{}] {
	return ContainsKeyAnswerToExpectation{question: q}
}
