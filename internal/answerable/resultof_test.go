package answerable

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verity-bdd/verity-bdd/internal/core"
)

func TestResultOf_BasicFunctionality(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("successful function execution", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("test value", func(ctx context.Context, actor core.Actor) (int, error) {
			return 42, nil
		})

		// Test AnsweredBy
		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, 42, result)

		// Test Description
		desc := q.Description()
		require.Equal(t, "test value", desc)
	})

	t.Run("function returns error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("function error")
		q := ResultOf("failing operation", func(ctx context.Context, actor core.Actor) (string, error) {
			return "", testErr
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.Error(t, err)
		require.Equal(t, testErr, err)
		require.Equal(t, "", result) // zero value on error
	})

	t.Run("function with different types", func(t *testing.T) {
		t.Parallel()
		// Test string result
		q1 := ResultOf("string value", func(ctx context.Context, actor core.Actor) (string, error) {
			return "hello", nil
		})
		result1, err1 := q1.AnsweredBy(context.Background(), actor)
		require.NoError(t, err1)
		require.Equal(t, "hello", result1)
		require.Equal(t, "string value", q1.Description())

		// Test boolean result
		q2 := ResultOf("boolean value", func(ctx context.Context, actor core.Actor) (bool, error) {
			return true, nil
		})
		result2, err2 := q2.AnsweredBy(context.Background(), actor)
		require.NoError(t, err2)
		require.Equal(t, true, result2)
		require.Equal(t, "boolean value", q2.Description())

		// Test float result
		q3 := ResultOf("float value", func(ctx context.Context, actor core.Actor) (float64, error) {
			return 3.14, nil
		})
		result3, err3 := q3.AnsweredBy(context.Background(), actor)
		require.NoError(t, err3)
		require.Equal(t, 3.14, result3)
		require.Equal(t, "float value", q3.Description())

		// Test struct result
		q4 := ResultOf("user struct", func(ctx context.Context, actor core.Actor) (TestUser, error) {
			return TestUser{Name: "John", Age: 30}, nil
		})
		result4, err4 := q4.AnsweredBy(context.Background(), actor)
		require.NoError(t, err4)
		require.Equal(t, TestUser{Name: "John", Age: 30}, result4)
		require.Equal(t, "user struct", q4.Description())
	})
}

func TestResultOf_ActorParameterUsage(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("function uses actor name", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("actor greeting", func(ctx context.Context, actor core.Actor) (string, error) {
			return "Hello, " + actor.Name(), nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, "Hello, TestActor", result)
	})

	t.Run("different actors produce different results", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("actor name", func(ctx context.Context, actor core.Actor) (string, error) {
			return actor.Name(), nil
		})

		actor1 := &mockActor{name: "Actor1"}
		actor2 := &mockActor{name: "Actor2"}

		result1, err1 := q.AnsweredBy(context.Background(), actor1)
		require.NoError(t, err1)
		require.Equal(t, "Actor1", result1)

		result2, err2 := q.AnsweredBy(context.Background(), actor2)
		require.NoError(t, err2)
		require.Equal(t, "Actor2", result2)
	})
}

func TestResultOf_ErrorHandling(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("standard error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("standard error")
		q := ResultOf("error operation", func(ctx context.Context, actor core.Actor) (int, error) {
			return 0, testErr
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.Error(t, err)
		require.Equal(t, testErr, err)
		require.Equal(t, 0, result)
	})

	t.Run("wrapped error", func(t *testing.T) {
		t.Parallel()
		wrappedErr := errors.New("wrapped error")
		q := ResultOf("wrapped operation", func(ctx context.Context, actor core.Actor) (string, error) {
			return "", wrappedErr
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.Error(t, err)
		require.Equal(t, wrappedErr, err)
		require.Equal(t, "", result)
	})

	t.Run("nil error with value", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("successful operation", func(ctx context.Context, actor core.Actor) (int, error) {
			return 100, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, 100, result)
	})
}

func TestResultOf_ComplexOperations(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("calculations", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("sum calculation", func(ctx context.Context, actor core.Actor) (int, error) {
			a, b := 10, 20
			return a + b, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, 30, result)
	})

	t.Run("slice operations", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("slice processing", func(ctx context.Context, actor core.Actor) ([]int, error) {
			input := []int{1, 2, 3, 4, 5}
			var result []int
			for _, v := range input {
				if v%2 == 0 {
					result = append(result, v*2)
				}
			}
			return result, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, []int{4, 8}, result)
	})

	t.Run("map operations", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("map transformation", func(ctx context.Context, actor core.Actor) (map[string]int, error) {
			input := map[string]int{"a": 1, "b": 2, "c": 3}
			result := make(map[string]int)
			for k, v := range input {
				result[k+"-new"] = v * 2
			}
			return result, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		expected := map[string]int{"a-new": 2, "b-new": 4, "c-new": 6}
		require.Equal(t, expected, result)
	})
}

func TestResultOf_DescriptionConsistency(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	descriptions := []string{
		"simple description",
		"description with spaces",
		"description-with-dashes",
		"description_with_underscores",
		"description with numbers 123",
		"Кириллическое описание",
		"emoji 🚀 description",
	}

	for _, desc := range descriptions {
		t.Run("description: "+desc, func(t *testing.T) {
			t.Parallel()
			q := ResultOf(desc, func(ctx context.Context, actor core.Actor) (int, error) {
				return 1, nil
			})

			// Test that description is preserved exactly
			require.Equal(t, desc, q.Description())

			// Test that function still works
			result, err := q.AnsweredBy(context.Background(), actor)
			require.NoError(t, err)
			require.Equal(t, 1, result)
		})
	}
}

func TestResultOf_GenericTypeInference(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("integer type inference", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("integer", func(ctx context.Context, actor core.Actor) (int, error) {
			return 42, nil
		})

		var result int
		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("string type inference", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("string", func(ctx context.Context, actor core.Actor) (string, error) {
			return "hello", nil
		})

		var result string
		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, "hello", result)
	})

	t.Run("pointer type inference", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("pointer", func(ctx context.Context, actor core.Actor) (*string, error) {
			s := "hello pointer"
			return &s, nil
		})

		var result *string
		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "hello pointer", *result)
	})
}

func TestResultOf_EdgeCases(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("empty description", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("", func(ctx context.Context, actor core.Actor) (int, error) {
			return 1, nil
		})

		require.Equal(t, "", q.Description())
		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, 1, result)
	})

	t.Run("nil function parameter", func(t *testing.T) {
		t.Parallel()
		// This should panic if we don't handle nil, but let's see what happens
		require.Panics(t, func() {
			var nilFunc func(context.Context, core.Actor) (int, error) = nil
			ResultOf("nil function", nilFunc)
		})
	})

	t.Run("function that returns nil", func(t *testing.T) {
		t.Parallel()
		q := ResultOf("nil result", func(ctx context.Context, actor core.Actor) (*TestUser, error) {
			return nil, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Nil(t, result)
	})
}

// Integration test to ensure ResultOf works with the broader ecosystem
func TestResultOf_IntegrationWithQuestionInterface(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	// Test that ResultOf creates a proper Question[T]
	q := ResultOf("integration test", func(ctx context.Context, actor core.Actor) (int, error) {
		return 123, nil
	})

	// Test interface compliance
	require.Implements(t, (*core.Question[int])(nil), q)

	// Test that it works with the interface methods
	result, err := q.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, 123, result)

	description := q.Description()
	require.Equal(t, "integration test", description)
}

// Test to demonstrate usage patterns
func TestResultOf_UsageExamples(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	t.Run("computation example", func(t *testing.T) {
		t.Parallel()
		// Example: computing something complex
		q := ResultOf("user age category", func(ctx context.Context, actor core.Actor) (string, error) {
			age := 25
			switch {
			case age < 18:
				return "minor", nil
			case age < 65:
				return "adult", nil
			default:
				return "senior", nil
			}
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, "adult", result)
		require.Equal(t, "user age category", q.Description())
	})

	t.Run("data transformation example", func(t *testing.T) {
		t.Parallel()
		// Example: transforming data
		q := ResultOf("uppercase words", func(ctx context.Context, actor core.Actor) ([]string, error) {
			words := []string{"hello", "world", "golang"}
			for i, word := range words {
				words[i] = strings.ToUpper(word)
			}
			return words, nil
		})

		result, err := q.AnsweredBy(context.Background(), actor)
		require.NoError(t, err)
		require.Equal(t, []string{"HELLO", "WORLD", "GOLANG"}, result)
		require.Equal(t, "uppercase words", q.Description())
	})
}
