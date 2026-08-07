// Package core implements the Screenplay Pattern for acceptance testing in Go.
//
// The Screenplay Pattern is an alternative to traditional Page Object Model that focuses
// on what users can do with a system rather than how they interact with specific UI elements.
// This package provides the fundamental interfaces and implementations for actor-centric testing.
//
// Core Concepts:
//
//	Actor    - Represents a user or system interacting with the application under test
//	Activity - An action that an actor can perform (interactions and tasks)
//	Question - Enables actors to retrieve information about system state
//	Ability  - Capabilities that actors possess (defined in abilities package)
//
// Basic Usage:
//
//	// Create an actor with specific abilities
//	test := verity.NewVerityTest(t, verity.Scene{})
//	actor := test.ActorCalled("TestUser").WhoCan(
//		api.CallAnApiAt("https://api.example.com"),
//		db.ConnectToDatabase("postgres://localhost/test"),
//	)
//
//	// Perform simple interactions
//	actor.AttemptsTo(
//		core.Do("sends GET request", func(ctx context.Context, a core.Actor) error {
//			return api.SendGetRequest("/users").PerformAs(ctx, a)
//		}),
//	)
//
//	// Create composed tasks
//	actor.AttemptsTo(
//		core.TaskWhere("creates and verifies user",
//			core.Do("creates user via API", createUser),
//			core.Do("verifies user in database", verifyUser),
//		),
//	)
//
//	// Ask questions about system state
//	userCount := core.QuestionAbout("user count", func(_ context.Context, actor core.Actor) (int, error) {
//		db := actor.AbilityTo(&database.DatabaseAbility{}).(database.DatabaseAbility)
//		return db.QueryRow("SELECT COUNT(*) FROM users").Int()
//	})
//
//	count, err := userCount.AnsweredBy(ctx, actor)
//	if err != nil {
//		return fmt.Errorf("failed to get user count: %w", err)
//	}
//
// Activity Types:
//
//	Interaction - Low-level, atomic operations (single API calls, database queries)
//	Task        - High-level, business-focused activities composed of multiple interactions
//
//	// Interaction example
//	sendRequest := core.Do("sends POST request", func(ctx context.Context, actor core.Actor) error {
//		return api.SendPostRequest("/users", userData).PerformAs(ctx, actor)
//	})
//
//	// Task example
//	createUserFlow := core.TaskWhere("creates new user account",
//		core.Do("validates user data", validateData),
//		sendRequest,
//		core.Do("verifies user creation", verifyCreation),
//	)
//
// Questions and Type Safety:
//
//	Questions use Go generics for type-safe answers about system state:
//
//	// Type-safe question with generic parameter
//	userName := core.QuestionAbout("current user name", func(ctx context.Context, actor core.Actor) (string, error) {
//		session := actor.AbilityTo(&auth.SessionAbility{}).(auth.SessionAbility)
//		return session.GetCurrentUser().Name, nil
//	})
//
//	// Complex type question
//	userProfile := core.QuestionAbout("user profile", func(ctx context.Context, actor core.Actor) (*UserProfile, error) {
//		db := actor.AbilityTo(&database.DatabaseAbility{}).(database.DatabaseAbility)
//		return db.GetUserProfile(actor.Name())
//	})
//
//	// Questions can be used directly or with expectations
//	actor.AttemptsTo(
//		ensure.That(userName, expectations.Contains("John")),
//		ensure.That(userProfile, expectations.HasField("Email", expectations.IsNotEmpty())),
//	)
//
// Failure Modes:
//
//	Activities can specify how failures should be handled:
//
//	// FailFast (default) - stops execution on first failure
//	actor.AttemptsTo(
//		core.Do("critical operation", criticalStep), // stops if this fails
//		core.Do("cleanup operation", cleanupStep),    // won't execute if above fails
//	)
//
//	// ErrorButContinue - logs error but continues execution
//	actor.AttemptsTo(
//		core.Do("non-critical step", nonCriticalStep).WithFailureMode(core.NonCritical()),
//		core.Do("another step", anotherStep), // executes even if above fails
//	)
//
//	// Ignore - completely ignores failures
//	actor.AttemptsTo(
//		core.Do("optional cleanup", optionalCleanup).WithFailureMode(core.Optional()),
//	)
//
// Integration with Other Packages:
//
//	abilities  - Define what actors can do (API calls, database operations, etc.)
//	expectations - Provide assertion capabilities for questions and activities
//	testing    - Offer TestContext API for simplified test setup and management
//
// Best Practices:
//
//  1. Use descriptive names for actors, activities, and questions
//  2. Prefer tasks over long sequences of interactions for business logic
//  3. Use questions to separate data retrieval from assertions
//  4. Choose appropriate failure modes based on business criticality
//  5. Keep interactions focused on single operations
//  6. Use type-safe questions with generics when possible
//
// Thread Safety:
//
//	All core interfaces are designed to be thread-safe when used properly.
//	Actor implementations use internal synchronization for ability management.
//	Activities and questions should be stateless or handle their own synchronization.
package core

import (
	"context"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
)

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
//	// Using TestContext API (recommended)
//	test := verity.NewVerityTest(t, verity.Scene{})
//	actor := test.ActorCalled("WebUser").WhoCan(
//		web.BrowseTheWebWith(selenium.ChromeDriver()),
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
//		core.Do("creates customer order", func(ctx context.Context, a core.Actor) error {
//			return createOrder(orderData).PerformAs(ctx, a)
//		}),
//		core.Do("verifies order in database", func(ctx context.Context, a core.Actor) error {
//			return verifyOrder(orderId).PerformAs(ctx, a)
//		}),
//	)
//
//	// Access abilities directly
//	apiAbility, err := actor.AbilityTo(&api.CallAnAPI{})
//	if err != nil {
//		return fmt.Errorf("actor lacks API ability: %w", err)
//	}
//
//	// Use ability for custom operations
//	response, err := apiAbility.(api.CallAnAPI).SendRequest(request)
//
// Thread Safety:
//
//	Actor implementations are thread-safe for ability management.
//	Activities are executed sequentially unless explicitly designed for concurrency.
type Actor interface {
	// Context returns the actor's context for cancellation and timeout.
	//
	// Returns:
	//   - context.Context: The context associated with this actor
	Context() context.Context

	// Name returns the actor's name for identification in logs and test reports.
	//
	// Returns:
	//   - string: The actor's unique identifier
	//
	// Example:
	//
	//	test := verity.NewVerityTest(t, verity.Scene{})
	//	actor := test.ActorCalled("TestUser")
	//	fmt.Println(actor.Name()) // Output: "TestUser"
	Name() string

	// WhoCan gives the actor additional abilities to interact with the system.
	// Returns the same actor instance for method chaining.
	//
	// Parameters:
	//   - abilities: One or more abilities to add to the actor
	//
	// Returns:
	//   - Actor: The same actor instance with new abilities added
	//
	// Example:
	//
	//	test := verity.NewVerityTest(t, verity.Scene{})
	//	actor := test.ActorCalled("FullStackTester").WhoCan(
	//		api.CallAnApiAt("https://api.example.com"),
	//		db.ConnectToDatabase("postgres://localhost/test"),
	//		fs.ManageFilesIn("/tmp/test"),
	//	)
	WhoCan(abilities ...abilities.Ability) Actor

	// AbilityTo retrieves a specific ability from the actor by type.
	// Returns an error if the actor doesn't have the requested ability.
	//
	// Parameters:
	//   - ability: A zero-value instance of the ability type to retrieve
	//
	// Returns:
	//   - abilities.Ability: The requested ability instance
	//   - error: Error if the ability is not found
	//
	// Example:
	//
	//	// Get API ability
	//	apiAbility, err := actor.AbilityTo(&api.CallAnAPI{})
	//	if err != nil {
	//		return fmt.Errorf("actor needs API ability: %w", err)
	//	}
	//
	//	// Use the ability
	//	response, err := apiAbility.(api.CallAnAPI).SendRequest(request)
	//
	// Common pattern for ability access:
	//
	//	ability, err := actor.AbilityTo(&targetType{})
	//	if err != nil {
	//		return fmt.Errorf("actor does not have required ability: %w", err)
	//	}
	//	specificAbility := ability.(TargetType)
	AbilityTo(ability abilities.Ability) (abilities.Ability, error)

	// AttemptsTo performs one or more activities sequentially.
	// Stops execution immediately if any activity fails (unless using custom failure modes).
	//
	// Parameters:
	//   - activities: One or more activities to perform
	//
	// Returns:
	//   - error: The first error encountered during activity execution
	//
	// Example:
	//
	//	err := actor.AttemptsTo(
	//		core.Do("logs into system", login),
	//		core.Do("creates user account", createUser),
	//		core.Do("verifies user creation", verifyUser),
	//	)
	//	if err != nil {
	//		return fmt.Errorf("user creation flow failed: %w", err)
	//	}
	//
	// With custom failure modes:
	//
	//	actor.AttemptsTo(
	//		core.Do("critical step", criticalStep),
	//		core.Do("optional cleanup", cleanup).WithFailureMode(core.Optional()),
	//		core.Do("final verification", verification),
	//	)
	AttemptsTo(activities ...Activity)
}

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
//	// Simple interaction using core.Do
//	sendRequest := core.Do("sends GET request", func(ctx context.Context, actor core.Actor) error {
//		return api.SendGetRequest("/users").PerformAs(ctx, actor)
//	})
//
//	// Custom interaction type
//	type SendRequestActivity struct {
//		method string
//		path   string
//	}
//
//	func (s *SendRequestActivity) PerformAs(ctx context.Context, actor core.Actor) error {
//		// implementation
//	}
//
//	// Composed task using core.TaskWhere
//	createUserFlow := core.TaskWhere("creates new user account",
//		core.Do("validates user data", validateData),
//		core.Do("creates user via API", createUser),
//		core.Do("verifies user in database", verifyUser),
//	)
//
// Activity Lifecycle:
//
//  1. Actor calls AttemptsTo() with one or more activities
//  2. Each activity's PerformAs() method is called with the actor
//  3. Activity uses actor's abilities to perform its action
//  4. Activity returns success or error
//  5. Actor handles errors based on activity's FailureMode
//
// Error Handling:
//
//	Activities should return descriptive errors with context:
//
//	func (a *MyActivity) PerformAs(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(&api.CallAnAPI{})
//		if err != nil {
//			return fmt.Errorf("actor lacks API ability: %w", err)
//		}
//
//		// Perform action...
//		if err != nil {
//			return fmt.Errorf("failed to send request: %w", err)
//		}
//
//		return nil
//	}
type Activity interface {
	// PerformAs executes the activity as the given actor.
	// This is where the activity's main logic is implemented.
	//
	// The activity should:
	// 1. Access required abilities from the actor
	// 2. Perform its intended action
	// 3. Return nil on success or descriptive error on failure
	//
	// Parameters:
	//   - actor: The actor performing this activity
	//   - ctx: Context for cancellation and timeout
	//
	// Returns:
	//   - error: Nil on success, error details on failure
	//
	// Example:
	//
	//	func (s *SendRequestActivity) PerformAs(ctx context.Context, actor core.Actor) error {
	//		ability, err := actor.AbilityTo(&api.CallAnAPI{})
	//		if err != nil {
	//			return fmt.Errorf("actor needs API ability: %w", err)
	//		}
	//
	//		api := ability.(api.CallAnAPI)
	//		return api.SendRequest(s.method, s.path, ctx)
	//	}
	PerformAs(ctx context.Context, actor Actor) error

	// Description returns a human-readable description of the activity.
	// This description is used in test reports and logging.
	//
	// Returns:
	//   - string: Human-readable description of what the activity does
	//
	// Examples:
	//
	//	"creates user account via API"
	//	"verifies user exists in database"
	//	"sends POST request to /orders"
	//	"logs into system with valid credentials"
	Description() string

	// FailureMode returns how the activity should handle failures.
	// This determines whether execution stops on errors or continues.
	//
	// Returns:
	//   - FailureMode: How to handle failures (FailFast, ErrorButContinue, Ignore)
	//
	// Default behavior:
	//	- Most activities use FailFast (stop on first error)
	//	- Can be overridden with WithFailureMode() on core.Do activities
	//
	// Example:
	//
	//	func (t *task) FailureMode() core.FailureMode {
	//		return core.FailFast // Default for tasks
	//	}
	FailureMode() FailureMode
}

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
//	sendGetRequest := core.Do("sends GET request to /users", func(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(&api.CallAnAPI{})
//		if err != nil {
//			return fmt.Errorf("actor needs API ability: %w", err)
//		}
//		return ability.(api.CallAnAPI).SendGetRequest("/users")
//	})
//
//	// Database query interaction
//	queryUser := core.Do("queries user from database", func(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(&db.DatabaseAbility{})
//		if err != nil {
//			return fmt.Errorf("actor needs database ability: %w", err)
//		}
//		return ability.(db.DatabaseAbility).QueryUser(userId)
//	})
//
//	// File operation interaction
//	readConfig := core.Do("reads configuration file", func(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(&fs.FileSystemAbility{})
//		if err != nil {
//			return fmt.Errorf("actor needs file system ability: %w", err)
//		}
//		return ability.(fs.FileSystemAbility).ReadFile("config.json")
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
//	func (s *SendEmailActivity) PerformAs(ctx context.Context, actor core.Actor) error {
//		// Implementation for sending email
//	}
//
//	func (s *SendEmailActivity) Description() string {
//		return fmt.Sprintf("sends email to %s with subject '%s'", s.to, s.subject)
//	}
//
//	func (s *SendEmailActivity) FailureMode() core.FailureMode {
//		return core.FailFast // Email sending is typically critical
//	}
type Interaction interface {
	Activity
}

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
//	registerUser := core.TaskWhere("registers new user account",
//		core.Do("validates user input", validateUserData),
//		core.Do("creates user via API", createUserInAPI),
//		core.Do("verifies user in database", verifyUserInDB),
//		core.Do("sends welcome email", sendWelcomeEmail),
//	)
//
//	// Order placement task
//	placeOrder := core.TaskWhere("places customer order",
//		core.Do("validates product availability", checkInventory),
//		core.Do("calculates order total", calculateTotal),
//		core.Do("creates order in system", createOrder),
//		core.Do("processes payment", processPayment),
//		core.Do("sends order confirmation", sendConfirmation),
//	)
//
//	// Data migration task
//	migrateUserData := core.TaskWhere("migrates user data from legacy system",
//		core.Do("connects to legacy database", connectToLegacyDB),
//		core.Do("exports user data", exportUserData),
//		core.Do("transforms data format", transformData),
//		core.Do("imports to new system", importToNewSystem),
//		core.Do("verifies migration success", verifyMigration),
//	)
//
// Custom Task Types:
//
//	type CreateUserTask struct {
//		userData UserData
//	}
//
//	func (c *CreateUserTask) PerformAs(ctx context.Context, actor core.Actor) error {
//		return actor.AttemptsTo(
//			core.Do("validates user data", func(ctx context.Context, a core.Actor) error {
//				return validateUserData(c.userData)
//			}),
//			core.Do("creates user in API", func(ctx context.Context, a core.Actor) error {
//				return createUserInAPI(c.userData)
//			}),
//			core.Do("verifies user exists", func(ctx context.Context, a core.Actor) error {
//				return verifyUserExists(c.userData.Email)
//			}),
//		)
//	}
//
//	func (c *CreateUserTask) Description() string {
//		return "creates new user account with validation and verification"
//	}
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
type Task interface {
	Activity
}

// Question enables actors to retrieve information from the system.
// Questions use Go generics to provide type-safe answers about system state.
// They separate the concern of data retrieval from assertion logic.
//
// Creating Questions:
//
//	// Using core.QuestionAbout (convenience factory)
//	userCount := core.QuestionAbout("user count", func(_ context.Context, actor core.Actor) (int, error) {
//		db := actor.AbilityTo(&database.DatabaseAbility{}).(database.DatabaseAbility)
//		return db.QueryRow("SELECT COUNT(*) FROM users").Int()
//	})
//
//	// Using core.QuestionAbout
//	userName := core.QuestionAbout("current user name", func(ctx context.Context, actor core.Actor) (string, error) {
//		session := actor.AbilityTo(&auth.SessionAbility{}).(auth.SessionAbility)
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
//	fmt.Printf("Total users: %d\n", count)
//
//	// With expectations (recommended)
//	actor.AttemptsTo(
//		ensure.That(userCount, expectations.GreaterThan(0)),
//		ensure.That(userName, expectations.Contains("admin")),
//	)
//
// Question Examples:
//
//	// Simple type question
//	isSystemOnline := core.QuestionAbout("system online status", func(_ context.Context, actor core.Actor) (bool, error) {
//		ability, err := actor.AbilityTo(&health.HealthCheckAbility{})
//		if err != nil {
//			return false, err
//		}
//		return ability.(health.HealthCheckAbility).IsOnline()
//	})
//
//	// Complex type question
//	userProfile := core.QuestionAbout("user profile", func(_ context.Context, actor core.Actor) (*UserProfile, error) {
//		db := actor.AbilityTo(&database.DatabaseAbility{}).(database.DatabaseAbility)
//		return db.GetUserProfile(actor.Name())
//	})
//
//	// Collection question
//	activeOrders := core.QuestionAbout("active orders", func(_ context.Context, actor core.Actor) ([]Order, error) {
//		api := actor.AbilityTo(&api.CallAnAPI{}).(api.CallAnAPI)
//		response, err := api.Get("/orders?status=active")
//		if err != nil {
//			return nil, err
//		}
//		return parseOrders(response.Body)
//	})
//
//	// Error-state question
//	lastError := core.QuestionAbout("last system error", func(_ context.Context, actor core.Actor) (*ErrorInfo, error) {
//		log := actor.AbilityTo(&logging.LogAbility{}).(logging.LogAbility)
//		return log.GetLastError()
//	})
//
// Question Design Patterns:
//
//  1. State Questions - Query current system state
//
//  2. Calculation Questions - Compute derived values
//
//  3. Validation Questions - Check conditions
//
//  4. History Questions - Query past events
//
//     // State Question
//     systemStatus := core.QuestionAbout("system status", func(_ context.Context, actor core.Actor) (SystemStatus, error) {
//     monitor := actor.AbilityTo(&monitoring.Ability{}).(monitoring.Ability)
//     return monitor.GetCurrentStatus()
//     })
//
//     // Calculation Question
//     averageResponseTime := core.QuestionAbout("average response time", func(_ context.Context, actor core.Actor) (time.Duration, error) {
//     metrics := actor.AbilityTo(&metrics.Ability{}).(metrics.Ability)
//     return metrics.CalculateAverageResponseTime(time.Hour)
//     })
//
//     // Validation Question
//     hasValidLicense := core.QuestionAbout("has valid license", func(_ context.Context, actor core.Actor) (bool, error) {
//     license := actor.AbilityTo(&license.Ability{}).(license.Ability)
//     return license.IsValid()
//     })
//
// Best Practices:
//
//  1. Use descriptive question descriptions
//  2. Return specific types rather than interface{}
//  3. Handle errors gracefully with context
//  4. Keep questions focused on single concerns
//  5. Use type parameters for compile-time safety
//  6. Cache expensive operations when appropriate
//
// Thread Safety:
//
//	Questions should be stateless or handle their own synchronization.
//	When called by multiple actors concurrently, they should not share mutable state.
type Question[T any] interface {
	// AnsweredBy returns the answer when asked by the given actor.
	// This is where the question's main logic is implemented.
	//
	// The question should:
	// 1. Access required abilities from the actor
	// 2. Perform its query or calculation
	// 3. Return typed result and any error
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - actor: The actor asking the question
	//
	// Returns:
	//   - T: The typed answer to the question
	//   - error: Error if the question cannot be answered
	//
	// Example:
	//
	//	func (q *userCountQuestion) AnsweredBy(ctx context.Context, actor core.Actor) (int, error) {
	//		db, err := actor.AbilityTo(&database.DatabaseAbility{})
	//		if err != nil {
	//			return 0, fmt.Errorf("actor needs database ability: %w", err)
	//		}
	//
	//		return db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Int()
	//	}
	//
	// Usage:
	//
	//	count, err := question.AnsweredBy(ctx, actor)
	//	if err != nil {
	//		return fmt.Errorf("failed to get user count: %w", err)
	//	}
	//	fmt.Printf("User count: %d\n", count)
	AnsweredBy(ctx context.Context, actor Actor) (T, error)

	// Description returns a human-readable description of what the question asks.
	// This description is used in test reports and assertion messages.
	//
	// Returns:
	//   - string: Human-readable description of the question
	//
	// Examples:
	//
	//	"number of users in system"
	//	"current system online status"
	//	"user profile for current actor"
	//	"average response time in last hour"
	//	"last error from system logs"
	Description() string
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
//	// JSON serialization
//	jsonData, _ := json.Marshal(result)
//	fmt.Println(string(jsonData))
//
//	// Human-readable output
//	fmt.Printf("Test %s: %s (%v)\n", result.Name, result.Status, result.Duration)
type TestResult struct {
	// Name is the human-readable name or description of the test
	Name string `json:"name"`

	// Status indicates the current state of test execution
	Status Status `json:"status"`

	// Duration is the total time taken for test execution
	Duration time.Duration `json:"duration"`

	// Error contains any error that occurred during test execution (if any)
	// This field is omitted from JSON when nil due to omitempty tag
	Error error `json:"error,omitempty"`
}

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
//		Name:     "API connectivity test",
//		Status:   StatusRunning,
//		Duration: 0,
//	}
//
//	// Update status based on test outcome
//	if testError != nil {
//		result.Status = StatusFailed
//		result.Error = testError
//	} else {
//		result.Status = StatusPassed
//	}
//
//	// Get human-readable status
//	fmt.Printf("Test status: %s\n", result.Status.String())
type Status int

const (
	// StatusPending indicates the test has been created but not yet started
	// This is the initial state for all tests
	StatusPending Status = iota

	// StatusRunning indicates the test is currently executing
	// Set when test execution begins
	StatusRunning

	// StatusPassed indicates the test completed successfully
	// All assertions passed and no errors occurred
	StatusPassed

	// StatusFailed indicates the test completed with errors
	// One or more assertions failed or an error occurred
	StatusFailed

	// StatusSkipped indicates the test was not executed
	// Typically due to preconditions not being met
	StatusSkipped
)

// String returns a human-readable string representation of the status.
//
// Returns:
//   - string: Human-readable status name
//
// Examples:
//
//	fmt.Println(StatusPending.String())   // "pending"
//	fmt.Println(StatusRunning.String())   // "running"
//	fmt.Println(StatusPassed.String())    // "passed"
//	fmt.Println(StatusFailed.String())     // "failed"
//	fmt.Println(StatusSkipped.String())   // "skipped"
//	fmt.Println(Status(999).String())    // "unknown" (for invalid values)
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusPassed:
		return "passed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}
