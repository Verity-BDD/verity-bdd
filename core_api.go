package verity

import (
	"context"

	internalabilities "github.com/verity-bdd/verity-bdd/internal/abilities"
	internalcore "github.com/verity-bdd/verity-bdd/internal/core"
)

// Ability enables an actor to interact with a specific interface of the system.
type Ability = internalabilities.Ability

// Actor represents a person or external system interacting with the system under test.
// Actors are the central figures in the Screenplay Pattern - they have abilities that enable
// them to perform activities and ask questions about the system state.
//
// Creating Actors:
//
//	// Basic actor creation
//	test := verity.NewVerityTest(t, verity.Scene{})
//	actor := test.ActorCalled("TestUser")
//
//	// Actor with abilities
//	actor := test.ActorCalled("APITester").WhoCan(
//		api.CallAnApiAt("https://api.example.com"),
//		db.ConnectToDatabase("postgres://localhost/test"),
//	)
//
// Actor Methods:
//
//	Name() returns the actor's identifier for logging and debugging.
//	WhoCan() adds abilities to the actor, returning the same actor for chaining.
//	AbilityTo() retrieves a specific ability by type for use in activities.
//	AttemptsTo() executes one or more activities sequentially.
//
// Example Usage:
//
//	test := verity.NewVerityTest(t, verity.Scene{})
//	actor := test.ActorCalled("OrderManager").WhoCan(
//		api.CallAnApiAt("https://api.shop.com"),
//		db.ConnectToDatabase("postgres://localhost/shop"),
//	)
//
//	// Perform activities
//	err := actor.AttemptsTo(
//		Do("creates customer order", func(ctx context.Context, a Actor) error {
//			return createOrder(orderData).PerformAs(ctx, a)
//		}),
//		Do("verifies order in database", func(ctx context.Context, a Actor) error {
//			return verifyOrder(orderId).PerformAs(ctx, a)
//		}),
//	)
//
// Thread Safety:
//
//	Actor implementations are thread-safe for ability management.
//	Activities are executed sequentially unless explicitly designed for concurrency.
type Actor = internalcore.Actor

// Activity represents an action that an actor can perform.
// Activities are the building blocks of test scenarios in the Screenplay Pattern.
// They define what actors do rather than how they interact with specific interfaces.
//
// Activity Types:
//
//	Interaction - Low-level, atomic operations (single API calls, database queries)
//	Task        - High-level, business-focused activities composed of multiple interactions
//
// Creating Activities:
//
//	// Simple interaction using Do
//	sendRequest := Do("sends GET request", func(ctx context.Context, actor Actor) error {
//		return api.SendGetRequest("/users").PerformAs(ctx, actor)
//	})
//
//	// Composed task using TaskWhere
//	createUserFlow := TaskWhere("creates new user account",
//		Do("validates user data", validateData),
//		Do("creates user via API", createUser),
//		Do("verifies user in database", verifyUser),
//	)
//
// Activity Lifecycle:
//
//  1. Actor calls AttemptsTo() with one or more activities
//  2. Each activity's PerformAs() method is called with the actor
//  3. Activity uses actor's abilities to perform its action
//  4. Activity returns success or error
//  5. Actor handles errors based on activity's FailureMode
type Activity = internalcore.Activity

// Interaction represents a low-level activity (atomic operation).
// Interactions are single, focused operations that typically involve
// one system call or interface interaction.
//
// Characteristics of Interactions:
//
//   - Atomic: Cannot be broken down into smaller operations
//   - Focused: Do one thing well
//   - Low-level: Direct system interactions
//   - Reusable: Can be composed into tasks
//
// Examples of Interactions:
//
//	// API call interaction
//	sendGetRequest := Do("sends GET request to /users", func(ctx context.Context, actor Actor) error {
//		ability, err := AbilityOf[api.CallAnAPI](actor)
//		if err != nil {
//			return fmt.Errorf("actor needs API ability: %w", err)
//		}
//		return ability.SendGetRequest("/users")
//	})
//
// Custom Interaction Types:
//
//	type SendEmailActivity struct {
//		to      string
//		subject string
//		body    string
//	}
//
//	func (s *SendEmailActivity) PerformAs(ctx context.Context, actor Actor) error {
//		// Implementation for sending email
//	}
//
//	func (s *SendEmailActivity) Description() string {
//		return fmt.Sprintf("sends email to %s with subject '%s'", s.to, s.subject)
//	}
//
//	func (s *SendEmailActivity) FailureMode() FailureMode {
//		return FailFast
//	}
type Interaction = internalcore.Interaction

// Task represents a high-level business-focused activity composed of interactions.
// Tasks represent meaningful business outcomes rather than technical operations.
// They are composed of multiple interactions that work together to achieve a goal.
//
// Characteristics of Tasks:
//
//   - Business-focused: Describe what the user accomplishes
//   - Composed: Made up of multiple interactions
//   - High-level: Abstract away technical details
//   - Meaningful: Represent valuable user outcomes
//
// Examples of Tasks:
//
//	// User registration task
//	registerUser := TaskWhere("registers new user account",
//		Do("validates user input", validateUserData),
//		Do("creates user via API", createUserInAPI),
//		Do("verifies user in database", verifyUserInDB),
//		Do("sends welcome email", sendWelcomeEmail),
//	)
//
// Task vs Interaction Guidelines:
//
//	Use Interactions for:
//	- Single API calls
//	- Database queries
//	- File operations
//	- UI interactions
//
//	Use Tasks for:
//	- User registration flows
//	- Order processing
//	- Data migration
//	- Complex business workflows
type Task = internalcore.Task

// Question enables actors to retrieve information from the system.
// Questions use Go generics to provide type-safe answers about system state.
// They separate the concern of data retrieval from assertion logic.
//
// Creating Questions:
//
//	// Using QuestionAbout (convenience factory)
//	userCount := QuestionAbout("user count", func(ctx context.Context, actor Actor) (int, error) {
//		db, err := AbilityOf[database.DatabaseAbility](actor)
//		if err != nil {
//			return 0, err
//		}
//		return db.QueryRow("SELECT COUNT(*) FROM users").Int()
//	})
//
//	// Using QuestionAbout
//	userName := QuestionAbout("current user name", func(ctx context.Context, actor Actor) (string, error) {
//		session, err := AbilityOf[auth.SessionAbility](actor)
//		if err != nil {
//			return "", err
//		}
//		return session.GetCurrentUser().Name, nil
//	})
//
// Using Questions:
//
//	// Direct usage
//	count, err := userCount.AnsweredBy(ctx, actor)
//	if err != nil {
//		return fmt.Errorf("failed to get user count: %w", err)
//	}
//
//	// With expectations (recommended)
//	actor.AttemptsTo(
//		ensure.That(userCount, expectations.GreaterThan(0)),
//		ensure.That(userName, expectations.ContainsSubstring("admin")),
//	)
//
// Thread Safety:
//
//	Questions should be stateless or handle their own synchronization.
//	When called by multiple actors concurrently, they should not share mutable state.
type Question[T any] interface {
	Description() string
	AnsweredBy(ctx context.Context, actor Actor) (T, error)
}

// TestResult represents the outcome of a test execution.
// This struct provides comprehensive information about test execution
// including timing, status, and error details.
//
// Usage:
//
//	result := &TestResult{
//		Name:     "user registration test",
//		Status:   StatusPassed,
//		Duration: 2 * time.Second,
//	}
//
//	// Human-readable output
//	fmt.Printf("Test %s: %s (%v)\n", result.Name, result.Status, result.Duration)
type TestResult = internalcore.TestResult

// Status represents the test execution status.
// Status values follow the standard test lifecycle states.
//
// Status Flow:
//
//	StatusPending → StatusRunning → (StatusPassed | StatusFailed | StatusSkipped)
//
// Usage:
//
//	result := &TestResult{
//		Name:   "API connectivity test",
//		Status: StatusRunning,
//	}
//
//	// Update status based on test outcome
//	if testError != nil {
//		result.Status = StatusFailed
//		result.Error = testError
//	} else {
//		result.Status = StatusPassed
//	}
type Status = internalcore.Status

// FailureMode defines how activities should handle failures.
// This type determines whether test execution continues or stops
// when an activity encounters an error.
//
// Default Behavior:
//
//	All activities use FailFast mode unless explicitly overridden.
type FailureMode = internalcore.FailureMode

const (
	// StatusPending indicates the test has been created but not yet started.
	// This is the initial state for all tests.
	StatusPending = internalcore.StatusPending

	// StatusRunning indicates the test is currently executing.
	// Set when test execution begins.
	StatusRunning = internalcore.StatusRunning

	// StatusPassed indicates the test completed successfully.
	// All assertions passed and no errors occurred.
	StatusPassed = internalcore.StatusPassed

	// StatusFailed indicates the test completed with errors.
	// One or more assertions failed or an error occurred.
	StatusFailed = internalcore.StatusFailed

	// StatusSkipped indicates the test was not executed.
	// Typically due to preconditions not being met.
	StatusSkipped = internalcore.StatusSkipped
)

const (
	// FailFast stops execution immediately when an activity fails.
	// This is the default and most commonly used failure mode.
	//
	// Use Cases:
	//	- Critical setup operations
	//	- Main test scenarios
	//	- Dependencies required for subsequent steps
	//
	// Behavior:
	//	- Stops execution immediately on error
	//	- Returns the first error encountered
	//	- Subsequent activities are not executed
	FailFast = internalcore.FailFast

	// ErrorButContinue logs the error but continues with remaining activities.
	// Use this for non-critical operations where you want to know about
	// failures but don't want them to stop the entire test.
	//
	// Use Cases:
	//	- Monitoring and metrics collection
	//	- Non-essential notifications
	//	- Audit logging
	//
	// Behavior:
	//	- Logs the error but continues execution
	//	- Later activities continue to execute
	ErrorButContinue = internalcore.ErrorButContinue

	// Ignore completely ignores the failure and continues.
	// Use this for truly optional operations where failure is acceptable
	// or expected in certain scenarios.
	//
	// Use Cases:
	//	- Optional cleanup operations
	//	- Best-effort notifications
	//	- Precondition checks that may legitimately fail
	//
	// Behavior:
	//	- Completely ignores any errors
	//	- Never returns errors from this activity
	//	- Execution continues regardless of success or failure
	Ignore = internalcore.Ignore
)

// Do creates a new interaction with the given description and perform function.
// This is the primary factory function for creating simple, atomic activities.
// Interactions created with Do are ideal for quick, focused operations.
//
// Parameters:
//   - description: Human-readable description of what the interaction does
//   - perform: Function that executes the interaction logic
//
// Returns:
//   - Interaction: A new interaction that executes the provided function
//
// Usage Examples:
//
//	// Simple API call interaction
//	sendGetRequest := Do("sends GET request to /users", func(ctx context.Context, actor Actor) error {
//		api, err := AbilityOf[api.CallAnAPI](actor)
//		if err != nil {
//			return fmt.Errorf("actor needs API ability: %w", err)
//		}
//		return api.SendGetRequest("/users")
//	})
//
//	// As part of a task
//	userWorkflow := TaskWhere("user login workflow",
//		Do("loads user credentials", loadCredentials),
//		Do("authenticates with API", authenticate),
//		Do("verifies session", verifySession),
//	)
func Do(description string, perform func(context.Context, Actor) error) Interaction {
	return internalcore.Do(description, perform)
}

// TaskWhere creates a new task with the given description and activities.
// This is the factory function for creating composed tasks that represent
// meaningful business operations.
//
// Parameters:
//   - description: Human-readable description of what the task accomplishes
//   - activities: One or more activities that compose this task
//
// Returns:
//   - Task: A new task instance that executes activities sequentially
//
// Usage Examples:
//
//	registerUser := TaskWhere("registers new user account",
//		Do("validates user data", validateUserData),
//		Do("creates user via API", createUser),
//		Do("verifies user in database", verifyUser),
//	)
//
//	err := actor.AttemptsTo(registerUser)
//	if err != nil {
//		return fmt.Errorf("test workflow failed: %w", err)
//	}
func TaskWhere(description string, activities ...Activity) Task {
	return internalcore.TaskWhere(description, activities...)
}

// Critical returns a failure mode that stops execution on failure.
// This is a semantic function that returns FailFast mode.
// Use this when you want to explicitly indicate critical operations.
//
// Returns:
//   - FailureMode: Always returns FailFast
//
// Usage:
//
//	actor.AttemptsTo(
//		Do("establishes database connection", connectDB).WithFailureMode(Critical()),
//		Do("creates user account", createUser),
//	)
func Critical() FailureMode {
	return internalcore.Critical()
}

// NonCritical returns a failure mode that logs errors but continues.
// This is a semantic function that returns ErrorButContinue mode.
// Use this for operations that should be noted but not stop execution.
//
// Returns:
//   - FailureMode: Always returns ErrorButContinue
//
// Usage:
//
//	actor.AttemptsTo(
//		Do("main business operation", businessLogic),
//		Do("collects usage metrics", collectMetrics).WithFailureMode(NonCritical()),
//		Do("verifies result", verifyResult),
//	)
func NonCritical() FailureMode {
	return internalcore.NonCritical()
}

// Optional returns a failure mode that ignores errors completely.
// This is a semantic function that returns Ignore mode.
// Use this for operations where failure is acceptable or expected.
//
// Returns:
//   - FailureMode: Always returns Ignore
//
// Usage:
//
//	actor.AttemptsTo(
//		Do("main test execution", mainTest),
//		Do("attempts to cleanup resources", cleanup).WithFailureMode(Optional()),
//		Do("final assertion", finalAssertion),
//	)
func Optional() FailureMode {
	return internalcore.Optional()
}

// AbilityName returns the readable name of the provided ability instance.
func AbilityName(ability Ability) string {
	return internalcore.AbilityName(ability)
}

// QuestionAbout creates a new question with the given description and ask function.
// It panics if ask is nil.
//
// Type Parameters:
//   - T: The type of answer this question returns
//
// Parameters:
//   - description: Human-readable description of what the question asks
//   - ask: Function that takes a context and actor, returning the typed answer
//
// Returns:
//   - Question[T]: A new question that returns type T when answered
//
// Usage Examples:
//
//	isHealthy := QuestionAbout("system health status", func(ctx context.Context, actor Actor) (bool, error) {
//		health, err := AbilityOf[monitoring.HealthAbility](actor)
//		if err != nil {
//			return false, err
//		}
//		return health.IsHealthy()
//	})
//
//	// With expectations
//	actor.AttemptsTo(
//		ensure.That(isHealthy, expectations.IsTrue()),
//	)
func QuestionAbout[T any](description string, ask func(ctx context.Context, actor Actor) (T, error)) Question[T] {
	return internalcore.QuestionAbout(description, ask)
}

// AbilityOf returns the first ability of type T held by the actor, or an error
// with a human-friendly message when missing.
//
// Type Parameters:
//   - T: The ability type to retrieve
//
// Parameters:
//   - actor: The actor whose ability to retrieve
//
// Returns:
//   - T: The typed ability instance
//   - error: Error if the actor does not have the requested ability
//
// Usage:
//
//	apiAbility, err := AbilityOf[api.CallAnAPI](actor)
//	if err != nil {
//		return fmt.Errorf("actor needs API ability: %w", err)
//	}
//	response, err := apiAbility.SendGetRequest("/users")
func AbilityOf[T Ability](actor Actor) (T, error) {
	return internalcore.AbilityOf[T](actor)
}
