// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// StartPlaybackTransport serves only existing opaque media tickets. WebKit's
// native media engine cannot reliably consume Wails custom-scheme streams.
// A loopback HTTP stream preserves bounded range reads without loading a whole
// recording into a browser Blob or exposing workspace paths.
func StartPlaybackTransport(b *Bridge) (func(), error) {
	b.mu.Lock()
	if b.playbackOrigin != "" {
		b.mu.Unlock()
		return nil, errors.New("playback transport already started")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		b.mu.Unlock()
		return nil, err
	}
	authority := listener.Addr().String()
	b.playbackOrigin = "http://" + authority
	b.mu.Unlock()
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       15 * time.Second,
		MaxHeaderBytes:    8192,
		Handler: http.HandlerFunc(func(out http.ResponseWriter, request *http.Request) {
			peer, _, err := net.SplitHostPort(request.RemoteAddr)
			if err != nil || !net.ParseIP(peer).IsLoopback() || request.Host != authority {
				out.WriteHeader(http.StatusForbidden)
				return
			}
			// Media elements need no CORS permission. Reject explicit foreign
			// origins; no general-purpose browser-readable endpoint is exposed.
			origin := request.Header.Get("Origin")
			if origin != "" && origin != "null" && origin != "wails://wails" && origin != "http://wails.localhost" && origin != "https://wails.localhost" {
				out.WriteHeader(http.StatusForbidden)
				return
			}
			PlaybackServer{Bridge: b}.ServeHTTP(out, request)
		}),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.Serve(listener)
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			ClosePlaybackHandles(b)
			server.Close()
			<-done
		})
	}, nil
}
