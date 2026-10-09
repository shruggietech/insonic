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
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

func defaults(c Config) (Config, error) {
	if c.MaxInputBytes == 0 {
		c.MaxInputBytes = 64 << 30
	}
	if c.MaxDurationUS == 0 {
		c.MaxDurationUS = 3_600_000_000
	}
	if c.MaxOutputBytes == 0 {
		c.MaxOutputBytes = 16 << 20
	}
	if c.TimeoutMS == 0 {
		c.TimeoutMS = 600_000
	}
	if c.Threads == 0 {
		c.Threads = 2
	}
	if c.MaxInputBytes < 1 || c.MaxInputBytes > 64<<30 || c.MaxDurationUS < 1 || c.MaxDurationUS > 604_800_000_000 || c.MaxOutputBytes < 1 || c.MaxOutputBytes > 16<<20 || c.TimeoutMS < 1 || c.TimeoutMS > 86_400_000 || c.Threads < 1 || c.Threads > 64 {
		return c, contracts.Fail("invalid_request")
	}
	return c, nil
}

func verifyPinned(ctx context.Context, file library.PinnedFile) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if !filepath.IsAbs(file.Path) || len(file.SHA256) != 64 {
		return contracts.Fail("unavailable")
	}
	f, err := os.Open(file.Path)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return contracts.Fail("unavailable")
	}
	h := sha256.New()
	buffer := make([]byte, 128<<10)
	for {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		n, e := f.Read(buffer)
		if n > 0 {
			h.Write(buffer[:n])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return contracts.Fail("unavailable")
		}
	}
	if hex.EncodeToString(h.Sum(nil)) != file.SHA256 {
		return contracts.Fail("incompatible_version")
	}
	return nil
}

// VerifyFFmpeg shares the pinned executable contract with native playback.
func VerifyFFmpeg(ctx context.Context, tool library.Tool) error { return verifyTool(ctx, tool) }

func verifyTool(ctx context.Context, tool library.Tool) error {
	if tool.Version == "" || tool.Interpreter != nil {
		return contracts.Fail("unavailable")
	}
	if err := verifyPinned(ctx, library.PinnedFile{Path: tool.Path, SHA256: tool.SHA256}); err != nil {
		return err
	}
	for _, file := range tool.SupportFiles {
		if err := verifyPinned(ctx, file); err != nil {
			return err
		}
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := process.Capture(checkCtx, process.Spec{Executable: tool.Path, Args: []string{"-version"}, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 65536})
	if err != nil {
		return err
	}
	first := strings.SplitN(string(result.Stdout), "\n", 2)[0]
	parts := strings.Fields(first)
	if len(parts) < 3 || parts[0] != "ffmpeg" || parts[1] != "version" || parts[2] != tool.Version {
		return contracts.Fail("incompatible_version")
	}
	return nil
}

func sourceClock(stream library.Stream) (SourceMap, error) {
	var start *big.Rat
	if stream.StartPTS != nil && stream.TimeBase != "" {
		base, ok := new(big.Rat).SetString(stream.TimeBase)
		if !ok || base.Sign() <= 0 {
			return SourceMap{}, contracts.Fail("invalid_request")
		}
		start = new(big.Rat).Mul(base, new(big.Rat).SetInt64(*stream.StartPTS))
	} else if stream.StartTime != "" {
		var ok bool
		start, ok = new(big.Rat).SetString(stream.StartTime)
		if !ok {
			return SourceMap{}, contracts.Fail("invalid_request")
		}
	} else {
		return SourceMap{}, contracts.Fail("timing_unavailable")
	}
	return SourceMap{StartNumerator: start.Num().String(), StartDenominator: start.Denom().String(), StreamIndex: stream.Index, SampleRate: 16000, Policy: "full-selected-stream;source-start-rational;sample-clock;no-seeking"}, nil
}

func pcmInfo(path string, maxDurationUS int64) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, contracts.Fail("unavailable")
	}
	defer f.Close()
	header := make([]byte, 12)
	if _, err = io.ReadFull(f, header); err != nil || string(header[:4]) != "RIFF" || string(header[8:]) != "WAVE" {
		return 0, contracts.Fail("invalid_audio")
	}
	info, err := f.Stat()
	if err != nil || info.Size() < 12 || info.Size() > maxDurationUS*16000/1_000_000*2+1<<20 {
		return 0, contracts.Fail("audio_limit")
	}
	formatOK := false
	dataFound := false
	var count int64
	for ordinal := 0; ordinal < 64; ordinal++ {
		chunk := make([]byte, 8)
		_, err = io.ReadFull(f, chunk)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, contracts.Fail("invalid_audio")
		}
		size := int64(binary.LittleEndian.Uint32(chunk[4:]))
		position, _ := f.Seek(0, io.SeekCurrent)
		if size > info.Size()-position {
			return 0, contracts.Fail("invalid_audio")
		}
		switch string(chunk[:4]) {
		case "fmt ":
			if size < 16 || size > 256 {
				return 0, contracts.Fail("invalid_audio")
			}
			format := make([]byte, size)
			if _, err = io.ReadFull(f, format); err != nil {
				return 0, contracts.Fail("invalid_audio")
			}
			formatOK = binary.LittleEndian.Uint16(format) == 1 && binary.LittleEndian.Uint16(format[2:]) == 1 && binary.LittleEndian.Uint32(format[4:]) == 16000 && binary.LittleEndian.Uint16(format[12:]) == 2 && binary.LittleEndian.Uint16(format[14:]) == 16
		case "data":
			if dataFound || size%2 != 0 {
				return 0, contracts.Fail("invalid_audio")
			}
			dataFound = true
			count = size / 2
			if _, err = f.Seek(size, io.SeekCurrent); err != nil {
				return 0, contracts.Fail("invalid_audio")
			}
		default:
			if _, err = f.Seek(size, io.SeekCurrent); err != nil {
				return 0, contracts.Fail("invalid_audio")
			}
		}
		if size%2 != 0 {
			if _, err = f.Seek(1, io.SeekCurrent); err != nil {
				return 0, contracts.Fail("invalid_audio")
			}
		}
	}
	if !formatOK || !dataFound {
		return 0, contracts.Fail("invalid_audio")
	}
	if count*1_000_000 > maxDurationUS*16000 {
		return 0, contracts.Fail("audio_limit")
	}
	return count, nil
}

func decodeAudio(ctx context.Context, tool library.Tool, source, destination string, index int, channel *int, config Config) (int64, error) {
	config, err := defaults(config)
	if err != nil {
		return 0, err
	}
	if index < 0 || channel != nil && (*channel < 0 || *channel > 63) {
		return 0, contracts.Fail("invalid_request")
	}
	if err = verifyTool(ctx, tool); err != nil {
		return 0, err
	}
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-threads", strconv.Itoa(config.Threads), "-i", source, "-map", fmt.Sprintf("0:%d", index), "-vn"}
	if channel != nil {
		args = append(args, "-af", fmt.Sprintf("pan=mono|c0=c%d", *channel))
	}
	args = append(args, "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-t", strconv.FormatFloat(float64(config.MaxDurationUS)/1e6+1, 'f', 6, 64), "-fs", strconv.FormatInt(config.MaxDurationUS*16000/1_000_000*2+1<<20, 10), "-f", "wav", "-y", destination)
	decodeCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMS)*time.Millisecond)
	defer cancel()
	_, err = process.Capture(decodeCtx, process.Spec{Executable: tool.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: config.MaxOutputBytes})
	if err != nil {
		os.Remove(destination)
		return 0, err
	}
	count, err := pcmInfo(destination, config.MaxDurationUS)
	if err != nil {
		os.Remove(destination)
	}
	return count, err
}

func (s *Service) Prepare(ctx context.Context, entry catalog.LibraryEntry, options AudioOptions) (_ *Session, returned error) {
	config, err := defaults(s.Config)
	if err != nil {
		return nil, err
	}
	if s.Artifacts == nil || s.Catalog == nil || entry.Size < 0 || entry.Size > config.MaxInputBytes {
		return nil, contracts.Fail("invalid_request")
	}
	var facts library.Facts
	if json.Unmarshal(entry.Facts, &facts) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var selected *library.Stream
	for i := range facts.Streams {
		stream := &facts.Streams[i]
		if stream.Kind == "audio" && (options.StreamIndex == nil || stream.Index == *options.StreamIndex) {
			selected = stream
			break
		}
	}
	if selected == nil || options.Channel != nil && (selected.Channels == nil || *options.Channel < 0 || int64(*options.Channel) >= *selected.Channels) {
		return nil, contracts.Fail("unsupported_audio")
	}
	mapping, err := sourceClock(*selected)
	if facts.Canonical != nil {
		found := false
		for _, track := range facts.Canonical.Tracks {
			if track.Index == selected.Index {
				start, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator)
				if !ok {
					return nil, contracts.Fail("timing_unavailable")
				}
				mapping = SourceMap{StartNumerator: start.Num().String(), StartDenominator: start.Denom().String(), StreamIndex: selected.Index, SampleRate: 16000, Policy: "full-selected-canonical-stream;original-start-rational;sample-clock;no-seeking"}
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
	mapping.Channel = options.Channel
	directory, err := os.MkdirTemp(s.Artifacts.Workspace.Control, "processing-")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	if err = workspace.SecureDirectory(directory, true); err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	if err = os.WriteFile(filepath.Join(directory, ".insonic-processing-session"), []byte("insonic-processing-scratch-v1\n"), 0600); err != nil {
		os.RemoveAll(directory)
		return nil, contracts.Fail("unavailable")
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &Session{service: s, directory: directory, ctx: sessionCtx, cancel: cancel, stop: make(chan struct{}), done: make(chan struct{}), SourceMap: mapping, Provenance: map[string]any{}, Diagnostics: []Diagnostic{}}
	go session.renew()
	defer func() {
		if returned != nil {
			session.Close()
		}
	}()
	source := entry.SourceLocator
	if entry.Mode == "copy" {
		if entry.OriginalPublicationID == nil {
			return nil, contracts.Fail("conflict")
		}
		materialized, e := session.materialize(sessionCtx, *entry.OriginalPublicationID, entry.Size)
		if e != nil {
			return nil, e
		}
		source = materialized.Path
	} else if entry.Mode != "reference" {
		return nil, contracts.Fail("invalid_request")
	}
	stage := filepath.Join(directory, "source")
	stageCtx, stageCancel := context.WithTimeout(sessionCtx, time.Duration(config.TimeoutMS)*time.Millisecond)
	err = stageSource(stageCtx, source, stage, entry.Size, entry.Digest)
	stageCancel()
	if err != nil {
		return nil, err
	}
	defer os.Remove(stage)
	digest, size := entry.Digest, entry.Size
	session.MappedPath = filepath.Join(directory, "mapped.wav")
	count, err := decodeAudio(sessionCtx, config.FFmpeg, stage, session.MappedPath, selected.Index, options.Channel, config)
	if err != nil {
		return nil, err
	}
	session.SourceMap.SampleCount = count
	file, err := os.Open(session.MappedPath)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	file.Close()
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	session.Provenance = map[string]any{"source_sha256": digest, "source_size": size, "mapped_audio_sha256": hex.EncodeToString(hash.Sum(nil)), "decoder": "ffmpeg", "decoder_version": config.FFmpeg.Version, "decoder_sha256": config.FFmpeg.SHA256, "audio_format": "pcm_s16le", "sample_rate": 16000, "channels": 1, "source_stream": selected.Index, "source_channel": options.Channel, "source_channels": selected.Channels, "source_time_base": selected.TimeBase, "source_start_pts": selected.StartPTS}
	if selected.Channels != nil && *selected.Channels > 1 && options.Channel == nil {
		session.Diagnostics = append(session.Diagnostics, Diagnostic{Code: "selected_stream_downmixed_to_mono", Count: *selected.Channels})
	}
	if count == 0 {
		session.Diagnostics = append(session.Diagnostics, Diagnostic{Code: "empty_decoded_audio", Count: 1})
	}
	return session, nil
}

func (s *Session) materialize(ctx context.Context, id string, size int64) (artifact.Materialization, error) {
	m, err := s.service.Artifacts.MaterializeBound(ctx, id, size)
	if err != nil {
		return m, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		s.service.Artifacts.Release(cleanup, id, m.Lease.ID)
		return m, contracts.Fail("cancelled")
	}
	s.leases = append(s.leases, m)
	return m, nil
}

type cancelReader struct {
	ctx    context.Context
	source io.Reader
}

func (r cancelReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}

// Stage at most the catalog's accepted size plus one byte. A growing referenced
// source cannot consume an unbounded copy before its identity is rejected.
func stageSource(ctx context.Context, source, destination string, size int64, digest string) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if !filepath.IsAbs(source) || size < 0 || size > 64<<30 || len(digest) != 64 {
		return contracts.Fail("invalid_request")
	}
	before, err := os.Lstat(source)
	if err != nil || !before.Mode().IsRegular() || before.Size() != size {
		return contracts.Fail("conflict")
	}
	input, err := os.Open(source)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil || !os.SameFile(before, opened) || opened.Size() != size {
		return contracts.Fail("conflict")
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(output, h), io.LimitReader(cancelReader{ctx, input}, size+1))
	syncErr := output.Sync()
	closeErr := output.Close()
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if copyErr != nil || syncErr != nil || closeErr != nil {
		return contracts.Fail("unavailable")
	}
	after, err := input.Stat()
	if err != nil {
		return contracts.Fail("unavailable")
	}
	current, err := os.Lstat(source)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(opened, current) || after.Size() != size || current.Size() != size || !opened.ModTime().Equal(after.ModTime()) || n != size || hex.EncodeToString(h.Sum(nil)) != digest {
		return contracts.Fail("conflict")
	}
	return nil
}

func (s *Session) renew() {
	defer close(s.done)
	interval := s.service.Artifacts.TTL / 3
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			leases := append([]artifact.Materialization(nil), s.leases...)
			s.mu.Unlock()
			for _, lease := range leases {
				if _, err := s.service.Artifacts.Renew(s.ctx, lease.PublicationID, lease.Lease.ID); err != nil {
					s.cancel()
					return
				}
			}
		}
	}
}

func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.cancel()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	s.operations.Wait()
	var failures []error
	s.mu.Lock()
	leases := append([]artifact.Materialization(nil), s.leases...)
	s.leases = nil
	s.mu.Unlock()
	for _, lease := range leases {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := s.service.Artifacts.Release(ctx, lease.PublicationID, lease.Lease.ID)
		cancel()
		if err != nil {
			failures = append(failures, err)
		}
	}
	if err := os.RemoveAll(s.directory); err != nil {
		failures = append(failures, contracts.Fail("unavailable"))
	}
	return errors.Join(failures...)
}

// RecoverScratch runs once after acquiring the workspace's exclusive runtime
// ownership and before starting workers. Disposable engine/native results have
// no accepted catalog authority and may be removed after an interrupted owner.
func (s *Service) RecoverScratch(ctx context.Context) error {
	if s.Artifacts == nil || s.Artifacts.Workspace == nil {
		return contracts.Fail("invalid_request")
	}
	control := s.Artifacts.Workspace.Control
	entries, err := os.ReadDir(control)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "processing-") {
			continue
		}
		suffix := strings.TrimPrefix(entry.Name(), "processing-")
		if suffix == "" {
			continue
		}
		valid := true
		for _, digit := range suffix {
			if digit < '0' || digit > '9' {
				valid = false
			}
		}
		if !valid {
			continue
		}
		root, err := os.OpenRoot(filepath.Join(control, entry.Name()))
		if err != nil {
			return contracts.Fail("unavailable")
		}
		marker, err := root.Open(".insonic-processing-session")
		if err != nil {
			root.Close()
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(marker, 65))
		marker.Close()
		root.Close()
		if err != nil || string(raw) != "insonic-processing-scratch-v1\n" {
			continue
		}
		if err = os.RemoveAll(filepath.Join(control, entry.Name())); err != nil {
			return contracts.Fail("unavailable")
		}
	}
	return nil
}
