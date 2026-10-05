package contracts

import (
	"strings"
	"testing"
)

func TestErrorBoundaryAndIdentities(t *testing.T) {
	for _, code := range []string{"invalid_request", "incompatible_version", "workspace_mismatch", "conflict", "unavailable", "secret-value"} {
		if strings.Contains(Fail(code).Error(), "secret-value") {
			t.Fatal("untrusted error text returned")
		}
	}
	seen := map[string]bool{}
	for range 1000 {
		id := ID()
		if !ValidID(id) || seen[id] {
			t.Fatal("bad identity")
		}
		seen[id] = true
	}
}
