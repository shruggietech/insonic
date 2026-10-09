// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/processing"
)

const maxPreviewBytes int64 = 2 << 30

func previewRequired(facts library.Facts, class string) bool {
	ext := strings.TrimPrefix(strings.ToLower(facts.Extension), ".")
	if class == "audio" {
		if ext != "wav" && ext != "mp3" && ext != "flac" && ext != "m4a" {
			return true
		}
		for _, stream := range facts.Streams {
			if stream.Kind != "audio" {
				continue
			}
			// WKWebView rejects the qualified six-channel FLAC master.
			// Present multichannel audio through the disposable stereo preview.
			if stream.Channels != nil && *stream.Channels > 2 {
				return true
			}
			if ext == "m4a" && stream.Codec != "aac" && stream.Codec != "mp3" {
				return true
			}
			if ext == "wav" && stream.Codec != "pcm_u8" && stream.Codec != "pcm_s16le" && stream.Codec != "pcm_s24le" && stream.Codec != "pcm_s32le" && stream.Codec != "pcm_f32le" {
				return true
			}
		}
		return false
	}
	if class != "video" {
		return false
	}
	if ext != "mp4" && ext != "m4v" && ext != "webm" {
		return true
	}
	for _, stream := range facts.Streams {
		if stream.Kind == "video" && ext != "webm" && stream.Codec != "h264" {
			return true
		}
		if stream.Kind == "audio" && stream.Codec != "aac" && stream.Codec != "mp3" && stream.Codec != "opus" && stream.Codec != "vorbis" {
			return true
		}
	}
	return false
}

func directPlaybackOffset(facts library.Facts) (float64, error) {
	found := false
	offset := float64(0)
	for _, stream := range facts.Streams {
		if stream.Kind != "audio" && stream.Kind != "video" {
			continue
		}
		start := stream.StartTime
		if start == "" || start == "N/A" {
			if stream.StartPTS == nil || stream.TimeBase == "" {
				continue
			}
			base, ok := new(big.Rat).SetString(stream.TimeBase)
			if !ok || base.Sign() <= 0 {
				return 0, contracts.Fail("timing_unavailable")
			}
			start = new(big.Rat).Mul(base, new(big.Rat).SetInt64(*stream.StartPTS)).FloatString(12)
		}
		value, e := probeSeconds(start, 0)
		if e != nil {
			return 0, e
		}
		if !found || value < offset {
			offset = value
			found = true
		}
	}
	return offset, nil
}

func previewArguments(source, output, class string, facts library.Facts) []string {
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-threads", "2", "-copyts", "-start_at_zero", "-i", source}
	audioMap := "0:a:0"
	for _, stream := range facts.Streams {
		if stream.Kind == "audio" {
			audioMap = "0:" + strconv.Itoa(stream.Index)
			break
		}
	}
	if class == "audio" {
		args = append(args, "-map", audioMap, "-vn", "-c:a", "pcm_s16le", "-ac", "2", "-ar", "48000", "-f", "wav")
	} else {
		codec := ""
		for _, stream := range facts.Streams {
			if stream.Kind == "video" {
				codec = stream.Codec
				break
			}
		}
		args = append(args, "-map", "0:v:0", "-map", audioMap+"?")
		if codec == "h264" {
			args = append(args, "-c:v", "copy")
		} else if runtime.GOOS == "darwin" {
			args = append(args, "-c:v", "h264_videotoolbox", "-b:v", "2M", "-pix_fmt", "yuv420p")
		} else {
			args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p")
		}
		args = append(args, "-c:a", "aac", "-ac", "2", "-b:a", "128k", "-movflags", "+faststart", "-f", "mp4")
	}
	return append(args, "-map_metadata", "-1", "-fs", strconv.FormatInt(maxPreviewBytes, 10), "-y", output)
}

type previewFormat struct {
	Start    string `json:"start_time"`
	Duration string `json:"duration"`
	Name     string `json:"format_name"`
}

type previewProbe struct {
	Format  previewFormat    `json:"format"`
	Streams []library.Stream `json:"streams"`
}

// Container durations do not share a universal origin: MOV reports content
// duration, while Matroska's duration includes the initial timestamp gap.
// Prefer the span of the streams that previewArguments actually selects.
func previewContentDuration(probe previewProbe, class string) (float64, error) {
	selected := map[string]bool{}
	first, last := math.Inf(1), math.Inf(-1)
	complete := true
	for _, stream := range probe.Streams {
		if stream.Kind != "audio" && (class != "video" || stream.Kind != "video") || selected[stream.Kind] {
			continue
		}
		selected[stream.Kind] = true
		start, e := probeSeconds(stream.StartTime, 0)
		if e != nil {
			return 0, e
		}
		duration, e := probeSeconds(stream.Duration, -1)
		if e != nil {
			return 0, e
		}
		if duration <= 0 && (strings.Contains(probe.Format.Name, "matroska") || strings.Contains(probe.Format.Name, "webm")) {
			var literal string
			json.Unmarshal(stream.Tags["DURATION"], &literal)
			parts := strings.Split(literal, ":")
			if len(parts) == 3 {
				h, e1 := strconv.ParseFloat(parts[0], 64)
				m, e2 := strconv.ParseFloat(parts[1], 64)
				sec, e3 := strconv.ParseFloat(parts[2], 64)
				if e1 == nil && e2 == nil && e3 == nil {
					duration = h*3600 + m*60 + sec - start
				}
			}
		}
		if duration <= 0 {
			complete = false
			continue
		}
		first = math.Min(first, start)
		last = math.Max(last, start+duration)
	}
	if complete && len(selected) != 0 {
		return last - first, nil
	}
	duration, e := probeSeconds(probe.Format.Duration, -1)
	if e != nil {
		return 0, e
	}
	if strings.Contains(probe.Format.Name, "matroska") || strings.Contains(probe.Format.Name, "webm") {
		start, e := probeSeconds(probe.Format.Start, 0)
		if e != nil {
			return 0, e
		}
		duration -= start
	}
	return duration, nil
}

func probePlayback(ctx context.Context, tool library.Tool, file string) (previewProbe, error) {
	var probe previewProbe
	result, err := process.Capture(ctx, process.Spec{Executable: tool.Path,
		Args:     []string{"-v", "error", "-show_entries", "format=format_name,start_time,duration:stream=index,codec_type,start_time,duration:stream_tags=DURATION", "-of", "json", file},
		CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10})
	if err != nil || catalog.ValidateJSON(result.Stdout) != nil || json.Unmarshal(result.Stdout, &probe) != nil {
		return probe, contracts.Fail("unavailable")
	}
	return probe, nil
}

func probeSeconds(value string, fallback float64) (float64, error) {
	if value == "" || value == "N/A" {
		return fallback, nil
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(result) || math.IsInf(result, 0) || math.Abs(result) > 365*24*60*60 {
		return 0, contracts.Fail("timing_unavailable")
	}
	return result, nil
}

// previewPlayback creates a disposable browser-compatible view of verified
// source bytes. It is media decoding/encoding only, never a model operation.
func (a *App) previewPlayback(ctx context.Context, p *playbackEntry, entry catalog.LibraryEntry) error {
	var facts library.Facts
	if json.Unmarshal(entry.Facts, &facts) != nil {
		return contracts.Fail("invalid_request")
	}
	if entry.Class == "audio" && p.streamIndex == nil {
		for _, stream := range facts.Streams {
			if stream.Kind == "audio" {
				index := stream.Index
				p.streamIndex = &index
				break
			}
		}
	}
	if p.streamIndex != nil {
		filtered := []library.Stream{}
		for _, stream := range facts.Streams {
			if stream.Kind == "video" || stream.Kind == "audio" && stream.Index == *p.streamIndex {
				filtered = append(filtered, stream)
			}
		}
		facts.Streams = filtered
	}
	originalOffset := float64(0)
	canonicalClock := false
	if facts.Canonical != nil {
		index := -1
		for _, stream := range facts.Streams {
			if stream.Kind == "audio" {
				index = stream.Index
				break
			}
		}
		for _, track := range facts.Canonical.Tracks {
			if track.Index == index {
				start, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator)
				if !ok {
					return contracts.Fail("invalid_timing")
				}
				originalOffset, _ = start.Float64()
				canonicalClock = true
				break
			}
		}
		if !canonicalClock {
			return contracts.Fail("timing_unavailable")
		}
	}
	offset, err := directPlaybackOffset(facts)
	if err != nil {
		return err
	}
	// HTML retains explicit nonnegative media timelines. A source's nonzero
	// first timestamp is therefore not an offset to subtract from untouched
	// media. Normalize such sources into a preview with a measured clock map.
	if !previewRequired(facts, entry.Class) && math.Abs(offset) <= 0.000001 && !(entry.Class == "video" && p.streamIndex != nil) {
		p.descriptor.TimelineOffsetSeconds = originalOffset
		return nil
	}
	tools, err := ReadMediaTools(a.Workspace)
	if err != nil {
		return err
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if err = processing.VerifyFFmpeg(bounded, tools.FFmpeg); err != nil {
		return err
	}
	if err = library.VerifyExtractor(bounded, tools.FFprobe, "ffprobe"); err != nil {
		return err
	}
	source, err := probePlayback(bounded, tools.FFprobe, p.materializedPath)
	if err != nil {
		return err
	}
	offset, err = probeSeconds(source.Format.Start, 0)
	if err != nil {
		return err
	}
	if p.streamIndex != nil {
		filtered := []library.Stream{}
		for _, stream := range source.Streams {
			if stream.Kind == "video" || stream.Index == *p.streamIndex {
				filtered = append(filtered, stream)
			}
		}
		source.Streams = filtered
	}
	sourceDuration, err := previewContentDuration(source, entry.Class)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Join(a.Workspace.Control, "artifact-scratch"), "playback-preview-"+a.Session+"-*")
	if err != nil {
		return contracts.Fail("unavailable")
	}
	path := file.Name()
	file.Close()
	success := false
	defer func() {
		if !success {
			os.Remove(path)
		}
	}()
	_, err = process.Run(bounded, process.Spec{Executable: tools.FFmpeg.Path,
		Args: previewArguments(p.materializedPath, path, entry.Class, facts), Directory: filepath.Dir(path),
		CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10})
	if err != nil {
		return err
	}
	proxy, err := probePlayback(bounded, tools.FFprobe, path)
	if err != nil {
		return err
	}
	proxyDuration, err := previewContentDuration(proxy, entry.Class)
	if err != nil || sourceDuration <= 0 || proxyDuration <= 0 || math.Abs(proxyDuration-sourceDuration) > 0.5 {
		return contracts.Fail("invalid_timing")
	}
	proxyStart, err := probeSeconds(proxy.Format.Start, 0)
	if err != nil {
		return err
	}
	bytes, err := os.Open(path)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer bytes.Close()
	info, err := bytes.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() >= maxPreviewBytes {
		return contracts.Fail("output_limit")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, bytes); err != nil {
		return contracts.Fail("unavailable")
	}
	p.proxyPath = path
	p.descriptor.Path = path
	p.descriptor.Digest = hex.EncodeToString(hash.Sum(nil))
	p.descriptor.Size = info.Size()
	p.descriptor.MIME = "video/mp4"
	if entry.Class == "audio" {
		p.descriptor.MIME = "audio/wav"
	}
	p.descriptor.Preview = true
	p.descriptor.TimelineOffsetSeconds = offset - proxyStart
	if canonicalClock {
		p.descriptor.TimelineOffsetSeconds = originalOffset - proxyStart
	}
	success = true
	return nil
}
