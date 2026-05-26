package expectations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-bdd/verity-bdd/internal/expectations"
)

// IsEmpty — slice tests (issue #21)

func TestIsEmpty_PassesOnEmptySlice(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[[]int]().Evaluate([]int{})
	assert.NoError(t, err)
}

func TestIsEmpty_FailsOnNonEmptySlice(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[[]int]().Evaluate([]int{1, 2, 3})
	require.Error(t, err)
	assert.Equal(t, "expected slice/array to be empty, but got 3 elements", err.Error())
}

// IsEmpty — string regression

func TestIsEmpty_PassesOnEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[string]().Evaluate("")
	assert.NoError(t, err)
}

func TestIsEmpty_FailsOnNonEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[string]().Evaluate("hello")
	require.Error(t, err)
	assert.Equal(t, "expected string to be empty, but got 'hello'", err.Error())
}

// IsEmpty — map regression

func TestIsEmpty_PassesOnEmptyMap(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[map[string]int]().Evaluate(map[string]int{})
	assert.NoError(t, err)
}

func TestIsEmpty_FailsOnNonEmptyMap(t *testing.T) {
	t.Parallel()
	err := expectations.IsEmpty[map[string]int]().Evaluate(map[string]int{"a": 1})
	require.Error(t, err)
	assert.Equal(t, "expected map to be empty, but got 1 elements", err.Error())
}

// IsEmpty — Description

func TestIsEmpty_Description(t *testing.T) {
	t.Parallel()
	desc := expectations.IsEmpty[[]int]().Description()
	assert.Equal(t, "is empty", desc)
}

// ArrayLengthEquals — slice tests

func TestArrayLengthEquals_PassesOnMatchingSlice(t *testing.T) {
	t.Parallel()
	err := expectations.ArrayLengthEquals[[]int](3).Evaluate([]int{1, 2, 3})
	assert.NoError(t, err)
}

func TestArrayLengthEquals_FailsOnWrongLengthSlice(t *testing.T) {
	t.Parallel()
	err := expectations.ArrayLengthEquals[[]int](3).Evaluate([]int{1, 2})
	require.Error(t, err)
	assert.Equal(t, "expected length to be 3, but got 2", err.Error())
}

// ArrayLengthEquals — string regression

func TestArrayLengthEquals_PassesOnMatchingString(t *testing.T) {
	t.Parallel()
	err := expectations.ArrayLengthEquals[string](5).Evaluate("hello")
	assert.NoError(t, err)
}

func TestArrayLengthEquals_FailsOnWrongLengthString(t *testing.T) {
	t.Parallel()
	err := expectations.ArrayLengthEquals[string](5).Evaluate("hi")
	require.Error(t, err)
	assert.Equal(t, "expected length to be 5, but got 2", err.Error())
}

// ArrayLengthEquals — Description

func TestArrayLengthEquals_Description(t *testing.T) {
	t.Parallel()
	desc := expectations.ArrayLengthEquals[[]int](3).Description()
	assert.Equal(t, "has length 3", desc)
}
