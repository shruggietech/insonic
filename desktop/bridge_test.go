package desktop

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestUnselectedWorkspaceDoesNotManufactureBehavior(t *testing.T) {
	result := (&Bridge{}).Show()
	if result.Version != contracts.Version || result.Error == nil || result.Error.Code != "not_found" {
		t.Fatal("missing workspace")
	}
}
