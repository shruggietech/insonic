// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/library"
)

func TestPlaybackMIMEUsesCanonicalBytesBeforeSourceObservations(t *testing.T) {
	for _, tc := range []struct{ format, want string }{{"flac", "audio/flac"}, {"mp3", "audio/mpeg"}, {"mka", "audio/x-matroska"}, {"unknown", "application/octet-stream"}} {
		t.Run(tc.format, func(t *testing.T) {
			facts, _ := json.Marshal(library.Facts{Canonical: &library.Canonical{Format: tc.format}, Extension: ".wav"})
			for _, metadata := range []json.RawMessage{json.RawMessage(`{"mime_resolution":{"preferred_mime_type":"audio/x-wav"}}`), json.RawMessage(`{"mime_resolution":{"preferred_mime_type":"video/mp4"}}`), nil, json.RawMessage(`{`)} {
				entry := catalog.LibraryEntry{Facts: facts, Metadata: metadata}
				if got := playbackMIME(entry); got != tc.want {
					t.Fatalf("format %s: got %s, want %s", tc.format, got, tc.want)
				}
				if string(entry.Metadata) != string(metadata) {
					t.Fatal("source observations mutated")
				}
			}
		})
	}
}
