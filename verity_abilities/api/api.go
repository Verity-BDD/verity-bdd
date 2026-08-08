package api

import (
	"net/http"

	verity "github.com/verity-bdd/verity-bdd"
	internalapi "github.com/verity-bdd/verity-bdd/internal/abilities/api"
)

// CallAnAPI enables an actor to make HTTP requests to APIs.
type CallAnAPI = internalapi.CallAnAPI

// RequestBuilder helps build HTTP requests with a fluent interface.
type RequestBuilder = internalapi.RequestBuilder

// RequestActivity is a unified HTTP request activity with a fluent interface.
type RequestActivity = internalapi.RequestActivity

// LastResponseStatus is a question that returns the status code of the last HTTP response.
type LastResponseStatus = internalapi.LastResponseStatus

// LastResponseBody is a question that returns the body of the last HTTP response.
type LastResponseBody = internalapi.LastResponseBody

// ResponseHeader is a question that returns a specific header value from the last HTTP response.
type ResponseHeader = internalapi.ResponseHeader

// JSONPath returns an untyped decoded JSON value at a dot-separated path.
// Object values decode as map[string]any, arrays as []any, numbers as float64,
// and an array wildcard segment can produce []any.
type JSONPath = internalapi.JSONPath

// ResponseTime is a placeholder question for request timing. Timing is not
// implemented, so AnsweredBy currently returns 0.
type ResponseTime = internalapi.ResponseTime

// Using creates a new CallAnAPI ability with the given HTTP client.
func Using(client *http.Client) CallAnAPI {
	return internalapi.Using(client)
}

// CallAnApiAt creates a new CallAnAPI ability with the given base URL.
// Panics if the base URL is invalid.
func CallAnApiAt(baseURL string) CallAnAPI {
	return internalapi.CallAnApiAt(baseURL)
}

// NewRequestBuilder creates a new request builder for the given HTTP method and URL.
func NewRequestBuilder(method, url string) *RequestBuilder {
	return internalapi.NewRequestBuilder(method, url)
}

// SendRequest creates an activity that sends the given HTTP request.
func SendRequest(request *http.Request) verity.Activity {
	return internalapi.SendRequest(request)
}

// SendGetRequest creates a GET request activity with a fluent interface for the given URL.
func SendGetRequest(url string) *RequestActivity {
	return internalapi.SendGetRequest(url)
}

// SendPostRequest creates a POST request activity with a fluent interface for the given URL.
func SendPostRequest(url string) *RequestActivity {
	return internalapi.SendPostRequest(url)
}

// SendPutRequest creates a PUT request activity with a fluent interface for the given URL.
func SendPutRequest(url string) *RequestActivity {
	return internalapi.SendPutRequest(url)
}

// SendDeleteRequest creates a DELETE request activity with a fluent interface for the given URL.
func SendDeleteRequest(url string) *RequestActivity {
	return internalapi.SendDeleteRequest(url)
}

// SendPatchRequest creates a PATCH request activity with a fluent interface for the given URL.
func SendPatchRequest(url string) *RequestActivity {
	return internalapi.SendPatchRequest(url)
}

// NewResponseHeader creates a new question that retrieves the named header from the last HTTP response.
func NewResponseHeader(key string) ResponseHeader {
	return internalapi.NewResponseHeader(key)
}

// NewJSONPath creates an untyped question for a dot-separated JSON path.
// Numeric path segments index arrays and "*" maps the remaining path across
// array elements, omitting elements for which that remaining path errors.
func NewJSONPath(path string) JSONPath {
	return internalapi.NewJSONPath(path)
}

// LastResponseBodyAsJSON creates a question that parses the last HTTP response body as JSON into type T.
func LastResponseBodyAsJSON[T any]() verity.Question[T] {
	return internalapi.NewResponseBodyAsJSON[T]()
}

// LastResponseStatusQ is a pre-built question that returns the status code of the last HTTP response.
var LastResponseStatusQ = internalapi.LastResponseStatusQ

// LastResponseBodyQ is a pre-built question that returns the body of the last HTTP response.
var LastResponseBodyQ = internalapi.LastResponseBodyQ

// ResponseTimeQ is the pre-built timing placeholder. It currently returns 0
// because request timing is not implemented.
var ResponseTimeQ = internalapi.ResponseTimeQ
