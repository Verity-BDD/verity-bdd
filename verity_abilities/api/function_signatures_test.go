package api_test

import (
	"net/http"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
)

var (
	_ func(*http.Client) api.CallAnAPI         = api.Using
	_ func(string) api.CallAnAPI               = api.CallAnApiAt
	_ func(string, string) *api.RequestBuilder = api.RequestFor
	_ func(*http.Request) verity.Activity      = api.SendRequest
	_ func(string) *api.RequestActivity        = api.SendGetRequest
	_ func(string) *api.RequestActivity        = api.SendPostRequest
	_ func(string) *api.RequestActivity        = api.SendPutRequest
	_ func(string) *api.RequestActivity        = api.SendDeleteRequest
	_ func(string) *api.RequestActivity        = api.SendPatchRequest
	_ func(string) api.ResponseHeader          = api.LastResponseHeader
	_ func(string) api.JSONPath                = api.LastResponseBodyAtJSONPath
)
