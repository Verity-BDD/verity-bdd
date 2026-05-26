package expectations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-bdd/verity-bdd/internal/expectations"
)

func TestNot_PassesWhenInnerFails(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Equals("foo")).Evaluate("bar")
	assert.NoError(t, err)
}

func TestNot_FailsWhenInnerPasses(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Equals("foo")).Evaluate("foo")
	require.Error(t, err)
	assert.Equal(t, "not equals foo: got foo", err.Error())
}

func TestNot_Description(t *testing.T) {
	t.Parallel()
	desc := expectations.Not(expectations.Equals("foo")).Description()
	assert.Equal(t, "not equals foo", desc)
}

func TestNot_IsEmpty_FailsOnEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.IsEmpty[string]()).Evaluate("")
	require.Error(t, err)
	assert.Equal(t, "not is empty: got ", err.Error())
}

func TestNot_IsEmpty_PassesOnNonEmptyString(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.IsEmpty[string]()).Evaluate("hello")
	assert.NoError(t, err)
}

func TestNot_DoubleNegation_PassesWhenInnerPasses(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Not(expectations.Equals("foo"))).Evaluate("foo")
	assert.NoError(t, err)
}

func TestNot_DoubleNegation_FailsWhenInnerFails(t *testing.T) {
	t.Parallel()
	err := expectations.Not(expectations.Not(expectations.Equals("foo"))).Evaluate("bar")
	require.Error(t, err)
	assert.Equal(t, "not not equals foo: got bar", err.Error())
}
