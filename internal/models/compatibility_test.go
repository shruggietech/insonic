// SPDX-License-Identifier: Apache-2.0
package models

import "testing"

func TestModelCompatibilityBeforeAcquisition(t *testing.T) {
	m := fixtureManifest()
	m.Capabilities = []string{"transcription"}
	m.Files = nil
	for _, role := range []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"} {
		m.Files = append(m.Files, File{Role: role, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 1, URL: "https://example.org/" + role})
	}
	if e := CheckCompatibility(m, "transcription", "faster-whisper", "1"); e != nil {
		t.Fatal(e)
	}
	m.Compatibility = &Compatibility{Adapter: "faster-whisper", ContractVersion: "1", Architecture: "unknown", RuntimeFormat: "ctranslate2"}
	if m.Validate() != nil {
		t.Fatal("unsupported architecture blocked registration")
	}
	if e := CheckCompatibility(m, "transcription", "faster-whisper", "1"); e == nil {
		t.Fatal("unknown architecture executed")
	}
	m.Compatibility.Architecture = "whisper"
	m.Files = m.Files[:3]
	if e := CheckCompatibility(m, "transcription", "faster-whisper", "1"); e == nil {
		t.Fatal("missing roles executed")
	}
	m.Files = append(m.Files, File{Role: "vocabulary.txt", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 1, URL: "https://example.org/vocab"})
	m.Compatibility.SampleRates = []int{16000}
	m.Compatibility.Channels = []int{1}
	if e := CheckCompatibility(m, "transcription", "faster-whisper", "1"); e != nil {
		t.Fatal(e)
	}
	m.Compatibility.Channels = []int{2}
	if e := CheckCompatibility(m, "transcription", "faster-whisper", "1"); e == nil {
		t.Fatal("audio incompatibility")
	}
	if e := CheckCompatibility(m, "diarization", "pyannote", "1"); e == nil {
		t.Fatal("wrong operation")
	}
}
