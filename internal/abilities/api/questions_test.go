package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestQuestionsErrorWhenNoResponse(t *testing.T) {
	t.Parallel()
	ab := Using(http.DefaultClient)
	actor := newStubActor("no-response", context.Background(), ab)

	if _, err := LastResponseStatusQ.AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected error when no response available")
	}
	if _, err := LastResponseBodyQ.AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected error when no response available")
	}
	if _, err := LastResponseHeader("X-Test").AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected error when no response available")
	}
}

func TestLastResponseBodyCanBeReadMultipleTimes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()

	actor := newStubActor("reader", context.Background(), ab)
	body1, err := LastResponseBodyQ.AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	body2, err := LastResponseBodyQ.AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	if body1 != "hello" || body2 != "hello" {
		t.Fatalf("expected consistent body, got %q and %q", body1, body2)
	}
}

func TestResponseBodyAsJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		case "/bad":
			_, _ = w.Write([]byte("not json"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	actor := newStubActor("json", context.Background(), ab)

	requestJSON := func(path string) {
		req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
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
	}

	requestJSON("/ok")
	answer, err := NewResponseBodyAsJSON[map[string]string]().AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected JSON parse to succeed, got error: %v", err)
	}
	if answer["message"] != "ok" {
		t.Fatalf("unexpected JSON message: %v", answer)
	}

	requestJSON("/bad")
	if _, err := NewResponseBodyAsJSON[map[string]string]().AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected error for invalid JSON body")
	}
}

func TestJSONPathTraversal(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"emails":["a@example.com","b@example.com"]},"items":[{"id":1},{"id":2}]}`))
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	actor := newStubActor("jsonpath", context.Background(), ab)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := ab.SendRequest(req, context.Background())
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	defer resp.Body.Close()

	email, err := LastResponseBodyAtJSONPath("user.emails.1").AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected email path to succeed: %v", err)
	}
	if email != "b@example.com" {
		t.Fatalf("unexpected email value: %v", email)
	}

	ids, err := LastResponseBodyAtJSONPath("items.*.id").AnsweredBy(context.Background(), actor)
	if err != nil {
		t.Fatalf("expected wildcard path to succeed: %v", err)
	}

	expected := []any{float64(1), float64(2)}
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("unexpected ids result: %#v", ids)
	}

	if _, err := LastResponseBodyAtJSONPath("user.emails.5").AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected out-of-bounds error")
	}
	if _, err := LastResponseBodyAtJSONPath("user.unknown").AnsweredBy(context.Background(), actor); err == nil {
		t.Fatalf("expected unknown path error")
	}
}
