// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
)

func localDefinition() Definition {
	return Definition{Recognition: Stage{Adapter: "faster-whisper", Version: "1", Mode: "local", ModelID: "11111111-1111-4111-8111-111111111111"}, Diarization: Stage{Adapter: "pyannote", Version: "1", Mode: "local", ModelID: "22222222-2222-4222-8222-222222222222"}}
}
func remoteStage(endpoint string) Stage {
	yes := true
	return Stage{Adapter: "insonic-http", Version: "1", Mode: "hosted", Endpoint: endpoint, RemoteModel: "test-model", Capabilities: []string{"transcription", "diarization"}, SupportsHints: &yes, MaxHintBytes: 512}
}
func TestElectionFreezesValuesAndRejectsMixedPreset(t *testing.T) {
	d := localDefinition()
	if err := Validate("local", d); err != nil {
		t.Fatal(err)
	}
	r := remoteStage("https://example.test/process")
	if _, err := Elect("local", d, Override{Recognition: &r}); err == nil {
		t.Fatal("mixed local preset accepted")
	}
	elected, err := Elect("custom", d, Override{Recognition: &r})
	if err != nil {
		t.Fatal(err)
	}
	r.Capabilities[0] = "changed"
	if elected.Recognition.Capabilities[0] != "transcription" {
		t.Fatal("election aliases mutable caller input")
	}
	bad := remoteStage("https://example.test/process?token=synthetic")
	if _, err := Elect("custom", d, Override{Recognition: &bad}); err == nil {
		t.Fatal("secret-bearing URL accepted")
	}
	bad = remoteStage("https://example.test/process")
	bad.ModelID = d.Recognition.ModelID
	if _, err := Elect("custom", d, Override{Recognition: &bad}); err == nil {
		t.Fatal("hosted local model registry reference accepted")
	}
	c, err := Capabilities(d.Recognition)
	if err != nil || !c.SupportsHints || c.MaxHintBytes != 200 {
		t.Fatalf("local capability: %+v %v", c, err)
	}
	d.Diarization.Diarization.Quality = &processing.QualityConfig{}
	if err := Validate("local", d); err == nil {
		t.Fatal("ambiguous stage quality election accepted")
	}
}

type fixtureSecrets struct{ calls int }

func (s *fixtureSecrets) Resolve(context.Context, string) ([]byte, error) {
	s.calls++
	return []byte("synthetic-fixture-token"), nil
}
func (*fixtureSecrets) Status(context.Context, string) (string, error) { return "available", nil }
func fixtureAudio(t *testing.T) (string, processing.SourceMap) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mapped.wav")
	raw := make([]byte, 44+32000)
	copy(raw, "RIFF")
	binary.LittleEndian.PutUint32(raw[4:], uint32(len(raw)-8))
	copy(raw[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(raw[16:], 16)
	binary.LittleEndian.PutUint16(raw[20:], 1)
	binary.LittleEndian.PutUint16(raw[22:], 1)
	binary.LittleEndian.PutUint32(raw[24:], 16000)
	binary.LittleEndian.PutUint32(raw[28:], 32000)
	binary.LittleEndian.PutUint16(raw[32:], 2)
	binary.LittleEndian.PutUint16(raw[34:], 16)
	copy(raw[36:], "data")
	binary.LittleEndian.PutUint32(raw[40:], 32000)
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return p, processing.SourceMap{StartNumerator: "1", StartDenominator: "3", SampleCount: 16000, SampleRate: 16000, StreamIndex: 0}
}
func TestHostedRecognitionUploadsMappedAudioAndProjectsExactClock(t *testing.T) {
	path, mapping := fixtureAudio(t)
	secrets := &fixtureSecrets{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic-fixture-token" {
			t.Error("explicit auth/method absent")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		var meta map[string]any
		if json.Unmarshal([]byte(r.FormValue("request")), &meta) != nil || meta["operation"] != "transcription" || meta["model"] != "test-model" || meta["contract_version"] != "1" {
			t.Error("protocol election absent")
		}
		opts := meta["options"].(map[string]any)
		if opts["context_digest"] != processing.HintsDigest([]string{"Exact Name"}) {
			t.Error("context election absent")
		}
		file, _, err := r.FormFile("audio")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		audio, _ := io.ReadAll(file)
		if len(audio) != 32044 {
			t.Error("mapped bytes changed")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"contract_version":"1","operation":"transcription","segments":[{"start_us":0,"end_us":500000,"text":"hello"}],"no_speech":false}`)
	}))
	defer server.Close()
	stage := remoteStage(server.URL)
	stage.CredentialID = "33333333-3333-4333-8333-333333333333"
	result, err := (&Executor{Secrets: secrets, Client: server.Client()}).Recognize(context.Background(), path, mapping, stage, processing.RecognitionOptions{Hints: []string{"Exact Name"}, ContextDigest: processing.HintsDigest([]string{"Exact Name"})})
	if err != nil || !strings.Contains(string(result.SRT), "00:00:00,334 --> 00:00:00,833") || secrets.calls != 1 {
		t.Fatalf("hosted result: %v", err)
	}
	if result.Provenance["adapter"] != "insonic-http" {
		t.Fatal("route provenance absent")
	}
}
func TestHostedRejectsRedirectWithoutSecondRequestAndRedactsErrors(t *testing.T) {
	path, mapping := fixtureAudio(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/error" {
			w.WriteHeader(403)
			io.WriteString(w, "synthetic-service-secret")
			return
		}
		w.Header().Set("Location", "/error")
		w.WriteHeader(307)
	}))
	defer server.Close()
	for _, suffix := range []string{"", "/error"} {
		_, err := (&Executor{Client: server.Client()}).Recognize(context.Background(), path, mapping, remoteStage(server.URL+suffix), processing.RecognitionOptions{})
		if err == nil || strings.Contains(err.Error(), "synthetic-service-secret") || strings.Contains(err.Error(), server.URL) {
			t.Fatal("provider error leaked or was accepted")
		}
	}
	if calls != 2 {
		t.Fatal("redirect followed")
	}
}
func TestHostedLimitsMalformedResultsAndNoSpeech(t *testing.T) {
	path, mapping := fixtureAudio(t)
	for _, body := range []string{`{"contract_version":"1","operation":"diarization","turns":[],"no_speech":true}`, `{"contract_version":"1","operation":"diarization","turns":[{"label":"A","start_us":0,"end_us":1000001}],"no_speech":false}`, `{"contract_version":"1","operation":"diarization","turns":[{"label":"A","start_us":0,"end_us":100000}],"no_speech":true}`, `{"contract_version":"1","operation":"diarization","turns":[],"no_speech":true,"no_speech":false}`, `{"contract_version":"2","operation":"diarization","turns":[],"no_speech":true}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
		stage := remoteStage(server.URL)
		result, err := (&Executor{Client: server.Client()}).Diarize(context.Background(), path, mapping, stage, processing.DiarizationOptions{})
		if body == `{"contract_version":"1","operation":"diarization","turns":[],"no_speech":true}` {
			if err != nil || !result.NoSpeech {
				t.Fatal("valid no speech refused")
			}
		} else if err == nil {
			t.Fatal("invalid response accepted")
		}
		stage.Limits.MaxResponseBytes = 16
		if _, err := (&Executor{Client: server.Client()}).Diarize(context.Background(), path, mapping, stage, processing.DiarizationOptions{}); err == nil {
			t.Fatal("response limit ignored")
		}
		stage.Limits.MaxAudioBytes = 16
		stage.Limits.MaxResponseBytes = 0
		if _, err := (&Executor{Client: server.Client()}).Diarize(context.Background(), path, mapping, stage, processing.DiarizationOptions{}); err == nil {
			t.Fatal("upload limit ignored")
		}
		server.Close()
	}
}
func TestHostedCancellationAndCIBoundary(t *testing.T) {
	path, mapping := fixtureAudio(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := (&Executor{Client: server.Client()}).Diarize(ctx, path, mapping, remoteStage(server.URL), processing.DiarizationOptions{})
	if typed, ok := err.(*contracts.Error); !ok || typed.Code != "cancelled" {
		t.Fatalf("cancellation: %v", err)
	}
	t.Setenv("CI", "true")
	secrets := &fixtureSecrets{}
	_, err = (&Executor{Secrets: secrets}).Diarize(context.Background(), "missing", mapping, remoteStage("https://example.test/process"), processing.DiarizationOptions{})
	if typed, ok := err.(*contracts.Error); !ok || typed.Code != "engine_ci_forbidden" || secrets.calls != 0 {
		t.Fatalf("CI launched production inference: %v", err)
	}
}

func TestHostedRequiresExplicitIntervalEvidenceAndRespectsDisabledQuality(t *testing.T) {
	path, mapping := fixtureAudio(t)
	for _, body := range []string{
		`{"contract_version":"1","operation":"diarization","turns":[{"label":"A","end_us":100000}],"no_speech":false}`,
		`{"contract_version":"1","operation":"diarization","turns":[{"label":"A","start_us":null,"end_us":100000}],"no_speech":false}`,
		`{"contract_version":"1","operation":"diarization","turns":[{"label":"A","start_us":0,"end_us":100000}],"no_speech":false}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
		disabled := false
		result, err := (&Executor{Client: server.Client()}).Diarize(context.Background(), path, mapping, remoteStage(server.URL), processing.DiarizationOptions{Quality: &processing.QualityConfig{Enabled: &disabled}})
		valid := strings.Contains(body, `"start_us":0`)
		if !valid && err == nil {
			t.Fatal("missing interval fabricated source-start evidence")
		}
		if valid && (err != nil || len(result.Diagnostics) != 0 || result.Provenance["quality_diagnostics_enabled"] != false) {
			t.Fatal("disabled analysis ignored")
		}
		server.Close()
	}
}
