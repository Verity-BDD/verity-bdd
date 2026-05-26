package answerable

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
)

// mockActor implements core.Actor for testing
type mockActor struct {
	name string
}

func (m *mockActor) Name() string {
	return m.name
}

func (m *mockActor) Context() context.Context {
	return context.Background()
}

func (m *mockActor) WhoCan(abilities ...abilities.Ability) core.Actor {
	return m
}

func (m *mockActor) AbilityTo(ability abilities.Ability) (abilities.Ability, error) {
	return nil, nil
}

func (m *mockActor) AttemptsTo(activities ...core.Activity) {
}

// Test types for comprehensive testing
type TestUser struct {
	Name string
	Age  int
}

func TestValueOf_BasicTypes(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"integer", 42, 42},
		{"string", "hello", "hello"},
		{"boolean", true, true},
		{"float", 3.14, 3.14},
		{"nil interface", (*TestUser)(nil), (*TestUser)(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := ValueOf(tt.value)

			// Test AnsweredBy
			result, err := q.AnsweredBy(context.Background(), actor)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)

			// Test Description
			desc := q.Description()
			require.Contains(t, desc, fmt.Sprintf("%v", tt.value))
			require.Contains(t, desc, fmt.Sprintf("%T", tt.value))
		})
	}
}

func TestValueOf_ErrorType(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{"standard error", errors.New("test error"), errors.New("test error")},
		{"formatted error", fmt.Errorf("formatted: %s", "error"), fmt.Errorf("formatted: %s", "error")},
		{"joined error", fmt.Errorf("wrapper: %w", errors.New("inner")), fmt.Errorf("wrapper: %w", errors.New("inner"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := ValueOf(tt.err)

			// Test AnsweredBy - error should be returned as value, not as error
			result, err := q.AnsweredBy(context.Background(), actor)
			require.NoError(t, err)
			require.Equal(t, tt.err, result)

			// Test Description should have special error formatting
			desc := q.Description()
			require.Contains(t, desc, "error")
			require.Contains(t, desc, tt.err.Error())
			require.Contains(t, desc, "(error)")
		})
	}
}

func TestValueOf_ComplexTypes(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	user := TestUser{Name: "John", Age: 30}
	q := ValueOf(user)

	result, err := q.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, user, result)

	desc := q.Description()
	require.Contains(t, desc, fmt.Sprintf("%v", user))
	require.Contains(t, desc, "answerable.TestUser")
}

func TestValueOf_PointersAndNil(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	// Test with pointer to value
	name := "test"
	q1 := ValueOf(&name)

	result, err := q1.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, &name, result)

	desc1 := q1.Description()
	require.Contains(t, desc1, fmt.Sprintf("%v", &name))
	require.Contains(t, desc1, "*string")

	// Test with nil pointer
	var ptr *string
	q2 := ValueOf(ptr)

	result2, err := q2.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, (*string)(nil), result2)

	desc2 := q2.Description()
	require.Contains(t, desc2, "<nil>")
	require.Contains(t, desc2, "*string")
}

func TestValueOf_SlicesAndMaps(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	// Slice
	slice := []int{1, 2, 3}
	q1 := ValueOf(slice)

	result, err := q1.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, slice, result)

	desc1 := q1.Description()
	require.Contains(t, desc1, fmt.Sprintf("%v", slice))
	require.Contains(t, desc1, "[]int")

	// Map
	m := map[string]int{"a": 1, "b": 2}
	q2 := ValueOf(m)

	result2, err := q2.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, m, result2)

	desc2 := q2.Description()
	require.Contains(t, desc2, fmt.Sprintf("%v", m))
	require.Contains(t, desc2, "map[string]int")
}

func TestValueOf_GenericTypeInference(t *testing.T) {
	t.Parallel()
	actor := &mockActor{name: "TestActor"}

	// Test that generic type inference works correctly

	// Integer
	intQuestion := ValueOf(123)
	var resultInt int
	resultInt, err := intQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, 123, resultInt)

	// String
	stringQuestion := ValueOf("test")
	var resultString string
	resultString, err = stringQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, "test", resultString)

	// Error - should work with error type
	errQuestion := ValueOf(errors.New("test"))
	var resultErr error
	resultErr, err = errQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, errors.New("test"), resultErr)
}

func TestValueOf_DescriptionFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"integer", 42, "42 (int)"},
		{"string", "hello", "hello (string)"},
		{"boolean", true, "true (bool)"},
		{"float", 3.14, "3.14 (float64)"},
		{"error", errors.New("test"), "error test (error)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			q := ValueOf(tt.value)
			desc := q.Description()
			require.Equal(t, tt.expected, desc)
		})
	}
}

// Integration test with ensure.That (if available)
func TestValueOf_IntegrationWithEnsure(t *testing.T) {
	t.Parallel()
	// This test demonstrates the intended usage pattern
	// Note: This would require importing expectations/ensure, but for now
	// we'll test that our Question interface works correctly

	actor := &mockActor{name: "TestActor"}

	// Create value questions
	intQuestion := ValueOf(42)
	stringQuestion := ValueOf("hello world")
	errorQuestion := ValueOf(errors.New("test error"))

	// Test that they can be answered correctly
	intResult, err := intQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, 42, intResult)

	stringResult, err := stringQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Equal(t, "hello world", stringResult)

	errorResult, err := errorQuestion.AnsweredBy(context.Background(), actor)
	require.NoError(t, err)
	require.Error(t, errorResult) // The error itself is the value
	require.Equal(t, "test error", errorResult.Error())
}

func TestIsError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"standard error", errors.New("test"), true},
		{"formatted error", fmt.Errorf("test"), true},
		{"nil", nil, false},
		{"string", "test", false},
		{"integer", 42, false},
		{"custom error type", &customError{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isError(tt.value)
			require.Equal(t, tt.expected, result)
		})
	}
}

// customError implements error interface for testing
type customError struct{}

func (e *customError) Error() string {
	return "custom error"
}
