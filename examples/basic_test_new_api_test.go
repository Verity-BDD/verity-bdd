package examples

import (
	"context"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// TestJSONPlaceholderBasicsNewAPI demonstrates basic API testing with JSONPlaceholder using new TestContext API
func TestJSONPlaceholderBasicsNewAPI(t *testing.T) {
	ctx := context.Background()
	test := verity.NewVerityTestWithContext(ctx, t)
	apiBaseURL := localJSONPlaceholderURL(t)

	apiTester := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt(apiBaseURL))

	// Test GET posts - should return existing posts
	apiTester.AttemptsTo(
		api.SendGetRequest("/posts"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("title")),
	)

	// Test GET users - should return existing users
	apiTester.AttemptsTo(
		api.SendGetRequest("/users"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("email")),
	)

	// The console output will now show detailed step-by-step execution
	// with emojis, timing, and activity tracking thanks to ConsoleReporter
}
