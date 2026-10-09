// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
)

type CanonicalTrack struct {
	SourceIndex      int    `json:"source_index"`
	Index            int    `json:"index"`
	StartNumerator   string `json:"source_start_numerator"`
	StartDenominator string `json:"source_start_denominator"`
}
type Canonical struct {
	Policy        string           `json:"policy"`
	Format        string           `json:"format"`
	SourceDigest  string           `json:"source_digest"`
	SourceSize    int64            `json:"source_size"`
	SourceStreams []Stream         `json:"source_streams"`
	Tracks        []CanonicalTrack `json:"tracks"`
	ToolVersion   string           `json:"tool_version"`
	ToolSHA256    string           `json:"tool_sha256"`
}

func streamStart(stream Stream) (*big.Rat, error) {
	if stream.StartPTS != nil && stream.TimeBase != "" {
		base, ok := new(big.Rat).SetString(stream.TimeBase)
		if ok && base.Sign() > 0 {
			return base.Mul(base, new(big.Rat).SetInt64(*stream.StartPTS)), nil
		}
	}
	if stream.StartTime != "" {
		if start, ok := new(big.Rat).SetString(stream.StartTime); ok {
			return start, nil
		}
	}
	return nil, contracts.Fail("timing_unavailable")
}
func canonicalPolicy(streams []Stream) (string, string, error) {
	audio := []Stream{}
	for _, stream := range streams {
		if stream.Kind == "audio" {
			audio = append(audio, stream)
		}
	}
	if len(audio) == 0 || len(audio) > 64 {
		return "", "", contracts.Fail("unsupported_audio")
	}
	allFLAC := true
	for _, stream := range audio {
		if stream.Channels == nil || *stream.Channels < 1 || *stream.Channels > 8 {
			return "", "", contracts.Fail("unsupported_audio")
		}
		if stream.Codec != "flac" {
			allFLAC = false
		}
		if strings.HasPrefix(stream.Codec, "pcm_f") {
			return "", "", contracts.Fail("unsupported_audio")
		}
	}
	if len(audio) > 1 {
		if allFLAC {
			return "mka", "copy", nil
		}
		return "mka", "flac", nil
	}
	if allFLAC {
		return "flac", "copy", nil
	}
	stream := audio[0]
	if stream.Channels == nil || *stream.Channels < 1 {
		return "", "", contracts.Fail("invalid_audio")
	}
	if *stream.Channels > 1 {
		return "flac", "flac", nil
	}
	if stream.SampleRate != "" {
		rate, err := strconv.Atoi(stream.SampleRate)
		if err != nil || rate < 8000 {
			return "", "", contracts.Fail("invalid_audio")
		}
		supported := false
		for _, candidate := range []int{8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000} {
			if rate == candidate {
				supported = true
			}
		}
		if !supported {
			return "flac", "flac", nil
		}
	}
	switch stream.Codec {
	case "mp3":
		return "mp3", "copy", nil
	case "aac", "ac3", "eac3", "opus", "vorbis", "mp2", "wmav1", "wmav2", "wmapro", "amr_nb", "amr_wb", "gsm":
		return "mp3", "libmp3lame", nil
	}
	return "flac", "flac", nil
}
func (s *Service) canonicalize(ctx context.Context, path, directory string, bundle captureBundle) (string, Facts, error) {
	format, codec, err := canonicalPolicy(bundle.Facts.Streams)
	if err != nil {
		return "", Facts{}, err
	}
	if err = verifyTool(ctx, s.Tools.FFmpeg, "ffmpeg"); err != nil {
		return "", Facts{}, err
	}
	output := filepath.Join(directory, "canonical."+format)
	args := []string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file", "-copyts", "-i", path}
	audio := []Stream{}
	for _, stream := range bundle.Facts.Streams {
		if stream.Kind == "audio" {
			audio = append(audio, stream)
			args = append(args, "-map", "0:"+strconv.Itoa(stream.Index))
		}
	}
	args = append(args, "-map_metadata", "-1")
	for i, stream := range audio {
		var language string
		json.Unmarshal(stream.Tags["language"], &language)
		if language != "" {
			args = append(args, "-metadata:s:a:"+strconv.Itoa(i), "language="+language)
		}
		disposition := "0"
		if stream.Disposition["default"] == 1 {
			disposition = "default"
		}
		if stream.Disposition["forced"] == 1 {
			if disposition == "0" {
				disposition = "forced"
			} else {
				disposition += "+forced"
			}
		}
		args = append(args, "-disposition:a:"+strconv.Itoa(i), disposition)
	}
	args = append(args, "-vn", "-sn", "-dn", "-map_chapters", "-1", "-c:a", codec, "-threads", "2")
	if codec == "libmp3lame" {
		args = append(args, "-q:a", "0")
	}
	if codec == "flac" {
		for i, stream := range audio {
			bits := stream.BitsPerSample
			if n, err := strconv.Atoi(stream.BitsPerRawSample); err == nil && n > bits {
				bits = n
			}
			if bits > 32 {
				return "", Facts{}, contracts.Fail("unsupported_audio")
			}
			if bits > 24 {
				args = append(args, "-bits_per_raw_sample:a:"+strconv.Itoa(i), strconv.Itoa(bits), "-strict:a:"+strconv.Itoa(i), "experimental")
			}
		}
	}
	if format == "mka" {
		args = append(args, "-f", "matroska", "-avoid_negative_ts", "disabled")
	} else {
		args = append(args, "-f", format)
	}
	args = append(args, "-fs", strconv.FormatInt(64<<30, 10), output)
	bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if _, err = process.Capture(bounded, process.Spec{Executable: s.Tools.FFmpeg.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10}); err != nil {
		return "", Facts{}, err
	}
	report := captureTool(ctx, "ffprobe", s.Tools.FFprobe, []string{"-v", "error", "-protocol_whitelist", "file", "-show_format", "-show_streams", "-of", "json", output}, s.MaxOutput)
	if report.State != "captured" {
		return "", Facts{}, contracts.Fail("invalid_audio")
	}
	facts, err := parseProbe(report.Stdout)
	if err != nil || len(facts.Streams) != len(audio) {
		return "", Facts{}, contracts.Fail("invalid_audio")
	}
	provenance := &Canonical{Policy: "canonical-audio-v1;stereo-flac;no-dsp;native-rates;independent-tracks", Format: format, SourceDigest: bundle.SourceDigest, SourceSize: bundle.SourceSize, SourceStreams: bundle.Facts.Streams, Tracks: []CanonicalTrack{}, ToolVersion: s.Tools.FFmpeg.Version, ToolSHA256: s.Tools.FFmpeg.SHA256}
	for i, stream := range facts.Streams {
		source := audio[i]
		expectedCodec := codec
		if codec == "copy" {
			expectedCodec = source.Codec
		}
		if codec == "libmp3lame" {
			expectedCodec = "mp3"
		}
		if stream.Kind != "audio" || stream.Codec != expectedCodec || stream.SampleRate != source.SampleRate || stream.Channels == nil || source.Channels == nil || *stream.Channels != *source.Channels {
			return "", Facts{}, contracts.Fail("invalid_audio")
		}
		start, err := streamStart(source)
		// Elementary WAV/FLAC streams have an intrinsic sample-zero origin and
		// may omit presentation timestamps entirely.
		if err != nil && (bundle.Facts.Container == "wav" || bundle.Facts.Container == "flac") {
			start = new(big.Rat)
			err = nil
		}
		if err != nil {
			return "", Facts{}, err
		}
		provenance.Tracks = append(provenance.Tracks, CanonicalTrack{source.Index, stream.Index, start.Num().String(), start.Denom().String()})
	}
	facts.Canonical = provenance
	facts.Extension = "." + format
	facts.Options = bundle.Facts.Options
	facts.Acquisition = bundle.Facts.Acquisition
	facts.FilesystemModified = bundle.Facts.FilesystemModified
	facts.SubtitleState = bundle.Facts.SubtitleState
	if info, err := os.Stat(output); err != nil || info.Size() < 1 || info.Size() >= 64<<30 {
		return "", Facts{}, contracts.Fail("audio_limit")
	}
	return output, facts, nil
}

// Scrub invocation paths while retaining captured source tags and exact dates.
func scrubBundle(bundle *captureBundle, paths ...string) {
	replace := func(value string) string {
		for _, path := range paths {
			if path != "" {
				value = strings.ReplaceAll(value, path, "<source>")
				value = strings.ReplaceAll(value, filepath.ToSlash(path), "<source>")
			}
		}
		return value
	}
	var walk func(any) any
	walk = func(value any) any {
		switch v := value.(type) {
		case map[string]any:
			for key, x := range v {
				if key == "filename" || strings.HasSuffix(key, "SourceFile") {
					v[key] = "<source>"
				} else {
					v[key] = walk(x)
				}
			}
		case []any:
			for i, x := range v {
				v[i] = walk(x)
			}
		case string:
			return replace(v)
		}
		return value
	}
	factsRaw := marshal(bundle.Facts)
	var factsValue any
	decoder := json.NewDecoder(bytes.NewReader(factsRaw))
	decoder.UseNumber()
	if decoder.Decode(&factsValue) == nil {
		clean, _ := json.Marshal(walk(factsValue))
		json.Unmarshal(clean, &bundle.Facts)
	}
	for i := range bundle.Metadata.Reports {
		r := &bundle.Metadata.Reports[i]
		if json.Valid(r.Stdout) {
			decoder := json.NewDecoder(strings.NewReader(string(r.Stdout)))
			decoder.UseNumber()
			var raw any
			if decoder.Decode(&raw) == nil {
				r.Stdout, _ = json.Marshal(walk(raw))
			}
		} else {
			r.Stdout = []byte(replace(string(r.Stdout)))
		}
		r.Stderr = []byte(replace(string(r.Stderr)))
		for j, arg := range r.Options {
			r.Options[j] = replace(arg)
		}
	}
	for i := range bundle.Metadata.Observations {
		o := &bundle.Metadata.Observations[i]
		for _, raw := range []*json.RawMessage{&o.Raw, &o.Normalized} {
			var value any
			decoder := json.NewDecoder(strings.NewReader(string(*raw)))
			decoder.UseNumber()
			if decoder.Decode(&value) == nil {
				*raw, _ = json.Marshal(walk(value))
			}
		}
	}
}
