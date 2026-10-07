// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type Service struct {
	Artifacts           *artifact.Service
	Catalog             catalog.Catalog
	Secrets             contracts.SecretProvider
	Tools               Tools
	MaxOutput           int
	AcquisitionAdapters map[string]AcquisitionAdapter
}
type EntryView struct {
	catalog.LibraryEntry
	Availability string `json:"availability"`
}

func NewService(a *artifact.Service, db catalog.Catalog, secrets contracts.SecretProvider, tools Tools) *Service {
	return &Service{Artifacts: a, Catalog: db, Secrets: secrets, Tools: tools, MaxOutput: 16 << 20, AcquisitionAdapters: map[string]AcquisitionAdapter{"http": httpAcquisition{Secrets: secrets}, "https": httpAcquisition{Secrets: secrets}}}
}
func (s *Service) Execute(ctx context.Context, work catalog.Work) (any, error) {
	switch work.Kind {
	case "media.import", "library.import":
		var req ImportRequest
		if strict(work.Payload, &req) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return s.Import(ctx, work, req)
	case "media.refresh", "library.refresh":
		var req RefreshRequest
		if strict(work.Payload, &req) != nil || !contracts.ValidID(req.MediaID) || !validOptions(req.Options) {
			return nil, contracts.Fail("invalid_request")
		}
		return s.Refresh(ctx, work, req)
	}
	return nil, contracts.Fail("invalid_request")
}
func (s *Service) checkpoint(ctx context.Context, work catalog.Work, phase string, result any) error {
	_, e := s.Catalog.CheckpointWork(ctx, work, phase, "running", marshal(result), time.Minute)
	return e
}
func (s *Service) Import(ctx context.Context, work catalog.Work, req ImportRequest) (ImportResult, error) {
	var result ImportResult
	if strict(work.Result, &result) == nil && result.CapturedTimezone != "" {
		req.Defaults.Timezone = result.CapturedTimezone
	}
	req, e := PrepareImport(req)
	if e != nil {
		return result, e
	}
	result.CapturedTimezone = req.Defaults.Timezone
	result.Items = []ItemResult{}
	entries, e := s.Catalog.Libraries(ctx)
	if e != nil {
		return result, e
	}
	known := newLibraryLookup(entries)
	if e = s.checkpoint(ctx, work, "admission-start", result); e != nil {
		return result, e
	}
	for ordinal, item := range req.Items {
		if ctx.Err() != nil {
			return result, contracts.Fail("cancelled")
		}
		out, e := s.admit(ctx, work, ordinal, item, merged(req.Defaults, item.Options), known)
		if e != nil {
			if ctx.Err() != nil {
				return result, contracts.Fail("cancelled")
			}
			out = ItemResult{Ordinal: ordinal, State: "failed", Error: code(e), QueuedJobIDs: []string{}}
			result.Partial = true
		}
		if out.CaptureState != "captured" && out.CaptureState != "no-embedded-metadata" {
			result.Partial = true
		}
		if out.DateUnresolved || out.DateState == "ambiguous" || out.DateState == "nonexistent" || out.DateState == "invalid" {
			result.Partial = true
		}
		if out.SubtitleState == "missing" {
			result.Partial = true
		}
		result.Items = append(result.Items, out)
		if out.MediaID != "" {
			if entry, e := s.Catalog.Library(ctx, out.MediaID); e == nil {
				known.remember(entry)
			}
		}
		if e = s.checkpoint(ctx, work, "item-"+strconv.Itoa(ordinal), result); e != nil {
			return result, e
		}
	}
	return result, nil
}
func code(e error) string {
	if c, ok := e.(*contracts.Error); ok {
		return c.Code
	}
	return "operation_failed"
}

type libraryLookup struct {
	source, content map[string]catalog.LibraryEntry
}

func newLibraryLookup(entries []catalog.LibraryEntry) *libraryLookup {
	k := &libraryLookup{source: map[string]catalog.LibraryEntry{}, content: map[string]catalog.LibraryEntry{}}
	for _, entry := range entries {
		k.remember(entry)
	}
	return k
}
func contentKey(digest string, size int64, mode string) string {
	return digest + ":" + strconv.FormatInt(size, 10) + ":" + mode
}
func (k *libraryLookup) remember(entry catalog.LibraryEntry) {
	key := contentKey(entry.Digest, entry.Size, entry.Mode)
	k.source[key+"\x00"+entry.SourceLocator] = entry
	if _, ok := k.content[key]; !ok {
		k.content[key] = entry
	}
}
func (s *Service) admit(ctx context.Context, work catalog.Work, ordinal int, item Item, options Options, known *libraryLookup) (out ItemResult, returned error) {
	entryID := DerivedID(work.ID, "media-"+strconv.Itoa(ordinal))
	reportID, e := s.candidateAttempt(ctx, DerivedID(work.ID, "report-"+strconv.Itoa(ordinal)))
	if e != nil {
		return ItemResult{}, e
	}
	candidates := []string{reportID}
	defer func() {
		if returned != nil {
			for _, id := range candidates {
				s.discardUnusedCandidate(id)
			}
		}
	}()
	if entry, e := s.Catalog.Library(ctx, entryID); e == nil {
		if e = s.Cleanup(ctx, entry.ID); e != nil {
			return ItemResult{}, e
		}
		return resultOf(ordinal, entry), nil
	}
	directory, e := os.MkdirTemp(s.Artifacts.Workspace.Control, "library-work-")
	if e != nil {
		return ItemResult{}, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	original := item.Source
	locator := item.Source
	mode := "copy"
	if options.Copy != nil && !*options.Copy {
		mode = "reference"
	}
	var acquisition *Acquisition
	if sourceURL, e := url.Parse(item.Source); e == nil && sourceURL.Scheme != "" && len(sourceURL.Scheme) > 1 {
		if mode == "reference" {
			return ItemResult{}, contracts.Fail("invalid_request")
		}
		adapterName := item.AcquisitionAdapter
		if adapterName == "" {
			adapterName = sourceURL.Scheme
		}
		adapter := s.AcquisitionAdapters[adapterName]
		if adapter == nil {
			return ItemResult{}, contracts.Fail("unavailable")
		}
		original = filepath.Join(directory, "acquired"+filepath.Ext(sourceURL.Path))
		receipt, e := adapter.Acquire(ctx, item.Source, item.CredentialID, original, options)
		if e != nil {
			return ItemResult{}, e
		}
		locator = receipt.SourceLocator
		acquisition = &receipt
	} else {
		original, e = filepath.Abs(original)
		if e != nil {
			return ItemResult{}, contracts.Fail("invalid_request")
		}
		locator = original
	}
	stage, digest, size, e := s.Artifacts.Stage(ctx, original)
	if e != nil {
		return ItemResult{}, e
	}
	stagePath := stage.Name()
	stage.Close()
	defer os.Remove(stagePath)
	var equal *catalog.LibraryEntry
	key := contentKey(digest, size, mode)
	if entry, ok := known.content[key]; ok {
		value := entry
		equal = &value
	}
	// Same source admission reuses its entry even with explicit date input; dates
	// become a correction rather than silently creating another library item.
	if !item.NewEntry {
		if entry, ok := known.source[key+"\x00"+locator]; ok && item.Subtitle == "" {
			if hasOrigin(options) && !sameOwnerDate(entry, options) {
				entry = applyOrigin(entry, DerivedID(work.ID, "date-"+strconv.Itoa(ordinal)), options)
				updated, e := s.Catalog.CommitLibrary(ctx, work, entry)
				if e != nil {
					return ItemResult{}, e
				}
				entry = updated
			}
			return resultOf(ordinal, entry), nil
		}
	}
	var publicationID *string
	assetID := DerivedID(work.ID, "asset-"+strconv.Itoa(ordinal))
	if equal != nil {
		assetID = equal.AssetID
		if mode == "copy" {
			publicationID = equal.OriginalPublicationID
		}
	}
	if mode == "copy" && publicationID == nil {
		id, e := s.candidateAttempt(ctx, DerivedID(work.ID, "original-"+strconv.Itoa(ordinal)))
		if e != nil {
			return ItemResult{}, e
		}
		candidates = append(candidates, id)
		pub, e := s.Artifacts.Publish(ctx, id, stagePath, "original-media")
		if e != nil {
			return ItemResult{}, e
		}
		publicationID = &pub.ID
	}
	var subtitlePublication *string
	subtitleState := ""
	if item.Subtitle != "" {
		subtitle, e := filepath.Abs(item.Subtitle)
		if e != nil {
			return ItemResult{}, contracts.Fail("invalid_request")
		}
		if _, statErr := os.Stat(subtitle); os.IsNotExist(statErr) {
			subtitleState = "missing"
		} else {
			id, e := s.candidateAttempt(ctx, DerivedID(work.ID, "subtitle-"+strconv.Itoa(ordinal)))
			if e != nil {
				return ItemResult{}, e
			}
			candidates = append(candidates, id)
			pub, e := s.Artifacts.Publish(ctx, id, subtitle, "supplied-subtitle")
			if e != nil {
				return ItemResult{}, e
			}
			subtitlePublication = &pub.ID
			subtitleState = "available"
		}
	}
	bundle, e := s.captureOrRecover(ctx, work, reportID, stagePath, original, options, directory)
	if e != nil {
		return ItemResult{}, e
	}
	if bundle.Facts.Acquisition == nil {
		bundle.Facts.Acquisition = acquisition
	}
	if acquisition != nil {
		bundle.Facts.FilesystemModified = ""
	}
	bundle.Facts.SubtitleState = subtitleState
	// Store the complete report envelope after acquisition facts have been bound.
	if e = s.publishBundle(ctx, reportID, bundle, directory); e != nil {
		return ItemResult{}, e
	}
	// Recheck the source binding after extraction; reports always describe the
	// staged digest, and a changed input requires a fresh admission attempt.
	currentDigest, currentSize, e := identity(ctx, original)
	if e != nil || currentDigest != digest || currentSize != size {
		return ItemResult{}, contracts.Fail("conflict")
	}
	dates := s.dates(bundle, options, nil)
	title := item.Title
	if title == "" {
		title = filepath.Base(original)
		if acquisition != nil {
			if u, e := url.Parse(locator); e == nil {
				title = filepath.Base(u.Path)
			}
		}
	}
	if title == "" || title == "." {
		title = "Imported media"
	}
	class := "unknown"
	for _, stream := range bundle.Facts.Streams {
		if stream.Kind == "video" {
			class = "video"
			break
		}
		if stream.Kind == "audio" {
			class = "audio"
		}
	}
	if class == "unknown" {
		if strings.HasPrefix(bundle.Metadata.MIME.Preferred, "audio/") {
			class = "audio"
		}
		if strings.HasPrefix(bundle.Metadata.MIME.Preferred, "video/") {
			class = "video"
		}
	}
	entry := catalog.LibraryEntry{ID: entryID, AssetID: assetID, Title: title, Class: class, Mode: mode, SourceLocator: locator, Digest: digest, Size: size, OriginalPublicationID: publicationID, SubtitlePublicationID: subtitlePublication, DurationUS: bundle.Facts.DurationUS, Facts: marshal(bundle.Facts), Metadata: marshal(bundle.Metadata), Dates: marshal(dates), ReportPublicationIDs: marshal([]string{reportID})}
	entry, e = s.Catalog.CommitLibrary(ctx, work, entry)
	if e != nil {
		return ItemResult{}, e
	}
	if e = s.Cleanup(ctx, entry.ID); e != nil {
		return ItemResult{}, e
	}
	return resultOf(ordinal, entry), nil
}
func sameOwnerDate(entry catalog.LibraryEntry, options Options) bool {
	expected := ownerDate(options)
	state := decodeDates(entry.Dates)
	if options.DatePrecedence != "" && options.DatePrecedence != state.Policy {
		return false
	}
	if state.Selected == nil {
		return false
	}
	d := *state.Selected
	d.ID = ""
	// An owner correction includes its explicit interpretation provenance, even
	// when two choices happen to produce the same recording instant.
	return d.Basis == "owner" && string(marshal(d)) == string(marshal(expected))
}
func resultOf(ordinal int, entry catalog.LibraryEntry) ItemResult {
	var metadata Metadata
	json.Unmarshal(entry.Metadata, &metadata)
	dates := decodeDates(entry.Dates)
	dateState := "unknown"
	dateUnresolved := false
	for _, date := range dates.Observations {
		if date.Basis != "filesystem-modified" && (date.State == "unknown" || date.State == "ambiguous" || date.State == "nonexistent" || date.State == "invalid") {
			dateUnresolved = true
		}
	}
	if dates.Selected != nil {
		dateState = dates.Selected.State
	} else {
		for _, date := range dates.Observations {
			if date.Basis == "owner" {
				dateState = date.State
			}
		}
	}
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	return ItemResult{Ordinal: ordinal, MediaID: entry.ID, Digest: entry.Digest, State: "admitted", CaptureState: metadata.CaptureState, DateState: dateState, DateUnresolved: dateUnresolved, SubtitleState: facts.SubtitleState, QueuedJobIDs: []string{}}
}
func (s *Service) captureOrRecover(ctx context.Context, work catalog.Work, reportID, path, original string, options Options, directory string) (captureBundle, error) {
	if p, e := s.Catalog.Publication(ctx, reportID); e == nil {
		if p.State == "pending" {
			if _, e = s.Artifacts.Reconcile(ctx, reportID); e != nil {
				return captureBundle{}, e
			}
		}
		materialized, e := s.Artifacts.MaterializeBound(ctx, reportID, p.Size)
		if e != nil {
			return captureBundle{}, e
		}
		defer s.Artifacts.Release(context.Background(), reportID, materialized.Lease.ID)
		data, e := os.ReadFile(materialized.Path)
		var bundle captureBundle
		if e != nil || strict(data, &bundle) != nil {
			return bundle, contracts.Fail("operation_failed")
		}
		digest, size, e := identity(ctx, path)
		if e != nil || bundle.SourceDigest != digest || bundle.SourceSize != size {
			return bundle, contracts.Fail("conflict")
		}
		return bundle, nil
	}
	return s.capture(ctx, path, original, options)
}
func (s *Service) publishBundle(ctx context.Context, id string, bundle captureBundle, directory string) error {
	if p, e := s.Catalog.Publication(ctx, id); e == nil {
		if p.State == "available" {
			return nil
		}
		_, e = s.Artifacts.Reconcile(ctx, id)
		return e
	}
	path := filepath.Join(directory, "capture.json")
	if e := os.WriteFile(path, marshal(bundle), 0600); e != nil {
		return contracts.Fail("unavailable")
	}
	_, e := s.Artifacts.Publish(ctx, id, path, "media-metadata-report")
	return e
}
func (s *Service) dates(bundle captureBundle, options Options, owners []Date) Dates {
	observations := append([]Date{}, owners...)
	observations = append(observations, embeddedDates(bundle.Metadata.Observations, options)...)
	if hasOrigin(options) {
		d := ownerDate(options)
		d.ID = contracts.ID()
		observations = append([]Date{d}, observations...)
	}
	if bundle.Facts.FilesystemModified != "" {
		observations = append(observations, ResolveDate(bundle.Facts.FilesystemModified, "instant", "UTC", "", "", "filesystem-modified"))
	}
	return SelectDates(observations, options.DatePrecedence)
}
func (s *Service) Refresh(ctx context.Context, work catalog.Work, req RefreshRequest) (out catalog.LibraryEntry, returned error) {
	entry, e := s.Catalog.Library(ctx, req.MediaID)
	if e != nil {
		return entry, e
	}
	reportID, e := s.candidateAttempt(ctx, DerivedID(work.ID, "refresh-report"))
	if e != nil {
		return entry, e
	}
	defer func() {
		if returned != nil {
			s.discardUnusedCandidate(reportID)
		}
	}()
	currentReports, _ := catalog.PublicationIDs(entry.ReportPublicationIDs)
	for _, id := range currentReports {
		if id == reportID {
			if e = s.Cleanup(ctx, entry.ID); e != nil {
				return entry, e
			}
			return entry, nil
		}
	}
	directory, e := os.MkdirTemp(s.Artifacts.Workspace.Control, "library-refresh-")
	if e != nil {
		return entry, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	source := entry.SourceLocator
	if entry.Mode == "copy" {
		materialized, e := s.Artifacts.MaterializeBound(ctx, *entry.OriginalPublicationID, entry.Size)
		if e != nil {
			return entry, e
		}
		defer s.Artifacts.Release(context.Background(), *entry.OriginalPublicationID, materialized.Lease.ID)
		source = materialized.Path
	}
	stage, digest, size, e := s.Artifacts.Stage(ctx, source)
	if e != nil {
		return entry, e
	}
	stage.Close()
	defer os.Remove(stage.Name())
	if digest != entry.Digest || size != entry.Size {
		return entry, contracts.Fail("conflict")
	}
	var oldFacts Facts
	json.Unmarshal(entry.Facts, &oldFacts)
	options := merged(oldFacts.Options, req.Options)
	bundle, e := s.captureOrRecover(ctx, work, reportID, stage.Name(), entry.SourceLocator, options, directory)
	if e != nil {
		return entry, e
	}
	bundle.Facts.Acquisition = oldFacts.Acquisition
	bundle.Facts.SubtitleState = oldFacts.SubtitleState
	if entry.Mode == "copy" || bundle.Facts.FilesystemModified == "" {
		bundle.Facts.FilesystemModified = oldFacts.FilesystemModified
	}
	if e = s.publishBundle(ctx, reportID, bundle, directory); e != nil {
		return entry, e
	}
	owners := []Date{}
	for _, d := range decodeDates(entry.Dates).Observations {
		if d.Basis == "owner" {
			owners = append(owners, d)
		}
	}
	// An owner's previous import literal is already represented in owners.
	options.OriginatedAt = req.Options.OriginatedAt
	options.OriginatedOn = req.Options.OriginatedOn
	options.OriginatedEarliest = req.Options.OriginatedEarliest
	options.OriginatedLatest = req.Options.OriginatedLatest
	entry.Facts = marshal(bundle.Facts)
	entry.Metadata = marshal(bundle.Metadata)
	entry.Dates = marshal(s.dates(bundle, options, owners))
	entry.DurationUS = bundle.Facts.DurationUS
	entry.ReportPublicationIDs = marshal([]string{reportID})
	entry, e = s.Catalog.CommitLibrary(ctx, work, entry)
	if e != nil {
		return entry, e
	}
	if e = s.Cleanup(ctx, entry.ID); e != nil {
		return entry, e
	}
	return entry, nil
}

// A crash keeps a live candidate recoverable. An observed failed attempt
// retires or aborts its candidate, and a retry advances past the tombstone.
func (s *Service) candidateAttempt(ctx context.Context, base string) (string, error) {
	id := base
	for attempt := 1; ; attempt++ {
		p, e := s.Catalog.Publication(ctx, id)
		if code(e) == "not_found" {
			return id, nil
		}
		if e != nil {
			return "", e
		}
		if p.State == "retiring" {
			// Resume an observed failure's durable retirement before attempting a
			// replacement. The catalog rechecks current references and all leases.
			p, e = s.Artifacts.RetireCurrent(ctx, id)
			if e != nil {
				return "", e
			}
		}
		if p.State != "retired" && p.State != "aborted" {
			return id, nil
		}
		id = DerivedID(base, "attempt-"+strconv.Itoa(attempt))
	}
}
func (s *Service) discardUnusedCandidate(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if p, e := s.Catalog.Publication(ctx, id); e == nil {
		if p.State == "pending" {
			_, _ = s.Artifacts.Abort(ctx, id)
			return
		}
		if p.State != "available" && p.State != "retiring" && p.State != "missing" {
			return
		}
		// RetireCurrent atomically refuses all selected library report references.
		// This also handles an uncertain commit whose transaction actually won.
		_, _ = s.Artifacts.RetireCurrent(ctx, id)
	}
}
func (s *Service) Cleanup(ctx context.Context, entryID string) error {
	cleanups, e := s.Catalog.Cleanups(ctx)
	if e != nil {
		return e
	}
	for _, cleanup := range cleanups {
		if cleanup.EntryID != entryID || cleanup.State == "done" {
			continue
		}
		if _, e = s.Artifacts.RetireCurrent(ctx, cleanup.ID); e != nil {
			return e
		}
		if e = s.Catalog.FinishCleanup(ctx, cleanup.ID); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) List(ctx context.Context) ([]EntryView, error) {
	entries, e := s.Catalog.Libraries(ctx)
	if e != nil {
		return nil, e
	}
	out := []EntryView{}
	for _, entry := range entries {
		out = append(out, EntryView{entry, s.availability(ctx, entry)})
	}
	return out, nil
}
func (s *Service) Show(ctx context.Context, id string) (EntryView, error) {
	entry, e := s.Catalog.Library(ctx, id)
	if e != nil {
		return EntryView{}, e
	}
	return EntryView{entry, s.availability(ctx, entry)}, nil
}
func (s *Service) Metadata(ctx context.Context, id string) (Metadata, error) {
	entry, e := s.Catalog.Library(ctx, id)
	var out Metadata
	if e != nil {
		return out, e
	}
	if strict(entry.Metadata, &out) != nil {
		return out, contracts.Fail("operation_failed")
	}
	return out, nil
}
func (s *Service) availability(ctx context.Context, entry catalog.LibraryEntry) string {
	if entry.Mode == "copy" {
		if entry.OriginalPublicationID == nil {
			return "missing"
		}
		if p, e := s.Catalog.Publication(ctx, *entry.OriginalPublicationID); e == nil && p.State == "available" {
			return "available"
		}
		return "missing"
	}
	digest, size, e := identity(ctx, entry.SourceLocator)
	if e != nil {
		return "missing"
	}
	if digest != entry.Digest || size != entry.Size {
		return "changed"
	}
	return "available"
}
func identity(ctx context.Context, path string) (string, int64, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", 0, contracts.Fail("unavailable")
	}
	defer f.Close()
	stat, e := f.Stat()
	if e != nil || !stat.Mode().IsRegular() {
		return "", 0, contracts.Fail("invalid_request")
	}
	h := sha256.New()
	buffer := make([]byte, 1<<20)
	var n int64
	for {
		if ctx.Err() != nil {
			return "", 0, contracts.Fail("cancelled")
		}
		count, e := f.Read(buffer)
		if count > 0 {
			h.Write(buffer[:count])
			n += int64(count)
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", 0, contracts.Fail("operation_failed")
		}
	}
	after, e := f.Stat()
	if e != nil || after.Size() != n || !stat.ModTime().Equal(after.ModTime()) {
		return "", 0, contracts.Fail("conflict")
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
func (s *Service) Relocate(ctx context.Context, op, id string, expected int64, path string) (catalog.LibraryEntry, error) {
	entry, e := s.Catalog.Library(ctx, id)
	if e != nil {
		return entry, e
	}
	if entry.Mode != "reference" {
		return entry, contracts.Fail("invalid_request")
	}
	absolute, e := filepath.Abs(path)
	if e != nil {
		return entry, contracts.Fail("invalid_request")
	}
	stage, digest, size, e := s.Artifacts.Stage(ctx, absolute)
	if e != nil {
		return entry, e
	}
	stage.Close()
	os.Remove(stage.Name())
	if digest != entry.Digest || size != entry.Size {
		return entry, contracts.Fail("conflict")
	}
	entry.SourceLocator = absolute
	return s.Catalog.UpdateLibrary(ctx, op, expected, entry)
}
func (s *Service) SetOrigin(ctx context.Context, op, id string, expected int64, options Options) (catalog.LibraryEntry, error) {
	entry, e := s.Catalog.Library(ctx, id)
	if e != nil {
		return entry, e
	}
	if !validOptions(options) || !hasOrigin(options) {
		return entry, contracts.Fail("invalid_request")
	}
	entry = applyOrigin(entry, op, options)
	return s.Catalog.UpdateLibrary(ctx, op, expected, entry)
}
func applyOrigin(entry catalog.LibraryEntry, op string, options Options) catalog.LibraryEntry {
	state := decodeDates(entry.Dates)
	d := ownerDate(options)
	d.ID = DerivedID(op, "owner-date")
	// Preserve incoming conflicts and prior owner observations while selecting the
	// newly entered owner's value with deterministic precedence.
	observations := append([]Date{d}, state.Observations...)
	policy := options.DatePrecedence
	if policy == "" {
		policy = state.Policy
	}
	entry.Dates = marshal(SelectDates(observations, policy))
	return entry
}
