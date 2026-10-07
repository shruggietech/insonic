// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestAcquisitionCredentialTransportBeforeIntentAndSecretResolution(t *testing.T) {
	elected := true
	credential := contracts.ID()
	for _, source := range []string{"http://example.test/media", "http://127.0.0.1/media"} {
		for _, options := range []Options{{}, {LocalHTTP: &elected}} {
			if source == "http://127.0.0.1/media" && options.LocalHTTP != nil {
				continue
			}
			if _, e := PrepareImport(ImportRequest{Defaults: options, Items: []Item{{Source: source, CredentialID: credential}}}); e == nil {
				t.Fatal("insecure credential intent accepted")
			}
			secret := &acquisitionSecret{}
			_, e := (httpAcquisition{Secrets: secret}).Acquire(context.Background(), source, credential, filepath.Join(t.TempDir(), "media"), options)
			if e == nil || secret.resolved != nil {
				t.Fatal("credential resolved before transport rejection")
			}
		}
	}
	for _, item := range []Item{{Source: "http://example.test/media"}, {Source: "https://example.test/media", CredentialID: credential}, {Source: "http://127.0.0.1/media", CredentialID: credential, Options: Options{LocalHTTP: &elected}}} {
		if _, e := PrepareImport(ImportRequest{Items: []Item{item}}); e != nil {
			t.Fatal("elected source transport rejected")
		}
	}
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-access" {
			t.Error("TLS credential missing")
		}
		w.Write([]byte("media"))
	}))
	defer tls.Close()
	if _, e := (httpAcquisition{Secrets: &acquisitionSecret{}, Client: tls.Client()}).Acquire(context.Background(), tls.URL, credential, filepath.Join(t.TempDir(), "media"), Options{}); e != nil {
		t.Fatal(e)
	}
}

func TestAcquisitionDeclaredAndUnknownByteLimits(t *testing.T) {
	for _, declared := range []bool{true, false} {
		for _, size := range []int{4, 5} {
			t.Run(strconv.FormatBool(declared)+strconv.Itoa(size), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if declared {
						w.Header().Set("Content-Length", strconv.Itoa(size))
					} else {
						w.(http.Flusher).Flush()
					}
					w.Write(make([]byte, size))
				}))
				defer server.Close()
				max := int64(4)
				destination := filepath.Join(t.TempDir(), "media")
				_, e := (httpAcquisition{}).Acquire(context.Background(), server.URL, "", destination, Options{AcquisitionMaxBytes: &max})
				if size == 4 {
					if e != nil {
						t.Fatal(e)
					}
					stat, e := os.Stat(destination)
					if e != nil || stat.Size() != 4 {
						t.Fatal("at-limit bytes lost")
					}
				} else {
					if code(e) != "output_limit" {
						t.Fatalf("over-limit response accepted: %v", e)
					}
					if _, e = os.Stat(destination); !os.IsNotExist(e) {
						t.Fatal("over-limit scratch retained")
					}
				}
			})
		}
	}
}

func TestAcquisitionTimeoutAndCancellationRemovePartialBytes(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(strconv.FormatBool(cancelled), func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.(http.Flusher).Flush()
				w.Write([]byte("partial"))
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			if cancelled {
				go func() { <-started; stop() }()
			}
			timeout := int64(60)
			destination := filepath.Join(t.TempDir(), "media")
			begin := time.Now()
			_, e := (httpAcquisition{}).Acquire(ctx, server.URL, "", destination, Options{AcquisitionTimeoutMS: &timeout})
			if e == nil || time.Since(begin) > time.Second {
				t.Fatal("stream not bounded by cancellation/deadline")
			}
			if _, e = os.Stat(destination); !os.IsNotExist(e) {
				t.Fatal("partial scratch retained")
			}
		})
	}
}

func TestAcquisitionOptionsOverrideInheritanceAndRejectNonpositive(t *testing.T) {
	enabled := true
	disabled := false
	max := int64(4096)
	timeout := int64(2000)
	itemMax := int64(1024)
	got := merged(Options{LocalHTTP: &enabled, AcquisitionMaxBytes: &max, AcquisitionTimeoutMS: &timeout}, Options{LocalHTTP: &disabled, AcquisitionMaxBytes: &itemMax})
	if *got.LocalHTTP || *got.AcquisitionMaxBytes != 1024 || *got.AcquisitionTimeoutMS != 2000 {
		t.Fatal("acquisition item inheritance incorrect")
	}
	for _, value := range []int64{0, -1} {
		if validOptions(Options{AcquisitionMaxBytes: &value}) || validOptions(Options{AcquisitionTimeoutMS: &value}) {
			t.Fatal("nonpositive acquisition bound accepted")
		}
	}
}
