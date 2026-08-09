package api

import (
	"net/http"

	"github.com/verity-bdd/verity-bdd/internal/core"
)

// SendRequest creates a SendRequest interaction (exported function)
func SendRequest(req *http.Request) core.Activity {
	return a(req)
}

// SendGetRequest creates GET request activity with fluent interface
func SendGetRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: RequestFor("GET", url),
	}
}

// SendPostRequest creates POST request activity with fluent interface
func SendPostRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: RequestFor("POST", url),
	}
}

// SendPutRequest creates PUT request activity with fluent interface
func SendPutRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: RequestFor("PUT", url),
	}
}

// SendDeleteRequest creates DELETE request activity with fluent interface
func SendDeleteRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: RequestFor("DELETE", url),
	}
}

// SendPatchRequest creates PATCH request activity with fluent interface
func SendPatchRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: RequestFor("PATCH", url),
	}
}
