// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// ObserveExitPlaybackQualification is deliberately unbound. The fixture owns a
// real ticket and checks the production shutdown path after Wails exits.
func ObserveExitPlaybackQualification(b *Bridge, media string, revision int64) (string, func() error, error) {
	response := b.Playback(media, revision, 0, "")
	if response.Error != nil {
		return "", nil, response.Error
	}
	result, ok := response.Result.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("playback descriptor missing")
	}
	address, _ := result["url"].(string)
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "http" {
		return "", nil, fmt.Errorf("playback URL unavailable")
	}
	id := ticketID(parsed.Path)
	b.mu.RLock()
	ticket, found := b.tickets[id]
	b.mu.RUnlock()
	if !found || ticket.Context.Err() != nil {
		return "", nil, fmt.Errorf("playback ticket unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", nil, err
	}
	request.Header.Set("Range", "bytes=0-15")
	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	served, err := client.Do(request)
	if err != nil {
		return "", nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(served.Body, 17))
	served.Body.Close()
	if served.StatusCode != http.StatusPartialContent || readErr != nil || len(data) != 16 {
		return "", nil, fmt.Errorf("playback range was not served")
	}
	return ticket.Descriptor.PlaybackID, func() error {
		b.mu.RLock()
		_, retained := b.tickets[id]
		b.mu.RUnlock()
		if retained || ticket.Context.Err() != context.Canceled {
			return fmt.Errorf("playback ticket survived GUI shutdown")
		}
		connection, err := net.DialTimeout("tcp", parsed.Host, 200*time.Millisecond)
		if err == nil {
			connection.Close()
			return fmt.Errorf("playback listener survived GUI shutdown")
		}
		return nil
	}, nil
}
