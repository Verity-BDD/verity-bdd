package verity_expectations_test

import (
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

var (
	_ func(string) ensure.Expectation[interface{}]      = ve.ContainsKey
	_ func(interface{}) ensure.Expectation[interface{}] = ve.IsGreaterThan
	_ func(interface{}) ensure.Expectation[interface{}] = ve.IsLessThan
)
