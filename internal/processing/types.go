// SPDX-License-Identifier: Apache-2.0
// Package processing provides disposable mapped-audio and local inference
// sessions. It does not persist transcripts, assignments, or current results.
package processing

import (
	"context"
	"sync"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/library"
)

type Config struct {
	FFmpeg            library.Tool       `json:"ffmpeg"`
	RecognitionPython library.PinnedFile `json:"recognition_python"`
	DiarizationPython library.PinnedFile `json:"diarization_python"`
	Worker            library.PinnedFile `json:"worker"`
	MaxInputBytes     int64              `json:"max_input_bytes,omitempty"`
	MaxDurationUS     int64              `json:"max_duration_us,omitempty"`
	MaxOutputBytes    int                `json:"max_output_bytes,omitempty"`
	TimeoutMS         int64              `json:"timeout_ms,omitempty"`
	Threads           int                `json:"threads,omitempty"`
}

type AudioOptions struct {
	// Nil selects the first audio stream; an explicit value is an absolute
	// ffprobe stream index, including when the source also contains video.
	StreamIndex *int `json:"stream_index,omitempty"`
	// Nil explicitly downmixes selected stream channels to mono. A value
	// selects one zero-based source channel before resampling.
	Channel *int `json:"channel,omitempty"`
}

type SourceMap struct {
	StartNumerator   string `json:"start_numerator"`
	StartDenominator string `json:"start_denominator"`
	SampleCount      int64  `json:"sample_count"`
	SampleRate       int64  `json:"sample_rate"`
	StreamIndex      int    `json:"stream_index"`
	Channel          *int   `json:"channel"`
	Policy           string `json:"policy"`
}

type Diagnostic struct {
	Code  string   `json:"code"`
	Count int64    `json:"count,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type RecognitionOptions struct {
	Device   string `json:"device,omitempty"`
	Language string `json:"language,omitempty"`
}

type DiarizationOptions struct {
	Device      string `json:"device,omitempty"`
	MinSpeakers int    `json:"min_speakers,omitempty"`
	MaxSpeakers int    `json:"max_speakers,omitempty"`
}

type Turn struct {
	Label   string `json:"label"`
	StartUS int64  `json:"start_us"`
	EndUS   int64  `json:"end_us"`
}

type RecognitionResult struct {
	SRT         []byte         `json:"-"`
	NoSpeech    bool           `json:"no_speech"`
	Provenance  map[string]any `json:"provenance"`
	Diagnostics []Diagnostic   `json:"diagnostics"`
}

type DiarizationResult struct {
	Turns       []Turn         `json:"-"`
	NoSpeech    bool           `json:"no_speech"`
	Provenance  map[string]any `json:"provenance"`
	Diagnostics []Diagnostic   `json:"diagnostics"`
}

type Service struct {
	Artifacts *artifact.Service
	Catalog   catalog.Catalog
	Config    Config
}

type Session struct {
	MappedPath  string
	SourceMap   SourceMap
	Provenance  map[string]any
	Diagnostics []Diagnostic
	service     *Service
	directory   string
	ctx         context.Context
	cancel      context.CancelFunc
	leases      []artifact.Materialization
	mu          sync.Mutex
	closed      bool
	stop        chan struct{}
	done        chan struct{}
	operations  sync.WaitGroup
}
