// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/process"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func TestCanonicalPolicy(t *testing.T) {
	mono, stereo := int64(1), int64(2)
	for _, test := range []struct {
		streams       []Stream
		format, codec string
	}{
		{[]Stream{{Kind: "audio", Codec: "mp3", Channels: &mono}}, "mp3", "copy"},
		{[]Stream{{Kind: "audio", Codec: "aac", Channels: &mono}}, "mp3", "libmp3lame"},
		{[]Stream{{Kind: "audio", Codec: "aac", Channels: &stereo}}, "flac", "flac"},
		{[]Stream{{Kind: "audio", Codec: "pcm_s16le", Channels: &mono}}, "flac", "flac"},
		{[]Stream{{Kind: "audio", Codec: "mp3", Channels: &mono}, {Kind: "audio", Codec: "mp3", Channels: &mono}}, "mka", "flac"},
	} {
		format, codec, err := canonicalPolicy(test.streams)
		if err != nil || format != test.format || codec != test.codec {
			t.Fatal(format, codec, err)
		}
	}
	if _, _, err := canonicalPolicy(nil); err == nil {
		t.Fatal("no audio accepted")
	}
}
func TestNativeCanonicalAdmissionPreservationAndReplay(t *testing.T) {
	tools := nativeTools(t)
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = tools
	source := wavFixture(t)
	raw, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "input.cueson.json")
	os.WriteFile(transcript, raw, 0600)
	request := ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Source: source, Transcript: transcript}}}
	work := claimLibraryWork(t, db, "media.import", request)
	ctx := context.Background()
	value, err := s.Execute(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	result := value.(ImportResult)
	if len(result.Items) != 1 || result.Items[0].State != "admitted" {
		t.Fatalf("admission %+v", result)
	}
	entry, err := db.Library(ctx, result.Items[0].MediaID)
	if err != nil {
		t.Fatal(err)
	}
	current, err := db.Recording(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current.Document, bytes.TrimSpace(raw)) || current.SourceRevision != entry.Revision {
		t.Fatal("preservation/source binding")
	}
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	if facts.Canonical == nil || facts.Canonical.Format != "flac" || entry.SourceLocator != "" || entry.SubtitlePublicationID != nil || len(facts.Streams) != 1 || facts.Streams[0].Codec != "flac" {
		t.Fatalf("canonical %+v", facts)
	}
	durable, err := db.Work(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(durable.Payload, []byte(filepath.Base(source))) || bytes.Contains(durable.Payload, []byte(filepath.Base(transcript))) {
		t.Fatal("durable source retained")
	}
	os.Remove(source)
	os.Remove(transcript)
	replay, err := s.Execute(ctx, durable)
	if err != nil {
		t.Fatal(err)
	}
	if replay.(ImportResult).Items[0].MediaID != entry.ID {
		t.Fatal("replay reacquired")
	}
	snapshot, err := db.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "restore.sqlite"), snapshot.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	encoded, _ := json.Marshal(snapshot)
	json.Unmarshal(encoded, &snapshot)
	if err = fresh.Restore(ctx, snapshot); err != nil {
		t.Fatalf("portable proof %v", err)
	}
}
func TestNativeStereoUsesFLAC(t *testing.T) {
	tools := nativeTools(t)
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = tools
	source := wavFixture(t)
	data, _ := os.ReadFile(source)
	binary.LittleEndian.PutUint16(data[22:24], 2)
	binary.LittleEndian.PutUint32(data[28:32], 192000)
	binary.LittleEndian.PutUint16(data[32:34], 4)
	os.WriteFile(source, data, 0600)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source}}})
	value, err := s.Execute(context.Background(), work)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(ImportResult).Items[0]
	if out.State != "admitted" {
		t.Fatalf("stereo %+v", out)
	}
	entry, _ := db.Library(context.Background(), out.MediaID)
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	if facts.Canonical.Format != "flac" || *facts.Streams[0].Channels != 2 {
		t.Fatal("stereo changed")
	}
	pubs, err := catalog.PublicationIDs(entry.ReportPublicationIDs)
	if err != nil || len(pubs) != 1 {
		t.Fatal("report missing")
	}
}
func TestTranscriptSelectionAndFormat(t *testing.T) {
	index := 2
	tracks := []EmbeddedTrack{{Index: 1, Supported: true}, {Index: 2, Supported: true, Default: true, Language: "en"}, {Index: 3, Codec: "dvd_subtitle"}}
	chosen, err := selectEmbedded(tracks, Options{})
	if err != nil || chosen.Index != 2 {
		t.Fatal("default selection", err)
	}
	chosen, err = selectEmbedded(tracks, Options{SubtitleStreamIndex: &index, SubtitleLanguage: "EN"})
	if err != nil || chosen.Index != 2 {
		t.Fatal("explicit selection", err)
	}
	if _, err = transcriptFormat([]byte("<html>error</html>"), "srt", "x.srt"); err == nil {
		t.Fatal("HTML accepted")
	}
	if !contracts.ValidLocalSpeakerID("voice one") {
		t.Fatal("local token")
	}
}
func mediaFixtureCommand(t *testing.T, tool Tool, args ...string) {
	t.Helper()
	all := append([]string{"-nostdin", "-v", "error", "-y"}, args...)
	if _, err := process.Capture(context.Background(), process.Spec{Executable: tool.Path, Args: all, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10}); err != nil {
		t.Fatal(err)
	}
}
func TestNativeCanonicalGroupedOffsetsAndMP3(t *testing.T) {
	tools := nativeTools(t)
	ctx := context.Background()
	s, db := libraryFixture(t)
	s.Tools = tools
	s.legacyFixture = false
	wav := wavFixture(t)
	dir := t.TempDir()
	group := filepath.Join(dir, "tracks.mka")
	mediaFixtureCommand(t, tools.FFmpeg, "-copyts", "-itsoffset", "2", "-i", wav, "-itsoffset", "5", "-i", wav, "-map", "0:a:0", "-map", "1:a:0", "-c:a", "flac", "-metadata:s:a:0", "language=eng", "-metadata:s:a:1", "language=fra", "-disposition:a:0", "default", "-disposition:a:1", "0", group)
	result, err := s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: group}}}))
	if err != nil {
		t.Fatal(err)
	}
	item := result.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("group %+v", item)
	}
	entry, _ := db.Library(ctx, item.MediaID)
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	if len(facts.Streams) != 2 || facts.Canonical.Format != "mka" || facts.Canonical.Tracks[0].StartNumerator != "2" || facts.Canonical.Tracks[1].StartNumerator != "5" {
		t.Fatalf("group clock %+v", facts)
	}
	materialized, err := s.Artifacts.MaterializeBound(ctx, *entry.OriginalPublicationID, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Artifacts.Release(ctx, *entry.OriginalPublicationID, materialized.Lease.ID)
	for i, language := range []string{"eng", "fra"} {
		var value string
		json.Unmarshal(facts.Streams[i].Tags["language"], &value)
		if value != language || facts.Streams[i].Codec != "flac" || *facts.Streams[i].Channels != 1 {
			t.Fatal("track metadata changed", facts.Streams[i])
		}
		decoded := filepath.Join(dir, "decoded"+string(rune('0'+i))+".pcm")
		mediaFixtureCommand(t, tools.FFmpeg, "-i", materialized.Path, "-map", "0:"+string(rune('0'+i)), "-c:a", "pcm_s16le", "-f", "s16le", decoded)
		raw, _ := os.ReadFile(decoded)
		if len(raw) != 960 {
			t.Fatalf("track %d truncated or padded: %d", i, len(raw))
		}
	}
	mp3 := filepath.Join(dir, "mono.mp3")
	mediaFixtureCommand(t, tools.FFmpeg, "-i", wav, "-c:a", "libmp3lame", "-q:a", "0", mp3)
	result, err = s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: mp3}}}))
	if err != nil {
		t.Fatal(err)
	}
	item = result.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("mp3 %+v", item)
	}
	entry, _ = db.Library(ctx, item.MediaID)
	json.Unmarshal(entry.Facts, &facts)
	if facts.Canonical.Format != "mp3" || facts.Streams[0].Codec != "mp3" {
		t.Fatal("MP3 tier")
	}
	aac := filepath.Join(dir, "mono.m4a")
	mediaFixtureCommand(t, tools.FFmpeg, "-i", wav, "-c:a", "aac", aac)
	result, err = s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: aac}}}))
	if err != nil {
		t.Fatal(err)
	}
	item = result.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("V0 %+v", item)
	}
}

func TestNativeCanonicalIntegerPrecisionAndSurround(t *testing.T) {
	tools := nativeTools(t)
	ctx := context.Background()
	s, db := libraryFixture(t)
	s.Tools = tools
	s.legacyFixture = false
	wav := wavFixture(t)
	data, _ := os.ReadFile(wav)
	// Nonzero low-order 32-bit samples detect implicit reduction to 24 bits.
	binary.LittleEndian.PutUint16(data[34:36], 32)
	binary.LittleEndian.PutUint16(data[32:34], 4)
	binary.LittleEndian.PutUint32(data[28:32], 192000)
	for i := 44; i < len(data); i += 4 {
		binary.LittleEndian.PutUint32(data[i:i+4], uint32(i*7919+1))
	}
	os.WriteFile(wav, data, 0600)
	value, err := s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wav}}}))
	if err != nil {
		t.Fatal(err)
	}
	item := value.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("32 bit %+v", item)
	}
	entry, _ := db.Library(ctx, item.MediaID)
	materialized, err := s.Artifacts.MaterializeBound(ctx, *entry.OriginalPublicationID, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Artifacts.Release(ctx, *entry.OriginalPublicationID, materialized.Lease.ID)
	decoded := filepath.Join(t.TempDir(), "integer.pcm")
	mediaFixtureCommand(t, tools.FFmpeg, "-i", materialized.Path, "-c:a", "pcm_s32le", "-f", "s32le", decoded)
	got, _ := os.ReadFile(decoded)
	if !bytes.Equal(got, data[44:]) {
		t.Fatal("integer precision reduced")
	}
	surround := wavFixture(t)
	data, _ = os.ReadFile(surround)
	binary.LittleEndian.PutUint16(data[22:24], 6)
	binary.LittleEndian.PutUint16(data[32:34], 12)
	binary.LittleEndian.PutUint32(data[28:32], 576000)
	os.WriteFile(surround, data, 0600)
	value, err = s.Execute(ctx, claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: surround}}}))
	if err != nil {
		t.Fatal(err)
	}
	item = value.(ImportResult).Items[0]
	if item.State != "admitted" {
		t.Fatalf("surround %+v", item)
	}
	entry, _ = db.Library(ctx, item.MediaID)
	var facts Facts
	json.Unmarshal(entry.Facts, &facts)
	if *facts.Streams[0].Channels != 6 || facts.Streams[0].Codec != "flac" {
		t.Fatal("surround changed")
	}
}
