package examples

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
	"github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
)

// TestConsoleReportingDemo demonstrates console reporting features
func TestConsoleReportingDemo(t *testing.T) {
	// Create custom console reporter with different output
	reporter := console_reporter.NewConsoleReporter()

	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
	apiBaseURL := localJSONPlaceholderURL(t)

	apiTester := test.ActorCalled("DemoAPITester").WhoCan(api.CallAnApiAt(apiBaseURL))

	// This will show detailed console output with emojis and timing
	apiTester.AttemptsTo(
		api.SendGetRequest("/posts"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("title")),
	)

	// Another sequence to demonstrate step tracking
	apiTester.AttemptsTo(
		api.SendGetRequest("/posts/1"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
		ensure.That(api.LastResponseBody{}, expectations.ContainsSubstring("userId")),
	)
}

// TestReportingToFile demonstrates outputting report to file
func TestReportingToFile(t *testing.T) {
	// Create file for output in current directory
	file, err := os.Create("test_report.txt")
	require.NoError(t, err)

	// Create reporter that writes to file
	reporter := console_reporter.NewConsoleReporter()
	reporter.SetOutput(file)

	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
	apiBaseURL := localJSONPlaceholderURL(t)

	apiTester := test.ActorCalled("FileReporter").WhoCan(api.CallAnApiAt(apiBaseURL))

	apiTester.AttemptsTo(
		api.SendGetRequest("/users"),
		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
	)

	// Ensure file is closed and flushed
	file.Close()

	// Verify file was created and contains content
	content, err := os.ReadFile("test_report.txt")
	require.NoError(t, err)
	require.Contains(t, string(content), "Starting: TestReportingToFile")
	require.Contains(t, string(content), "FileReporter sends GET request")
}
