// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/workspace"
)

var rolePattern = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._/-]*$`)
var languagePattern = regexp.MustCompile(`^[a-z]{2,3}$`)

func validRole(role string) bool {
	if len(role) == 0 || len(role) > 128 || !rolePattern.MatchString(role) || path.Clean(role) != role || strings.HasSuffix(role, "/") {
		return false
	}
	for _, part := range strings.Split(role, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") {
			return false
		}
		name := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if name == "CON" || name == "PRN" || name == "AUX" || name == "NUL" || len(name) == 4 && (strings.HasPrefix(name, "COM") || strings.HasPrefix(name, "LPT")) && name[3] >= '1' && name[3] <= '9' {
			return false
		}
	}
	return true
}

func (s *Session) model(ctx context.Context, id, capability string) (string, map[string]any, error) {
	if !contracts.ValidID(id) {
		return "", nil, contracts.Fail("invalid_request")
	}
	install, err := s.service.Catalog.BaseModel(ctx, id)
	if err != nil {
		return "", nil, err
	}
	var manifest models.Manifest
	if install.State != "available" || models.DecodeManifest(install.Manifest, &manifest) != nil || manifest.Digest() != install.Digest {
		return "", nil, contracts.Fail("model_unavailable")
	}
	capable := false
	for _, value := range manifest.Capabilities {
		if value == capability {
			capable = true
		}
	}
	if !capable {
		return "", nil, contracts.Fail("unsupported_capability")
	}
	ids, err := catalog.PublicationIDs(install.PublicationIDs)
	if err != nil || len(ids) != len(manifest.Files) {
		return "", nil, contracts.Fail("conflict")
	}
	roles := map[string]bool{}
	for _, file := range manifest.Files {
		if !validRole(file.Role) {
			return "", nil, contracts.Fail("invalid_request")
		}
		key := strings.ToLower(file.Role)
		if roles[key] {
			return "", nil, contracts.Fail("invalid_request")
		}
		roles[key] = true
	}
	required := []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"}
	if capability == "diarization" {
		required = []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"}
	}
	for _, role := range required {
		if !roles[role] {
			return "", nil, contracts.Fail("model_unavailable")
		}
	}
	directory, err := os.MkdirTemp(s.directory, "model-")
	if err != nil {
		return "", nil, contracts.Fail("unavailable")
	}
	failed := true
	defer func() {
		if failed {
			os.RemoveAll(directory)
		}
	}()
	if err = workspace.SecureDirectory(directory, true); err != nil {
		return "", nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", nil, contracts.Fail("unavailable")
	}
	defer root.Close()
	for index, file := range manifest.Files {
		if ctx.Err() != nil {
			return "", nil, contracts.Fail("cancelled")
		}
		if file.Size < 0 || file.Size > 64<<30 {
			return "", nil, contracts.Fail("input_limit")
		}
		publication, err := s.service.Catalog.Publication(ctx, ids[index])
		if err != nil {
			return "", nil, err
		}
		if publication.State != "available" || publication.Digest != file.SHA256 || publication.Size != file.Size {
			return "", nil, contracts.Fail("conflict")
		}
		materialized, err := s.materialize(ctx, ids[index], file.Size)
		if err != nil {
			return "", nil, err
		}
		parent := path.Dir(file.Role)
		if parent != "." {
			prefix := ""
			for _, part := range strings.Split(parent, "/") {
				prefix = path.Join(prefix, part)
				if err = root.Mkdir(prefix, 0700); err != nil && !os.IsExist(err) {
					return "", nil, contracts.Fail("unavailable")
				}
			}
		}
		output, err := root.OpenFile(file.Role, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", nil, contracts.Fail("unavailable")
		}
		input, err := os.Open(materialized.Path)
		if err != nil {
			output.Close()
			return "", nil, contracts.Fail("unavailable")
		}
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(output, h), io.LimitReader(cancelReader{ctx, input}, file.Size+1))
		input.Close()
		syncErr := output.Sync()
		closeErr := output.Close()
		if ctx.Err() != nil {
			return "", nil, contracts.Fail("cancelled")
		}
		if err != nil || syncErr != nil || closeErr != nil || n != file.Size || hex.EncodeToString(h.Sum(nil)) != file.SHA256 {
			return "", nil, contracts.Fail("conflict")
		}
	}
	failed = false
	return directory, map[string]any{"model_id": id, "model_name": manifest.Name, "model_version": manifest.ModelVersion, "model_revision": manifest.Revision, "model_manifest_sha256": install.Digest, "model_license": manifest.License}, nil
}

func (s *Session) begin(ctx context.Context) (context.Context, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil || ctx.Err() != nil {
		return nil, nil, contracts.Fail("cancelled")
	}
	s.operations.Add(1)
	operationCtx, cancel := context.WithCancel(s.ctx)
	stop := context.AfterFunc(ctx, cancel)
	return operationCtx, func() { stop(); cancel(); s.operations.Done() }, nil
}

func (s *Session) worker(ctx context.Context, operation string, id string, options map[string]any) ([]byte, map[string]any, error) {
	// Refuse before even selecting or materializing model bytes. The Python
	// guard remains independent so direct maintainer worker entry also refuses CI.
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL"} {
		if os.Getenv(name) != "" {
			return nil, nil, contracts.Fail("engine_ci_forbidden")
		}
	}
	config, err := defaults(s.service.Config)
	if err != nil {
		return nil, nil, err
	}
	python := config.RecognitionPython
	capability := "transcription"
	if operation == "diarize" {
		python = config.DiarizationPython
		capability = "diarization"
	}
	if err = verifyPinned(ctx, python); err != nil {
		return nil, nil, err
	}
	if err = verifyPinned(ctx, config.Worker); err != nil {
		return nil, nil, err
	}
	directory, provenance, err := s.model(ctx, id, capability)
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(directory)
	request := map[string]any{"operation": operation, "audio_path": s.MappedPath, "model_path": directory, "threads": config.Threads, "max_duration_us": config.MaxDurationUS, "source_start_numerator": s.SourceMap.StartNumerator, "source_start_denominator": s.SourceMap.StartDenominator}
	for key, value := range options {
		request[key] = value
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, nil, contracts.Fail("invalid_request")
	}
	workerCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMS)*time.Millisecond)
	defer cancel()
	result, err := process.Capture(workerCtx, process.Spec{Executable: python.Path, Args: []string{"-I", "-B", config.Worker.Path}, Directory: s.directory, Env: []string{"HF_HUB_OFFLINE=1", "TRANSFORMERS_OFFLINE=1", "HF_HUB_DISABLE_TELEMETRY=1", "PYANNOTE_METRICS_ENABLED=0", "PYTHONIOENCODING=utf-8", "TOKENIZERS_PARALLELISM=false"}, Input: bytes.NewReader(encoded), MaxOutput: config.MaxOutputBytes})
	if err != nil {
		// Only documented compact worker error codes escape. Discard raw engine
		// stderr so request/secret values cannot enter durable diagnostics.
		var failure struct {
			Error string `json:"error"`
		}
		lines := bytes.Split(bytes.TrimSpace(result.Stderr), []byte("\n"))
		if len(lines) > 0 && strict(lines[len(lines)-1], &failure) == nil {
			switch failure.Error {
			case "engine_ci_forbidden", "engine_unavailable", "model_unavailable", "incompatible_version", "unsupported_device", "invalid_embedding", "invalid_timing", "audio_limit":
				return nil, nil, contracts.Fail(failure.Error)
			}
		}
		return nil, nil, err
	}
	provenance["python_sha256"] = python.SHA256
	provenance["worker_sha256"] = config.Worker.SHA256
	return result.Stdout, provenance, nil
}

func strict(raw []byte, out any) error {
	if len(raw) > 16<<20 || catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}

func (s *Session) Recognize(ctx context.Context, modelID string, options RecognitionOptions) (RecognitionResult, error) {
	operationCtx, end, err := s.begin(ctx)
	if err != nil {
		return RecognitionResult{}, err
	}
	defer end()
	if options.Device == "" {
		options.Device = "cpu"
	}
	if options.Language == "" {
		options.Language = "en"
	}
	if options.Device != "cpu" && options.Device != "cuda" || options.Language != "auto" && !languagePattern.MatchString(options.Language) {
		return RecognitionResult{}, contracts.Fail("invalid_request")
	}
	raw, provenance, err := s.worker(operationCtx, "transcribe", modelID, map[string]any{"device": options.Device, "language": options.Language})
	if err != nil {
		return RecognitionResult{}, err
	}
	var output struct {
		SRT         string         `json:"srt"`
		NoSpeech    bool           `json:"no_speech"`
		Provenance  map[string]any `json:"provenance"`
		Diagnostics []Diagnostic   `json:"diagnostics"`
	}
	if strict(raw, &output) != nil || len(output.SRT) > 8<<20 || len(output.Diagnostics) > 20000 || !validProvenance(output.Provenance) || !validDiagnostics(output.Diagnostics) || output.NoSpeech && output.SRT != "" {
		return RecognitionResult{}, contracts.Fail("invalid_engine_output")
	}
	for key, value := range output.Provenance {
		provenance[key] = value
	}
	return RecognitionResult{SRT: []byte(output.SRT), NoSpeech: output.NoSpeech, Provenance: provenance, Diagnostics: output.Diagnostics}, nil
}

func decodeDiarization(raw []byte, durationUS int64) (DiarizationResult, error) {
	var output struct {
		Turns       []Turn         `json:"turns"`
		NoSpeech    bool           `json:"no_speech"`
		Provenance  map[string]any `json:"provenance"`
		Diagnostics []Diagnostic   `json:"diagnostics"`
	}
	if strict(raw, &output) != nil || len(output.Turns) > 20000 || len(output.Diagnostics) > 20000 || !validProvenance(output.Provenance) || !validDiagnostics(output.Diagnostics) || output.NoSpeech && len(output.Turns) != 0 {
		return DiarizationResult{}, contracts.Fail("invalid_engine_output")
	}
	for _, turn := range output.Turns {
		if turn.Label == "" || len(turn.Label) > 256 || strings.ContainsAny(turn.Label, "\r\n\x00") || turn.StartUS < 0 || turn.EndUS <= turn.StartUS || turn.EndUS > durationUS {
			return DiarizationResult{}, contracts.Fail("invalid_engine_output")
		}
	}
	sort.Slice(output.Turns, func(i, j int) bool {
		a, b := output.Turns[i], output.Turns[j]
		if a.StartUS != b.StartUS {
			return a.StartUS < b.StartUS
		}
		if a.EndUS != b.EndUS {
			return a.EndUS < b.EndUS
		}
		return a.Label < b.Label
	})
	return DiarizationResult{Turns: output.Turns, NoSpeech: output.NoSpeech, Provenance: output.Provenance, Diagnostics: output.Diagnostics}, nil
}

func (s *Session) Diarize(ctx context.Context, modelID string, options DiarizationOptions) (DiarizationResult, error) {
	operationCtx, end, err := s.begin(ctx)
	if err != nil {
		return DiarizationResult{}, err
	}
	defer end()
	if options.Device == "" {
		options.Device = "cpu"
	}
	if options.Device != "cpu" && options.Device != "cuda" || options.MinSpeakers < 0 || options.MaxSpeakers < 0 || options.MinSpeakers > 64 || options.MaxSpeakers > 64 || options.MaxSpeakers > 0 && options.MinSpeakers > options.MaxSpeakers {
		return DiarizationResult{}, contracts.Fail("invalid_request")
	}
	raw, provenance, err := s.worker(operationCtx, "diarize", modelID, map[string]any{"device": options.Device, "min_speakers": options.MinSpeakers, "max_speakers": options.MaxSpeakers})
	if err != nil {
		return DiarizationResult{}, err
	}
	output, err := decodeDiarization(raw, s.SourceMap.SampleCount*1_000_000/s.SourceMap.SampleRate)
	if err != nil {
		return output, err
	}
	for key, value := range output.Provenance {
		provenance[key] = value
	}
	output.Provenance = provenance
	output.Turns, output.Diagnostics, err = sourceTurns(output.Turns, s.SourceMap, output.Diagnostics)
	if err != nil {
		return DiarizationResult{}, err
	}
	output.Provenance["source_clock_projection"] = "exact-source-origin;ceil-start/floor-end-microsecond"
	return output, nil
}

func roundedRat(value *big.Rat, ceil bool) (int64, error) {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if remainder.Sign() != 0 {
		if ceil && value.Sign() > 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
		if !ceil && value.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	if !quotient.IsInt64() {
		return 0, contracts.Fail("invalid_timing")
	}
	return quotient.Int64(), nil
}

func sourceTurns(turns []Turn, mapping SourceMap, diagnostics []Diagnostic) ([]Turn, []Diagnostic, error) {
	origin, ok := new(big.Rat).SetString(mapping.StartNumerator + "/" + mapping.StartDenominator)
	if !ok {
		return nil, nil, contracts.Fail("invalid_timing")
	}
	origin.Mul(origin, new(big.Rat).SetInt64(1_000_000))
	out := make([]Turn, 0, len(turns))
	for _, turn := range turns {
		start, err := roundedRat(new(big.Rat).Add(origin, new(big.Rat).SetInt64(turn.StartUS)), true)
		if err != nil {
			return nil, nil, err
		}
		end, err := roundedRat(new(big.Rat).Add(origin, new(big.Rat).SetInt64(turn.EndUS)), false)
		if err != nil {
			return nil, nil, err
		}
		if start < 0 || end < 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "negative_source_interval_unrepresentable", Count: 1})
			continue
		}
		if start >= end {
			diagnostics = append(diagnostics, Diagnostic{Code: "source_interval_collapsed_at_microsecond", Count: 1})
			continue
		}
		turn.StartUS, turn.EndUS = start, end
		out = append(out, turn)
	}
	return out, diagnostics, nil
}
