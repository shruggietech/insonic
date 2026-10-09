// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"encoding/json"
	"os"
	"testing"
)

func testDocument(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func mutateDocument(t *testing.T, mutate func(map[string]json.RawMessage)) []byte {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(testDocument(t), &fields); err != nil {
		t.Fatal(err)
	}
	mutate(fields)
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestOfflineCurrentValidation(t *testing.T) {
	if err := ValidateDocument(testDocument(t)); err != nil {
		t.Fatal(err)
	}
	annotated := mutateDocument(t, func(fields map[string]json.RawMessage) {
		var cues []map[string]json.RawMessage
		json.Unmarshal(fields["cues"], &cues)
		cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"11111111-1111-4111-8111-111111111111","start_milliseconds":100,"end_milliseconds":400},{"speaker_id":"22222222-2222-4222-8222-222222222222","start_milliseconds":200,"end_milliseconds":600},{"speaker_id":"11111111-1111-4111-8111-111111111111"}]`)
		fields["cues"], _ = json.Marshal(cues)
		fields["media_timing"] = json.RawMessage(`{"duration_milliseconds":1000}`)
	})
	if err := ValidateDocument(annotated); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		assignments string
		media       string
	}{
		{"leading-whitespace-id", `[{"speaker_id":" global-name"}]`, `{"duration_milliseconds":1000}`},
		{"outside-cue", `[{"speaker_id":"11111111-1111-4111-8111-111111111111","start_milliseconds":100,"end_milliseconds":1001}]`, `{"duration_milliseconds":2000}`},
		{"outside-media", `[{"speaker_id":"11111111-1111-4111-8111-111111111111","start_milliseconds":100,"end_milliseconds":600}]`, `{"duration_milliseconds":500}`},
		{"half-pair", `[{"speaker_id":"11111111-1111-4111-8111-111111111111","start_milliseconds":100}]`, `{"duration_milliseconds":1000}`},
		{"collapsed", `[{"speaker_id":"11111111-1111-4111-8111-111111111111","start_milliseconds":100,"end_milliseconds":100}]`, `{"duration_milliseconds":1000}`},
		{"overflow", `[]`, `{"duration_milliseconds":9223372036854775807,"timeline_start_milliseconds":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := mutateDocument(t, func(fields map[string]json.RawMessage) {
				var cues []map[string]json.RawMessage
				json.Unmarshal(fields["cues"], &cues)
				cues[0]["speaker_attributions"] = json.RawMessage(tc.assignments)
				fields["cues"], _ = json.Marshal(cues)
				fields["media_timing"] = json.RawMessage(tc.media)
			})
			if ValidateDocument(data) == nil {
				t.Fatal("invalid current document accepted")
			}
		})
	}
}
func TestOfflineRejectsHistoricalIdentityAndSourceCorruption(t *testing.T) {
	for _, mutate := range []func(map[string]json.RawMessage){
		func(fields map[string]json.RawMessage) { fields["schema_version"] = json.RawMessage(`"0.0.1"`) },
		func(fields map[string]json.RawMessage) {
			fields["$schema"] = json.RawMessage(`"https://attacker.invalid/schema"`)
		},
		func(fields map[string]json.RawMessage) {
			var source map[string]json.RawMessage
			json.Unmarshal(fields["source"], &source)
			var assets []map[string]json.RawMessage
			json.Unmarshal(source["assets"], &assets)
			assets[0]["data_base64"] = json.RawMessage(`"AAAA"`)
			source["assets"], _ = json.Marshal(assets)
			fields["source"], _ = json.Marshal(source)
		},
	} {
		if ValidateDocument(mutateDocument(t, mutate)) == nil {
			t.Fatal("bad source/identity accepted")
		}
	}
	data := testDocument(t)
	duplicate := append([]byte(`{"schema_version":"0.0.1",`), data[1:]...)
	if ValidateDocument(duplicate) == nil {
		t.Fatal("duplicate property accepted")
	}
}
func TestOfflineAllowsUnknownDurationAndMeasuredZero(t *testing.T) {
	data := mutateDocument(t, func(fields map[string]json.RawMessage) {
		fields["media_timing"] = json.RawMessage(`{"duration_milliseconds":0}`)
	})
	if err := ValidateDocument(data); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDocument(testDocument(t)); err != nil {
		t.Fatal(err)
	}
}

func TestOfflineRejectsLossyUnicodeAndTimestampPrecision(t *testing.T) {
	invalid := mutateDocument(t, func(fields map[string]json.RawMessage) {
		var metadata map[string]json.RawMessage
		json.Unmarshal(fields["metadata"], &metadata)
		metadata["title"] = json.RawMessage(`"\ud800"`)
		fields["metadata"], _ = json.Marshal(metadata)
	})
	if ValidateDocument(invalid) == nil {
		t.Fatal("accepted lossy surrogate escape")
	}
	invalid = mutateDocument(t, func(fields map[string]json.RawMessage) {
		var source map[string]json.RawMessage
		json.Unmarshal(fields["source"], &source)
		var assets []map[string]json.RawMessage
		json.Unmarshal(source["assets"], &assets)
		var timestamps map[string]json.RawMessage
		json.Unmarshal(assets[0]["timestamps"], &timestamps)
		timestamps["modified"] = json.RawMessage(`{"iso":"2026-10-07T20:46:13.3479284001Z","unix_ns":1791405973347928400}`)
		assets[0]["timestamps"], _ = json.Marshal(timestamps)
		source["assets"], _ = json.Marshal(assets)
		fields["source"], _ = json.Marshal(source)
	})
	if ValidateDocument(invalid) == nil {
		t.Fatal("accepted truncated timestamp precision")
	}
}
