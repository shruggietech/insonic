// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

const windowSourceChunk = 1 << 20

// verifiedWindowSource holds at most 2 MiB of digests for the 64 GiB input cap.
// Every later pass verifies a complete bounded chunk before supplying any of
// its bytes to a decoder. No canonical-size local snapshot is created.
type verifiedWindowSource struct {
	size   int64
	digest string
	open   func(context.Context) (io.ReadCloser, error)
	chunks [][32]byte
	header mp3HeaderProbe
}

func (s *verifiedWindowSource) verify(ctx context.Context) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if s.size < 0 || s.size > 64<<30 || !pinnedDigest.MatchString(s.digest) || s.open == nil {
		return contracts.Fail("invalid_request")
	}
	r, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer r.Close()
	h := sha256.New()
	buffer := make([]byte, windowSourceChunk)
	s.chunks = nil
	s.header = mp3HeaderProbe{}
	for remaining := s.size; remaining > 0; {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		n := min(int64(len(buffer)), remaining)
		if _, err = io.ReadFull(cancelReader{ctx, r}, buffer[:n]); err != nil {
			if ctx.Err() != nil {
				return contracts.Fail("cancelled")
			}
			return contracts.AssetFailure("asset_size", "artifact", "object")
		}
		h.Write(buffer[:n])
		s.header.Write(buffer[:n])
		s.chunks = append(s.chunks, sha256.Sum256(buffer[:n]))
		remaining -= n
	}
	var extra [1]byte
	n, e := r.Read(extra[:])
	if n != 0 || e != io.EOF {
		return contracts.AssetFailure("asset_size", "artifact", "object")
	}
	if hex.EncodeToString(h.Sum(nil)) != s.digest {
		return contracts.AssetFailure("asset_hash", "artifact", "object")
	}
	return nil
}
func (s *verifiedWindowSource) reader(ctx context.Context) (*verifiedChunkReader, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	r, err := s.open(ctx)
	if err != nil {
		return nil, err
	}
	return &verifiedChunkReader{ctx: ctx, source: s, reader: r, buffer: make([]byte, windowSourceChunk)}, nil
}

type verifiedChunkReader struct {
	ctx       context.Context
	source    *verifiedWindowSource
	reader    io.ReadCloser
	buffer    []byte
	chunk     int
	offset    int
	available int
	err       error
}

func (r *verifiedChunkReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.ctx.Err() != nil {
		return 0, contracts.Fail("cancelled")
	}
	if r.err != nil {
		return 0, r.err
	}
	if r.offset == r.available {
		if r.chunk >= len(r.source.chunks) {
			return 0, io.EOF
		}
		n := min(int64(len(r.buffer)), r.source.size-int64(r.chunk)*windowSourceChunk)
		if _, err := io.ReadFull(cancelReader{r.ctx, r.reader}, r.buffer[:n]); err != nil {
			r.err = contracts.AssetFailure("asset_size", "artifact", "object")
			if r.ctx.Err() != nil {
				r.err = contracts.Fail("cancelled")
			}
			return 0, r.err
		}
		if sha256.Sum256(r.buffer[:n]) != r.source.chunks[r.chunk] {
			r.err = contracts.AssetFailure("asset_hash", "artifact", "object")
			return 0, r.err
		}
		r.chunk++
		r.offset = 0
		r.available = int(n)
	}
	n := copy(p, r.buffer[r.offset:r.available])
	r.offset += n
	return n, nil
}
func (r *verifiedChunkReader) Close() error { return r.reader.Close() }

type WindowSession struct {
	Plan      ProcessingPlan
	service   *Service
	parent    *Session
	entry     catalog.LibraryEntry
	source    *verifiedWindowSource
	config    Config
	mu        sync.Mutex
	closed    bool
	finished  bool
	last      int
	active    *Session
	decoder   *windowDecoder
	tail      []byte
	tailStart int64
	mp3       *mp3Gapless
}

func newWindowAudioSession(ctx context.Context, s *Service, parent string) (*Session, error) {
	dir, err := os.MkdirTemp(parent, "processing-")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	if err = workspace.SecureDirectory(dir, true); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if err = os.WriteFile(filepath.Join(dir, ".insonic-processing-session"), []byte("insonic-processing-scratch-v1\n"), 0600); err != nil {
		os.RemoveAll(dir)
		return nil, contracts.Fail("unavailable")
	}
	work, cancel := context.WithCancel(ctx)
	session := &Session{service: s, directory: dir, ctx: work, cancel: cancel, stop: make(chan struct{}), done: make(chan struct{}), Provenance: map[string]any{}, Diagnostics: []Diagnostic{}}
	go session.renew()
	return session, nil
}

func (s *Service) PrepareWindows(ctx context.Context, entry catalog.LibraryEntry, options WindowOptions, identity PlanIdentity) (_ *WindowSession, returned error) {
	config, err := defaults(s.Config)
	if err != nil {
		return nil, err
	}
	options, err = NormalizeWindowOptions(options)
	if err != nil {
		return nil, err
	}
	if s.Artifacts == nil || s.Artifacts.Workspace == nil || s.Artifacts.Store == nil || s.Catalog == nil || entry.Size < 0 || entry.Size > config.MaxInputBytes {
		return nil, contracts.Fail("invalid_request")
	}
	if identity.SourceID == "" {
		identity.SourceID = entry.ID
	}
	if identity.SourceDigest == "" {
		identity.SourceDigest = entry.Digest
	}
	if identity.SourceRevision == 0 {
		identity.SourceRevision = entry.Revision
	}
	if identity.SourceID != entry.ID || identity.SourceDigest != entry.Digest || identity.SourceRevision != entry.Revision {
		return nil, contracts.Fail("conflict")
	}
	var facts library.Facts
	if json.Unmarshal(entry.Facts, &facts) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	selections := options.Tracks
	if len(selections) == 0 {
		for _, stream := range facts.Streams {
			if stream.Kind == "audio" {
				selections = []TrackSelection{{StreamIndex: stream.Index}}
				break
			}
		}
	}
	if len(selections) == 0 {
		return nil, contracts.Fail("unsupported_audio")
	}
	options.Tracks = selections
	parent, err := newWindowAudioSession(ctx, s, s.Artifacts.Workspace.Control)
	if err != nil {
		return nil, err
	}
	long := &WindowSession{service: s, parent: parent, entry: entry, config: config, last: -1}
	defer func() {
		if returned != nil {
			long.Close()
		}
	}()
	long.source, err = long.sourceReader(entry)
	if err != nil {
		return nil, err
	}
	if err = long.source.verify(parent.ctx); err != nil {
		return nil, err
	}
	long.mp3 = long.source.header.gapless(long.source.size)
	if err = verifyTool(parent.ctx, config.FFmpeg); err != nil {
		return nil, err
	}
	tracks := make([]TrackPlan, 0, len(selections))
	for _, selection := range selections {
		var selected *library.Stream
		for i := range facts.Streams {
			stream := &facts.Streams[i]
			if stream.Kind == "audio" && stream.Index == selection.StreamIndex {
				selected = stream
				break
			}
		}
		if selected == nil || selection.Channel != nil && (selected.Channels == nil || int64(*selection.Channel) >= *selected.Channels) {
			return nil, contracts.Fail("unsupported_audio")
		}
		rate, err := strconv.ParseInt(selected.SampleRate, 10, 64)
		if err != nil || rate < 1 || rate > 768000 {
			return nil, contracts.Fail("unsupported_audio")
		}
		mapping, err := sourceClock(*selected)
		if facts.Canonical != nil {
			found := false
			for _, track := range facts.Canonical.Tracks {
				if track.Index == selected.Index {
					origin, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator)
					if !ok {
						return nil, contracts.Fail("invalid_timing")
					}
					mapping.StartNumerator = origin.Num().String()
					mapping.StartDenominator = origin.Denom().String()
					err = nil
					found = true
					break
				}
			}
			if !found {
				return nil, contracts.Fail("timing_unavailable")
			}
		}
		if err != nil {
			return nil, err
		}
		count, err := long.countTrack(parent.ctx, selection, rate)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, TrackPlan{StreamIndex: selected.Index, Channel: selection.Channel, SourceSampleRate: rate, SourceSampleCount: count, StartNumerator: mapping.StartNumerator, StartDenominator: mapping.StartDenominator, EndSample: count})
	}
	long.Plan, err = BuildWindowPlan(identity, options, tracks)
	if err != nil {
		return nil, err
	}
	return long, nil
}

func (s *WindowSession) sourceReader(entry catalog.LibraryEntry) (*verifiedWindowSource, error) {
	source := &verifiedWindowSource{size: entry.Size, digest: entry.Digest}
	if entry.Mode == "copy" {
		if entry.OriginalPublicationID == nil {
			return nil, contracts.Fail("conflict")
		}
		p, err := s.service.Catalog.Publication(s.parent.ctx, *entry.OriginalPublicationID)
		if err != nil {
			return nil, err
		}
		if p.State != "available" || p.Size != entry.Size || p.Digest != entry.Digest {
			return nil, contracts.Fail("conflict")
		}
		id := contracts.ID()
		lease, err := s.service.Catalog.ArtifactLease(s.parent.ctx, p.ID, id, id, s.service.Artifacts.TTL, false)
		if err != nil {
			return nil, err
		}
		s.parent.mu.Lock()
		s.parent.leases = append(s.parent.leases, artifact.Materialization{PublicationID: p.ID, Lease: lease})
		s.parent.mu.Unlock()
		source.open = func(ctx context.Context) (io.ReadCloser, error) {
			stat, err := s.service.Artifacts.Store.Stat(ctx, p)
			if err != nil {
				return nil, err
			}
			if stat.Size != p.Size || p.Version != "" && stat.Version != p.Version {
				return nil, contracts.AssetFailure("artifact_identity", "artifact", "object")
			}
			return s.service.Artifacts.Store.OpenRange(ctx, p, 0, p.Size)
		}
	} else if entry.Mode == "reference" {
		if !filepath.IsAbs(entry.SourceLocator) {
			return nil, contracts.Fail("invalid_request")
		}
		source.open = func(ctx context.Context) (io.ReadCloser, error) {
			if ctx.Err() != nil {
				return nil, contracts.Fail("cancelled")
			}
			before, err := os.Lstat(entry.SourceLocator)
			if err != nil || !before.Mode().IsRegular() || before.Size() != entry.Size {
				return nil, contracts.Fail("conflict")
			}
			file, err := os.Open(entry.SourceLocator)
			if err != nil {
				return nil, contracts.Fail("unavailable")
			}
			info, err := file.Stat()
			if err != nil || !os.SameFile(before, info) || info.Size() != entry.Size {
				file.Close()
				return nil, contracts.Fail("conflict")
			}
			return file, nil
		}
	} else {
		return nil, contracts.Fail("invalid_request")
	}
	return source, nil
}

func windowDecodeArgs(index int, channel *int, rate int64, threads int) []string {
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-threads", strconv.Itoa(threads), "-protocol_whitelist", "pipe", "-i", "pipe:0", "-map", fmt.Sprintf("0:%d", index), "-vn", "-sn", "-dn"}
	if channel != nil {
		args = append(args, "-af", fmt.Sprintf("pan=mono|c0=c%d", *channel))
	}
	return append(args, "-ac", "1", "-ar", strconv.FormatInt(rate, 10), "-c:a", "pcm_s16le", "-f", "s16le", "pipe:1")
}

type windowCountWriter struct {
	count, limit int64
	err          error
}

func (w *windowCountWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.limit-w.count {
		w.err = contracts.Fail("audio_limit")
		return 0, w.err
	}
	w.count += int64(len(p))
	return len(p), nil
}

func (s *WindowSession) decodeInput(ctx context.Context) (*verifiedChunkReader, error) {
	return s.source.reader(ctx)
}
func (s *WindowSession) decodeArgs(index int, channel *int, rate int64) []string {
	args := windowDecodeArgs(index, channel, rate, s.config.Threads)
	if s.mp3 != nil {
		filter := fmt.Sprintf("atrim=end_sample=%d", s.mp3.samples)
		if channel != nil {
			filter = fmt.Sprintf("pan=mono|c0=c%d,", *channel) + filter
		}
		found := false
		for i := range args {
			if args[i] == "-af" {
				args[i+1] = filter
				found = true
				break
			}
		}
		if !found {
			args = append(args[:len(args)-1], "-af", filter, args[len(args)-1])
		}
	}
	return args
}
func (s *WindowSession) countTrack(ctx context.Context, selection TrackSelection, rate int64) (int64, error) {
	bounded, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutMS)*time.Millisecond)
	defer cancel()
	reader, err := s.decodeInput(bounded)
	if err != nil {
		return 0, err
	}
	var input io.Reader
	if reader != nil {
		input = reader
		defer reader.Close()
		stop := context.AfterFunc(bounded, func() { reader.Close() })
		defer stop()
	}
	maximum := s.config.MaxDurationUS * rate / 1000000
	decodeMaximum := maximum
	if s.mp3 != nil {
		decodeMaximum += s.mp3.untrimmed - s.mp3.samples
	}
	counter := &windowCountWriter{limit: decodeMaximum * 2}
	_, err = process.Stream(bounded, process.Spec{Executable: s.config.FFmpeg.Path, Args: windowDecodeArgs(selection.StreamIndex, selection.Channel, rate, s.config.Threads), Input: input, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: s.config.MaxOutputBytes}, counter)
	if reader != nil && reader.err != nil {
		return 0, reader.err
	}
	if counter.err != nil {
		return 0, counter.err
	}
	if err != nil {
		return 0, err
	}
	if counter.count%2 != 0 {
		return 0, contracts.Fail("invalid_audio")
	}
	count := counter.count / 2
	if s.mp3 != nil {
		if rate != s.mp3.rate || count != s.mp3.untrimmed && count != s.mp3.samples {
			return 0, contracts.Fail("invalid_audio")
		}
		count = s.mp3.samples
	}
	if count > maximum {
		return 0, contracts.Fail("audio_limit")
	}
	return count, nil
}

type windowDecoder struct {
	track    int
	reader   *io.PipeReader
	cancel   context.CancelFunc
	done     chan error
	position int64
	finished bool
	err      error
}

func (s *WindowSession) startDecoder(track int) (*windowDecoder, error) {
	ctx, cancel := context.WithCancel(s.parent.ctx)
	source, err := s.decodeInput(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	reader, writer := io.Pipe()
	decoder := &windowDecoder{track: track, reader: reader, cancel: cancel, done: make(chan error, 1)}
	stop := context.AfterFunc(ctx, func() {
		reader.CloseWithError(contracts.Fail("cancelled"))
		writer.CloseWithError(contracts.Fail("cancelled"))
		if source != nil {
			source.Close()
		}
	})
	t := s.Plan.Tracks[track]
	var input io.Reader
	if source != nil {
		input = source
	}
	go func() {
		_, err := process.Stream(ctx, process.Spec{Executable: s.config.FFmpeg.Path, Args: s.decodeArgs(t.StreamIndex, t.Channel, 16000), Input: input, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: s.config.MaxOutputBytes}, writer)
		if source != nil && source.err != nil {
			err = source.err
		}
		if source != nil {
			source.Close()
		}
		writer.CloseWithError(err)
		stop()
		decoder.done <- err
	}()
	return decoder, nil
}
func (d *windowDecoder) close() error {
	if d.finished {
		return d.err
	}
	d.cancel()
	d.reader.Close()
	d.err = <-d.done
	d.finished = true
	return d.err
}

func (s *WindowSession) finishDecoder(ctx context.Context) error {
	d := s.decoder
	if d == nil {
		return nil
	}
	if d.finished {
		return d.err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutMS)*time.Millisecond)
	defer cancel()
	stop := context.AfterFunc(ctx, d.cancel)
	defer stop()
	t := s.Plan.Tracks[d.track]
	maximum := (t.SourceSampleCount*16000 + t.SourceSampleRate - 1) / t.SourceSampleRate
	remaining := &windowCountWriter{limit: max(int64(0), maximum-d.position) * 2}
	_, err := io.Copy(remaining, d.reader)
	if err != nil {
		d.cancel()
	}
	processErr := <-d.done
	d.finished = true
	d.err = errors.Join(err, processErr)
	if ctx.Err() != nil {
		d.err = contracts.Fail("cancelled")
		return d.err
	}
	if d.err != nil {
		return d.err
	}
	total := d.position + remaining.count/2
	minimum := t.SourceSampleCount * 16000 / t.SourceSampleRate
	if remaining.count%2 != 0 || total < minimum || total > maximum {
		d.err = contracts.Fail("invalid_audio")
	}
	return d.err
}

// Finish proves successful resampler termination and sample counts before the
// caller accepts a transcript. Close is cancellation/cleanup, not acceptance.
func (s *WindowSession) Finish(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.parent.ctx.Err() != nil || ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if s.finished {
		return nil
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutMS)*time.Millisecond)
	defer cancel()
	if err := s.finishDecoder(bounded); err != nil {
		return err
	}
	s.finished = true
	return nil
}

// Window prepares one bounded input. Calls advance monotonically; the caller
// retries inference on the returned Session rather than decoding it again.
func (s *WindowSession) Window(ctx context.Context, index int) (_ *Session, returned error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.finished || s.parent.ctx.Err() != nil || ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if index < 0 || index >= len(s.Plan.Windows) || index <= s.last {
		return nil, contracts.Fail("invalid_request")
	}
	if s.active != nil {
		if err := s.active.Close(); err != nil {
			return nil, err
		}
		s.active = nil
	}
	w := s.Plan.Windows[index]
	if s.decoder == nil || s.decoder.track != w.TrackOrdinal {
		if s.decoder != nil {
			if err := s.finishDecoder(ctx); err != nil {
				return nil, err
			}
			s.decoder.close()
		}
		var err error
		s.decoder, err = s.startDecoder(w.TrackOrdinal)
		if err != nil {
			return nil, err
		}
		s.tail = nil
		s.tailStart = 0
	}
	work, cancel := context.WithTimeout(s.parent.ctx, time.Duration(s.config.TimeoutMS)*time.Millisecond)
	defer cancel()
	stopCaller := context.AfterFunc(ctx, cancel)
	defer stopCaller()
	decoder := s.decoder
	stopRead := context.AfterFunc(work, decoder.cancel)
	defer stopRead()
	if decoder.position < w.MappedStartSample {
		if _, err := io.CopyN(io.Discard, decoder.reader, (w.MappedStartSample-decoder.position)*2); err != nil {
			return nil, contracts.Fail("invalid_audio")
		}
		decoder.position = w.MappedStartSample
		s.tail = nil
	}
	data := make([]byte, (w.MappedEndSample-w.MappedStartSample)*2)
	copied := int64(0)
	if decoder.position > w.MappedStartSample {
		if w.MappedStartSample < s.tailStart || decoder.position > s.tailStart+int64(len(s.tail))/2 {
			return nil, contracts.Fail("invalid_timing")
		}
		end := min(decoder.position, w.MappedEndSample)
		copied = end - w.MappedStartSample
		copy(data[:copied*2], s.tail[(w.MappedStartSample-s.tailStart)*2:(end-s.tailStart)*2])
	}
	if copied < int64(len(data))/2 {
		if _, err := io.ReadFull(decoder.reader, data[copied*2:]); err != nil {
			if work.Err() != nil {
				return nil, contracts.Fail("cancelled")
			}
			return nil, err
		}
		decoder.position = w.MappedEndSample
	}
	// Retain at most both context margins, independent of recording length.
	tailSamples := min(int64(len(data))/2, (*s.Plan.Options.ContextUS*16000/1000000)*2+2)
	s.tail = append(s.tail[:0], data[len(data)-int(tailSamples*2):]...)
	s.tailStart = w.MappedEndSample - tailSamples
	child, err := newWindowAudioSession(s.parent.ctx, s.service, s.parent.directory)
	if err != nil {
		return nil, err
	}
	defer func() {
		if returned != nil {
			child.Close()
		}
	}()
	child.MappedPath = filepath.Join(child.directory, "mapped.wav")
	child.SourceMap = w.Mapping
	if err = writeWindowWAV(child.MappedPath, data); err != nil {
		return nil, err
	}
	file, err := os.Open(child.MappedPath)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	h := sha256.New()
	_, err = io.Copy(h, file)
	file.Close()
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	child.Provenance = map[string]any{"source_sha256": s.entry.Digest, "source_size": s.entry.Size, "mapped_audio_sha256": hex.EncodeToString(h.Sum(nil)), "decoder": "ffmpeg", "decoder_version": s.config.FFmpeg.Version, "decoder_sha256": s.config.FFmpeg.SHA256, "sample_rate": 16000, "channels": 1, "source_stream": w.StreamIndex, "source_channel": w.Channel, "window_id": w.ID, "plan_sha256": s.Plan.Digest, "core_start_sample": w.CoreStartSample, "core_end_sample": w.CoreEndSample, "context_start_sample": w.ContextStartSample, "context_end_sample": w.ContextEndSample, "source_sample_rate": w.SourceSampleRate}
	if len(data) == 0 {
		child.Diagnostics = append(child.Diagnostics, Diagnostic{Code: "empty_decoded_audio", Count: 1})
	}
	s.active = child
	s.last = index
	return child, nil
}

func writeWindowWAV(path string, data []byte) error {
	if len(data)%2 != 0 || int64(len(data)) > 300*2*16000*2 {
		return contracts.Fail("audio_limit")
	}
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(len(data)+36))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], 16000)
	binary.LittleEndian.PutUint32(header[28:], 32000)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(data)))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	_, err = f.Write(header)
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return contracts.Fail("unavailable")
	}
	return nil
}

func (s *WindowSession) Close() error {
	if s == nil || s.parent == nil {
		return nil
	}
	s.parent.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.decoder != nil {
		s.decoder.close()
		s.decoder = nil
	}
	var childErr error
	if s.active != nil {
		childErr = s.active.Close()
		s.active = nil
	}
	s.tail = nil
	return errors.Join(childErr, s.parent.Close())
}
