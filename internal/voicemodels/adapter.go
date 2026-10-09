// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/process"
)

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func text(v string, n int) bool {
	return v != "" && len(v) <= n && utf8.ValidString(v) && v == strings.TrimSpace(v) && !strings.ContainsFunc(v, unicode.IsControl)
}
func normalizeLimits(v Limits) (Limits, error) {
	if v.MaxInputBytes == 0 {
		v.MaxInputBytes = 128 << 20
	}
	if v.MaxOutputBytes == 0 {
		v.MaxOutputBytes = 16 << 20
	}
	if v.TimeoutMS == 0 {
		v.TimeoutMS = 600000
	}
	if v.MaxInputBytes < 1 || v.MaxInputBytes > 64<<30 || v.MaxOutputBytes < 1 || v.MaxOutputBytes > 64<<30 || v.TimeoutMS < 1 || v.TimeoutMS > 86400000 {
		return v, contracts.Fail("invalid_request")
	}
	return v, nil
}
func ValidateAdapter(a Adapter) error {
	if a.ContractVersion != "1" || !text(a.Architecture, 128) || len(a.OutputKinds) == 0 || len(a.OutputKinds) > 32 || len(a.Consumers) > 32 || len(a.Arguments) > 32 || len(a.SupportFiles) > 64 || len(a.License) > 1024 {
		return contracts.Fail("invalid_request")
	}
	if _, err := normalizeLimits(a.Limits); err != nil {
		return err
	}
	for _, set := range [][]string{a.OutputKinds, a.Consumers, a.SupportedOperations} {
		seen := map[string]bool{}
		for _, v := range set {
			if !text(v, 128) || seen[v] {
				return contracts.Fail("invalid_request")
			}
			seen[v] = true
		}
	}
	if a.ID == "pyannote-profile" {
		if a.Mode != "local" || a.Architecture != "pyannote" || a.Executable.Path != "" || a.Endpoint != "" || a.CredentialID != "" || len(a.Arguments) > 0 || len(a.SupportFiles) > 0 || a.NeedsText || a.SupportsResume || len(a.OutputKinds) != 1 || a.OutputKinds[0] != "voice-embedding" {
			return contracts.Fail("unsupported_capability")
		}
		return nil
	}
	if a.ID != "insonic-speaker" {
		return contracts.Fail("unsupported_capability")
	}
	if a.Mode == "local" {
		if !filepath.IsAbs(a.Executable.Path) || !digestPattern.MatchString(a.Executable.SHA256) || a.Endpoint != "" || a.CredentialID != "" || a.RemoteModel != "" || a.UpstreamRevision != "" {
			return contracts.Fail("invalid_request")
		}
		for _, v := range a.Arguments {
			if !text(v, 2048) {
				return contracts.Fail("invalid_request")
			}
		}
		for _, v := range a.SupportFiles {
			if !filepath.IsAbs(v.Path) || !digestPattern.MatchString(v.SHA256) {
				return contracts.Fail("invalid_request")
			}
		}
	} else if a.Mode == "hosted" {
		limits, _ := normalizeLimits(a.Limits)
		if limits.MaxInputBytes > 128<<20 || limits.MaxOutputBytes > 16<<20 {
			return contracts.Fail("input_limit")
		}
		if a.Executable.Path != "" || a.Executable.SHA256 != "" || len(a.Arguments) > 0 || len(a.SupportFiles) > 0 || !text(a.RemoteModel, 256) || !text(a.UpstreamRevision, 512) || a.CredentialID != "" && !contracts.ValidID(a.CredentialID) {
			return contracts.Fail("invalid_request")
		}
		u, err := contracts.SourceURL(a.Endpoint)
		if err != nil || u.RawQuery != "" || u.ForceQuery {
			return contracts.Fail("invalid_request")
		}
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "https" && (u.Scheme != "http" || !(u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())) {
			return contracts.Fail("invalid_request")
		}
	} else {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func strict(raw []byte, out any) error {
	if catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_engine_output")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_engine_output")
	}
	return nil
}
func inCI() bool {
	for _, k := range []string{"CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}
func pinned(ctx context.Context, f library.PinnedFile) error {
	if !filepath.IsAbs(f.Path) || !digestPattern.MatchString(f.SHA256) {
		return contracts.Fail("invalid_request")
	}
	in, err := os.Open(f.Path)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return contracts.Fail("unavailable")
	}
	h := sha256.New()
	if _, err = io.Copy(h, &contextReader{ctx: ctx, reader: in}); err != nil {
		return contracts.Fail("cancelled")
	}
	if hex.EncodeToString(h.Sum(nil)) != f.SHA256 {
		return contracts.Fail("incompatible_version")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(b)
}
func (s *Service) invoke(ctx context.Context, a Adapter, req AdapterRequest) (Output, error) {
	if err := ValidateAdapter(a); err != nil {
		return Output{}, err
	}
	limits, _ := normalizeLimits(a.Limits)
	bounded, cancel := context.WithTimeout(ctx, time.Duration(limits.TimeoutMS)*time.Millisecond)
	defer cancel()
	if a.Mode == "local" {
		if s.LocalRunner != nil {
			return s.LocalRunner(bounded, a, req)
		}
		if inCI() {
			return Output{}, contracts.Fail("engine_ci_forbidden")
		}
		if err := pinned(bounded, a.Executable); err != nil {
			return Output{}, err
		}
		for _, f := range a.SupportFiles {
			if err := pinned(bounded, f); err != nil {
				return Output{}, err
			}
		}
		raw, err := json.Marshal(req)
		if err != nil || int64(len(raw)) > limits.MaxInputBytes {
			return Output{}, contracts.Fail("input_limit")
		}
		result, err := process.Capture(bounded, process.Spec{Executable: a.Executable.Path, Args: a.Arguments, Directory: req.OutputDirectory, CleanEnv: true, Env: append(process.LocalEnvironment(), "HF_HUB_OFFLINE=1", "TRANSFORMERS_OFFLINE=1", "PYTHONIOENCODING=utf-8"), Input: bytes.NewReader(raw), MaxOutput: 16 << 20})
		if err != nil {
			return Output{}, err
		}
		var out Output
		if strict(result.Stdout, &out) != nil {
			return out, contracts.Fail("invalid_engine_output")
		}
		return out, nil
	}
	return s.hosted(bounded, a, req, limits)
}
func (s *Service) hosted(ctx context.Context, a Adapter, req AdapterRequest, limits Limits) (Output, error) {
	if s.Client == nil && inCI() {
		return Output{}, contracts.Fail("engine_ci_forbidden")
	}
	req.RemoteModel = a.RemoteModel
	req.UpstreamRevision = a.UpstreamRevision
	req.OutputDirectory = ""
	req.BaseModelDirectory = ""
	// Media bytes are sent only to the selected route. Local paths never leave it.
	type upload struct {
		PreparedInput
		Data []byte `json:"data"`
	}
	inputs := make([]upload, 0, len(req.Inputs))
	var total int64
	allInputs := append([]PreparedInput{}, req.Inputs...)
	for _, input := range req.BaseFiles {
		input.ID = "base/" + input.ID
		allInputs = append(allInputs, input)
	}
	for _, input := range allInputs {
		f, err := os.Open(input.Path)
		if err != nil {
			return Output{}, contracts.Fail("unavailable")
		}
		data, err := io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, reader: f}, limits.MaxInputBytes-total+1))
		f.Close()
		total += int64(len(data))
		if err != nil || total > limits.MaxInputBytes {
			return Output{}, contracts.Fail("input_limit")
		}
		if int64(len(data)) != input.Size || hash(data) != input.SHA256 {
			return Output{}, contracts.Fail("conflict")
		}
		input.Path = ""
		inputs = append(inputs, upload{input, data})
	}
	requestDoc := struct {
		AdapterRequest
		Uploads []upload `json:"uploads"`
	}{req, inputs}
	requestDoc.Inputs = nil
	requestDoc.BaseFiles = nil
	if requestDoc.Checkpoint != nil {
		c := *requestDoc.Checkpoint
		c.Artifact.Path = ""
		requestDoc.Checkpoint = &c
		total += int64(len(c.Artifact.Data))
		if total > limits.MaxInputBytes {
			return Output{}, contracts.Fail("input_limit")
		}
	}
	raw, err := json.Marshal(requestDoc)
	if err != nil || int64(len(raw)) > limits.MaxInputBytes*2+1<<20 {
		return Output{}, contracts.Fail("input_limit")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, bytes.NewReader(raw))
	if err != nil {
		return Output{}, contracts.Fail("invalid_request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if a.CredentialID != "" {
		if s.Secrets == nil {
			return Output{}, contracts.Fail("unavailable")
		}
		secret, err := s.Secrets.Resolve(ctx, a.CredentialID)
		if err != nil {
			return Output{}, contracts.Fail("unavailable")
		}
		valid := len(secret) > 0 && len(secret) <= 2048
		for _, v := range secret {
			if v <= 32 || v >= 127 {
				valid = false
			}
		}
		if valid {
			request.Header.Set("Authorization", "Bearer "+string(secret))
		}
		for i := range secret {
			secret[i] = 0
		}
		if !valid {
			return Output{}, contracts.Fail("invalid_request")
		}
		defer request.Header.Del("Authorization")
	}
	client := http.Client{}
	if s.Client != nil {
		client = *s.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return Output{}, contracts.Fail("cancelled")
		}
		return Output{}, contracts.Fail("unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Output{}, contracts.Fail("operation_failed")
	}
	max := limits.MaxOutputBytes
	if max > 16<<20 {
		max = 16 << 20
	}
	reply, err := io.ReadAll(io.LimitReader(response.Body, max+1))
	if err != nil {
		return Output{}, contracts.Fail("unavailable")
	}
	if int64(len(reply)) > max {
		return Output{}, contracts.Fail("output_limit")
	}
	var out Output
	if strict(reply, &out) != nil {
		return out, contracts.Fail("invalid_engine_output")
	}
	return out, nil
}
func hash(raw []byte) string         { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func adapterDigest(a Adapter) string { raw, _ := json.Marshal(a); return hash(raw) }
func opaqueHandle(value string) bool {
	return text(value, 2048) && !strings.ContainsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || !unicode.IsPrint(r) }) && !strings.Contains(value, "://") && !strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "\\") && !strings.ContainsAny(value, "?#")
}
func validateOutput(a Adapter, o Output) error {
	if o.HostedHandle != "" && !opaqueHandle(o.HostedHandle) {
		return contracts.Fail("invalid_engine_output")
	}
	if o.State != "" && o.State != "completed" && o.State != "checkpointed" {
		return contracts.Fail("invalid_engine_output")
	}
	if o.ContractVersion != "1" || o.Architecture != a.Architecture || len(o.Artifacts) > 64 || len(o.Checkpoints) > 64 || len(o.Diagnostics) > 128 || len(o.HostedHandle) > 2048 || len(o.SupportedOperations) < 1 || len(o.SupportedOperations) > 32 {
		return contracts.Fail("invalid_engine_output")
	}
	allowed := false
	for _, kind := range a.OutputKinds {
		allowed = allowed || kind == o.Kind
	}
	if !allowed {
		return contracts.Fail("unsupported_capability")
	}
	for _, v := range o.Consumers {
		found := false
		for _, want := range a.Consumers {
			found = found || v == want
		}
		if !found {
			return contracts.Fail("unsupported_capability")
		}
	}
	if (len(o.Artifacts) == 0 && o.HostedHandle == "" && (o.State != "checkpointed" || len(o.Checkpoints) == 0)) || o.HostedHandle != "" && a.Mode != "hosted" || o.State == "checkpointed" && (len(o.Artifacts) > 0 || o.HostedHandle != "") {
		return contracts.Fail("invalid_engine_output")
	}
	roles := map[string]bool{}
	for _, f := range o.Artifacts {
		if !models.ValidRole(f.Role) || roles[strings.ToLower(f.Role)] || !text(f.Format, 128) || !digestPattern.MatchString(f.SHA256) || f.Size < 1 || f.Size > 64<<30 {
			return contracts.Fail("invalid_engine_output")
		}
		roles[strings.ToLower(f.Role)] = true
	}
	for _, op := range o.SupportedOperations {
		if !text(op, 128) || len(a.SupportedOperations) > 0 && !contains(a.SupportedOperations, op) {
			return contracts.Fail("unsupported_capability")
		}
	}
	for _, d := range o.Diagnostics {
		if !text(d.Code, 128) || d.Count < 0 || d.Value != nil && (math.IsNaN(*d.Value) || math.IsInf(*d.Value, 0)) {
			return contracts.Fail("invalid_engine_output")
		}
	}
	return nil
}
