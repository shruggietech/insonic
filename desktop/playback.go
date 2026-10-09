// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

type playbackDescriptor struct {
	StreamIndex           *int    `json:"stream_index,omitempty"`
	PlaybackID            string  `json:"playback_id"`
	MediaID               string  `json:"media_id"`
	Revision              int64   `json:"revision"`
	RecordingRevision     int64   `json:"recording_revision"`
	DocumentDigest        string  `json:"document_digest"`
	Path                  string  `json:"path"`
	Digest                string  `json:"digest"`
	SizeBytes             int64   `json:"size_bytes"`
	MIMEType              string  `json:"mime_type"`
	Preview               bool    `json:"preview"`
	SourceDigest          string  `json:"source_digest"`
	SourceSizeBytes       int64   `json:"source_size_bytes"`
	TimelineOffsetSeconds float64 `json:"timeline_offset_seconds"`
}

type playbackTicket struct {
	Workspace  *workspace.Workspace
	Descriptor playbackDescriptor
	Expires    time.Time
	Context    context.Context
	Cancel     context.CancelFunc
}

func (b *Bridge) callPlayback(w *workspace.Workspace, operation, media string, data any) contracts.Response {
	raw, _ := json.Marshal(data)
	req := contracts.Request{WorkspaceID: w.Config.WorkspaceID, Operation: operation, ItemID: media, Data: raw}
	b.mu.RLock()
	cli, fixture := b.CLIExecutable, b.playbackCall
	b.mu.RUnlock()
	if fixture != nil {
		return fixture(w, req)
	}
	return (&Bridge{Workspace: w, CLIExecutable: cli}).operate(req)
}

func decodePlayback(response contracts.Response) (playbackDescriptor, bool) {
	var result playbackDescriptor
	raw, err := json.Marshal(response.Result)
	if response.Error != nil || err != nil || json.Unmarshal(raw, &result) != nil {
		return result, false
	}
	typeName, _, err := mime.ParseMediaType(result.MIMEType)
	valid := contracts.ValidID(result.PlaybackID) && contracts.ValidID(result.MediaID) &&
		filepath.IsAbs(result.Path) && len(result.Digest) == 64 && result.SizeBytes > 0 &&
		err == nil && (strings.HasPrefix(typeName, "audio/") || strings.HasPrefix(typeName, "video/"))
	return result, valid
}

// Playback mediates a shared runtime handle into a client-scoped native URL.
// Paths and cache leases remain native-only and cannot be selected by the browser.
func (b *Bridge) Playback(mediaID string, revision, recordingRevision int64, documentDigest string) contracts.Response {
	return b.playback(mediaID, revision, recordingRevision, documentDigest, nil)
}
func (b *Bridge) PlaybackTrack(mediaID string, revision, recordingRevision int64, documentDigest string, streamIndex int) contracts.Response {
	if streamIndex < 0 {
		return b.failed(contracts.Fail("invalid_request"))
	}
	return b.playback(mediaID, revision, recordingRevision, documentDigest, &streamIndex)
}
func (b *Bridge) playback(mediaID string, revision, recordingRevision int64, documentDigest string, streamIndex *int) contracts.Response {
	w := SelectedWorkspace(b)
	selected := &Bridge{Workspace: w}
	if w == nil {
		return selected.failed(contracts.Fail("not_found"))
	}
	if !contracts.ValidID(mediaID) || revision < 1 || recordingRevision < 0 || len(documentDigest) != 0 && len(documentDigest) != 64 {
		return selected.failed(contracts.Fail("invalid_request"))
	}
	data := map[string]any{"revision": revision}
	if streamIndex != nil {
		data["stream_index"] = *streamIndex
	}
	if recordingRevision > 0 {
		data["recording_revision"] = recordingRevision
	}
	if documentDigest != "" {
		data["document_digest"] = documentDigest
	}
	response := b.callPlayback(w, "media.playback", mediaID, data)
	if response.Error != nil {
		return response
	}
	descriptor, ok := decodePlayback(response)
	if !ok || streamIndex != nil && (descriptor.StreamIndex == nil || *descriptor.StreamIndex != *streamIndex) || descriptor.MediaID != mediaID || descriptor.Revision != revision ||
		recordingRevision != 0 && descriptor.RecordingRevision != recordingRevision ||
		documentDigest != "" && descriptor.DocumentDigest != documentDigest {
		if contracts.ValidID(descriptor.PlaybackID) {
			b.callPlayback(w, "media.playback-close", mediaID, map[string]any{"playback_id": descriptor.PlaybackID})
		}
		return selected.failed(contracts.Fail("unavailable"))
	}
	ctx, cancel := context.WithCancel(context.Background())
	ticket := playbackTicket{Workspace: w, Descriptor: descriptor, Expires: time.Now().Add(2 * time.Hour), Context: ctx, Cancel: cancel}
	id := contracts.ID()
	b.mu.Lock()
	if b.Workspace != w {
		b.mu.Unlock()
		b.closeTicket(ticket)
		return selected.failed(contracts.Fail("workspace_mismatch"))
	}
	if b.tickets == nil {
		b.tickets = make(map[string]playbackTicket)
	}
	expired := []playbackTicket{}
	for key, old := range b.tickets {
		if time.Now().After(old.Expires) {
			expired = append(expired, old)
			delete(b.tickets, key)
		}
	}
	if len(b.tickets) >= 64 {
		b.mu.Unlock()
		b.closeTicket(ticket)
		for _, old := range expired {
			b.closeTicket(old)
		}
		return selected.failed(contracts.Fail("input_limit"))
	}
	b.tickets[id] = ticket
	origin := b.playbackOrigin
	b.mu.Unlock()
	for _, old := range expired {
		b.closeTicket(old)
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID,
		Result: map[string]any{"url": origin + "/media/" + id, "media_id": mediaID, "revision": descriptor.Revision,
			"recording_revision": descriptor.RecordingRevision, "document_digest": descriptor.DocumentDigest,
			"digest": descriptor.Digest, "size_bytes": descriptor.SizeBytes, "mime_type": descriptor.MIMEType,
			"source_digest": descriptor.SourceDigest, "source_size_bytes": descriptor.SourceSizeBytes, "timeline_offset_seconds": descriptor.TimelineOffsetSeconds,
			"preview": descriptor.Preview, "expires_at": ticket.Expires.UTC().Format(time.RFC3339)}}
}

func (b *Bridge) closeTicket(ticket playbackTicket) {
	if ticket.Cancel != nil {
		ticket.Cancel()
	}
	if ticket.Workspace != nil {
		b.callPlayback(ticket.Workspace, "media.playback-close", ticket.Descriptor.MediaID,
			map[string]any{"playback_id": ticket.Descriptor.PlaybackID})
	}
}

func ticketID(url string) string {
	if !strings.HasPrefix(url, "/media/") {
		return ""
	}
	id := strings.TrimPrefix(url, "/media/")
	if !contracts.ValidID(id) {
		return ""
	}
	return id
}

// Only this process's transport origin is accepted by native close/check calls.
func (b *Bridge) playbackID(address string) string {
	if strings.HasPrefix(address, "/media/") {
		return ticketID(address)
	}
	parsed, err := url.Parse(address)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
		return ""
	}
	b.mu.RLock()
	origin := b.playbackOrigin
	b.mu.RUnlock()
	if origin == "" || parsed.Scheme+"://"+parsed.Host != origin {
		return ""
	}
	return ticketID(parsed.Path)
}

func (b *Bridge) ClosePlayback(url string) contracts.Response {
	id := b.playbackID(url)
	if id == "" {
		return (&Bridge{}).failed(contracts.Fail("invalid_request"))
	}
	b.mu.Lock()
	ticket, exists := b.tickets[id]
	delete(b.tickets, id)
	b.mu.Unlock()
	if exists {
		b.closeTicket(ticket)
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: map[string]any{"closed": true}}
}

// ClosePlaybackHandles releases ephemeral viewing resources without cancelling jobs.
func ClosePlaybackHandles(b *Bridge) {
	b.mu.Lock()
	tickets := b.tickets
	b.tickets = make(map[string]playbackTicket)
	b.mu.Unlock()
	for _, ticket := range tickets {
		b.closeTicket(ticket)
	}
}

type PlaybackServer struct{ Bridge *Bridge }

// VerifyPlayback lets a mounted media element revoke already buffered bytes
// after another client replaces current evidence. Runtime authority stays shared.
func (b *Bridge) VerifyPlayback(url string) contracts.Response {
	id := b.playbackID(url)
	if id == "" {
		return (&Bridge{}).failed(contracts.Fail("invalid_request"))
	}
	b.mu.RLock()
	ticket, exists := b.tickets[id]
	current := b.Workspace
	b.mu.RUnlock()
	selected := &Bridge{Workspace: ticket.Workspace}
	if !exists || current != ticket.Workspace || time.Now().After(ticket.Expires) {
		b.ClosePlayback(url)
		return selected.failed(contracts.Fail("not_found"))
	}
	response := b.callPlayback(ticket.Workspace, "media.playback-check", ticket.Descriptor.MediaID,
		map[string]any{"playback_id": ticket.Descriptor.PlaybackID})
	verified, ok := decodePlayback(response)
	b.mu.RLock()
	active, live := b.tickets[id]
	sameWorkspace := b.Workspace == ticket.Workspace
	b.mu.RUnlock()
	if !ok || verified != ticket.Descriptor || !live || !sameWorkspace || active.Descriptor != ticket.Descriptor {
		b.ClosePlayback(url)
		return selected.failed(contracts.Fail("conflict"))
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: ticket.Workspace.Config.WorkspaceID,
		Result: map[string]bool{"valid": true}}
}

func (s PlaybackServer) ServeHTTP(out http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		out.Header().Set("Allow", "GET, HEAD")
		out.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := ticketID(request.URL.Path)
	if id == "" || request.URL.RawQuery != "" || s.Bridge == nil {
		http.NotFound(out, request)
		return
	}
	b := s.Bridge
	b.mu.RLock()
	ticket, exists := b.tickets[id]
	current := b.Workspace
	b.mu.RUnlock()
	if !exists || current != ticket.Workspace || time.Now().After(ticket.Expires) {
		out.WriteHeader(http.StatusGone)
		return
	}
	check := func() bool {
		return b.VerifyPlayback(request.URL.Path).Error == nil
	}
	if !check() {
		b.ClosePlayback(request.URL.Path)
		out.WriteHeader(http.StatusGone)
		return
	}
	file, err := os.Open(ticket.Descriptor.Path)
	if err != nil {
		out.WriteHeader(http.StatusGone)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != ticket.Descriptor.SizeBytes {
		out.WriteHeader(http.StatusGone)
		return
	}
	ctx, cancel := context.WithCancel(request.Context())
	defer cancel()
	// Long media range responses keep the shared cache lease alive. A relevant
	// correction closes this reader rather than continuing superseded evidence.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticket.Context.Done():
				file.Close()
				return
			case <-ticker.C:
				if !check() {
					file.Close()
					return
				}
			}
		}
	}()
	defer func() { cancel(); <-done }()
	out.Header().Set("Content-Type", ticket.Descriptor.MIMEType)
	out.Header().Set("Cache-Control", "no-store")
	out.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(out, request, "media", time.Time{}, file)
}
