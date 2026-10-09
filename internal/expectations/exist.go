package expectations

import (
	"context"
	"fmt"
	"reflect"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/expectations/ensure"
)

// ExistExpectation checks whether a value is not nil.
type ExistExpectation[T any] struct{}

// Evaluate reports an error when actual is nil, including a typed nil held by an interface.
func (ExistExpectation[T]) Evaluate(_ context.Context, _ core.Actor, actual T) error {
	value := reflect.ValueOf(actual)
	if !value.IsValid() || (isNilable(value.Kind()) && value.IsNil()) {
		return fmt.Errorf("expected value to exist, but got nil")
	}
	return nil
}

// Description returns the expectation description.
func (ExistExpectation[T]) Description() string {
	return "exists"
}

// Exist creates an expectation that checks whether a value is not nil.
// Non-nilable values always exist.
func Exist[T any]() ensure.Expectation[T] {
	return ExistExpectation[T]{}
}

func isNilable(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}
