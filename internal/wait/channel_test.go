package wait_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/verity-bdd/verity-bdd/internal/core"
	"github.com/verity-bdd/verity-bdd/internal/wait"
)

// TestChannelActivity_ReceivesValue: send message to buffered channel before PerformAs → returns nil
func TestChannelActivity_ReceivesValue(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "hello"

	activity := wait.UntilReceived(ch)
	err := activity.PerformAs(context.Background(), &stubActor{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestChannelActivity_TimeoutWithNoSend: For(60ms), nobody sends → error contains "timed out after"
func TestChannelActivity_TimeoutWithNoSend(t *testing.T) {
	ch := make(chan string)
	activity := wait.UntilReceived(ch).For(60 * time.Millisecond)

	err := activity.PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out after") {
		t.Fatalf("expected error to contain 'timed out after', got: %v", err)
	}
}

// TestChannelActivity_ContextCancelledBeforeSend: cancel ctx before sending → error contains "cancelled" but NOT "timed out"
func TestChannelActivity_ContextCancelledBeforeSend(t *testing.T) {
	ch := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	activity := wait.UntilReceived(ch).For(10 * time.Second)

	// cancel before PerformAs returns; use a goroutine to cancel quickly
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := activity.PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}
	if strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected cancellation error, not timeout: %v", err)
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

// TestChannelActivity_ContextAlreadyCancelledOnEntry: pre-cancelled ctx passed in → error contains "cancelled"
func TestChannelActivity_ContextAlreadyCancelledOnEntry(t *testing.T) {
	ch := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling PerformAs

	activity := wait.UntilReceived(ch).For(10 * time.Second)
	err := activity.PerformAs(ctx, &stubActor{})

	if err == nil {
		t.Fatal("expected error for pre-cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected error to mention cancellation, got: %v", err)
	}
}

// TestChannelActivity_ClosedChannelReturnsError: close channel, use For(10s) to avoid race with timeout → error contains "channel closed"
func TestChannelActivity_ClosedChannelReturnsError(t *testing.T) {
	ch := make(chan string)
	close(ch)

	activity := wait.UntilReceived(ch).For(10 * time.Second)
	err := activity.PerformAs(context.Background(), &stubActor{})

	if err == nil {
		t.Fatal("expected error for closed channel, got nil")
	}
	if !strings.Contains(err.Error(), "channel closed") {
		t.Fatalf("expected error to mention 'channel closed', got: %v", err)
	}
}

// TestChannelActivity_ForOverridesDefaultTimeout: For(60ms) terminates well under 1s
func TestChannelActivity_ForOverridesDefaultTimeout(t *testing.T) {
	ch := make(chan string)
	start := time.Now()

	activity := wait.UntilReceived(ch).For(60 * time.Millisecond)
	_ = activity.PerformAs(context.Background(), &stubActor{})

	elapsed := time.Since(start)
	if elapsed > 1*time.Second {
		t.Fatalf("For(60ms) did not override default 5s timeout, elapsed: %v", elapsed)
	}
}

// TestChannelActivity_Description: Description() contains the timeout duration
func TestChannelActivity_Description(t *testing.T) {
	ch := make(chan string)
	activity := wait.UntilReceived(ch).For(30 * time.Second)

	desc := activity.Description()
	if !strings.Contains(desc, "30s") {
		t.Fatalf("expected description to contain '30s', got: %q", desc)
	}
}

// TestChannelActivity_FailureMode: FailureMode() returns core.FailFast
func TestChannelActivity_FailureMode(t *testing.T) {
	ch := make(chan string)
	activity := wait.UntilReceived(ch)

	if mode := activity.FailureMode(); mode != core.FailFast {
		t.Fatalf("expected FailFast, got %v", mode)
	}
}

// TestChannelActivity_BufferedChannelAlreadyHasValue: buffered channel with value already in it → returns immediately without blocking
func TestChannelActivity_BufferedChannelAlreadyHasValue(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 42

	start := time.Now()
	activity := wait.UntilReceived(ch).For(10 * time.Second)
	err := activity.PerformAs(context.Background(), &stubActor{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected immediate return for buffered channel with value, elapsed: %v", elapsed)
	}
}
