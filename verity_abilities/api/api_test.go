package api_test

import (
	"strings"
	"testing"

	"github.com/verity-bdd/verity-bdd/verity_abilities/api"
)

func TestSendPatchRequestIsExposed(t *testing.T) {
	t.Parallel()

	activity := api.SendPatchRequest("/users/1")

	if got := activity.Description(); !strings.Contains(got, "PATCH") {
		t.Fatalf("expected description to mention PATCH method, got %q", got)
	}
}
