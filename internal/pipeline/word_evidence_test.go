// SPDX-License-Identifier: Apache-2.0
package pipeline

import "testing"

func TestHostedWordsRequireExplicitIntervals(t *testing.T) {
	for _, word := range []string{`{"end_us":1,"text":"x"}`, `{"start_us":null,"end_us":1,"text":"x"}`, `{"start_us":0,"text":"x"}`, `{"start_us":0,"end_us":1}`} {
		raw := []byte(`{"segments":[{"start_us":0,"end_us":1,"text":"x","words":[` + word + `]}]}`)
		if requiredIntervals(raw, "transcription") {
			t.Fatalf("incomplete word accepted: %s", word)
		}
	}
	if !requiredIntervals([]byte(`{"segments":[{"start_us":0,"end_us":1,"text":"x","words":[{"start_us":0,"end_us":1,"text":"x"}]}]}`), "transcription") {
		t.Fatal("valid word rejected")
	}
}
