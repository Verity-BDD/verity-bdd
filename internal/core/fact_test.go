package core

import (
	"context"
	"testing"
)

func TestFactFactoriesRejectNilCallbacks(t *testing.T) {
	t.Parallel()

	setup := func(context.Context, Actor) error { return nil }
	teardown := func(context.Context, Actor) error { return nil }
	tests := []struct {
		name      string
		wantPanic string
		create    func()
	}{
		{
			name:      "setup-only nil setup",
			wantPanic: "FactAbout: setup function cannot be nil",
			create:    func() { FactAbout("fact", nil) },
		},
		{
			name:      "paired nil setup",
			wantPanic: "FactAboutWithTeardown: setup function cannot be nil",
			create:    func() { FactAboutWithTeardown("fact", nil, teardown) },
		},
		{
			name:      "paired nil teardown",
			wantPanic: "FactAboutWithTeardown: teardown function cannot be nil",
			create:    func() { FactAboutWithTeardown("fact", setup, nil) },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if got := recover(); got != test.wantPanic {
					t.Fatalf("panic = %v, want %q", got, test.wantPanic)
				}
			}()
			test.create()
		})
	}
}
