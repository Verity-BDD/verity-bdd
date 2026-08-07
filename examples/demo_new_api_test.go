package examples

import (
	"context"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// TestNewAPIDemonstration demonstrates the new TestContext API without require.NoError
func TestNewAPIDemonstration(t *testing.T) {
	// Create VerityTest context - no more manual error handling!
	ctx := context.Background()
	test := verity.NewVerityTestWithContext(ctx, t)
	apiBaseURL := localJSONPlaceholderURL(t)

	// Create actor through test context
	apiTester := test.ActorCalled("APITester").WhoCan(api.CallAnApiAt(apiBaseURL))

	// Chain activities without require.NoError - errors are handled automatically!
	apiTester.AttemptsTo(
		api.SendGetRequest("/posts"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("title")),
	)

	// Multiple actors are supported
	user := test.ActorCalled("RegularUser").WhoCan(api.CallAnApiAt(apiBaseURL))

	user.AttemptsTo(
		api.SendGetRequest("/users"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("email")),
	)
}
