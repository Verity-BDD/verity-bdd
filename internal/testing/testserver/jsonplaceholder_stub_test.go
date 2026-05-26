package testserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestJSONPlaceholderStubSupportsRequiredEndpoints(t *testing.T) {
	t.Parallel()
	baseURL := StartJSONPlaceholderStub(t)

	assertStatusAndBodyContains(t, http.MethodGet, baseURL+"/posts", nil, http.StatusOK, "title")
	assertStatusAndBodyContains(t, http.MethodGet, baseURL+"/posts/1", nil, http.StatusOK, "sunt aut facere")
	assertStatusAndBodyContains(t, http.MethodGet, baseURL+"/posts/2", nil, http.StatusOK, "title")
	assertStatusAndBodyContains(t, http.MethodGet, baseURL+"/users", nil, http.StatusOK, "email")

	postBody := map[string]any{
		"title":  "Test Post",
		"body":   "Test body",
		"userId": 1,
	}
	bodyJSON, err := json.Marshal(postBody)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	assertStatusAndBodyContains(t, http.MethodPost, baseURL+"/posts", bytes.NewReader(bodyJSON), http.StatusCreated, "Test Post")
	assertStatusAndBodyContains(t, http.MethodPost, baseURL+"/posts", bytes.NewReader(bodyJSON), http.StatusCreated, "userId")
}

func assertStatusAndBodyContains(t *testing.T, method string, url string, body io.Reader, expectedStatus int, expectedBodyPart string) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if !bytes.Contains(responseBody, []byte(expectedBodyPart)) {
		t.Fatalf("expected response body to contain %q, got %s", expectedBodyPart, string(responseBody))
	}
}
