// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"testing/fstest"
)

func TestOfflineHelpBeforeWorkspaceAndOpeningFailure(t *testing.T) {
	files := fstest.MapFS{"offline-manifest.json": {Data: []byte("{\"version\":\"" + contracts.Version + "\",\"entrypoint\":\"docs/index.html\"}")}, "docs/index.html": {Data: []byte("<h1>installed help</h1>")}}
	b := &Bridge{helpAssets: files}
	response := b.OpenHelp()
	if response.Error != nil || response.Result.(map[string]any)["url"] != "/assets/help/docs/index.html" {
		t.Fatal(response)
	}
	delete(files, "docs/index.html")
	response = b.OpenHelp()
	if response.Error == nil || response.Error.Reason != "documentation_open" {
		t.Fatal(response)
	}
}
