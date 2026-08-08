// Package testing provides the TestContext API for simplified testing in Go.
//
// The TestContext API reports activity errors according to each activity's
// FailureMode through the testing.TB-like interface.
//
// Key Features:
//
//   - Failure-mode-aware error handling through TestContext
//   - Automatic actor cleanup through TestContext.Cleanup
//   - Integrated reporting capabilities
//   - Support for multiple actors in single test
//   - Thread-safe actor management
//
// Basic Usage:
//
//	test := verity.NewVerityTest(t, verity.Scene{})
//
//	actor := test.ActorCalled("APITester").WhoCan(
//		api.CallAnApiAt("https://api.example.com"),
//	)
//
//	actor.AttemptsTo(
//		api.SendGetRequest("/users"),
//		ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
//	)
//
// Multiple Actors:
//
//	test := verity.NewVerityTest(t, verity.Scene{})
//
//	admin := test.ActorCalled("Admin").WhoCan(api.CallAnApiAt(apiURL))
//	user := test.ActorCalled("User").WhoCan(api.CallAnApiAt(apiURL))
//
//	admin.AttemptsTo(api.SendPostRequest("/users").WithBody(userData))
//	user.AttemptsTo(api.SendGetRequest("/users/1"))
//
// Custom Reporting:
//
//	reporter := custom.NewJSONReporter()
//	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
//
// Error Handling:
//
//	Unlike the legacy API where errors need to be manually handled:
//
//	// AttemptsTo applies the activity's failure mode and returns no error.
//	actor.AttemptsTo(activity)
//	// FailFast: Errorf + FailNow; ErrorButContinue: Errorf; Ignore: Logf only.
//
// Thread Safety:
//
//	All actor operations are thread-safe. Multiple goroutines can safely
//	use actors created from the same VerityTest instance.
package testing

//go:generate go run go.uber.org/mock/mockgen@latest -source=context.go -destination=mocks/mock_test_context.go -package=mocks

// TestContext provides the testing hooks used by actors to handle activity errors.
// AttemptsTo applies the activity's FailureMode without returning an error.
//
// FailFast reports with Errorf and stops with FailNow; ErrorButContinue reports
// with Errorf and continues; Ignore logs with Logf and does not fail the test.
type TestContext interface {
	// Name returns the name of the test
	Name() string

	// Logf logs a formatted message
	Logf(format string, args ...interface{})

	// Errorf logs a formatted error message and marks the test as failed
	Errorf(format string, args ...interface{})

	// FailNow marks the test as failed and stops execution
	FailNow()

	// Failed returns true if the test has already failed
	Failed() bool

	Cleanup(func())

	Helper()
}

// Advanced Usage Examples:
//
// Concurrent Testing:
//
//	test := verity.NewVerityTest(t, verity.Scene{})
//
//	var wg sync.WaitGroup
//	actor := test.ActorCalled("ConcurrentUser").WhoCan(api.CallAnApiAt(apiURL))
//
//	for i := 0; i < 5; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//			actor.AttemptsTo(
//				api.SendGetRequest(fmt.Sprintf("/items/%d", id)),
//				ensure.That(api.LastResponseStatus{}, expectations.Equals(200)),
//			)
//		}(i)
//	}
//	wg.Wait()
//
// Error Scenarios:
//
//	test := verity.NewVerityTest(t, verity.Scene{})
//
//	actor := test.ActorCalled("ErrorProneUser").WhoCan(api.CallAnApiAt("https://invalid.example.com"))
//
//	// SendGetRequest is FailFast, so an error reports with Errorf and calls FailNow.
//	actor.AttemptsTo(api.SendGetRequest("/endpoint"))
//
// Custom Reporters:
//
//	reporter := &customReporter{output: os.Stdout}
//	test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)
//
//	actor := test.ActorCalled("ReportedUser").WhoCan(api.CallAnApiAt(apiURL))
//	actor.AttemptsTo(api.SendGetRequest("/users"))
