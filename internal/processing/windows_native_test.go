// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

func nativeWindowTool(t *testing.T) library.Tool {
	t.Helper()
	file := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if file == "" {
		t.Skip("exact media tools not elected")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var tools struct {
		FFmpeg library.Tool `json:"ffmpeg"`
	}
	if json.Unmarshal(raw, &tools) != nil || tools.FFmpeg.Path == "" {
		t.Fatal("missing FFmpeg pin")
	}
	return tools.FFmpeg
}
func nativeWindowFixture(t *testing.T, tool library.Tool, format string, seconds string) (string, library.Facts) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "source."+format)
	channels := int64(2)
	if format == "mp3" || format == "mka" {
		channels = 1
	}
	duration, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		t.Fatal(err)
	}
	input := nativeWindowWAV(t, 44100, int(channels), int64(duration*44100))
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", input, "-c:a", "flac", "-y", file}
	facts := library.Facts{Container: format, Extension: "." + format, Streams: []library.Stream{{Index: 0, Kind: "audio", SampleRate: "44100", Channels: &channels}}, Canonical: &library.Canonical{Tracks: []library.CanonicalTrack{{Index: 0, SourceIndex: 0, StartNumerator: "96001", StartDenominator: "48000"}}}}
	if format == "mp3" {
		args = []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", input, "-c:a", "libmp3lame", "-q:a", "0", "-y", file}
	}
	if format == "mka" {
		second := nativeWindowWAV(t, 48000, 1, 53904)
		args = []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-copyts", "-itsoffset", "2", "-i", input, "-itsoffset", "5", "-i", second, "-map", "0:a:0", "-map", "1:a:0", "-c:a", "flac", "-y", file}
		facts.Streams = append(facts.Streams, library.Stream{Index: 1, Kind: "audio", SampleRate: "48000", Channels: &channels})
		facts.Canonical.Tracks = []library.CanonicalTrack{{Index: 0, SourceIndex: 0, StartNumerator: "2", StartDenominator: "1"}, {Index: 1, SourceIndex: 1, StartNumerator: "5", StartDenominator: "1"}}
	}

	if result, err := process.Capture(context.Background(), process.Spec{Executable: tool.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 65536}); err != nil {
		t.Fatalf("fixture: %v: %s", err, result.Stderr)
	}
	return file, facts
}
func nativeWindowWAV(t *testing.T, rate, channels int, count int64) string {
	t.Helper()
	data := make([]byte, int(count)*channels*2)
	for sample := int64(0); sample < count; sample++ {
		for ch := 0; ch < channels; ch++ {
			value := int16((sample*int64(997+ch*83))%32760 - 16380)
			binary.LittleEndian.PutUint16(data[(int(sample)*channels+ch)*2:], uint16(value))
		}
	}
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(len(data)+36))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(header[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(data)))
	path := filepath.Join(t.TempDir(), "input.wav")
	if err := os.WriteFile(path, append(header, data...), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func windowServiceFixture(t *testing.T, tool library.Tool, file string, facts library.Facts) (*Service, catalog.LibraryEntry) {
	t.Helper()
	ctx := context.Background()
	w, err := workspace.Init(t.TempDir(), "windows")
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	db, err := catalog.OpenWorkspace(ctx, w, nil, false)
	if err != nil {
		t.Fatalf("catalog open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.RegisterWorkspace(ctx, w); err != nil {
		t.Fatalf("workspace registration: %v", err)
	}
	a, err := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if err != nil {
		t.Fatalf("artifact service: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	p, err := a.Publish(ctx, contracts.ID(), file, "canonical-audio")
	if err != nil {
		t.Fatalf("publication: %v", err)
	}
	raw, _ := json.Marshal(facts)
	entry := catalog.LibraryEntry{ID: contracts.ID(), Revision: 1, Mode: "copy", Size: p.Size, Digest: p.Digest, OriginalPublicationID: &p.ID, Facts: raw}
	return &Service{Artifacts: a, Catalog: db, Config: Config{FFmpeg: tool, TimeoutMS: 120000, MaxDurationUS: 120000000}}, entry
}
func windowPCM(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for offset := 12; offset+8 <= len(raw); {
		n := int(binary.LittleEndian.Uint32(raw[offset+4:]))
		if offset+8+n > len(raw) {
			t.Fatal("invalid WAV")
		}
		if string(raw[offset:offset+4]) == "data" {
			return raw[offset+8 : offset+8+n]
		}
		offset += 8 + n + n%2
	}
	t.Fatal("PCM absent")
	return nil
}
func windowOracle(t *testing.T, tool library.Tool, path string, index int, channel *int) []byte {
	t.Helper()
	args := windowDecodeArgs(index, channel, 16000, 2)
	for i := range args {
		if args[i] == "pipe:0" {
			args[i] = path
		}
		if args[i] == "pipe" {
			args[i] = "file,pipe"
		}
	}
	var output bytes.Buffer
	if _, err := process.Stream(context.Background(), process.Spec{Executable: tool.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 65536}, &output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestNativeWindowPCMMatchesContinuousFullDecode(t *testing.T) {
	tool := nativeWindowTool(t)
	for _, format := range []string{"flac", "mp3", "mka"} {
		t.Run(format, func(t *testing.T) {
			file, facts := nativeWindowFixture(t, tool, format, "3.217")
			s, entry := windowServiceFixture(t, tool, file, facts)
			options := WindowOptions{CoreUS: 1000000}
			for _, stream := range facts.Streams {
				options.Tracks = append(options.Tracks, TrackSelection{StreamIndex: stream.Index})
			}
			identity := windowTestIdentity()
			identity.SourceID = entry.ID
			identity.SourceDigest = entry.Digest
			windows, err := s.PrepareWindows(context.Background(), entry, options, identity)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.Close()
			oracles := map[int][]byte{}
			for _, stream := range facts.Streams {
				oracles[stream.Index] = windowOracle(t, tool, file, stream.Index, nil)
			}
			for i, w := range windows.Plan.Windows {
				session, err := windows.Window(context.Background(), i)
				if err != nil {
					t.Fatalf("window %d: %v", i, err)
				}
				pcm := windowPCM(t, session.MappedPath)
				expected := oracles[w.StreamIndex][w.MappedStartSample*2 : w.MappedEndSample*2]
				if !bytes.Equal(pcm, expected) {
					t.Fatalf("window %d PCM differs from full decode", i)
				}
				if err = session.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err = windows.Finish(context.Background()); err != nil {
				t.Fatal("finish", err)
			}
		})
	}
}

func TestNativeWindowScratchBoundAndResumeSkip(t *testing.T) {
	tool := nativeWindowTool(t)
	var peaks []int64
	for _, duration := range []string{"3.1", "31"} {
		file, facts := nativeWindowFixture(t, tool, "flac", duration)
		s, entry := windowServiceFixture(t, tool, file, facts)
		identity := windowTestIdentity()
		identity.SourceID = entry.ID
		identity.SourceDigest = entry.Digest
		windows, err := s.PrepareWindows(context.Background(), entry, WindowOptions{CoreUS: 1000000}, identity)
		if err != nil {
			t.Fatal(err)
		}
		peak := int64(0)
		for i := range windows.Plan.Windows {
			if i%3 == 0 && i < len(windows.Plan.Windows)-1 {
				continue
			}
			session, err := windows.Window(context.Background(), i)
			if err != nil {
				t.Fatal(i, err)
			}
			var size int64
			filepath.WalkDir(windows.parent.directory, func(path string, d fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if !d.IsDir() {
					info, e := d.Info()
					if e != nil {
						return e
					}
					size += info.Size()
				}
				return nil
			})
			peak = max(peak, size)
			if err = session.Close(); err != nil {
				t.Fatal(err)
			}
		}
		if err = windows.Finish(context.Background()); err != nil {
			t.Fatal(err)
		}
		dir := windows.parent.directory
		if err = windows.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(dir); !os.IsNotExist(err) {
			t.Fatal("owned scratch retained")
		}
		peaks = append(peaks, peak)
	}
	if peaks[0] > 65000 || peaks[1] > 65000 || peaks[1] > peaks[0]+64 {
		t.Fatal("scratch grew with duration", peaks)
	}
}

func TestNativeWindowRejectsChangedSourceBeforeInference(t *testing.T) {
	tool := nativeWindowTool(t)
	file, facts := nativeWindowFixture(t, tool, "flac", "3.1")
	s, entry := windowServiceFixture(t, tool, file, facts)
	entry.Mode = "reference"
	entry.OriginalPublicationID = nil
	entry.SourceLocator = file
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	entry.Digest = hex.EncodeToString(digest[:])
	identity := windowTestIdentity()
	identity.SourceID = entry.ID
	identity.SourceDigest = entry.Digest
	windows, err := s.PrepareWindows(context.Background(), entry, WindowOptions{CoreUS: 1000000}, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.Close()
	raw[len(raw)/2] ^= 1
	if err = os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = windows.Window(context.Background(), 0); err == nil {
		t.Fatal("mutated source reached inference")
	}
	if !strings.Contains(contracts.SafeError(err).Code, "asset") && contracts.SafeError(err).Code != "invalid_audio" {
		t.Log("source rejection", err)
	}
}

func TestNativeWindowSelectedChannelsAndRangeMatchFullDecode(t *testing.T) {
	tool := nativeWindowTool(t)
	file, facts := nativeWindowFixture(t, tool, "flac", "3.217")
	s, entry := windowServiceFixture(t, tool, file, facts)
	identity := windowTestIdentity()
	identity.SourceID, identity.SourceDigest = entry.ID, entry.Digest
	left, right, end := 0, 1, int64(110251)
	options := WindowOptions{CoreUS: 1000000, Tracks: []TrackSelection{{StreamIndex: 0, Channel: &left, StartSample: 4409, EndSample: &end}, {StreamIndex: 0, Channel: &right, StartSample: 4409, EndSample: &end}}}
	windows, err := s.PrepareWindows(context.Background(), entry, options, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.Close()
	oracles := [][]byte{windowOracle(t, tool, file, 0, &left), windowOracle(t, tool, file, 0, &right)}
	for i, w := range windows.Plan.Windows {
		session, err := windows.Window(context.Background(), i)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(windowPCM(t, session.MappedPath), oracles[w.TrackOrdinal][w.MappedStartSample*2:w.MappedEndSample*2]) {
			t.Fatal("channel or selected endpoint mismatch", i)
		}
	}
	if err = windows.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNativeWindowMP3RatesMatchGaplessFullDecode(t *testing.T) {
	tool := nativeWindowTool(t)
	for _, rate := range []int{16000, 22050, 48000} {
		t.Run(strconv.Itoa(rate), func(t *testing.T) {
			input := nativeWindowWAV(t, rate, 1, int64(rate*21/10))
			file := filepath.Join(t.TempDir(), "source.mp3")
			args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", input, "-c:a", "libmp3lame", "-b:a", "96k", "-y", file}
			if _, err := process.Capture(context.Background(), process.Spec{Executable: tool.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 65536}); err != nil {
				t.Fatal(err)
			}
			mono := int64(1)
			facts := library.Facts{Container: "mp3", Extension: ".mp3", Streams: []library.Stream{{Index: 0, Kind: "audio", SampleRate: strconv.Itoa(rate), Channels: &mono, TimeBase: "1/1000000", StartPTS: new(int64)}}}
			s, entry := windowServiceFixture(t, tool, file, facts)
			identity := windowTestIdentity()
			identity.SourceID, identity.SourceDigest = entry.ID, entry.Digest
			windows, err := s.PrepareWindows(context.Background(), entry, WindowOptions{CoreUS: 1000000}, identity)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.Close()
			oracle := windowOracle(t, tool, file, 0, nil)
			for i, w := range windows.Plan.Windows {
				session, err := windows.Window(context.Background(), i)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(windowPCM(t, session.MappedPath), oracle[w.MappedStartSample*2:w.MappedEndSample*2]) {
					t.Fatal("gapless sample mismatch", i)
				}
			}
			if err = windows.Finish(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeWindowCancellationReleasesLeaseAndScratch(t *testing.T) {
	tool := nativeWindowTool(t)
	file, facts := nativeWindowFixture(t, tool, "flac", "3.217")
	s, entry := windowServiceFixture(t, tool, file, facts)
	identity := windowTestIdentity()
	identity.SourceID, identity.SourceDigest = entry.ID, entry.Digest
	ctx, cancel := context.WithCancel(context.Background())
	windows, err := s.PrepareWindows(ctx, entry, WindowOptions{CoreUS: 1000000}, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.Close()
	if _, err = s.Catalog.ClaimRetirement(ctx, *entry.OriginalPublicationID, contracts.ID(), 0, time.Minute); err == nil {
		t.Fatal("live source lease did not fence retirement")
	}
	if _, err = windows.Window(ctx, 0); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err = windows.Finish(context.Background()); err == nil {
		t.Fatal("cancelled decoding accepted")
	}
	dir := windows.parent.directory
	if err = windows.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("cancelled scratch retained")
	}
	if _, err = s.Catalog.ClaimRetirement(context.Background(), *entry.OriginalPublicationID, contracts.ID(), 0, time.Minute); err != nil {
		t.Fatal("closed source still leased", err)
	}
}

type countedWindowStore struct {
	artifact.Store
	opens int
	sizes []int64
}

func (s *countedWindowStore) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	s.opens++
	s.sizes = append(s.sizes, length)
	return s.Store.OpenRange(ctx, p, offset, length)
}

func TestNativeWindowReadsSourcePerTrackNotPerWindow(t *testing.T) {
	tool := nativeWindowTool(t)
	file, facts := nativeWindowFixture(t, tool, "flac", "11.1")
	s, entry := windowServiceFixture(t, tool, file, facts)
	counted := &countedWindowStore{Store: s.Artifacts.Store}
	s.Artifacts.Store = counted
	identity := windowTestIdentity()
	identity.SourceID, identity.SourceDigest = entry.ID, entry.Digest
	windows, err := s.PrepareWindows(context.Background(), entry, WindowOptions{CoreUS: 1000000}, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.Close()
	for i := range windows.Plan.Windows {
		if _, err = windows.Window(context.Background(), i); err != nil {
			t.Fatal(err)
		}
	}
	if err = windows.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if counted.opens != 3 {
		t.Fatal("expected verify, native count, continuous resample reads", counted.opens)
	}
	for _, size := range counted.sizes {
		if size != entry.Size {
			t.Fatal("unexpected source range", size)
		}
	}
}

func TestNativeWindowS3StreamingAndChangedObject(t *testing.T) {
	tool := nativeWindowTool(t)
	for _, changed := range []bool{false, true} {
		t.Run(strconv.FormatBool(changed), func(t *testing.T) {
			file, facts := nativeWindowFixture(t, tool, "flac", "3.217")
			s, entry := windowServiceFixture(t, tool, file, facts)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var mutate atomic.Bool
			var gets atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
				w.Header().Set("ETag", `"fixture"`)
				w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
				if r.Method == "HEAD" {
					return
				}
				gets.Add(1)
				data := raw
				if mutate.Load() {
					data = append([]byte(nil), raw...)
					data[len(data)/2] ^= 1
				}
				w.Write(data)
			}))
			defer server.Close()
			store, err := artifact.NewS3(context.Background(), artifact.S3Config{Endpoint: server.URL, Bucket: "fixture", Region: "us-east-1", Authentication: "anonymous", AddressingStyle: "path"}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Artifacts.Store.Close(); err != nil {
				t.Fatal(err)
			}
			s.Artifacts.Store = store
			identity := windowTestIdentity()
			identity.SourceID, identity.SourceDigest = entry.ID, entry.Digest
			windows, err := s.PrepareWindows(context.Background(), entry, WindowOptions{CoreUS: 1000000}, identity)
			if err != nil {
				t.Fatal(err)
			}
			defer windows.Close()
			mutate.Store(changed)
			if changed {
				if _, err = windows.Window(context.Background(), 0); err == nil {
					t.Fatal("changed S3 object reached inference")
				}
				return
			}
			oracle := windowOracle(t, tool, file, 0, nil)
			for i, w := range windows.Plan.Windows {
				session, err := windows.Window(context.Background(), i)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(windowPCM(t, session.MappedPath), oracle[w.MappedStartSample*2:w.MappedEndSample*2]) {
					t.Fatal("S3 streamed PCM differs", i)
				}
			}
			if err = windows.Finish(context.Background()); err != nil {
				t.Fatal(err)
			}
			if gets.Load() != 3 {
				t.Fatal("S3 object reread per window", gets.Load())
			}
		})
	}
}
