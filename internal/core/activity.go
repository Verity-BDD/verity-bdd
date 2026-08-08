package core

import (
	"context"
	"fmt"
)

type activityPerformer interface {
	PerformActivity(ctx context.Context, activity Activity) error
}

func performActivity(ctx context.Context, actor Actor, activity Activity) error {
	if performer, ok := actor.(activityPerformer); ok {
		return performer.PerformActivity(ctx, activity)
	}

	return activity.PerformAs(ctx, actor)
}

// This file provides concrete implementations of the Activity interface
// defined in interfaces.go. These implementations enable the creation
// of both atomic interactions and composed tasks for test scenarios.
//
// Key Implementations:
//
//	task        - High-level business activity composed of multiple interactions
//	interaction - Low-level atomic operation with custom perform function
//
// Factory Functions:
//
//	TaskWhere() - Creates composed tasks from multiple activities
//	Do()        - Creates simple interactions with custom perform functions
//
// Usage Examples:
//
//	// Create a simple interaction
//	sendRequest := core.Do("sends GET request", func(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(api.Using(nil))
//		if err != nil {
//			return fmt.Errorf("actor needs API ability: %w", err)
//		}
//		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "/users", nil)
//		if err != nil {
//			return fmt.Errorf("create GET request: %w", err)
//		}
//		_, err = ability.(api.CallAnAPI).SendRequest(request, ctx)
//		return err
//	})
//
//	// Create a composed task
//	createUser := core.TaskWhere("creates new user account",
//		core.Do("validates user data", validateUserData),
//		sendRequest,
//		core.Do("verifies user creation", verifyUser),
//	)
//
//	// Execute activities; failures are reported through the test context.
//	actor.AttemptsTo(sendRequest, createUser)
//
// Error Handling:
//
//	Built-in tasks and interactions use FailFast. The Interaction returned by
//	Do has no WithFailureMode method; custom Activity implementations can return
//	a different FailureMode.

// task implements the Task interface for composed activities.
// Tasks represent high-level business operations that consist of multiple
// smaller activities executed sequentially.
//
// Type task is private - use TaskWhere() factory function to create instances.
type task struct {
	// description provides a human-readable description of the task's purpose
	description string

	// activities contains the sequence of activities that compose this task
	activities []Activity
}

// Description returns the task's human-readable description.
// This description is used in test reports and logging.
//
// Returns:
//   - string: Description of what the task accomplishes
//
// Example:
//
//	task := core.TaskWhere("creates user account", activities...)
//	fmt.Println(task.Description()) // "creates user account"
func (t *task) Description() string {
	return t.description
}

// PerformAs executes the task as the given actor by running all activities sequentially.
// Activities are executed in the order they were provided to TaskWhere().
// Execution stops immediately if any activity fails (FailFast behavior).
//
// Parameters:
//   - actor: The actor performing this task
//
// Returns:
//   - error: Descriptive error if any activity fails, nil if all succeed
//
// Example:
//
//	func (t *task) PerformAs(ctx context.Context, actor core.Actor) error {
//		for _, activity := range t.activities {
//			if err := activity.PerformAs(ctx, actor); err != nil {
//				return fmt.Errorf("task '%s' failed during activity '%s': %w",
//					t.Description(), activity.Description(), err)
//			}
//		}
//		return nil
//	}
//
// Error Context:
//
//	The returned error includes:
//	- The task description for identification
//	- The specific activity that failed
//	- The original error wrapped with context
func (t *task) PerformAs(ctx context.Context, actor Actor) error {
	for _, activity := range t.activities {
		err := performActivity(ctx, actor, activity)
		if err != nil {
			return fmt.Errorf("task '%s' failed during activity '%s': %w",
				t.Description(), activity.Description(), err)
		}
	}
	return nil
}

// FailureMode returns the failure mode for tasks.
// Tasks use FailFast mode by default, meaning execution stops on first error.
//
// Returns:
//   - FailureMode: Always returns FailFast for task implementations
//
// This ensures that all activities in a task must complete successfully
// for the task to be considered successful.
func (t *task) FailureMode() FailureMode {
	return FailFast
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
//	// Simple task with multiple activities
//	registerUser := core.TaskWhere("registers new user account",
//		core.Do("validates user data", validateUserData),
//		core.Do("creates user via API", createUser),
//		core.Do("verifies user in database", verifyUser),
//	)
//
//	// Complex workflow task
//	placeOrder := core.TaskWhere("places customer order",
//		core.Do("checks product inventory", checkInventory),
//		core.Do("calculates order total", calculateTotal),
//		core.Do("creates order record", createOrder),
//		core.Do("processes payment", processPayment),
//		core.Do("sends confirmation email", sendConfirmation),
//	)
//
//	// Data migration task
//	migrateData := core.TaskWhere("migrates user data from legacy system",
//		core.Do("exports data from legacy system", exportLegacyData),
//		core.Do("transforms data format", transformData),
//		core.Do("imports to new system", importToNewSystem),
//		core.Do("verifies migration success", verifyMigration),
//	)
//
// Task Execution:
//
//	actor.AttemptsTo(
//		registerUser,
//		placeOrder,
//		migrateData,
//	)
//
// Best Practices:
//
//  1. Use descriptive, business-focused descriptions
//  2. Keep activities focused on single responsibilities
//  3. Order activities logically (prerequisites first)
//  4. Include verification activities to ensure success
//  5. Avoid too many activities in a single task (prefer breaking down)
func TaskWhere(description string, activities ...Activity) Task {
	return &task{
		description: description,
		activities:  activities,
	}
}

// interaction implements the Interaction interface for atomic operations.
// Interactions are low-level, focused activities that typically perform
// a single operation or system call.
//
// Type interaction is private - use Do() factory function to create instances.
type interaction struct {
	// description provides a human-readable description of the interaction
	description string

	// perform is the function that executes when the interaction is performed
	perform func(ctx context.Context, actor Actor) error
}

// Do creates a new interaction with the given description and perform function.
// This is the primary factory function for creating simple, atomic activities.
// Interactions created with Do() are ideal for quick, focused operations.
//
// Parameters:
//   - description: Human-readable description of what the interaction does
//   - perform: Function that executes the interaction logic (receives context first, then actor)
//
// Returns:
//   - Interaction: A new interaction that executes the provided function
//
// Usage Examples:
//
//	// Simple API call interaction
//	sendGetRequest := core.Do("sends GET request to /users", func(ctx context.Context, actor core.Actor) error {
//		ability, err := actor.AbilityTo(api.Using(nil))
//		if err != nil {
//			return fmt.Errorf("actor needs API ability: %w", err)
//		}
//		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "/users", nil)
//		if err != nil {
//			return fmt.Errorf("create GET request: %w", err)
//		}
//		_, err = ability.(api.CallAnAPI).SendRequest(request, ctx)
//		return err
//	})
//
//	// Database query interaction
//	queryUser := core.Do("queries user from database", func(ctx context.Context, actor core.Actor) error {
//		db, err := actor.AbilityTo(&database.DatabaseAbility{})
//		if err != nil {
//			return fmt.Errorf("actor needs database ability: %w", err)
//		}
//		return db.(database.DatabaseAbility).QueryUser(userId)
//	})
//
//	// File operation interaction
//	readConfig := core.Do("reads configuration file", func(ctx context.Context, actor core.Actor) error {
//		fs, err := actor.AbilityTo(&filesystem.FileSystemAbility{})
//		if err != nil {
//			return fmt.Errorf("actor needs file system ability: %w", err)
//		}
//		return fs.(filesystem.FileSystemAbility).ReadFile("config.json")
//	})
//
//	// Custom business logic interaction
//	validateEmail := core.Do("validates email format", func(ctx context.Context, actor core.Actor) error {
//		email := getEmailFromContext()
//		if !isValidEmail(email) {
//			return fmt.Errorf("invalid email format: %s", email)
//		}
//		return nil
//	})
//
//	// System status check interaction
//	checkHealth := core.Do("checks system health", func(ctx context.Context, actor core.Actor) error {
//		health, err := actor.AbilityTo(&monitoring.HealthAbility{})
//		if err != nil {
//			return fmt.Errorf("actor needs health check ability: %w", err)
//		}
//
//		isHealthy := health.(monitoring.HealthAbility).IsHealthy()
//		if !isHealthy {
//			return fmt.Errorf("system is not healthy")
//		}
//		return nil
//	})
//
// Using Interactions:
//
//	actor.AttemptsTo(
//		sendGetRequest,
//		queryUser,
//		readConfig,
//		validateEmail,
//		checkHealth,
//	)
//
//	// Or as part of a task
//	userWorkflow := core.TaskWhere("user login workflow",
//		core.Do("loads user credentials", loadCredentials),
//		core.Do("authenticates with API", authenticate),
//		core.Do("verifies session", verifySession),
//	)
//
// Interaction Guidelines:
//
//  1. Keep interactions focused on single operations
//  2. Use descriptive, action-oriented descriptions
//  3. Handle errors with proper context
//  4. Access abilities safely and check for their existence
//  5. Avoid complex logic in interactions (prefer tasks for workflows)
func Do(description string, perform func(ctx context.Context, actor Actor) error) Interaction {
	return &interaction{
		description: description,
		perform:     perform,
	}
}

// Description returns the interaction's human-readable description.
// This description is used in test reports and logging output.
//
// Returns:
//   - string: Description of what the interaction does
//
// Example:
//
//	interaction := core.Do("sends POST request", sendPostFunction)
//	fmt.Println(interaction.Description()) // "sends POST request"
func (i *interaction) Description() string {
	return i.description
}

// PerformAs executes the interaction as the given actor.
// This method simply calls the perform function provided to Do().
//
// Parameters:
//   - actor: The actor performing this interaction
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - error: Whatever error the perform function returns
//
// Note: Error handling and wrapping should be done in the perform function
// to provide proper context about what went wrong.
func (i *interaction) PerformAs(ctx context.Context, actor Actor) error {
	return i.perform(ctx, actor)
}

// FailureMode returns the failure mode for interactions.
// Interactions use FailFast mode by default, meaning errors stop execution.
//
// Returns:
//   - FailureMode: Always returns FailFast for interaction implementations
//
// This ensures that interactions fail immediately if something goes wrong,
// which is appropriate for atomic operations.
func (i *interaction) FailureMode() FailureMode {
	return FailFast
}
