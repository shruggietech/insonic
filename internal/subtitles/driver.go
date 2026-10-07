// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
)

type Tool struct {
	Executable       string `json:"executable"`
	ExecutableSHA256 string `json:"executable_sha256"`
}
type Driver struct {
	tool     Tool
	mu       sync.Mutex
	verified bool
}
type Export struct {
	Bytes       []byte       `json:"bytes"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func New(tool Tool) (*Driver, error) {
	if !filepath.IsAbs(tool.Executable) || !digestPattern.MatchString(tool.ExecutableSHA256) {
		return nil, contracts.Fail("invalid_request")
	}
	driver := &Driver{tool: tool}
	if err := driver.checkExecutable(); err != nil {
		return nil, err
	}
	return driver, nil
}
func (d *Driver) checkExecutable() error {
	if d == nil {
		return contracts.Fail("invalid_request")
	}
	info, err := os.Lstat(d.tool.Executable)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return contracts.Fail("unavailable")
	}
	file, err := os.Open(d.tool.Executable)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(file, (64<<20)+1)); err != nil {
		return contracts.Fail("unavailable")
	}
	if hex.EncodeToString(hash.Sum(nil)) != d.tool.ExecutableSHA256 {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (d *Driver) run(ctx context.Context, directory string, args ...string) (process.CaptureResult, error) {
	if ctx == nil {
		return process.CaptureResult{}, contracts.Fail("invalid_request")
	}
	if err := d.checkExecutable(); err != nil {
		return process.CaptureResult{}, err
	}
	boundedContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return process.Capture(boundedContext, process.Spec{Executable: d.tool.Executable, Args: append([]string{"--no-color"}, args...), Directory: directory, MaxOutput: MaxDocumentBytes})
}
func (d *Driver) Verify(ctx context.Context) error {
	if ctx == nil || d == nil {
		return contracts.Fail("invalid_request")
	}
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if err := d.checkExecutable(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.verified {
		return nil
	}
	version, err := d.run(ctx, "", "version")
	if err != nil {
		return err
	}
	if !bytes.Equal(version.Stdout, []byte(SchemaVersion+"\n")) {
		return contracts.Fail("incompatible_version")
	}
	schema, err := d.run(ctx, "", "schema")
	if err != nil {
		return err
	}
	if !bytes.Equal(schema.Stdout, schemaBytes) {
		return contracts.Fail("incompatible_version")
	}
	d.verified = true
	return nil
}
func privateInput(data []byte, suffix string) (directory, path string, err error) {
	if len(data) > MaxDocumentBytes {
		return "", "", contracts.Fail("output_limit")
	}
	directory, err = os.MkdirTemp("", "insonic-subtitles-")
	if err != nil {
		return "", "", contracts.Fail("operation_failed")
	}
	path = filepath.Join(directory, "source"+suffix)
	if err = os.WriteFile(path, data, 0600); err != nil {
		os.RemoveAll(directory)
		return "", "", contracts.Fail("operation_failed")
	}
	return directory, path, nil
}
func nativeFormat(format string) string {
	switch strings.ToLower(format) {
	case "srt", "subrip":
		return "srt"
	case "vtt", "webvtt":
		return "vtt"
	case "ass":
		return "ass"
	case "ssa":
		return "ssa"
	}
	return ""
}
func (d *Driver) Ingest(ctx context.Context, source []byte, format string) (json.RawMessage, error) {
	format = nativeFormat(format)
	if format == "" || len(source) == 0 {
		return nil, contracts.Fail("invalid_request")
	}
	if err := d.Verify(ctx); err != nil {
		return nil, err
	}
	directory, path, err := privateInput(source, "."+format)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	captured, err := d.run(ctx, directory, "encode", "--format", format, "--output", "-", path)
	if err != nil {
		return nil, err
	}
	if err = validateDocument(captured.Stdout, true); err != nil {
		return nil, err
	}
	var decoded semanticDocument
	json.Unmarshal(captured.Stdout, &decoded)
	if len(decoded.Cues) == 0 {
		return nil, nil
	}
	return bytes.Clone(captured.Stdout), nil
}
func (d *Driver) Validate(ctx context.Context, document []byte) error {
	if err := ValidateDocument(document); err != nil {
		return err
	}
	if err := d.Verify(ctx); err != nil {
		return err
	}
	directory, path, err := privateInput(document, ".cueson.json")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	_, err = d.run(ctx, directory, "validate", "--format", "cueson", path)
	return err
}
func (d *Driver) Inspect(ctx context.Context, document []byte) (json.RawMessage, error) {
	if err := ValidateDocument(document); err != nil {
		return nil, err
	}
	if err := d.Verify(ctx); err != nil {
		return nil, err
	}
	directory, path, err := privateInput(document, ".cueson.json")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	captured, err := d.run(ctx, directory, "inspect", "--format", "cueson", "--json", path)
	if err != nil {
		return nil, err
	}
	if !json.Valid(captured.Stdout) {
		return nil, contracts.Fail("operation_failed")
	}
	return bytes.Clone(captured.Stdout), nil
}
func (d *Driver) Assemble(ctx context.Context, document []byte, durationNS *int64, turns []Turn, participation []Participation) (Assembly, error) {
	result, err := AssembleDocument(document, durationNS, turns, participation)
	if err != nil {
		return result, err
	}
	if err = d.Validate(ctx, result.Document); err != nil {
		result.Document = nil
		return result, err
	}
	return result, nil
}
func omissionDiagnostics(document []byte, conversion bool) ([]Diagnostic, error) {
	var doc semanticDocument
	json.Unmarshal(document, &doc)
	total := 0
	if doc.Media != nil {
		total++
	}
	for _, cue := range doc.Cues {
		total += len(cue.Attributions)
	}
	if total > 8192 {
		return []Diagnostic{{Code: "export_loss_limit", Message: "Complete export accounting exceeds the configured limit; no output is published."}}, contracts.Fail("output_limit")
	}
	result := []Diagnostic{}
	prefix := "consumer_"
	if conversion {
		prefix = "conversion_"
	}
	appendOne := func(code, pointer, message string) error {
		if len(result) >= 8192 {
			return contracts.Fail("output_limit")
		}
		result = append(result, Diagnostic{Code: prefix + code, Pointer: pointer, Message: message})
		return nil
	}
	if doc.Media != nil {
		if err := appendOne("media_timing_omitted", "/media_timing", "Declared media timing has no native subtitle representation."); err != nil {
			return result, err
		}
	}
	for i, cue := range doc.Cues {
		for j := range cue.Attributions {
			if err := appendOne("speaker_attribution_omitted", "/cues/"+integer(i)+"/speaker_attributions/"+integer(j), "A consumer speaker assignment has no native subtitle representation."); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

var warningCode = regexp.MustCompile(`^warning: ([a-z][a-z0-9_]*):`)

func addUpstreamWarnings(diagnostics []Diagnostic, stderr []byte) ([]Diagnostic, error) {
	accounted := map[string]bool{}
	for _, item := range diagnostics {
		accounted[item.Code] = true
	}
	for _, line := range strings.Split(string(stderr), "\n") {
		match := warningCode.FindStringSubmatch(line)
		if len(match) != 2 || accounted[match[1]] {
			continue
		}
		if len(diagnostics) >= 8192 {
			return []Diagnostic{{Code: "export_loss_limit", Message: "Complete export accounting exceeds the configured limit; no output is published."}}, contracts.Fail("output_limit")
		}
		message := strings.TrimSpace(strings.TrimPrefix(line, match[0]))
		if len(message) > 4096 {
			return []Diagnostic{{Code: "export_loss_limit", Message: "Complete export accounting exceeds the configured limit; no output is published."}}, contracts.Fail("output_limit")
		}
		diagnostics = append(diagnostics, Diagnostic{Code: match[1], Message: message})
	}
	return diagnostics, nil
}
func (d *Driver) Export(ctx context.Context, document []byte, format string, strict bool) (Export, error) {
	result := Export{Diagnostics: []Diagnostic{}}
	if err := ValidateDocument(document); err != nil {
		return result, err
	}
	if err := d.Verify(ctx); err != nil {
		return result, err
	}
	if format == "cueson" || format == "cue-json" || format == "json" {
		if err := d.Validate(ctx, document); err != nil {
			return result, err
		}
		result.Bytes = bytes.Clone(document)
		return result, nil
	}
	target := nativeFormat(format)
	if target == "" {
		return result, contracts.Fail("invalid_request")
	}
	var identity struct {
		Format string `json:"format"`
	}
	json.Unmarshal(document, &identity)
	conversion := nativeFormat(identity.Format) != target
	var err error
	result.Diagnostics, err = omissionDiagnostics(document, conversion)
	if err != nil {
		return result, err
	}
	directory, path, err := privateInput(document, ".cueson.json")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(directory)
	command := "render"
	if conversion {
		command = "convert"
	}
	args := []string{command, "--to", target}
	if strict {
		args = append(args, "--strict")
	}
	args = append(args, "--output", "-", path)
	captured, err := d.run(ctx, directory, args...)
	var diagnosticErr error
	result.Diagnostics, diagnosticErr = addUpstreamWarnings(result.Diagnostics, captured.Stderr)
	if diagnosticErr != nil {
		return result, diagnosticErr
	}
	if err != nil {
		return result, err
	}
	result.Bytes = bytes.Clone(captured.Stdout)
	return result, nil
}
