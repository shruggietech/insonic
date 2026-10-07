// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

type Acquisition struct {
	Adapter           string            `json:"adapter"`
	Version           string            `json:"version"`
	SourceLocator     string            `json:"source_locator"`
	RetrievedAt       string            `json:"retrieved_at"`
	DeclaredMIME      string            `json:"declared_mime_type"`
	PublisherMetadata map[string]string `json:"publisher_metadata"`
}

// AcquisitionAdapter writes complete original bytes and returns nonsecret facts.
// Adapters resolve authorization through credential IDs, never provenance data.
type AcquisitionAdapter interface {
	Acquire(context.Context, string, string, string) (Acquisition, error)
}
type httpAcquisition struct {
	Secrets contracts.SecretProvider
	Client  *http.Client
}

func stableLocator(source string) (string, error) {
	u, e := contracts.SourceURL(source)
	if e != nil || u.Scheme == "" {
		return "", contracts.Fail("invalid_request")
	}
	return u.String(), nil
}
func (a httpAcquisition) Acquire(ctx context.Context, source, credential, destination string) (Acquisition, error) {
	locator, e := stableLocator(source)
	if e != nil {
		return Acquisition{}, e
	}
	u, _ := contracts.SourceURL(source)
	if u.Scheme != "https" && u.Scheme != "http" {
		return Acquisition{}, contracts.Fail("invalid_request")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if e != nil {
		return Acquisition{}, contracts.Fail("invalid_request")
	}
	if credential != "" {
		if a.Secrets == nil {
			return Acquisition{}, contracts.Fail("unavailable")
		}
		value, e := a.Secrets.Resolve(ctx, credential)
		if e != nil {
			return Acquisition{}, contracts.Fail("unavailable")
		}
		if len(value) == 0 || strings.ContainsAny(string(value), "\r\n") {
			return Acquisition{}, contracts.Fail("invalid_request")
		}
		req.Header.Set("Authorization", "Bearer "+string(value))
		for i := range value {
			value[i] = 0
		}
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 0, CheckRedirect: func(next *http.Request, previous []*http.Request) error {
			if _, e := contracts.SourceURL(next.URL.String()); e != nil {
				return e
			}
			if len(previous) > 10 {
				return contracts.Fail("operation_failed")
			}
			if next.URL.Host != previous[0].URL.Host {
				next.Header.Del("Authorization")
			}
			if previous[0].URL.Scheme == "https" && next.URL.Scheme != "https" {
				return contracts.Fail("operation_failed")
			}
			if next.URL.Scheme != "http" && next.URL.Scheme != "https" {
				return contracts.Fail("operation_failed")
			}
			return nil
		}}
	}
	response, e := client.Do(req)
	if e != nil {
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	f, e := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return Acquisition{}, contracts.Fail("unavailable")
	}
	n, e := io.Copy(f, response.Body)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil || (response.ContentLength >= 0 && n != response.ContentLength) {
		os.Remove(destination)
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	return Acquisition{Adapter: "http", Version: "1", SourceLocator: locator, RetrievedAt: time.Now().UTC().Format(time.RFC3339Nano), DeclaredMIME: response.Header.Get("Content-Type"), PublisherMetadata: map[string]string{"last_modified": response.Header.Get("Last-Modified"), "etag": response.Header.Get("ETag")}}, nil
}
