// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
)

// Executor implements only explicitly elected hosted routes. Client permits
// deterministic injected HTTP fixtures; application construction leaves it nil.
// No capabilities inspection requests, retries, discovery or fallback occur.
type Executor struct {
	Secrets contracts.SecretProvider
	Client  *http.Client
}
type requestAudio struct {
	SampleRate  int64  `json:"sample_rate"`
	SampleCount int64  `json:"sample_count"`
	Format      string `json:"format"`
}
type workerRequest struct {
	Version   string       `json:"contract_version"`
	Operation string       `json:"operation"`
	Model     string       `json:"model"`
	Audio     requestAudio `json:"audio"`
	Options   any          `json:"options"`
}
type workerResponse struct {
	Version   string                           `json:"contract_version"`
	Operation string                           `json:"operation"`
	Segments  *[]processing.RecognitionSegment `json:"segments,omitempty"`
	Turns     *[]processing.Turn               `json:"turns,omitempty"`
	NoSpeech  *bool                            `json:"no_speech"`
}

func ci() bool {
	for _, key := range []string{"CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL"} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}
func (e *Executor) call(ctx context.Context, path string, mapping processing.SourceMap, stage Stage, operation string, options any) (workerResponse, error) {
	var out workerResponse
	if e == nil || ctx == nil {
		return out, contracts.Fail("invalid_request")
	}
	// The production route categorically refuses CI before files, credentials or
	// requests. Explicit injected clients are used only by deterministic fixtures.
	if e.Client == nil && ci() {
		return out, contracts.Fail("engine_ci_forbidden")
	}
	if ctx.Err() != nil {
		return out, contracts.Fail("cancelled")
	}
	if stage.Adapter != "insonic-http" || stage.Mode != "hosted" {
		return out, contracts.Fail("unsupported_capability")
	}
	if err := ValidateStage(stage, operation); err != nil {
		return out, err
	}
	if _, err := processing.MappedDuration(mapping); err != nil {
		return out, err
	}
	limits, _ := NormalizeLimits(stage.Limits)
	bounded, cancel := context.WithTimeout(ctx, time.Duration(limits.TimeoutMS)*time.Millisecond)
	defer cancel()
	file, err := os.Open(path)
	if err != nil {
		return out, contracts.Fail("unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return out, contracts.Fail("invalid_audio")
	}
	if info.Size() > limits.MaxAudioBytes {
		return out, contracts.Fail("input_limit")
	}
	if err = processing.ValidateMappedAudio(path, mapping); err != nil {
		return out, err
	}
	metadata, err := json.Marshal(workerRequest{Version: "1", Operation: operation, Model: stage.RemoteModel, Audio: requestAudio{mapping.SampleRate, mapping.SampleCount, "wav-pcm-s16le-mono"}, Options: options})
	if err != nil || len(metadata) > 64<<10 {
		return out, contracts.Fail("input_limit")
	}
	// Prefix and suffix are bounded metadata. Audio streams from the leased file
	// without a goroutine or an unbounded in-memory multipart body.
	var envelope bytes.Buffer
	writer := multipart.NewWriter(&envelope)
	if err = writer.WriteField("request", string(metadata)); err != nil {
		return out, contracts.Fail("operation_failed")
	}
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", `form-data; name="audio"; filename="mapped.wav"`)
	headers.Set("Content-Type", "audio/wav")
	if _, err = writer.CreatePart(headers); err != nil {
		return out, contracts.Fail("operation_failed")
	}
	prefix := append([]byte(nil), envelope.Bytes()...)
	if writer.Close() != nil {
		return out, contracts.Fail("operation_failed")
	}
	suffix := append([]byte(nil), envelope.Bytes()[len(prefix):]...)
	body := io.MultiReader(bytes.NewReader(prefix), io.LimitReader(file, info.Size()), bytes.NewReader(suffix))
	request, err := http.NewRequestWithContext(bounded, http.MethodPost, stage.Endpoint, body)
	if err != nil {
		return out, contracts.Fail("invalid_request")
	}
	request.ContentLength = int64(len(prefix)) + info.Size() + int64(len(suffix))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	if stage.CredentialID != "" {
		if e.Secrets == nil {
			return out, contracts.Fail("unavailable")
		}
		secret, err := e.Secrets.Resolve(bounded, stage.CredentialID)
		if err != nil {
			return out, contracts.Fail("unavailable")
		}
		valid := len(secret) > 0 && len(secret) <= 2048
		for _, b := range secret {
			if b <= 32 || b >= 127 {
				valid = false
			}
		}
		if valid {
			request.Header.Set("Authorization", "Bearer "+string(secret))
		}
		for index := range secret {
			secret[index] = 0
		}
		if !valid {
			return out, contracts.Fail("invalid_request")
		}
		defer request.Header.Del("Authorization")
	}
	client := http.Client{}
	if e.Client != nil {
		client = *e.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	timeout := time.Duration(limits.TimeoutMS) * time.Millisecond
	if client.Timeout == 0 || client.Timeout > timeout {
		client.Timeout = timeout
	}
	response, err := client.Do(request)
	if err != nil {
		if bounded.Err() != nil {
			return out, contracts.Fail("cancelled")
		}
		return out, contracts.Fail("unavailable")
	}
	defer response.Body.Close()
	// Never retain or expose provider error text, URLs, headers or raw bodies.
	if response.StatusCode != http.StatusOK {
		return out, contracts.Fail("operation_failed")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, limits.MaxResponseBytes+1))
	if bounded.Err() != nil {
		return out, contracts.Fail("cancelled")
	}
	if err != nil {
		return out, contracts.Fail("unavailable")
	}
	if int64(len(raw)) > limits.MaxResponseBytes {
		return out, contracts.Fail("output_limit")
	}
	if catalog.ValidateJSON(raw) != nil {
		return out, contracts.Fail("invalid_engine_output")
	}
	if !requiredIntervals(raw, operation) {
		return out, contracts.Fail("invalid_engine_output")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil || decoder.Decode(new(any)) != io.EOF || out.Version != "1" || out.Operation != operation || out.NoSpeech == nil {
		return workerResponse{}, contracts.Fail("invalid_engine_output")
	}
	if operation == "transcription" && (out.Segments == nil || out.Turns != nil) || operation == "diarization" && (out.Turns == nil || out.Segments != nil) {
		return workerResponse{}, contracts.Fail("invalid_engine_output")
	}
	return out, nil
}

// Plain numeric fields otherwise decode an absent/null start as zero. Require
// explicit interval evidence rather than manufacturing a source-start boundary.
func requiredIntervals(raw []byte, operation string) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return false
	}
	key := "segments"
	required := []string{"start_us", "end_us", "text"}
	if operation == "diarization" {
		key = "turns"
		required = []string{"start_us", "end_us", "label"}
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(fields[key], &items) != nil || items == nil || len(items) > 20000 {
		return false
	}
	for _, item := range items {
		for _, name := range required {
			value, ok := item[name]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return false
			}
		}
	}
	return true
}
func routeProvenance(stage Stage) map[string]any {
	return map[string]any{"adapter": "insonic-http", "adapter_contract_version": "1", "execution_mode": "hosted", "remote_model": stage.RemoteModel}
}
func (e *Executor) Recognize(ctx context.Context, path string, mapping processing.SourceMap, stage Stage, options processing.RecognitionOptions) (processing.RecognitionResult, error) {
	capability, err := Capabilities(stage)
	if err != nil {
		return processing.RecognitionResult{}, err
	}
	if options.Device != "" {
		return processing.RecognitionResult{}, contracts.Fail("unsupported_device")
	}
	options, err = processing.ValidateRecognitionOptions(options, capability.MaxHintBytes)
	if err != nil {
		return processing.RecognitionResult{}, err
	}
	out, err := e.call(ctx, path, mapping, stage, "transcription", options)
	if err != nil {
		return processing.RecognitionResult{}, err
	}
	result, err := processing.RecognitionFromSegments(*out.Segments, *out.NoSpeech, mapping)
	if err != nil {
		return result, err
	}
	for key, value := range routeProvenance(stage) {
		result.Provenance[key] = value
	}
	result.Provenance["context_digest"] = options.ContextDigest
	result.Provenance["hint_count"] = len(options.Hints)
	return result, nil
}
func (e *Executor) Diarize(ctx context.Context, path string, mapping processing.SourceMap, stage Stage, options processing.DiarizationOptions) (processing.DiarizationResult, error) {
	if options.Device != "" {
		return processing.DiarizationResult{}, contracts.Fail("unsupported_device")
	}
	if err := processing.ValidateDiarizationOptions(options); err != nil {
		return processing.DiarizationResult{}, err
	}
	remoteOptions := options
	// Quality is measured locally on returned mapped-relative intervals and is
	// not a model option or an instruction to the configured remote worker.
	remoteOptions.Quality = nil
	out, err := e.call(ctx, path, mapping, stage, "diarization", remoteOptions)
	if err != nil {
		return processing.DiarizationResult{}, err
	}
	duration, err := processing.MappedDuration(mapping)
	if err != nil {
		return processing.DiarizationResult{}, err
	}
	quality, err := processing.ApplyDiarizationQuality(processing.DiarizationResult{Turns: *out.Turns, NoSpeech: *out.NoSpeech}, duration, options.Quality)
	if err != nil {
		return processing.DiarizationResult{}, err
	}
	result, err := processing.DiarizationFromTurns(*out.Turns, *out.NoSpeech, mapping)
	if err != nil {
		return result, err
	}
	for key, value := range routeProvenance(stage) {
		result.Provenance[key] = value
	}
	result.Diagnostics = append(result.Diagnostics, quality.Diagnostics...)
	result.Provenance["quality_diagnostics_enabled"] = quality.Provenance["quality_diagnostics_enabled"]
	return result, nil
}
