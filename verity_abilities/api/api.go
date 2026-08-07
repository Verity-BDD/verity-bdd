package api

import (
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

// JSONPath is a question that returns the value at a specified JSON path in the last HTTP response body.
type JSONPath = internalapi.JSONPath

// ResponseTime is a question that returns the response time of the last HTTP request.
type ResponseTime = internalapi.ResponseTime

// Using creates a new CallAnAPI ability with the given HTTP client.
var Using = internalapi.Using

// CallAnApiAt creates a new CallAnAPI ability with the given base URL.
// Panics if the base URL is invalid.
var CallAnApiAt = internalapi.CallAnApiAt

// NewRequestBuilder creates a new request builder for the given HTTP method and URL.
var NewRequestBuilder = internalapi.NewRequestBuilder

// SendRequest creates an activity that sends the given HTTP request.
var SendRequest = internalapi.SendRequest

// SendGetRequest creates a GET request activity with a fluent interface for the given URL.
var SendGetRequest = internalapi.SendGetRequest

// SendPostRequest creates a POST request activity with a fluent interface for the given URL.
var SendPostRequest = internalapi.SendPostRequest

// SendPutRequest creates a PUT request activity with a fluent interface for the given URL.
var SendPutRequest = internalapi.SendPutRequest

// SendDeleteRequest creates a DELETE request activity with a fluent interface for the given URL.
var SendDeleteRequest = internalapi.SendDeleteRequest

// SendPatchRequest creates a PATCH request activity with a fluent interface for the given URL.
var SendPatchRequest = internalapi.SendPatchRequest

// NewResponseHeader creates a new question that retrieves the named header from the last HTTP response.
var NewResponseHeader = internalapi.NewResponseHeader

// NewJSONPath creates a new question that extracts the value at the given JSON path from the last HTTP response body.
var NewJSONPath = internalapi.NewJSONPath

// LastResponseBodyAsJSON creates a question that parses the last HTTP response body as JSON into type T.
func LastResponseBodyAsJSON[T any]() verity.Question[T] {
	return internalapi.NewResponseBodyAsJSON[T]()
}

// LastResponseStatusQ is a pre-built question that returns the status code of the last HTTP response.
var LastResponseStatusQ = internalapi.LastResponseStatusQ

// LastResponseBodyQ is a pre-built question that returns the body of the last HTTP response.
var LastResponseBodyQ = internalapi.LastResponseBodyQ

// ResponseTimeQ is a pre-built question that returns the response time of the last HTTP request.
var ResponseTimeQ = internalapi.ResponseTimeQ
