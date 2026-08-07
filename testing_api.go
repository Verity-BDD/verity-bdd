package verity

import (
	"context"

	internaltesting "github.com/verity-bdd/verity-bdd/internal/testing"
	reporting "github.com/verity-bdd/verity-bdd/verity_reporting"
)

// ReporterProvider provides access to reporter adapter.
type ReporterProvider = internaltesting.ReporterProvider

// DefaultAbilityFactory creates a default ability for the actor name.
type DefaultAbilityFactory = internaltesting.DefaultAbilityFactory

// Scene configures VerityTest runtime behavior.
type Scene = internaltesting.Scene

// VerityTest manages the lifecycle of test actors and provides the TestContext API.
// This interface serves as the main entry point for using the simplified testing approach.
//
// Lifecycle Management:
//  1. Create test instance with NewVerityTest() or NewVerityTestWithReporter()
//  2. Create actors using ActorCalled()
//  3. Execute test activities
//  4. Call Shutdown() to clean up resources (typically via defer)
//
// Thread Safety:
//
//	All VerityTest methods are thread-safe. Multiple goroutines can safely
//	create and use actors from the same test instance.
type VerityTest = internaltesting.VerityTest

// TestContext provides a testing.TB wrapper for automatic error handling.
// This interface enables the TestContext API where test failures are automatically
// handled without the need for manual error checking.
//
// Methods automatically call t.Helper() and t.Fatalf() on errors,
// eliminating the need for require.NoError() calls in test code.
type TestContext = internaltesting.TestContext

// NewVerityTest creates a new VerityTest instance.
func NewVerityTest(t TestContext, scene Scene) VerityTest {
	return internaltesting.NewVerityTest(t, scene)
}

// NewVerityTestWithContext creates a new VerityTest instance using the provided context.
func NewVerityTestWithContext(ctx context.Context, t TestContext) VerityTest {
	return internaltesting.NewVerityTestWithContext(ctx, t)
}

// NewVerityTestWithReporter creates a new VerityTest instance with a reporter.
func NewVerityTestWithReporter(ctx context.Context, t TestContext, reporter reporting.Reporter) VerityTest {
	return internaltesting.NewVerityTestWithReporter(ctx, t, reporter)
}
