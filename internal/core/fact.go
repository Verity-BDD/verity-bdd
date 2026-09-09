package core

import "context"

type fact struct {
	description string
	setup       func(context.Context, Actor) error
	teardown    func(context.Context, Actor) error
}

func (f *fact) Description() string {
	return f.description
}

func (f *fact) Setup(ctx context.Context, actor Actor) error {
	return f.setup(ctx, actor)
}

func (f *fact) Teardown(ctx context.Context, actor Actor) error {
	if f.teardown == nil {
		return nil
	}
	return f.teardown(ctx, actor)
}

// FactAbout creates a setup-only fact about an actor. Its teardown is a no-op.
// It panics if setup is nil.
func FactAbout(description string, setup func(context.Context, Actor) error) Fact {
	if setup == nil {
		panic("FactAbout: setup function cannot be nil")
	}
	return &fact{description: description, setup: setup}
}
