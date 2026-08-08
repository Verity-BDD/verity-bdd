package core

import (
	"context"
)

// This file provides concrete implementations of the Question interface
// defined in interfaces.go. These implementations enable type-safe
// queries about system state using Go generics.
//
// Key Implementation:
//
//	question[T] - Generic question implementation with custom ask function
//
// Factory Functions:
//
//	QuestionAbout() - Creates a new question with a description and ask function
//
// Usage Examples:
//
//	// Create a question using QuestionAbout
//	userCount := core.QuestionAbout[int]("number of users", func(_ context.Context, actor core.Actor) (int, error) {
//		ability, err := actor.AbilityTo(&database.DatabaseAbility{})
//		if err != nil {
//			return 0, err
//		}
//		return ability.(database.DatabaseAbility).QueryRow("SELECT COUNT(*) FROM users").Int()
//	})
//
//	// Another question using QuestionAbout
//	userName := core.QuestionAbout("current user name", func(_ context.Context, actor core.Actor) (string, error) {
//		ability, err := actor.AbilityTo(&auth.SessionAbility{})
//		if err != nil {
//			return "", err
//		}
//		return ability.(auth.SessionAbility).GetCurrentUser().Name, nil
//	})
//
// Type Safety:
//
//	// Compile-time type checking ensures correct return types
//	var count int
//	var name string
//	var profile *UserProfile
//	var orders []Order
//	var isActive bool
//
//	// Each question returns its specific type
//	count, err := userCount.AnsweredBy(ctx, actor)    // int, error
//	name, err := userName.AnsweredBy(ctx, actor)      // string, error
//
// Using Questions with Expectations:
//
//	actor.AttemptsTo(
//		ensure.That(userCount, expectations.Equals(count)),
//		ensure.That(userName, expectations.ContainsSubstring("admin")),
//	)
//

// question implements the Question interface for type-safe system queries.
// This generic implementation allows creating questions that return
// specific types while maintaining the Question interface contract.
//
// Type question is private - use the QuestionAbout() factory function.
type question[T any] struct {
	// description provides a human-readable description of what the question asks
	description string

	// ask is the function that executes when the question is answered
	ask func(ctx context.Context, actor Actor) (T, error)
}

// Description returns the question's human-readable description as-is.
//
// Returns:
//   - string: Original description provided during question creation
//
// Example:
//
//	q := core.QuestionAbout("user count", getUserCount)
//	fmt.Println(q.Description()) // "user count"
func (q *question[T]) Description() string {
	return q.description
}

// AnsweredBy returns the answer when asked by the given actor.
// This method executes the ask function provided to QuestionAbout().
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
//		return q.ask(ctx, actor)
//	}
//
// Usage:
//
//	count, err := question.AnsweredBy(ctx, actor)
//	if err != nil {
//		return fmt.Errorf("failed to answer question '%s': %w", question.Description(), err)
//	}
//	fmt.Printf("Answer: %v\n", count)
func (q *question[T]) AnsweredBy(ctx context.Context, actor Actor) (T, error) {
	return q.ask(ctx, actor)
}

// QuestionAbout creates a new question with the given description and ask function.
// This is the factory function for creating typed questions.
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
//		ensure.That(isHealthy, expectations.Equals(true)),
//	)
func QuestionAbout[T any](description string, ask func(ctx context.Context, actor Actor) (T, error)) Question[T] {
	if ask == nil {
		panic("QuestionAbout: function parameter cannot be nil")
	}

	return &question[T]{
		description: description,
		ask:         ask,
	}
}
