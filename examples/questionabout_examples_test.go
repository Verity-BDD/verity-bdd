package examples

import (
	"context"
	"fmt"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	expectations "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

// TestQuestionAbout demonstrates usage of verity.QuestionAbout for dynamic assertions
func TestQuestionAbout(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("QuestionAboutTester")

	// Basic usage with different data types
	actor.AttemptsTo(
		// Simple calculation
		ensure.That(
			verity.QuestionAbout("calculated value", func(ctx context.Context, actor verity.Actor) (int, error) {
				return 42, nil
			}),
			expectations.Equals(42),
		),

		// String manipulation
		ensure.That(
			verity.QuestionAbout("greeting message", func(ctx context.Context, actor verity.Actor) (string, error) {
				return "Hello, " + actor.Name(), nil
			}),
			expectations.Contains("Hello, QuestionAboutTester"),
		),

		// Boolean logic
		ensure.That(
			verity.QuestionAbout("validation result", func(ctx context.Context, actor verity.Actor) (bool, error) {
				name := actor.Name()
				return len(name) > 5, nil
			}),
			expectations.Equals(true),
		),

		// Float calculation
		ensure.That(
			verity.QuestionAbout("division result", func(ctx context.Context, actor verity.Actor) (float64, error) {
				return 10.0 / 2.0, nil
			}),
			expectations.Equals(5.0),
		),
	)
}

// TestQuestionAboutCalculations demonstrates using QuestionAbout for calculations
func TestQuestionAboutCalculations(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	dataProcessor := test.ActorCalled("DataProcessor")

	// Example: Processing numeric data
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	dataProcessor.AttemptsTo(
		ensure.That(
			verity.QuestionAbout("sum of even numbers", func(ctx context.Context, actor verity.Actor) (int, error) {
				sum := 0
				for _, num := range numbers {
					if num%2 == 0 {
						sum += num
					}
				}
				return sum, nil
			}),
			expectations.Equals(30), // 2+4+6+8+10 = 30
		),

		ensure.That(
			verity.QuestionAbout("average of numbers", func(ctx context.Context, actor verity.Actor) (float64, error) {
				if len(numbers) == 0 {
					return 0, fmt.Errorf("no numbers to calculate average")
				}
				sum := 0
				for _, num := range numbers {
					sum += num
				}
				return float64(sum) / float64(len(numbers)), nil
			}),
			expectations.Equals(5.5),
		),

		ensure.That(
			verity.QuestionAbout("maximum value", func(ctx context.Context, actor verity.Actor) (int, error) {
				if len(numbers) == 0 {
					return 0, fmt.Errorf("no numbers to find maximum")
				}
				max := numbers[0]
				for _, num := range numbers {
					if num > max {
						max = num
					}
				}
				return max, nil
			}),
			expectations.Equals(10),
		),
	)
}

// TestQuestionAboutErrorHandling demonstrates error handling in QuestionAbout functions
func TestQuestionAboutErrorHandling(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("ErrorTestActor")

	// Example: Functions that can return errors
	actor.AttemptsTo(
		// This will succeed
		ensure.That(
			verity.QuestionAbout("successful operation", func(ctx context.Context, actor verity.Actor) (string, error) {
				return "success", nil
			}),
			expectations.Equals("success"),
		),

		// Example: Conditional logic with error handling
		ensure.That(
			verity.QuestionAbout("safe division", func(ctx context.Context, actor verity.Actor) (float64, error) {
				numerator, denominator := 10, 2
				if denominator == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return float64(numerator) / float64(denominator), nil
			}),
			expectations.Equals(5.0),
		),

		// Example: String manipulation with validation
		ensure.That(
			verity.QuestionAbout("validated string", func(ctx context.Context, actor verity.Actor) (string, error) {
				input := "test"
				if input == "" {
					return "", fmt.Errorf("input cannot be empty")
				}
				return fmt.Sprintf("processed: %s", input), nil
			}),
			expectations.Equals("processed: test"),
		),
	)
}

// TestQuestionAboutWithActor demonstrates using actor properties in QuestionAbout
func TestQuestionAboutWithActor(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor1 := test.ActorCalled("Actor1")
	actor2 := test.ActorCalled("Actor2")

	// Both actors use the same QuestionAbout question but get different results
	greetingQuestion := verity.QuestionAbout("personalized greeting", func(ctx context.Context, actor verity.Actor) (string, error) {
		return fmt.Sprintf("Hello, %s!", actor.Name()), nil
	})

	actor1.AttemptsTo(
		ensure.That(greetingQuestion, expectations.Equals("Hello, Actor1!")),
	)

	actor2.AttemptsTo(
		ensure.That(greetingQuestion, expectations.Equals("Hello, Actor2!")),
	)

	// Different actors can have different calculations
	nameLengthQuestion := verity.QuestionAbout("name length", func(ctx context.Context, actor verity.Actor) (int, error) {
		return len(actor.Name()), nil
	})

	actor1.AttemptsTo(
		ensure.That(nameLengthQuestion, expectations.Equals(6)), // "Actor1"
	)

	actor2.AttemptsTo(
		ensure.That(nameLengthQuestion, expectations.Equals(6)), // "Actor2"
	)
}

// TestQuestionAboutComplexTypes demonstrates using QuestionAbout with complex data structures
func TestQuestionAboutComplexTypes(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("ComplexTypeTester")

	type User struct {
		Name  string
		Age   int
		Email string
	}

	actor.AttemptsTo(
		// Create and validate a struct
		ensure.That(
			verity.QuestionAbout("user creation", func(ctx context.Context, actor verity.Actor) (User, error) {
				return User{
					Name:  "John Doe",
					Age:   30,
					Email: "john@example.com",
				}, nil
			}),
			expectations.Equals(User{Name: "John Doe", Age: 30, Email: "john@example.com"}),
		),

		// Work with slices
		ensure.That(
			verity.QuestionAbout("user names list", func(ctx context.Context, actor verity.Actor) ([]string, error) {
				users := []User{
					{Name: "Alice", Age: 25, Email: "alice@example.com"},
					{Name: "Bob", Age: 30, Email: "bob@example.com"},
					{Name: "Charlie", Age: 35, Email: "charlie@example.com"},
				}

				names := make([]string, len(users))
				for i, user := range users {
					names[i] = user.Name
				}
				return names, nil
			}),
			expectations.Satisfies("contains Bob", func(names []string) error {
				for _, name := range names {
					if name == "Bob" {
						return nil
					}
				}
				return fmt.Errorf("expected to find Bob in names, but got %v", names)
			}),
		),

		// Work with maps
		ensure.That(
			verity.QuestionAbout("user data map", func(ctx context.Context, actor verity.Actor) (map[string]interface{}, error) {
				return map[string]interface{}{
					"name":    "Jane Doe",
					"age":     28,
					"active":  true,
					"balance": 1000.50,
				}, nil
			}),
			expectations.Equals(map[string]interface{}{
				"name":    "Jane Doe",
				"age":     28,
				"active":  true,
				"balance": 1000.50,
			}),
		),

		// Pointer operations
		ensure.That(
			verity.QuestionAbout("pointer to string", func(ctx context.Context, actor verity.Actor) (*string, error) {
				message := "Hello from pointer"
				return &message, nil
			}),
			expectations.Satisfies("is not nil", func(p *string) error {
				if p == nil {
					return fmt.Errorf("expected pointer to not be nil")
				}
				return nil
			}),
		),
	)
}

// TestQuestionAboutMixedWithStatic demonstrates mixing QuestionAbout with ValueOf
func TestQuestionAboutMixedWithStatic(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("MixedTester")

	// Mix of static values and QuestionAbout functions
	actor.AttemptsTo(
		// Static value
		ensure.That(answerable.ValueOf(42), expectations.Equals(42)),

		// Dynamic calculation using static values
		ensure.That(
			verity.QuestionAbout("double calculation", func(ctx context.Context, actor verity.Actor) (int, error) {
				staticValue := 42
				return staticValue * 2, nil
			}),
			expectations.Equals(84),
		),

		// Complex calculation
		ensure.That(
			verity.QuestionAbout("complex math", func(ctx context.Context, actor verity.Actor) (int, error) {
				a := 10
				b := 20
				c := a + b  // 30
				d := c * 2  // 60
				e := d - 10 // 50
				return e, nil
			}),
			expectations.Equals(50),
		),

		// String operations
		ensure.That(
			verity.QuestionAbout("string building", func(ctx context.Context, actor verity.Actor) (string, error) {
				parts := []string{"Hello", "from", "QuestionAbout"}
				result := ""
				for i, part := range parts {
					if i > 0 {
						result += " "
					}
					result += part
				}
				return result, nil
			}),
			expectations.Equals("Hello from QuestionAbout"),
		),
	)
}

// TestQuestionAboutEdgeCases demonstrates edge cases and special scenarios
func TestQuestionAboutEdgeCases(t *testing.T) {
	test := verity.NewVerityTest(t, verity.Scene{})

	actor := test.ActorCalled("EdgeCaseTester")

	actor.AttemptsTo(
		// Empty slice
		ensure.That(
			verity.QuestionAbout("empty slice", func(ctx context.Context, actor verity.Actor) ([]int, error) {
				return []int{}, nil
			}),
			expectations.Equals([]int{}),
		),

		// Nil pointer
		ensure.That(
			verity.QuestionAbout("nil pointer", func(ctx context.Context, actor verity.Actor) (*int, error) {
				return nil, nil
			}),
			expectations.Equals((*int)(nil)),
		),

		// Empty string
		ensure.That(
			verity.QuestionAbout("empty string", func(ctx context.Context, actor verity.Actor) (string, error) {
				return "", nil
			}),
			expectations.Equals(""),
		),

		// Zero values
		ensure.That(
			verity.QuestionAbout("zero int", func(ctx context.Context, actor verity.Actor) (int, error) {
				return 0, nil
			}),
			expectations.Equals(0),
		),

		ensure.That(
			verity.QuestionAbout("zero float", func(ctx context.Context, actor verity.Actor) (float64, error) {
				return 0.0, nil
			}),
			expectations.Equals(0.0),
		),

		ensure.That(
			verity.QuestionAbout("zero bool", func(ctx context.Context, actor verity.Actor) (bool, error) {
				return false, nil
			}),
			expectations.Equals(false),
		),
	)
}
