// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"io"
	"net"
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
	Acquire(context.Context, string, string, string, Options) (Acquisition, error)
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

const DefaultAcquisitionMaxBytes int64 = 64 << 30
const DefaultAcquisitionTimeoutMS int64 = 10 * 60 * 1000

func validateAcquisitionTransport(source, credential string, options Options) error {
	u, e := contracts.SourceURL(source)
	if e != nil {
		return e
	}
	if credential == "" || u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	loopback := strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()
	if u.Scheme == "http" && options.LocalHTTP != nil && *options.LocalHTTP && loopback {
		return nil
	}
	return contracts.Fail("invalid_request")
}
func (a httpAcquisition) Acquire(ctx context.Context, source, credential, destination string, options Options) (Acquisition, error) {
	if !validOptions(options) {
		return Acquisition{}, contracts.Fail("invalid_request")
	}
	if e := validateAcquisitionTransport(source, credential, options); e != nil {
		return Acquisition{}, e
	}
	maxBytes := DefaultAcquisitionMaxBytes
	if options.AcquisitionMaxBytes != nil {
		maxBytes = *options.AcquisitionMaxBytes
	}
	timeoutMS := DefaultAcquisitionTimeoutMS
	if options.AcquisitionTimeoutMS != nil {
		timeoutMS = *options.AcquisitionTimeoutMS
	}
	ctx, stop := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer stop()
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
		defer func() {
			for i := range value {
				value[i] = 0
			}
		}()
		if len(value) == 0 || strings.ContainsAny(string(value), "\r\n") {
			return Acquisition{}, contracts.Fail("invalid_request")
		}
		req.Header.Set("Authorization", "Bearer "+string(value))
		for i := range value {
			value[i] = 0
		}
	}
	client := http.Client{}
	if a.Client != nil {
		client = *a.Client
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if client.Timeout == 0 || client.Timeout > timeout {
		client.Timeout = timeout
	}
	previousRedirect := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, previous []*http.Request) error {
		if _, e := contracts.SourceURL(next.URL.String()); e != nil {
			return e
		}
		if len(previous) > 10 {
			return contracts.Fail("operation_failed")
		}
		if next.URL.Host != previous[0].URL.Host {
			next.Header.Del("Authorization")
		}
		if next.Header.Get("Authorization") != "" {
			if e := validateAcquisitionTransport(next.URL.String(), credential, options); e != nil {
				return e
			}
		}
		if previous[0].URL.Scheme == "https" && next.URL.Scheme != "https" {
			return contracts.Fail("operation_failed")
		}
		if next.URL.Scheme != "http" && next.URL.Scheme != "https" {
			return contracts.Fail("operation_failed")
		}
		if previousRedirect != nil {
			return previousRedirect(next, previous)
		}
		return nil
	}
	response, e := client.Do(req)
	if e != nil {
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	if response.ContentLength > maxBytes {
		return Acquisition{}, contracts.Fail("output_limit")
	}
	f, e := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return Acquisition{}, contracts.Fail("unavailable")
	}
	n, e := io.Copy(f, io.LimitReader(response.Body, maxBytes))
	over := false
	if e == nil && n == maxBytes {
		var extra [1]byte
		count, readErr := io.ReadFull(response.Body, extra[:])
		over = count > 0
		if readErr != nil && readErr != io.EOF {
			e = readErr
		}
	}
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil || over || (response.ContentLength >= 0 && n != response.ContentLength) {
		os.Remove(destination)
		if over {
			return Acquisition{}, contracts.Fail("output_limit")
		}
		return Acquisition{}, contracts.Fail("operation_failed")
	}
	return Acquisition{Adapter: "http", Version: "1", SourceLocator: locator, RetrievedAt: time.Now().UTC().Format(time.RFC3339Nano), DeclaredMIME: response.Header.Get("Content-Type"), PublisherMetadata: map[string]string{"last_modified": response.Header.Get("Last-Modified"), "etag": response.Header.Get("ETag")}}, nil
}
