// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/processing"
)

func TestPreviewPolicyRequiresPlayableContainerAndAudio(t *testing.T) {
	if !previewRequired(library.Facts{Extension: ".m4a", Streams: []library.Stream{{Kind: "audio", Codec: "alac"}}}, "audio") {
		t.Fatal("ALAC incorrectly assumed playable in every native webview")
	}
	if !previewRequired(library.Facts{Extension: ".mkv"}, "video") {
		t.Fatal("MKV incorrectly treated as browser media")
	}
	if !previewRequired(library.Facts{Extension: "mp4", Streams: []library.Stream{{Kind: "audio", Codec: "flac"}}}, "video") {
		t.Fatal("unsupported audio ignored")
	}
	if previewRequired(library.Facts{Extension: "mp4", Streams: []library.Stream{{Kind: "audio", Codec: "aac"}}}, "video") {
		t.Fatal("playable original transformed")
	}
	if previewRequired(library.Facts{Extension: "flac"}, "audio") {
		t.Fatal("supported audio transformed")
	}
	args := previewArguments("source with spaces", "output", "video", library.Facts{Streams: []library.Stream{{Kind: "video", Codec: "h264"}}})
	joined := strings.Join(args, " ")
	for _, required := range []string{"-nostdin", "-copyts -start_at_zero", "-c:v copy", "-c:a aac", "-fs", "-movflags +faststart"} {
		if !strings.Contains(joined, required) {
			t.Fatal("missing preview contract", required)
		}
	}
}

func TestDirectPlaybackClockUsesStreamOffsetAndExactPTS(t *testing.T) {
	pts := int64(9)
	for _, facts := range []library.Facts{
		{Streams: []library.Stream{{Kind: "audio", StartTime: "1.125"}}},
		{Streams: []library.Stream{{Kind: "audio", StartPTS: &pts, TimeBase: "1/8"}}},
		{Streams: []library.Stream{{Kind: "video", StartTime: "2"}, {Kind: "audio", StartTime: "1.125"}}},
	} {
		if value, e := directPlaybackOffset(facts); e != nil || value != 1.125 {
			t.Fatal("direct source clock offset", value, e)
		}
	}
}

func TestNativeDirectPlaybackNonzeroSourceClock(t *testing.T) {
	selected := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if selected == "" {
		t.Skip("exact native media tools fixture not selected")
	}
	raw, e := os.ReadFile(selected)
	if e != nil {
		t.Fatal(e)
	}
	var tools library.Tools
	if json.Unmarshal(raw, &tools) != nil {
		t.Fatal("selected tool config")
	}
	ctx, stop := context.WithTimeout(context.Background(), time.Minute)
	defer stop()
	if e = processing.VerifyFFmpeg(ctx, tools.FFmpeg); e != nil {
		t.Fatal(e)
	}
	if e = library.VerifyExtractor(ctx, tools.FFprobe, "ffprobe"); e != nil {
		t.Fatal(e)
	}
	source, e := filepath.Abs("../../tests/fixtures/media/speech.flac")
	if e != nil {
		t.Fatal(e)
	}
	for _, fixture := range []struct{ name, codec, format string }{{"clock.m4a", "aac", "mp4"}, {"clock.mka", "flac", "matroska"}} {
		t.Run(fixture.format, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), fixture.name)
			if _, e = process.Run(ctx, process.Spec{Executable: tools.FFmpeg.Path, Args: []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", source, "-t", "1", "-c:a", fixture.codec, "-output_ts_offset", "1.25", "-f", fixture.format, "-y", target}, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 64 << 10}); e != nil {
				t.Fatal(e)
			}
			probe, e := probePlayback(ctx, tools.FFprobe, target)
			if e != nil {
				t.Fatal(e)
			}
			start, e := probeSeconds(probe.Format.Start, 0)
			if e != nil || start <= 1 {
				t.Fatal("fixture lacks nonzero source clock", start, e)
			}
			a := configuredApp(t)
			if e = os.WriteFile(filepath.Join(a.Workspace.Control, "media-tools.json"), raw, 0600); e != nil {
				t.Fatal(e)
			}
			service, e := a.libraryService()
			if e != nil {
				t.Fatal(e)
			}
			claim, e := a.Catalog.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{}`))
			if e != nil {
				t.Fatal(e)
			}
			claim, e = a.Catalog.ClaimWork(ctx, claim.ID, contracts.ID(), time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			copyMode := false
			result, e := service.Import(ctx, claim, library.ImportRequest{Items: []library.Item{{Source: target, Options: library.Options{Copy: &copyMode}}}})
			if e != nil || len(result.Items) != 1 || result.Items[0].MediaID == "" {
				t.Fatal("clock fixture import", result, e)
			}
			entry, e := a.Catalog.Library(ctx, result.Items[0].MediaID)
			if e != nil {
				t.Fatal(e)
			}
			opened := realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision})
			if opened.Error != nil {
				t.Fatal(opened.Error)
			}
			d := opened.Result.(PlaybackDescriptor)
			if !d.Preview || d.MIME != "audio/wav" || math.Abs(d.TimelineOffsetSeconds-start) > 0.000001 {
				t.Fatal("nonzero native timeline was not normalized", d, start)
			}
			proxy, e := probePlayback(ctx, tools.FFprobe, d.Path)
			if e != nil {
				t.Fatal(e)
			}
			proxyStart, e := probeSeconds(proxy.Format.Start, 0)
			if e != nil || math.Abs(proxyStart) > 0.000001 {
				t.Fatal("preview clock is not zero", proxyStart, e)
			}
			decode := func(path string) []byte {
				t.Helper()
				out, e := process.Capture(ctx, process.Spec{Executable: tools.FFmpeg.Path, Args: []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-i", path, "-map", "0:a:0", "-c:a", "pcm_s16le", "-ac", "2", "-ar", "48000", "-f", "s16le", "pipe:1"}, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 8 << 20})
				if e != nil {
					t.Fatal(e)
				}
				return out.Stdout
			}
			sourcePCM, proxyPCM := decode(target), decode(d.Path)
			if len(sourcePCM) == 0 || !bytes.Equal(sourcePCM, proxyPCM) {
				t.Fatal("normalization changed decoded content", len(sourcePCM), len(proxyPCM))
			}
			// The source sample at native time start+0.25 must be the preview sample
			// at 0.25. Compare an audible sample window, independent of metadata.
			sourceCue := start + 0.25
			seek := sourceCue - d.TimelineOffsetSeconds
			index := int(math.Round(seek*48000)) * 4
			if index != 48000 || !bytes.Equal(sourcePCM[48000:52000], proxyPCM[index:index+4000]) {
				t.Fatal("source cue seeks to different decoded samples", seek, index)
			}
			duration, e := probeSeconds(proxy.Format.Duration, -1)
			if e != nil || math.Abs(duration-float64(len(sourcePCM))/(48000*4)) > 0.000001 {
				t.Fatal("normalization changed real sample duration", duration, len(sourcePCM), e)
			}
		})
	}
}

func TestPreviewDurationUsesSelectedStreamSpanOrContainerClock(t *testing.T) {
	for _, fixture := range []struct {
		probe previewProbe
		want  float64
	}{
		{probe: previewProbe{Format: previewFormat{Start: "1.25", Duration: "2.25", Name: "matroska,webm"}}, want: 1},
		{probe: previewProbe{Format: previewFormat{Start: "1.228", Duration: "1.021333", Name: "mov,mp4,m4a,3gp,3g2,mj2"}, Streams: []library.Stream{{Kind: "audio", StartTime: "1.228", Duration: "1.021333"}}}, want: 1.021333},
		{probe: previewProbe{Format: previewFormat{Start: "1", Duration: "6"}, Streams: []library.Stream{{Kind: "video", StartTime: "1", Duration: "3"}, {Kind: "audio", StartTime: "2", Duration: "3"}, {Kind: "audio", StartTime: "10", Duration: "100"}}}, want: 4},
	} {
		value, e := previewContentDuration(fixture.probe, "video")
		if e != nil || math.Abs(value-fixture.want) > 0.000001 {
			t.Fatal("content duration differs from selected stream span", value, fixture.want, e)
		}
	}
}

func TestPreviewSourceClockRejectsNonfiniteOrUnboundedOffsets(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "-Inf", "999999999999999999999"} {
		if _, err := probeSeconds(value, 0); err == nil {
			t.Fatal("invalid source clock", value)
		}
	}
	if value, err := probeSeconds("1.125", 0); err != nil || math.Abs(value-1.125) > 0.0001 {
		t.Fatal("valid original offset lost")
	}
}
