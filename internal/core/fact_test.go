package core

import "testing"

func TestFactAboutRejectsNilSetup(t *testing.T) {
	t.Parallel()

	defer func() {
		const want = "FactAbout: setup function cannot be nil"
		if got := recover(); got != want {
			t.Fatalf("panic = %v, want %q", got, want)
		}
	}()
	FactAbout("fact", nil)
}
