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
		builder: NewRequestBuilder("GET", url),
	}
}

// SendPostRequest creates POST request activity with fluent interface
func SendPostRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: NewRequestBuilder("POST", url),
	}
}

// SendPutRequest creates PUT request activity with fluent interface
func SendPutRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: NewRequestBuilder("PUT", url),
	}
}

// SendDeleteRequest creates DELETE request activity with fluent interface
func SendDeleteRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: NewRequestBuilder("DELETE", url),
	}
}

// SendPatchRequest creates PATCH request activity with fluent interface
func SendPatchRequest(url string) *RequestActivity {
	return &RequestActivity{
		builder: NewRequestBuilder("PATCH", url),
	}
}
