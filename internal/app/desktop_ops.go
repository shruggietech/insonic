// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

const playbackIdle = 5 * time.Minute
const maxPlaybackBytes int64 = 10 << 30

// PlaybackDescriptor is exchanged only with native clients. The desktop wraps
// its path in an opaque ticket and never supplies it to browser JavaScript.
type PlaybackDescriptor struct {
	PlaybackID            string  `json:"playback_id"`
	MediaID               string  `json:"media_id"`
	Revision              int64   `json:"revision"`
	RecordingRevision     int64   `json:"recording_revision"`
	DocumentDigest        string  `json:"document_digest"`
	Path                  string  `json:"path"`
	Digest                string  `json:"digest"`
	Size                  int64   `json:"size_bytes"`
	MIME                  string  `json:"mime_type"`
	SourceDigest          string  `json:"source_digest"`
	SourceSize            int64   `json:"source_size_bytes"`
	Preview               bool    `json:"preview"`
	TimelineOffsetSeconds float64 `json:"timeline_offset_seconds"`
	PublicationID         string  `json:"publication_id,omitempty"`
	LeaseID               string  `json:"lease_id,omitempty"`
}
type playbackEntry struct {
	descriptor       PlaybackDescriptor
	touched          time.Time
	sourceLocator    string
	materializedPath string
	proxyPath        string
	snapshot         bool
}

func (a *App) desktopDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "settings.show":
		return a.settingsShow()
	case "settings.set":
		return a.settingsSet(req)
	case "recordings.cues":
		return a.recordingCues(req)
	case "media.capture":
		return a.mediaCapture(req)
	case "media.playback":
		return a.openPlayback(req)
	case "media.playback-check", "media.playback-close":
		var input struct {
			ID string `json:"playback_id"`
		}
		if strictPayload(req.Data, &input) != nil || !contracts.ValidID(input.ID) {
			return nil, contracts.Fail("invalid_request")
		}
		a.playbackMu.Lock()
		defer a.playbackMu.Unlock()
		p := a.playbacks[input.ID]
		if p == nil || p.descriptor.MediaID != req.ItemID {
			return nil, contracts.Fail("not_found")
		}
		if req.Operation == "media.playback-close" {
			a.releasePlayback(a.ctx, input.ID, p)
			return map[string]bool{"closed": true}, nil
		}
		if time.Since(p.touched) >= playbackIdle {
			a.releasePlayback(a.ctx, input.ID, p)
			return nil, contracts.Fail("not_found")
		}
		if e := a.checkPlayback(a.ctx, p); e != nil {
			a.releasePlayback(a.ctx, input.ID, p)
			return nil, e
		}
		p.touched = time.Now()
		return p.descriptor, nil
	}
	return nil, contracts.Fail("invalid_request")
}

func (a *App) openPlayback(req contracts.Request) (any, error) {
	var input struct {
		Revision          int64  `json:"revision"`
		RecordingRevision int64  `json:"recording_revision,omitempty"`
		DocumentDigest    string `json:"document_digest,omitempty"`
		MaxBytes          int64  `json:"max_bytes,omitempty"`
	}
	if strictPayload(req.Data, &input) != nil || input.Revision < 1 || input.RecordingRevision < 0 || (input.DocumentDigest != "" && !hashString.MatchString(input.DocumentDigest)) || (input.DocumentDigest != "" && input.RecordingRevision == 0) || input.MaxBytes < 0 || input.MaxBytes > maxPlaybackBytes {
		return nil, contracts.Fail("invalid_request")
	}
	if input.MaxBytes == 0 {
		input.MaxBytes = maxPlaybackBytes
	}
	a.playbackMu.Lock()
	if a.ctx.Err() != nil {
		a.playbackMu.Unlock()
		return nil, contracts.Fail("unavailable")
	}
	if len(a.playbacks)+a.playbackPreparing >= 64 {
		a.playbackMu.Unlock()
		return nil, contracts.Fail("output_limit")
	}
	a.playbackPreparing++
	a.playbackMu.Unlock()
	defer func() { a.playbackMu.Lock(); a.playbackPreparing--; a.playbackMu.Unlock() }()
	ctx, stop := context.WithTimeout(a.ctx, 10*time.Minute)
	defer stop()
	entry, e := a.Catalog.Library(ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if entry.Revision != input.Revision {
		return nil, contracts.Fail("conflict")
	}
	if entry.Size > input.MaxBytes {
		return nil, contracts.Fail("output_limit")
	}
	descriptor := PlaybackDescriptor{PlaybackID: contracts.ID(), MediaID: entry.ID, Revision: entry.Revision, Digest: entry.Digest, Size: entry.Size, SourceDigest: entry.Digest, SourceSize: entry.Size, MIME: playbackMIME(entry)}
	if input.RecordingRevision > 0 {
		record, e := a.Catalog.Recording(ctx, entry.ID)
		if e != nil {
			return nil, e
		}
		if record.Revision != input.RecordingRevision || record.SourceDigest != entry.Digest || (input.DocumentDigest != "" && input.DocumentDigest != record.DocumentDigest) {
			return nil, contracts.Fail("conflict")
		}
		descriptor.RecordingRevision = record.Revision
		descriptor.DocumentDigest = record.DocumentDigest
	}
	service, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	p := &playbackEntry{descriptor: descriptor, touched: time.Now(), sourceLocator: entry.SourceLocator}
	if entry.Mode == "copy" {
		if entry.OriginalPublicationID == nil {
			return nil, contracts.Fail("not_found")
		}
		publication, e := a.Catalog.Publication(ctx, *entry.OriginalPublicationID)
		if e != nil {
			return nil, e
		}
		if publication.Digest != entry.Digest || publication.Size != entry.Size || publication.State != "available" {
			return nil, contracts.Fail("conflict")
		}
		materialized, e := service.MaterializeBound(ctx, publication.ID, input.MaxBytes)
		if e != nil {
			return nil, e
		}
		p.descriptor.Path = materialized.Path
		p.descriptor.PublicationID = publication.ID
		p.descriptor.LeaseID = materialized.Lease.ID
	} else if entry.Mode == "reference" {
		staged, digest, size, e := service.StageBound(ctx, entry.SourceLocator, input.MaxBytes)
		if e != nil {
			return nil, e
		}
		path := staged.Name()
		staged.Close()
		if digest != entry.Digest || size != entry.Size {
			os.Remove(path)
			return nil, contracts.Fail("conflict")
		}
		// A distinctive disposable name enables interrupted-session cleanup without
		// touching artifact publication sources or other scratch users.
		target := filepath.Join(filepath.Dir(path), "playback-"+a.Session+"-"+descriptor.PlaybackID)
		if e = os.Rename(path, target); e != nil {
			os.Remove(path)
			return nil, contracts.Fail("unavailable")
		}
		p.descriptor.Path = target
		p.snapshot = true
	} else {
		return nil, contracts.Fail("invalid_request")
	}
	p.materializedPath = p.descriptor.Path
	success := false
	defer func() {
		if !success {
			cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			a.playbackMu.Lock()
			a.releasePlayback(cleanup, p.descriptor.PlaybackID, p)
			a.playbackMu.Unlock()
		}
	}()
	previewCtx, finish := keepPlaybackPreparationLease(ctx, service, p)
	defer finish()
	preview := a.playbackPreview
	if preview == nil {
		preview = a.previewPlayback
	}
	if e = preview(previewCtx, p, entry); e != nil {
		return nil, e
	}
	if e = finish(); e != nil {
		return nil, e
	}
	if e = a.checkPlayback(ctx, p); e != nil {
		return nil, e
	}
	a.playbackMu.Lock()
	if a.ctx.Err() != nil {
		a.playbackMu.Unlock()
		return nil, contracts.Fail("unavailable")
	}
	p.touched = time.Now()
	a.playbacks[p.descriptor.PlaybackID] = p
	a.playbackMu.Unlock()
	success = true
	return p.descriptor, nil
}

// Preview preparation can exceed a cache lease. Renew it independently while
// retaining neither the registry mutex nor the runtime's request admission lock.
func keepPlaybackPreparationLease(ctx context.Context, service *artifact.Service, p *playbackEntry) (context.Context, func() error) {
	if p.descriptor.PublicationID == "" {
		return ctx, func() error { return nil }
	}
	work, cancel := context.WithCancel(ctx)
	done, joined := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	id, lease := p.descriptor.PublicationID, p.descriptor.LeaseID
	interval := service.TTL / 3
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	go func() {
		defer close(joined)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				if _, e := service.Renew(work, id, lease); e != nil {
					result <- e
					cancel()
					return
				}
			}
		}
	}()
	var once sync.Once
	finish := func() error {
		once.Do(func() { close(done); <-joined; cancel() })
		select {
		case e := <-result:
			result <- e
			return e
		default:
			return nil
		}
	}
	return work, finish
}

func playbackMIME(entry catalog.LibraryEntry) string {
	var metadata library.Metadata
	if json.Unmarshal(entry.Metadata, &metadata) == nil {
		mime := metadata.MIME.Preferred
		if (strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/")) && len(mime) <= 128 && !strings.ContainsAny(mime, "\r\n\x00") {
			return mime
		}
	}
	var facts library.Facts
	json.Unmarshal(entry.Facts, &facts)
	switch strings.ToLower(facts.Extension) {
	case ".wav", "wav":
		return "audio/wav"
	case ".mp3", "mp3":
		return "audio/mpeg"
	case ".m4a", "m4a":
		return "audio/mp4"
	case ".flac", "flac":
		return "audio/flac"
	case ".ogg", "ogg", ".oga", "oga":
		return "audio/ogg"
	case ".mp4", "mp4", ".m4v", "m4v":
		return "video/mp4"
	case ".webm", "webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func (a *App) checkPlayback(ctx context.Context, p *playbackEntry) error {
	d := p.descriptor
	entry, e := a.Catalog.Library(ctx, d.MediaID)
	if e != nil {
		return e
	}
	if entry.Revision != d.Revision || entry.Digest != d.SourceDigest || entry.Size != d.SourceSize || entry.SourceLocator != p.sourceLocator {
		return contracts.Fail("conflict")
	}
	if d.RecordingRevision > 0 {
		record, e := a.Catalog.Recording(ctx, d.MediaID)
		if e != nil {
			return e
		}
		if record.Revision != d.RecordingRevision || record.DocumentDigest != d.DocumentDigest || record.SourceDigest != d.SourceDigest {
			return contracts.Fail("conflict")
		}
	}
	info, e := os.Lstat(d.Path)
	if e != nil || !info.Mode().IsRegular() || info.Size() != d.Size {
		return contracts.Fail("unavailable")
	}
	if d.PublicationID != "" {
		if entry.OriginalPublicationID == nil || *entry.OriginalPublicationID != d.PublicationID {
			return contracts.Fail("conflict")
		}
		s, e := a.artifactService()
		if e != nil {
			return e
		}
		if _, e = s.Renew(ctx, d.PublicationID, d.LeaseID); e != nil {
			return e
		}
	}
	return nil
}

func (a *App) releasePlayback(ctx context.Context, id string, p *playbackEntry) {
	delete(a.playbacks, id)
	if p.proxyPath != "" {
		os.Remove(p.proxyPath)
	}
	if p.snapshot {
		os.Remove(p.materializedPath)
		return
	}
	if p.descriptor.PublicationID != "" {
		if s, e := a.artifactService(); e == nil {
			s.Release(ctx, p.descriptor.PublicationID, p.descriptor.LeaseID)
		}
	}
}
func (a *App) maintainPlaybacks() {
	defer a.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case now := <-ticker.C:
			a.playbackMu.Lock()
			ctx, stop := context.WithTimeout(a.ctx, 8*time.Second)
			for id, p := range a.playbacks {
				if now.Sub(p.touched) >= playbackIdle || a.checkPlayback(ctx, p) != nil {
					a.releasePlayback(ctx, id, p)
				}
			}
			a.playbackMu.Unlock()
			stop()
		}
	}
}
func (a *App) closePlaybacks(ctx context.Context) {
	a.playbackMu.Lock()
	defer a.playbackMu.Unlock()
	for id, p := range a.playbacks {
		a.releasePlayback(ctx, id, p)
	}
}

// RecoverPlaybackScratch runs only while Serve holds the exclusive runtime
// owner lock. Interrupted playback snapshots are disposable; publication and
// materialization files use different names and are not touched.
func RecoverPlaybackScratch(w *workspace.Workspace) error {
	directory := filepath.Join(w.Control, "artifact-scratch")
	if _, e := os.Lstat(directory); os.IsNotExist(e) {
		return nil
	}
	if e := workspace.SecureDirectory(directory, false); e != nil {
		return e
	}
	root, e := os.OpenRoot(directory)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer root.Close()
	file, e := root.Open(".")
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer file.Close()
	for {
		entries, e := file.ReadDir(100)
		if e != nil && e != io.EOF {
			return contracts.Fail("unavailable")
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "playback-") && !entry.IsDir() {
				if e := root.Remove(entry.Name()); e != nil && !os.IsNotExist(e) {
					return contracts.Fail("unavailable")
				}
			}
		}
		if e == io.EOF {
			break
		}
	}
	return nil
}

var hashString = regexp.MustCompile(`^[a-f0-9]{64}$`)

type appearanceSettings struct {
	Theme         string `json:"theme"`
	ReducedMotion bool   `json:"reduced_motion"`
}

var settingFiles = map[string]string{"media_tools": "media-tools.json", "processing_tools": "processing-tools.json", "appearance": "appearance.json"}

func settingRead(path string) ([]byte, error) {
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	if len(raw) > 1<<20 {
		return nil, contracts.Fail("output_limit")
	}
	return raw, nil
}
func settingRevision(raw []byte) string {
	if len(raw) == 0 {
		return documentHash([]byte("null"))
	}
	if catalog.ValidateJSON(raw) != nil {
		return documentHash(raw)
	}
	var v any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.Decode(&v)
	canonical, _ := json.Marshal(v)
	return documentHash(canonical)
}
func (a *App) settingSection(section string) (map[string]any, error) {
	raw, e := settingRead(filepath.Join(a.Workspace.Control, settingFiles[section]))
	if e != nil {
		return nil, e
	}
	origin := "workspace"
	var value any
	if len(raw) > 0 {
		value, e = validateSetting(section, raw)
		if e != nil {
			return nil, e
		}
	} else {
		origin = "unconfigured"
		switch section {
		case "media_tools":
			tools, e := ReadMediaTools(a.Workspace)
			if e != nil {
				return nil, e
			}
			if tools.FFprobe.Path != "" || tools.ExifTool.Path != "" {
				value = tools
				origin = "package"
			}
		case "processing_tools":
			tools, e := ReadProcessingTools(a.Workspace)
			if e == nil {
				value = tools
				origin = "package"
			} else if typed, ok := e.(*contracts.Error); !ok || typed.Code != "unavailable" {
				return nil, e
			}
		case "appearance":
			value = appearanceSettings{Theme: "system"}
		}
	}
	return map[string]any{"revision": settingRevision(raw), "value": value, "origin": origin}, nil
}
func validateSetting(section string, raw []byte) (any, error) {
	switch section {
	case "appearance":
		var value appearanceSettings
		if strictPayload(raw, &value) != nil || (value.Theme != "system" && value.Theme != "light" && value.Theme != "dark") {
			return nil, contracts.Fail("invalid_request")
		}
		return value, nil
	case "media_tools":
		if schemas.ValidateMediaTools(raw) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		var value library.Tools
		if strictPayload(raw, &value) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		for _, tool := range []library.Tool{value.ExifTool, value.FFprobe, value.FFmpeg} {
			if tool.Path != "" && (!filepath.IsAbs(tool.Path) || !hashString.MatchString(tool.SHA256)) {
				return nil, contracts.Fail("invalid_request")
			}
			for _, pin := range tool.SupportFiles {
				if !filepath.IsAbs(pin.Path) {
					return nil, contracts.Fail("invalid_request")
				}
			}
			if tool.Interpreter != nil && !filepath.IsAbs(tool.Interpreter.Path) {
				return nil, contracts.Fail("invalid_request")
			}
		}
		return value, nil
	case "processing_tools":
		var value ProcessingTools
		if schemas.ValidateDocument(raw) != nil || strictPayload(raw, &value) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if e := ValidateProcessingTools(value); e != nil {
			return nil, e
		}
		return value, nil
	}
	return nil, contracts.Fail("invalid_request")
}

// ValidateSetting lets convenience CLI routes reject invalid input before
// ensuring the runtime, while the runtime remains the mutation authority.
func ValidateSetting(section string, raw []byte) error {
	_, err := validateSetting(section, raw)
	return err
}
func (a *App) settingsShow() (any, error) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	// Serialize readback with credential selection. Reading the persisted mode
	// neither accesses values nor replaces the live unlocked manager.
	if a.secretOwner != nil {
		a.secretOwner.mu.RLock()
		defer a.secretOwner.mu.RUnlock()
	}
	backend, err := secrets.ReadSelection(a.Workspace)
	if err != nil {
		return nil, err
	}
	sections := map[string]any{}
	for _, section := range []string{"media_tools", "processing_tools", "appearance"} {
		value, e := a.settingSection(section)
		if e != nil {
			raw, readErr := settingRead(filepath.Join(a.Workspace.Control, settingFiles[section]))
			if readErr != nil {
				return nil, readErr
			}
			typed, ok := e.(*contracts.Error)
			if !ok {
				typed = contracts.Fail("operation_failed")
			}
			value = map[string]any{"revision": settingRevision(raw), "value": nil, "origin": "workspace", "error": typed}
		}
		sections[section] = value
	}
	return map[string]any{"sections": sections, "credentials": map[string]any{"backend": backend}, "profiles": a.Workspace.Config.Profiles, "profile_configuration": "Backend configuration is readable here. Switching an existing workspace backend requires migration; use the advanced catalog configuration workflow."}, nil
}
func (a *App) settingsSet(req contracts.Request) (any, error) {
	var input struct {
		Section  string          `json:"section"`
		Revision string          `json:"revision"`
		Value    json.RawMessage `json:"value"`
	}
	if strictPayload(req.Data, &input) != nil || !hashString.MatchString(input.Revision) || settingFiles[input.Section] == "" {
		return nil, contracts.Fail("invalid_request")
	}
	reset := bytes.Equal(bytes.TrimSpace(input.Value), []byte("null"))
	var value any
	var e error
	if !reset {
		value, e = validateSetting(input.Section, input.Value)
		if e != nil {
			return nil, e
		}
	}
	normalized, _ := json.Marshal(value)
	next := settingRevision(normalized)
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	path := filepath.Join(a.Workspace.Control, settingFiles[input.Section])
	current, e := settingRead(path)
	if e != nil {
		return nil, e
	}
	if settingRevision(current) != input.Revision && settingRevision(current) != next {
		return nil, contracts.Fail("conflict")
	}
	if reset {
		if e = os.Remove(path); e != nil && !os.IsNotExist(e) {
			return nil, contracts.Fail("unavailable")
		}
		// The atomic removal is the reset publication. Defaults remain elected
		// from the current installation; none of their absolute paths are saved.
		section, e := a.settingSection(input.Section)
		if e != nil {
			return nil, e
		}
		return map[string]any{"section": input.Section, "revision": next, "value": nil, "origin": section["origin"], "applies": "next-operation"}, nil
	}
	if settingRevision(current) != next {
		tmp, e := os.CreateTemp(a.Workspace.Control, ".settings-*")
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
		name := tmp.Name()
		defer os.Remove(name)
		if e = tmp.Chmod(0600); e == nil {
			_, e = tmp.Write(append(normalized, '\n'))
		}
		if e == nil {
			e = tmp.Sync()
		}
		closeErr := tmp.Close()
		if e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(name, path)
		}
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
	}
	return map[string]any{"section": input.Section, "revision": next, "value": value, "origin": "workspace", "applies": "next-operation"}, nil
}

type cueProjection struct {
	Ordinal         int      `json:"ordinal"`
	ID              string   `json:"id"`
	Text            string   `json:"text"`
	Timing          any      `json:"timing"`
	SeekSeconds     *float64 `json:"seek_seconds"`
	LocalSpeakerIDs []string `json:"local_speaker_ids"`
	Attributions    []any    `json:"speaker_attributions"`
}

// mediaCapture returns exact current capture JSON as byte pages. Browsers can
// display it as text without rounding source nanosecond integers through Number.
func (a *App) mediaCapture(req contracts.Request) (any, error) {
	var input struct {
		Section  string `json:"section"`
		Revision int64  `json:"revision,omitempty"`
		Offset   int    `json:"offset,omitempty"`
		Limit    int    `json:"limit,omitempty"`
		Digest   string `json:"sha256,omitempty"`
	}
	if strictPayload(req.Data, &input) != nil || input.Revision < 0 || input.Offset < 0 || input.Limit < 0 || input.Limit > 64<<10 || input.Digest != "" && !hashString.MatchString(input.Digest) {
		return nil, contracts.Fail("invalid_request")
	}
	if input.Limit == 0 {
		input.Limit = 64 << 10
	}
	entry, e := a.Catalog.Library(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if input.Revision > 0 && input.Revision != entry.Revision {
		return nil, contracts.Fail("conflict")
	}
	var raw json.RawMessage
	switch input.Section {
	case "metadata":
		raw = entry.Metadata
	case "facts":
		raw = entry.Facts
	case "dates":
		raw = entry.Dates
	default:
		return nil, contracts.Fail("invalid_request")
	}
	digest := documentHash(raw)
	if input.Digest != "" && input.Digest != digest {
		return nil, contracts.Fail("conflict")
	}
	if input.Offset > len(raw) {
		return nil, contracts.Fail("invalid_request")
	}
	end := min(input.Offset+input.Limit, len(raw))
	return map[string]any{"media_id": entry.ID, "revision": entry.Revision, "section": input.Section, "sha256": digest, "encoding": "base64", "data": base64.StdEncoding.EncodeToString(raw[input.Offset:end]), "offset": input.Offset, "next_offset": end, "complete": end == len(raw)}, nil
}

func seekSeconds(milliseconds int64) *float64 {
	if milliseconds < 0 || milliseconds > 9007199254740991 {
		return nil
	}
	seconds := float64(milliseconds) / 1000
	if math.IsInf(seconds, 0) || math.IsNaN(seconds) {
		return nil
	}
	return &seconds
}
func (a *App) recordingCues(req contracts.Request) (any, error) {
	var input struct {
		Revision int64  `json:"revision,omitempty"`
		Digest   string `json:"document_digest,omitempty"`
		After    int    `json:"after_ordinal,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}
	if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if input.Limit == 0 {
		input.Limit = 100
	}
	if input.Revision < 0 || (input.Digest != "" && !hashString.MatchString(input.Digest)) || input.After < 0 || input.Limit < 1 || input.Limit > 100 {
		return nil, contracts.Fail("invalid_request")
	}
	record, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if (input.Revision > 0 && input.Revision != record.Revision) || (input.Digest != "" && input.Digest != record.DocumentDigest) {
		return nil, contracts.Fail("conflict")
	}
	var doc struct {
		Cues []struct {
			ID     string `json:"id"`
			Timing *struct {
				Start int64 `json:"start_milliseconds"`
				End   int64 `json:"end_milliseconds"`
			} `json:"timing"`
			Payload struct {
				Text string `json:"plain_text"`
			} `json:"payload"`
			Attributions []struct {
				Speaker string `json:"speaker_id"`
				Start   *int64 `json:"start_milliseconds"`
				End     *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
		Speakers []struct {
			ID   string  `json:"id"`
			Name *string `json:"name"`
		} `json:"speakers"`
	}
	if record.State == "ready" {
		if subtitles.ValidateDocument(record.Document) != nil || json.Unmarshal(record.Document, &doc) != nil {
			return nil, contracts.Fail("operation_failed")
		}
	}
	if input.After > len(doc.Cues) {
		return nil, contracts.Fail("invalid_request")
	}
	cues := []cueProjection{}
	voices := map[string]string{}
	nextOrdinal := input.After
	bytesTotal := 0
	for nextOrdinal < len(doc.Cues) && len(cues) < input.Limit {
		source := doc.Cues[nextOrdinal]
		cue := cueProjection{Ordinal: nextOrdinal, ID: source.ID, Text: source.Payload.Text, LocalSpeakerIDs: []string{}, Attributions: []any{}}
		if source.Timing != nil {
			cue.Timing = map[string]string{"start_milliseconds": strconv.FormatInt(source.Timing.Start, 10), "end_milliseconds": strconv.FormatInt(source.Timing.End, 10)}
			cue.SeekSeconds = seekSeconds(source.Timing.Start)
		}
		seen := map[string]bool{}
		for _, attribution := range source.Attributions {
			if !seen[attribution.Speaker] {
				cue.LocalSpeakerIDs = append(cue.LocalSpeakerIDs, attribution.Speaker)
				seen[attribution.Speaker] = true
				voices[attribution.Speaker] = ""
			}
			item := map[string]any{"local_speaker_id": attribution.Speaker}
			if attribution.Start != nil && attribution.End != nil {
				item["start_milliseconds"] = strconv.FormatInt(*attribution.Start, 10)
				item["end_milliseconds"] = strconv.FormatInt(*attribution.End, 10)
				item["seek_seconds"] = seekSeconds(*attribution.Start)
			}
			cue.Attributions = append(cue.Attributions, item)
		}
		raw, _ := json.Marshal(cue)
		if bytesTotal+len(raw) > 384<<10 {
			if len(cues) == 0 {
				return nil, contracts.Fail("output_limit")
			}
			break
		}
		cues = append(cues, cue)
		bytesTotal += len(raw)
		nextOrdinal++
	}
	for _, speaker := range doc.Speakers {
		if _, ok := voices[speaker.ID]; ok && speaker.Name != nil {
			voices[speaker.ID] = *speaker.Name
		}
	}
	ids := []string{}
	for id := range voices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	voiceList := []any{}
	for _, id := range ids {
		voiceList = append(voiceList, map[string]string{"id": id, "name": voices[id]})
	}
	var next any
	if nextOrdinal < len(doc.Cues) {
		next = nextOrdinal
	}
	return map[string]any{"recording_id": record.ID, "revision": record.Revision, "document_digest": record.DocumentDigest, "state": record.State, "cues": cues, "voices": voiceList, "next_ordinal": next}, nil
}
