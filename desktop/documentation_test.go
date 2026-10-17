// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestOfflineHelpPolicyDoesNotRestrictRuntimeRequests(t *testing.T) {
	handler := HelpAssetPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, name := range []string{"/assets/help/docs/index.html", "/assets/help/brand/fonts/fonts.css", "/runtime/operate"} {
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, httptest.NewRequest("GET", name, nil))
		policy := out.Header().Get("Content-Security-Policy")
		if strings.HasPrefix(name, "/assets/help/") {
			if !strings.Contains(policy, "connect-src 'none'") || !strings.Contains(policy, "font-src 'self'") {
				t.Fatal(name, policy)
			}
		} else if policy != "" {
			t.Fatal("help policy changed runtime access", policy)
		}
	}
}
