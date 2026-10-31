// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestImportDiarizationDeviceFlag(t *testing.T) {
	for _, device := range []string{"auto", "cpu", "cuda"} {
		_, _, data, err := parseDomain([]string{"transcript", "import", "caption.srt", "--record", contracts.ID(), "--attribution", "diarize", "--diarization-model-id", contracts.ID(), "--diarization-device", device})
		if err != nil {
			t.Fatal(device, err)
		}
		var request map[string]any
		if json.Unmarshal(data, &request) != nil || request["defaults"].(map[string]any)["diarization_device"] != device {
			t.Fatal(string(data))
		}
	}
	if _, _, _, err := parseDomain([]string{"media", "import", "source.wav", "--diarization-device", "other"}); err == nil {
		t.Fatal("unknown device accepted")
	}
}
