// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
)

type Observation struct {
	Family     string          `json:"family"`
	GroupPath  []string        `json:"group_path"`
	Tag        string          `json:"tag"`
	Occurrence int             `json:"occurrence"`
	Raw        json.RawMessage `json:"raw_value"`
	Normalized json.RawMessage `json:"normalized"`
}
type Extraction struct {
	Tool         string   `json:"tool"`
	Version      string   `json:"version"`
	BinarySHA256 string   `json:"binary_sha256"`
	Options      []string `json:"options"`
	Stdout       []byte   `json:"stdout"`
	Stderr       []byte   `json:"stderr"`
	State        string   `json:"state"`
	ExitCode     int      `json:"exit_code"`
	Truncated    bool     `json:"truncated"`
}
type Stream struct {
	Index         int                        `json:"index"`
	Kind          string                     `json:"codec_type"`
	Codec         string                     `json:"codec_name"`
	SampleRate    string                     `json:"sample_rate"`
	Channels      *int64                     `json:"channels"`
	ChannelLayout string                     `json:"channel_layout"`
	TimeBase      string                     `json:"time_base"`
	StartPTS      *int64                     `json:"start_pts"`
	DurationTS    *int64                     `json:"duration_ts"`
	StartTime     string                     `json:"start_time"`
	Duration      string                     `json:"duration"`
	Tags          map[string]json.RawMessage `json:"tags"`
}
type Facts struct {
	DurationUS         *int64       `json:"duration_us"`
	Streams            []Stream     `json:"streams"`
	Container          string       `json:"container"`
	SourceClock        string       `json:"source_clock"`
	DurationRounding   string       `json:"duration_rounding"`
	Extension          string       `json:"extension"`
	FilesystemModified string       `json:"filesystem_modified"`
	Acquisition        *Acquisition `json:"acquisition,omitempty"`
	SubtitleState      string       `json:"subtitle_state,omitempty"`
	Options            Options      `json:"import_options"`
}
type MIMEObservation struct {
	Basis string `json:"basis"`
	Value string `json:"value"`
}
type MIMEResolution struct {
	Resolver     string            `json:"resolver"`
	Version      string            `json:"version"`
	Preferred    string            `json:"preferred_mime_type"`
	Observations []MIMEObservation `json:"observations"`
	Disagreement bool              `json:"disagreement"`
}
type Metadata struct {
	CaptureState string          `json:"capture_state"`
	CapturedAt   catalog.Instant `json:"captured_at"`
	Reports      []Extraction    `json:"reports"`
	Observations []Observation   `json:"observations"`
	MIME         MIMEResolution  `json:"mime_resolution"`
	Warnings     []string        `json:"warnings"`
	Truncated    bool            `json:"truncated"`
}
type captureBundle struct {
	Metadata     Metadata `json:"metadata"`
	Facts        Facts    `json:"facts"`
	SourceDigest string   `json:"source_digest"`
	SourceSize   int64    `json:"source_size"`
}

// VerifyExtractor applies the same pinned extractor contract to runtime callers.
func VerifyExtractor(ctx context.Context, tool Tool, name string) error {
	return verifyTool(ctx, tool, name)
}

func verifyTool(ctx context.Context, tool Tool, name string) error {
	if !filepath.IsAbs(tool.Path) || tool.Version == "" || len(tool.SHA256) != 64 {
		return contracts.Fail("unavailable")
	}
	f, e := os.Open(tool.Path)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil || hex.EncodeToString(h.Sum(nil)) != tool.SHA256 {
		return contracts.Fail("incompatible_version")
	}
	for _, support := range tool.SupportFiles {
		if !filepath.IsAbs(support.Path) || len(support.SHA256) != 64 {
			return contracts.Fail("invalid_request")
		}
		file, e := os.Open(support.Path)
		if e != nil {
			return contracts.Fail("unavailable")
		}
		sum := sha256.New()
		_, readErr := io.Copy(sum, file)
		file.Close()
		if readErr != nil || hex.EncodeToString(sum.Sum(nil)) != support.SHA256 {
			return contracts.Fail("incompatible_version")
		}
	}
	if tool.Interpreter != nil {
		support := tool.Interpreter
		if !filepath.IsAbs(support.Path) || len(support.SHA256) != 64 {
			return contracts.Fail("invalid_request")
		}
		data, e := os.ReadFile(support.Path)
		if e != nil {
			return contracts.Fail("unavailable")
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != support.SHA256 {
			return contracts.Fail("incompatible_version")
		}
	}
	args := []string{"-version"}
	if name == "exiftool" {
		args = []string{"-config", "", "-ver"}
	}
	executable, effectiveArgs := toolCommand(tool, args)
	result, e := process.Capture(ctx, process.Spec{Executable: executable, Args: effectiveArgs, MaxOutput: 64 << 10, Env: toolEnvironment})
	if e != nil {
		return contracts.Fail("unavailable")
	}
	version := strings.TrimSpace(string(result.Stdout))
	if name == "ffprobe" {
		fields := strings.Fields(version)
		if len(fields) < 3 || fields[0] != "ffprobe" || fields[1] != "version" {
			return contracts.Fail("incompatible_version")
		}
		version = fields[2]
	}
	if version != tool.Version {
		return contracts.Fail("incompatible_version")
	}
	return nil
}

var toolEnvironment = []string{"PERL5OPT=", "PERL5LIB=", "PERLLIB=", "EXIFTOOL_HOME=", "LC_ALL=C", "LANG=C", "LD_PRELOAD=", "DYLD_INSERT_LIBRARIES="}

func toolCommand(tool Tool, args []string) (string, []string) {
	if tool.Interpreter != nil {
		return tool.Interpreter.Path, append([]string{tool.Path}, args...)
	}
	return tool.Path, args
}
func captureTool(ctx context.Context, name string, tool Tool, args []string, max int) Extraction {
	recorded := append([]string{}, args...)
	if len(recorded) > 0 {
		recorded[len(recorded)-1] = "<stable-original-bytes>"
	}
	out := Extraction{Tool: name, Version: tool.Version, BinarySHA256: tool.SHA256, Options: recorded, State: "unsupported", ExitCode: -1}
	timeout, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if verifyTool(timeout, tool, name) != nil {
		return out
	}
	executable, effectiveArgs := toolCommand(tool, args)
	result, e := process.Capture(timeout, process.Spec{Executable: executable, Args: effectiveArgs, MaxOutput: max, Env: toolEnvironment})
	out.Stdout = result.Stdout
	out.Stderr = result.Stderr
	out.ExitCode = result.ExitCode
	out.Truncated = result.Truncated
	out.State = "captured"
	if e != nil {
		out.State = "failed"
	}
	if result.Truncated {
		out.State = "partial"
	}
	return out
}
func (s *Service) capture(ctx context.Context, path, original string, options Options) (captureBundle, error) {
	if (s.Tools.Kind != "" && s.Tools.Kind != "media-tools") || (s.Tools.Version != "" && s.Tools.Version != contracts.Version) {
		return captureBundle{}, contracts.Fail("incompatible_version")
	}
	now := time.Now().UTC()
	bundle := captureBundle{Metadata: Metadata{CaptureState: "unsupported", CapturedAt: catalog.Instant{ISO: now.Format(time.RFC3339Nano), UnixNS: now.UnixNano()}, Reports: []Extraction{}, Observations: []Observation{}, Warnings: []string{}}, Facts: Facts{Streams: []Stream{}, SourceClock: "original-media-presentation-time", DurationRounding: "nearest-microsecond-half-up", Extension: strings.ToLower(filepath.Ext(original)), Options: options}}
	exifArgs := []string{"-config", "", "-json", "-a", "-G0:1:3:4:5", "-struct", "-u", "-ee3", "-api", "QuickTimeUTC=0", "--", path}
	probeArgs := []string{"-v", "error", "-protocol_whitelist", "file", "-show_error", "-show_format", "-show_streams", "-show_chapters", "-of", "json", path}
	exif := captureTool(ctx, "exiftool", s.Tools.ExifTool, exifArgs, s.MaxOutput)
	probe := captureTool(ctx, "ffprobe", s.Tools.FFprobe, probeArgs, s.MaxOutput)
	if ctx.Err() != nil {
		return bundle, contracts.Fail("cancelled")
	}
	bundle.Metadata.Reports = []Extraction{exif, probe}
	extractorMIME := ""
	if exif.State == "captured" {
		observations, mime, e := parseExif(exif.Stdout)
		if e != nil {
			exif.State = "failed"
			bundle.Metadata.Reports[0].State = "failed"
		} else {
			bundle.Metadata.Observations = observations
			extractorMIME = mime
		}
	}
	if probe.State == "captured" {
		facts, e := parseProbe(probe.Stdout)
		if e != nil {
			probe.State = "failed"
			bundle.Metadata.Reports[1].State = "failed"
		} else {
			facts.Extension = bundle.Facts.Extension
			facts.Options = options
			bundle.Facts = facts
			bundle.Metadata.Observations = append(bundle.Metadata.Observations, probeTagObservations(probe.Stdout)...)
		}
	}
	switch {
	case exif.Truncated || probe.Truncated:
		bundle.Metadata.CaptureState = "partial"
		bundle.Metadata.Truncated = true
	case exif.State == "captured" && probe.State == "captured":
		bundle.Metadata.CaptureState = "captured"
		if len(bundle.Metadata.Observations) == 0 {
			bundle.Metadata.CaptureState = "no-embedded-metadata"
		}
	case exif.State == "captured" || probe.State == "captured":
		bundle.Metadata.CaptureState = "partial"
	case exif.State == "failed" || probe.State == "failed":
		bundle.Metadata.CaptureState = "failed"
	}
	for _, report := range bundle.Metadata.Reports {
		if report.State != "captured" {
			bundle.Metadata.Warnings = append(bundle.Metadata.Warnings, report.Tool+": "+report.State)
		}
		if len(report.Stderr) > 0 {
			bundle.Metadata.Warnings = append(bundle.Metadata.Warnings, report.Tool+": diagnostics retained in raw report")
		}
	}
	if stat, e := os.Stat(original); e == nil {
		bundle.Facts.FilesystemModified = stat.ModTime().UTC().Format(time.RFC3339Nano)
	}
	bundle.Metadata.MIME = resolveMIME(path, bundle.Facts.Extension, extractorMIME, bundle.Facts.Container)
	digest, size, e := identity(ctx, path)
	if e != nil {
		return bundle, e
	}
	bundle.SourceDigest = digest
	bundle.SourceSize = size
	return bundle, nil
}
func parseExif(raw []byte) ([]Observation, string, error) {
	var objects []map[string]json.RawMessage
	if strict(raw, &objects) != nil || len(objects) != 1 {
		return nil, "", contracts.Fail("operation_failed")
	}
	observations := []Observation{}
	mime := ""
	keys := []string{}
	for key := range objects[0] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "SourceFile" {
			continue
		}
		parts := strings.Split(key, ":")
		tag := parts[len(parts)-1]
		family := parts[0]
		if tag == "MIMEType" {
			json.Unmarshal(objects[0][key], &mime)
		}
		if family == "File" || family == "System" || family == "ExifTool" || family == "Composite" {
			continue
		}
		groups := parts[:len(parts)-1]
		if len(groups) == 0 {
			groups = []string{family}
		}
		observations = append(observations, Observation{Family: family, GroupPath: groups, Tag: tag, Raw: objects[0][key], Normalized: marshal(map[string]any{"value": json.RawMessage(objects[0][key]), "method": "preserve-literal", "method_version": "1"})})
	}
	return observations, mime, nil
}
func parseProbe(raw []byte) (Facts, error) {
	var probe struct {
		Format struct {
			Duration string `json:"duration"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []Stream `json:"streams"`
	}
	// ffprobe fields vary by codec; retain complete bytes while interpreting known fields.
	if catalog.ValidateJSON(raw) != nil || json.Unmarshal(raw, &probe) != nil {
		return Facts{}, contracts.Fail("operation_failed")
	}
	duration, e := decimalUS(probe.Format.Duration)
	if e != nil {
		return Facts{}, e
	}
	return Facts{DurationUS: duration, Streams: probe.Streams, Container: probe.Format.Name, SourceClock: "original-media-presentation-time", DurationRounding: "nearest-microsecond-half-up"}, nil
}

var decimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:[eE][+-]?[0-9]{1,2})?$`)

func decimalUS(value string) (*int64, error) {
	if value == "" || value == "N/A" {
		return nil, nil
	}
	if len(value) > 128 || !decimalPattern.MatchString(value) {
		return nil, contracts.Fail("invalid_request")
	}
	rat, ok := new(big.Rat).SetString(value)
	if !ok || rat.Sign() < 0 {
		return nil, contracts.Fail("invalid_request")
	}
	numerator := new(big.Int).Mul(rat.Num(), big.NewInt(1000000))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, rat.Denom(), remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(rat.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return nil, contracts.Fail("invalid_request")
	}
	n := quotient.Int64()
	return &n, nil
}
func probeTagObservations(raw []byte) []Observation {
	var doc struct {
		Format struct {
			Tags map[string]json.RawMessage `json:"tags"`
		} `json:"format"`
		Streams []Stream `json:"streams"`
	}
	json.Unmarshal(raw, &doc)
	out := []Observation{}
	add := func(group string, tags map[string]json.RawMessage) {
		keys := []string{}
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, Observation{Family: "container", GroupPath: []string{"ffprobe", group}, Tag: k, Raw: tags[k], Normalized: marshal(map[string]any{"value": tags[k], "method": "preserve-literal", "method_version": "1"})})
		}
	}
	add("format", doc.Format.Tags)
	for _, stream := range doc.Streams {
		add("stream-"+new(big.Int).SetInt64(int64(stream.Index)).String(), stream.Tags)
	}
	return out
}
func embeddedDates(observations []Observation, options Options) []Date {
	out := []Date{}
	for _, observation := range observations {
		switch strings.ToLower(observation.Tag) {
		case "datetimeoriginal", "createdate", "creationdate", "creation_time", "recordingdate", "originationdate", "recordingtime":
		default:
			continue
		}
		var literal string
		if json.Unmarshal(observation.Raw, &literal) != nil {
			continue
		}
		value := literal
		if len(value) >= 10 && value[4] == ':' && value[7] == ':' {
			value = value[:4] + "-" + value[5:7] + "-" + value[8:]
		}
		if len(value) > 10 && value[10] == ' ' {
			value = value[:10] + "T" + value[11:]
		}
		precision := "instant"
		if len(value) == 10 {
			precision = "day"
		}
		basis := "embedded-import-zone"
		zone := options.Timezone
		if _, e := time.Parse(time.RFC3339Nano, value); e == nil {
			basis = "embedded-own-zone"
			zone = ""
		}
		d := ResolveDate(value, precision, zone, options.DSTFold, options.DSTGap, basis)
		d.Literal = literal
		out = append(out, d)
	}
	return out
}
func resolveMIME(path, extension, extractor, container string) MIMEResolution {
	out := MIMEResolution{Resolver: "insonic-media-type", Version: "1", Observations: []MIMEObservation{}}
	byExtension := map[string]string{".wav": "audio/wav", ".mp3": "audio/mpeg", ".flac": "audio/flac", ".mp4": "video/mp4", ".m4a": "audio/mp4", ".mov": "video/quicktime", ".ogg": "audio/ogg", ".webm": "video/webm", ".mkv": "video/x-matroska"}
	add := func(basis, value string) {
		if value != "" {
			out.Observations = append(out.Observations, MIMEObservation{basis, value})
		}
	}
	add("filename-extension", byExtension[extension])
	if f, e := os.Open(path); e == nil {
		buffer := make([]byte, 512)
		n, _ := f.Read(buffer)
		f.Close()
		add("byte-detected", http.DetectContentType(buffer[:n]))
	}
	add("extractor", extractor)
	for _, o := range out.Observations {
		if strings.HasPrefix(o.Value, "audio/") || strings.HasPrefix(o.Value, "video/") {
			out.Preferred = o.Value
		}
	}
	if out.Preferred == "" {
		for _, mapping := range []struct{ format, mime string }{{"wav", "audio/wav"}, {"mp3", "audio/mpeg"}, {"flac", "audio/flac"}, {"ogg", "audio/ogg"}, {"webm", "video/webm"}, {"matroska", "video/x-matroska"}, {"mov", "video/quicktime"}, {"mp4", "video/mp4"}} {
			if strings.Contains(container, mapping.format) {
				out.Preferred = mapping.mime
				break
			}
		}
	}
	if out.Preferred == "" {
		out.Preferred = "application/octet-stream"
	}
	for _, o := range out.Observations {
		if o.Value != out.Preferred {
			out.Disagreement = true
		}
	}
	return out
}
