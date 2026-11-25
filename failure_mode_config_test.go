package verity_test

import (
	"context"
	"testing"
	"time"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/wait"
	answerable "github.com/verity-bdd/verity-bdd/verity_answerable"
	ve "github.com/verity-bdd/verity-bdd/verity_expectations"
	"github.com/verity-bdd/verity-bdd/verity_expectations/ensure"
)

type failureModeTestContext struct {
	errors int
	logs   int
}

func (t *failureModeTestContext) Cleanup(func())        {}
func (t *failureModeTestContext) Errorf(string, ...any) { t.errors++ }
func (t *failureModeTestContext) FailNow()              { panic("FailNow must not be called") }
func (t *failureModeTestContext) Failed() bool          { return t.errors > 0 }
func (t *failureModeTestContext) Helper()               {}
func (t *failureModeTestContext) Logf(string, ...any)   { t.logs++ }
func (t *failureModeTestContext) Name() string          { return "failure-mode" }

func TestConfiguredFailureModesContinueAttemptsTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		activity   verity.Activity
		wantErrors int
		wantLogs   int
	}{
		{
			name: "wait non-critical",
			activity: wait.Until(answerable.ValueOf("not ready"), ve.Equals("ready")).
				WithFailureMode(verity.NonCritical()).
				For(time.Millisecond).
				CheckingEvery(time.Millisecond),
			wantErrors: 1,
		},
		{
			name: "ensure optional",
			activity: ensure.That(answerable.ValueOf("not ready"), ve.Equals("ready")).
				WithFailureMode(verity.Optional()),
			wantLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testContext := &failureModeTestContext{}
			test := verity.NewVerityTest(testContext, verity.Scene{})
			defer test.Shutdown()
			actor := test.ActorCalled("Ava")
			continued := false

			actor.AttemptsTo(tt.activity, verity.Do("records a later activity", func(context.Context, verity.Actor) error {
				continued = true
				return nil
			}))

			if !continued {
				t.Fatal("expected AttemptsTo to continue after configured failure")
			}
			if testContext.errors != tt.wantErrors {
				t.Fatalf("expected %d errors, got %d", tt.wantErrors, testContext.errors)
			}
			if testContext.logs != tt.wantLogs {
				t.Fatalf("expected %d logs, got %d", tt.wantLogs, testContext.logs)
			}
		})
	}
}

func TestWaitAndEnsurePreserveDefaultFailureModes(t *testing.T) {
	t.Parallel()

	waitActivity := wait.Until(answerable.ValueOf("not ready"), ve.Equals("ready"))
	if got := waitActivity.FailureMode(); got != verity.FailFast {
		t.Fatalf("wait default changed: got %v, want FailFast", got)
	}

	ensureActivity := ensure.That(answerable.ValueOf("not ready"), ve.Equals("ready"))
	if got := ensureActivity.FailureMode(); got != verity.ErrorButContinue {
		t.Fatalf("ensure default changed: got %v, want ErrorButContinue", got)
	}

	delayedEnsure := ensureActivity.After(time.Second)
	if got := delayedEnsure.FailureMode(); got != verity.FailFast {
		t.Fatalf("delayed ensure default changed: got %v, want FailFast", got)
	}
}
