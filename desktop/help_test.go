//go:build desktop

package desktop

import "testing"

func TestPackagedOfflinePagesResolveTheirLocalAssets(t *testing.T) {
	if err := verifyHelp(); err != nil {
		t.Fatal("offline asset graph is incomplete", err)
	}
}
