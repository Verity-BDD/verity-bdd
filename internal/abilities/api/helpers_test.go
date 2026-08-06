package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPatchRequestSendsPatchMethod(t *testing.T) {
	t.Parallel()
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	ab := Using(server.Client())
	actor := newStubActor("patcher", context.Background(), ab)

	activity := SendPatchRequest(server.URL)
	if err := activity.PerformAs(context.Background(), actor); err != nil {
		t.Fatalf("PerformAs returned error: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH method, got %s", gotMethod)
	}
}
