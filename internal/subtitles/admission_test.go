// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"strings"
	"testing"
)

func TestPreserveCurrentAndHistorical(t *testing.T) {
	current := testDocument(t)
	result, err := Preserve(current)
	if err != nil || !bytes.Equal(result.Document, current) || result.Translated {
		t.Fatalf("current preservation: %v", err)
	}
	paths, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if !strings.HasPrefix(path.Name(), "historical-") {
			continue
		}
		t.Run(path.Name(), func(t *testing.T) {
			original, _ := os.ReadFile("testdata/" + path.Name())
			result, err := Preserve(original)
			if err != nil || !result.Translated {
				t.Fatalf("translation: %v", err)
			}
			var before, after map[string]json.RawMessage
			json.Unmarshal(original, &before)
			json.Unmarshal(result.Document, &after)
			for key, raw := range before {
				if key == "$schema" || key == "schema_version" {
					continue
				}
				var a, b any
				d := json.NewDecoder(bytes.NewReader(raw))
				d.UseNumber()
				d.Decode(&a)
				d = json.NewDecoder(bytes.NewReader(after[key]))
				d.UseNumber()
				d.Decode(&b)
				one, _ := json.Marshal(a)
				two, _ := json.Marshal(b)
				if !bytes.Equal(one, two) {
					t.Fatalf("field changed: %s", key)
				}
			}
			before["schema_version"] = json.RawMessage(`"1.1.0"`)
			before["$schema"] = json.RawMessage(`"https://foreign.invalid/schema"`)
			bad, _ := json.Marshal(before)
			if _, err = Preserve(bad); err == nil {
				t.Fatal("foreign schema accepted")
			}
		})
	}
}
func TestAttributionAuthorityAndScope(t *testing.T) {
	data := mutateDocument(t, func(fields map[string]json.RawMessage) {
		var cues []map[string]json.RawMessage
		json.Unmarshal(fields["cues"], &cues)
		cues[0]["speakers"] = json.RawMessage(`[{"name":"Alex","origin":"native"}]`)
		fields["cues"], _ = json.Marshal(cues)
	})
	input, err := Preserve(data)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Attribute(input, contracts.ID(), "auto")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Attribute(input, contracts.ID(), "native")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.AttributionBasis) != 1 || first.AttributionBasis[0].SpeakerID == second.AttributionBasis[0].SpeakerID {
		t.Fatal("cross-recording identity collision")
	}
	if bytes.Contains(first.Document, []byte(`"start_milliseconds":null`)) {
		t.Fatal("invented time")
	}
	preserved, err := Attribute(first, contracts.ID(), "auto")
	if err != nil || !bytes.Equal(first.Document, preserved.Document) {
		t.Fatal("existing attribution changed")
	}
	off, err := Attribute(input, contracts.ID(), "off")
	if err != nil || !bytes.Equal(off.Document, data) {
		t.Fatal("off changed observations")
	}
}
func TestLocalSpeakerTokens(t *testing.T) {
	for _, token := range []string{"voice one", "日本語", "e\u0301", strings.Repeat("🙂", 256)} {
		if !contracts.ValidLocalSpeakerID(token) {
			t.Fatalf("rejected %q", token)
		}
		data := mutateDocument(t, func(fields map[string]json.RawMessage) {
			var cues []map[string]json.RawMessage
			json.Unmarshal(fields["cues"], &cues)
			cues[0]["speaker_attributions"], _ = json.Marshal([]attribution{{SpeakerID: token}})
			fields["cues"], _ = json.Marshal(cues)
		})
		if err := ValidateDocument(data); err != nil {
			t.Fatal(err)
		}
	}
	for _, token := range []string{"", " leading", "trailing\u3000", "a\u202eb", strings.Repeat("🙂", 257), string([]byte{0xff})} {
		if contracts.ValidLocalSpeakerID(token) {
			t.Fatalf("accepted %q", token)
		}
	}
}

func TestNativeLabelOriginsAndPartialNonClobber(t *testing.T) {
	data := mutateDocument(t, func(fields map[string]json.RawMessage) {
		var cues []map[string]json.RawMessage
		json.Unmarshal(fields["cues"], &cues)
		cues[0]["speakers"] = json.RawMessage(`[{"name":"Alex","origin":"native"},{"name":"Alex","origin":"native"},{"name":"alex","origin":"native"},{"name":"é","origin":"native"},{"name":"e\u0301","origin":"native"},{"name":"Alex","origin":"heuristic"}]`)
		fields["cues"], _ = json.Marshal(cues)
	})
	input, err := Preserve(data)
	if err != nil {
		t.Fatal(err)
	}
	recording := contracts.ID()
	auto, err := Attribute(input, recording, "auto")
	if err != nil || len(auto.AttributionBasis) != 5 {
		t.Fatal("case Unicode repetition or origin collapsed", err, len(auto.AttributionBasis))
	}
	native, err := Attribute(input, recording, "native")
	if err != nil || len(native.AttributionBasis) != 4 {
		t.Fatal("native origin election failed", err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(data, &fields)
	var cues []map[string]json.RawMessage
	json.Unmarshal(fields["cues"], &cues)
	cues[0]["speaker_attributions"] = json.RawMessage(`[{"speaker_id":"Supplied voice"}]`)
	fields["cues"], _ = json.Marshal(cues)
	data, _ = json.Marshal(fields)
	input, err = Preserve(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"auto", "native", "off"} {
		out, err := Attribute(input, recording, mode)
		if err != nil || !bytes.Equal(out.Document, data) {
			t.Fatal("supplied attribution clobbered", mode, err)
		}
	}
}
