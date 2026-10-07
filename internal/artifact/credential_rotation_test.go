// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestS3ResolvesReplacedCredentialForSubsequentRequests(t *testing.T) {
	var mu sync.Mutex
	headers := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		headers = append(headers, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Length", "6")
		w.Header().Set("Last-Modified", "Wed, 07 Oct 2026 12:00:00 GMT")
		w.Header().Set("ETag", `"fixture-etag"`)
	}))
	defer server.Close()
	raw, _ := json.Marshal(S3Credentials{"first-key", "fixture-one", ""})
	provider := secrets{"fixture": raw}
	s, e := NewS3(context.Background(), S3Config{Endpoint: server.URL, Bucket: "fixture", Region: "us-east-1", AddressingStyle: "path", Authentication: "credential", CredentialID: "fixture"}, provider, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.Stat(context.Background(), catalog.Publication{Key: "objects/source"}); e != nil {
		t.Fatal(e)
	}
	raw, _ = json.Marshal(S3Credentials{"replacement-key", "fixture-two", ""})
	provider["fixture"] = raw
	if _, e = s.Stat(context.Background(), catalog.Publication{Key: "objects/source"}); e != nil {
		t.Fatal(e)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(headers) != 2 || !strings.Contains(headers[0], "Credential=first-key/") || !strings.Contains(headers[1], "Credential=replacement-key/") {
		t.Fatal("subsequent request retained old auth")
	}
}
