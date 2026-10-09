// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
)

type EmbeddedTrack struct {
	Index     int    `json:"index"`
	Codec     string `json:"codec"`
	Language  string `json:"language,omitempty"`
	Default   bool   `json:"default"`
	Forced    bool   `json:"forced"`
	Supported bool   `json:"supported"`
	Selected  bool   `json:"selected"`
}

func embeddedTracks(streams []Stream) []EmbeddedTrack {
	tracks := []EmbeddedTrack{}
	for _, stream := range streams {
		if stream.Kind != "subtitle" {
			continue
		}
		var language string
		json.Unmarshal(stream.Tags["language"], &language)
		supported := false
		switch stream.Codec {
		case "subrip", "webvtt", "ass", "ssa", "mov_text", "text":
			supported = true
		}
		tracks = append(tracks, EmbeddedTrack{Index: stream.Index, Codec: stream.Codec, Language: language, Default: stream.Disposition["default"] == 1, Forced: stream.Disposition["forced"] == 1, Supported: supported})
	}
	sort.SliceStable(tracks, func(i, j int) bool { return tracks[i].Index < tracks[j].Index })
	return tracks
}
func selectEmbedded(tracks []EmbeddedTrack, options Options) (*EmbeddedTrack, error) {
	candidates := []EmbeddedTrack{}
	for _, track := range tracks {
		if options.SubtitleStreamIndex != nil && track.Index != *options.SubtitleStreamIndex {
			continue
		}
		if options.SubtitleLanguage != "" && !strings.EqualFold(track.Language, options.SubtitleLanguage) {
			continue
		}
		if !track.Supported {
			if options.SubtitleStreamIndex != nil {
				return nil, contracts.Fail("unsupported_capability")
			}
			continue
		}
		candidates = append(candidates, track)
	}
	if len(candidates) == 0 {
		if options.SubtitleStreamIndex != nil || options.SubtitleLanguage != "" {
			return nil, contracts.Fail("not_found")
		}
		return nil, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Default != candidates[j].Default {
			return candidates[i].Default
		}
		if candidates[i].Forced != candidates[j].Forced {
			return !candidates[i].Forced
		}
		return candidates[i].Index < candidates[j].Index
	})
	chosen := candidates[0]
	chosen.Selected = true
	return &chosen, nil
}
func (s *Service) extractEmbedded(ctx context.Context, path, directory string, track EmbeddedTrack) ([]byte, string, error) {
	format, codec := "srt", "srt"
	if track.Codec == "ass" || track.Codec == "ssa" {
		format = "ass"
		codec = "ass"
	}
	if track.Codec == "webvtt" {
		format = "vtt"
		codec = "webvtt"
	}
	if err := verifyTool(ctx, s.Tools.FFmpeg, "ffmpeg"); err != nil {
		return nil, "", err
	}
	output := filepath.Join(directory, "embedded."+format)
	max := int64(16 << 20)
	// -fs is a physical output bound; a boundary-sized result is rejected before ingestion.
	args := []string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file", "-copyts", "-i", path, "-map", "0:" + strconv.Itoa(track.Index), "-vn", "-an", "-dn", "-map_metadata", "-1", "-c:s", codec, "-fs", strconv.FormatInt(max, 10), output}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if _, err := process.Capture(bounded, process.Spec{Executable: s.Tools.FFmpeg.Path, Args: args, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10}); err != nil {
		return nil, "", err
	}
	data, err := readTranscript(output, max)
	if err != nil || int64(len(data)) >= max {
		return nil, "", contracts.Fail("output_limit")
	}
	return data, format, nil
}
