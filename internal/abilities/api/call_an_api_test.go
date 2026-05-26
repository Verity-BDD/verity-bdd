package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
)

type stubActor struct {
	name      string
	ctx       context.Context
	abilities []abilities.Ability
}

func newStubActor(name string, ctx context.Context, abilities ...abilities.Ability) *stubActor {
	return &stubActor{name: name, ctx: ctx, abilities: abilities}
}

func (a *stubActor) Name() string { return a.name }

func (a *stubActor) Context() context.Context { return a.ctx }

func (a *stubActor) WhoCan(abilities ...abilities.Ability) core.Actor {
	a.abilities = append(a.abilities, abilities...)
	return a
}

func (a *stubActor) AbilityTo(target abilities.Ability) (abilities.Ability, error) {
	for _, ability := range a.abilities {
		if fmt.Sprintf("%T", ability) == fmt.Sprintf("%T", target) {
			return ability, nil
		}
	}
	return nil, fmt.Errorf("actor '%s' can't %s. Did you give them the ability?", a.name, core.AbilityName(target))
}

func (a *stubActor) AttemptsTo(activities ...core.Activity) {
	for _, activity := range activities {
		_ = activity.PerformAs(a.ctx, a)
	}
}

func TestSendRequestStoresLastResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	req, err := http.NewRequest(http.MethodGet, server.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	last := ab.LastResponse()
	if last != resp {
		t.Fatalf("expected stored response to match returned response")
	}
	if last.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, last.StatusCode)
	}
}

func TestSendRequestResolvesRelativeURLWithBase(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Fatalf("expected path /hello, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	if err := ab.SetBaseURL(server.URL); err != nil {
		t.Fatalf("SetBaseURL returned error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/hello", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()
}

func TestCallAnApiAtResolvesRelativeURL(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/path" {
			t.Fatalf("expected path /path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	ab := CallAnApiAt(server.URL)
	req, err := http.NewRequest(http.MethodGet, "/path", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()
}

func TestSetBaseURLRejectsInvalidURL(t *testing.T) {
	t.Parallel()
	ab := Using(http.DefaultClient)
	if err := ab.SetBaseURL("example.com"); err == nil {
		t.Fatalf("expected error for base URL without scheme")
	}
}

func TestCallAnApiAtPanicsOnInvalidBaseURL(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for invalid base URL")
		}
	}()

	_ = CallAnApiAt("://bad")
}

func TestLastResponseIsSafeForConcurrentSendRequest(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	if err := ab.SetBaseURL(server.URL); err != nil {
		t.Fatalf("SetBaseURL returned error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, "/concurrent", nil)
			if err != nil {
				t.Errorf("failed to build request: %v", err)
				return
			}
			resp, err := ab.SendRequest(req, context.Background())
			if err != nil {
				t.Errorf("SendRequest returned error: %v", err)
				return
			}
			_ = resp.Body.Close()
		}()
	}

	wg.Wait()

	last := ab.LastResponse()
	if last == nil {
		t.Fatalf("expected last response to be stored")
	}
	if last.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, last.StatusCode)
	}
}
