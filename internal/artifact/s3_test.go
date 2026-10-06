// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type secrets map[string][]byte

func (s secrets) Resolve(ctx context.Context, id string) ([]byte, error) { return s[id], nil }
func (s secrets) Status(ctx context.Context, id string) (string, error)  { return "available", nil }
func TestS3RangeAndSigning(t *testing.T) {
	signed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signed = strings.Contains(r.Header.Get("Authorization"), "Credential=fixture-access/")
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", "6")
			return
		}
		w.Header().Set("Content-Length", "6")
		w.Write([]byte("abcdef"))
	}))
	defer server.Close()
	raw, _ := json.Marshal(S3Credentials{"fixture-access", "fixture-secret", ""})
	s, e := NewS3(context.Background(), S3Config{Endpoint: server.URL, Bucket: "fixture", Region: "us-east-1", AddressingStyle: "path", Authentication: "credential", CredentialID: "fixture"}, secrets{"fixture": raw}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.OpenRange(context.Background(), catalog.Publication{Key: "objects/object", Size: 6}, 1, 2); e == nil {
		t.Fatal("provider ignored range without detection")
	}
	if !signed {
		t.Fatal("explicit credentials not signed")
	}
}

func TestS3UnknownCompletionAbortStaysPending(t *testing.T) {
	var heads atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			heads.Add(1)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<Error><Code>NoSuchUpload</Code><Message>Gone</Message></Error>`))
	}))
	defer server.Close()
	s, e := NewS3(context.Background(), S3Config{Endpoint: server.URL, Bucket: "fixture", Region: "us-east-1", Authentication: "anonymous", AddressingStyle: "path"}, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	for _, requested := range []bool{false, true} {
		p := catalog.Publication{Key: "objects/uncertain", UploadID: "unknown", CompletionRequested: requested, State: "pending"}
		if e = s.Abort(context.Background(), p); e == nil {
			t.Fatal("uncertain completion declared aborted")
		}
	}
	if heads.Load() < 8 {
		t.Fatal("bounded visibility observations absent")
	}
}
func TestS3MissingCredentialAndURL(t *testing.T) {
	for _, c := range []S3Config{{Endpoint: "https://user:secret@example.com", Bucket: "bucket", Region: "region", Authentication: "anonymous", AddressingStyle: "path"}, {Endpoint: "https://example.com", Bucket: "bucket", Region: "region", Authentication: "credential", AddressingStyle: "path", CredentialID: "missing"}} {
		_, e := NewS3(context.Background(), c, nil, nil)
		if e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal("credential configuration admitted/leaked")
		}
	}
}

func TestS3RemoteHTTPRefusedBeforeCredentialUse(t *testing.T) {
	for _, endpoint := range []string{"http://storage.example.com", "http://203.0.113.4:9000", "http://[2001:db8::1]:9000"} {
		for _, authentication := range []string{"anonymous", "credential", "environment"} {
			if _, e := NewS3(context.Background(), S3Config{Endpoint: endpoint, Bucket: "fixture", Region: "us-east-1", Authentication: authentication, AddressingStyle: "path"}, nil, nil); e == nil {
				t.Fatal("remote HTTP accepted", endpoint, authentication)
			}
		}
	}
	for _, endpoint := range []string{"http://localhost:9000", "http://127.0.0.1:9000", "http://[::1]:9000", "http://10.1.2.3:9000", "https://storage.example.com"} {
		s, e := NewS3(context.Background(), S3Config{Endpoint: endpoint, Bucket: "fixture", Region: "us-east-1", Authentication: "anonymous", AddressingStyle: "path"}, nil, nil)
		if e != nil {
			t.Fatal("explicit local HTTP/remote HTTPS refused", e)
		}
		s.Close()
	}
}
