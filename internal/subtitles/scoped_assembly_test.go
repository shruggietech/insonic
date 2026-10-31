// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestScopedAssemblyKeepsConcurrentSourceTracksIndependent(t *testing.T) {
	var doc map[string]any
	decoder := json.NewDecoder(bytes.NewReader(testDocument(t)))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	cues := doc["cues"].([]any)
	copyBytes, _ := json.Marshal(cues[0])
	var second map[string]any
	json.Unmarshal(copyBytes, &second)
	second["id"], second["ordinal"], second["source_order"] = "cue-000001", 1, 1
	doc["cues"] = append(cues, second)
	doc["document"].(map[string]any)["cue_count"] = 2
	doc["stats"].(map[string]any)["cue_count"] = 2
	raw, _ := json.Marshal(doc)
	voices := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"}
	turns := []Turn{{voices[0], 0, 1000000000}, {voices[1], 0, 1000000000}}
	cueTracks := map[string]int{"cue-000000": 0, "cue-000001": 1}
	voiceTracks := map[string]int{voices[0]: 0, voices[1]: 1}
	result, err := AssembleDocumentScoped(raw, nil, turns, nil, cueTracks, voiceTracks)
	if err != nil {
		t.Fatal(err)
	}
	var accepted semanticDocument
	json.Unmarshal(result.Document, &accepted)
	for i, cue := range accepted.Cues {
		if len(cue.Attributions) != 1 || cue.Attributions[0].SpeakerID != voices[i] {
			t.Fatalf("cross-track voice assigned: %+v", accepted.Cues)
		}
	}
	delete(voiceTracks, voices[1])
	if _, err := AssembleDocumentScoped(raw, nil, turns, nil, cueTracks, voiceTracks); err == nil {
		t.Fatal("unknown source voice accepted")
	}
}
