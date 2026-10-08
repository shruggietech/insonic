// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLoopbackPlaybackStreamsRangesAndRejectsForeignAuthority(t *testing.T) {
	b, stale, path := playbackFixture(t)
	stop, err := StartPlaybackTransport(b)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	address := b.playbackOrigin + path
	client := &http.Client{Timeout: 5 * time.Second}
	request, _ := http.NewRequest(http.MethodGet, address, nil)
	request.Header.Set("Range", "bytes=2-4")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusPartialContent || string(data) != "234" || response.Header.Get("Content-Range") != "bytes 2-4/10" || response.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("native loopback ranged stream failed", response.StatusCode)
	}
	if b.VerifyPlayback(address).Error != nil || b.ClosePlayback("https://foreign.invalid"+path).Error == nil || b.VerifyPlayback(address).Error != nil {
		t.Fatal("transport identity was not bound to this process")
	}
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Host = "foreign.invalid" },
		func(r *http.Request) { r.Header.Set("Origin", "https://foreign.invalid") },
	} {
		request, _ = http.NewRequest(http.MethodGet, address, nil)
		mutate(request)
		response, err = client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		data, _ = io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden || len(data) != 0 {
			t.Fatal("foreign browser authority admitted")
		}
	}
	for _, suffix := range []string{"?path=private", "/extra"} {
		response, err = client.Get(address + suffix)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatal("unexpected media route accepted")
		}
	}
	stale.Store(true)
	response, err = client.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusGone {
		t.Fatal("stale native stream remained accessible")
	}
	stop()
	stop()
	if _, err = client.Get(address); err == nil {
		t.Fatal("native listener survived shutdown")
	}
}

func TestLoopbackPlaybackReturnsItsTransportURL(t *testing.T) {
	b, _, path := playbackFixture(t)
	stop, err := StartPlaybackTransport(b)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	descriptor := b.tickets[ticketID(path)].Descriptor
	response := b.Playback(descriptor.MediaID, descriptor.Revision, 0, "")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	address := response.Result.(map[string]any)["url"].(string)
	if !strings.HasPrefix(address, b.playbackOrigin+"/media/") || b.VerifyPlayback(address).Error != nil {
		t.Fatal("native media URL did not use scoped loopback transport")
	}
	if _, err = StartPlaybackTransport(b); err == nil {
		t.Fatal("second native transport started")
	}
}
