package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestBuilderAppliesHeadersAndBodies(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-One"); got != "1" {
			t.Fatalf("expected X-One header to be 1, got %s", got)
		}
		if got := r.Header.Get("X-Two"); got != "two" {
			t.Fatalf("expected X-Two header to be two, got %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %s", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"tester"}` {
			t.Fatalf("unexpected body: %s", string(body))
		}
	}))
	t.Cleanup(server.Close)

	builder := NewRequestBuilder(http.MethodPost, server.URL)
	builder.WithHeader("X-One", "1").WithHeaders(map[string]string{"X-Two": "two"})
	if err := builder.WithJSONBody(map[string]string{"name": "tester"}); err != nil {
		t.Fatalf("WithJSONBody returned error: %v", err)
	}

	req, err := builder.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	ab := Using(server.Client())
	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()
}

func TestRequestBuilderWithSetsBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "" {
			t.Fatalf("expected no Content-Type set for generic With body")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw-body" {
			t.Fatalf("unexpected body: %s", string(body))
		}
	}))
	t.Cleanup(server.Close)

	builder := NewRequestBuilder(http.MethodPost, server.URL)
	builder.With("raw-body")

	req, err := builder.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	ab := Using(server.Client())
	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()
}

func TestRequestActivityWithNilBuilderReturnsError(t *testing.T) {
	t.Parallel()
	activity := &RequestActivity{}
	actor := newStubActor("nil-builder", context.Background())

	err := activity.PerformAs(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error for nil builder")
	}
	if err.Error() != "request builder is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendRequestPerformAsRequiresAbility(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	interaction := &sendRequest{request: req}
	actor := newStubActor("no-ability", context.Background())

	err = interaction.PerformAs(context.Background(), actor)
	if err == nil {
		t.Fatalf("expected error when actor lacks ability")
	}
	if err.Error() == "" {
		t.Fatalf("expected meaningful error, got empty string")
	}
}
