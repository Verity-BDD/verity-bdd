package examples

import (
	"errors"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// TestAnswerableValueOf demonstrates the usage of answerable.ValueOf() API
// for creating Question[T] objects from static values.
//
// This enables the use of static values in ensure.That() assertions:
//
//	ensure.That(answerable.ValueOf(4), expectations.Equals(4))
//
// Previously, only core.Question[T] objects could be used in ensure.That(),
// but answerable.ValueOf() allows any static value to be wrapped as a Question.
func TestAnswerableValueOf(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("ValueTester")

	// Basic scalar values
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf(42), expectations.Equals(42)),
		ensure.That(answerable.ValueOf(3.14), expectations.Equals(3.14)),
		ensure.That(answerable.ValueOf("hello"), expectations.ContainsSubstring("hell")),
		ensure.That(answerable.ValueOf(true), expectations.Equals(true)),
	)

	// Complex types
	type Person struct {
		Name string
		Age  int
	}

	person := Person{Name: "Alice", Age: 25}
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf(person), expectations.Equals(Person{Name: "Alice", Age: 25})),
	)

	// Error values - check error message as string
	err := errors.New("connection failed")
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf(err.Error()), expectations.ContainsSubstring("connection")),
	)
}

// TestAnswerableWithMixedQuestions demonstrates mixing static value questions
// with traditional dynamic questions from API interactions.
func TestAnswerableWithMixedQuestions(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})
	apiBaseURL := localJSONPlaceholderURL(t)

	apiTester := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt(apiBaseURL))

	// Mix of static value questions (using answerable.ValueOf)
	// and dynamic questions (from API interactions)
	apiTester.AttemptsTo(
		// Dynamic: Get actual status from API
		api.SendGetRequest("/posts/1"),

		// Static: Compare against expected status code
		ensure.That(answerable.ValueOf(200), expectations.Equals(200)),

		// Dynamic: Get actual response body
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),

		// Static: Check response contains expected content
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("title")),

		// Static: Verify expected response structure
		ensure.That(answerable.ValueOf("userId"), expectations.ContainsSubstring("user")),
	)
}

// TestAnswerableDescriptions shows how answerable.ValueOf() generates
// clear descriptions that appear in test failure messages.
func TestAnswerableDescriptions(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("DescriptionTester")

	// These descriptions will appear in test output:
	// "42 (int)"
	// "hello (string)"
	// "error something went wrong (error)"
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf(42), expectations.Equals(42)),
		ensure.That(answerable.ValueOf("hello"), expectations.ContainsSubstring("hell")),
		ensure.That(answerable.ValueOf(errors.New("something went wrong").Error()), expectations.ContainsSubstring("something")),
	)
}

// TestAnswerableEdgeCases demonstrates handling of edge cases
func TestAnswerableEdgeCases(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("EdgeCaseTester")

	// Zero values
	actor.AttemptsTo(
		ensure.That(answerable.ValueOf(0), expectations.Equals(0)),
		ensure.That(answerable.ValueOf(false), expectations.Equals(false)),
		ensure.That(answerable.ValueOf(""), expectations.Equals("")),
	)
}
