// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"math"
	"testing"
)

func TestAdapterRejectsAmbiguousRoutesAndUnsupportedResume(t *testing.T) {
	a := Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "hosted", Architecture: "test", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, Endpoint: "https://example.com/train", RemoteModel: "trainer", UpstreamRevision: "abc"}
	if ValidateAdapter(a) != nil {
		t.Fatal("valid declared hosted adapter")
	}
	a.Endpoint = "https://example.com/train?token=secret"
	if ValidateAdapter(a) == nil {
		t.Fatal("query credential route accepted")
	}
	a.Endpoint = "https://example.com/train"
	a.Arguments = []string{"--secret"}
	if ValidateAdapter(a) == nil {
		t.Fatal("hosted/local settings mixed")
	}
}

func TestCosineDecisionNeverForcesRosterMembership(t *testing.T) {
	candidates := []Candidate{{SpeakerID: "a", VersionID: "v1", Vector: []float64{1, 0}}, {SpeakerID: "b", VersionID: "v2", Vector: []float64{0.99, 0.01}}}
	decision, err := Decide("local", []float64{1, 0}, 2_000_000, candidates, 0.7, 0.05, 1_000_000)
	if err != nil || decision.State != "ambiguous" {
		t.Fatalf("expected ambiguity: %+v %v", decision, err)
	}
	decision, err = Decide("local", []float64{-1, 0}, 2_000_000, candidates[:1], 0.7, 0.05, 1_000_000)
	if err != nil || decision.State != "unknown" {
		t.Fatalf("single member forced: %+v %v", decision, err)
	}
	decision, err = Decide("local", []float64{1, 0}, 100, candidates[:1], 0.7, 0.05, 1_000_000)
	if err != nil || decision.State != "unknown" {
		t.Fatalf("insufficient evidence accepted: %+v %v", decision, err)
	}
	if _, err = Decide("local", []float64{math.NaN(), 0}, 2_000_000, candidates, 0.7, 0.05, 1_000_000); err == nil {
		t.Fatal("nonfinite vector accepted")
	}
	identical := []Candidate{{SpeakerID: "a", Vector: []float64{1, 0}}, {SpeakerID: "b", Vector: []float64{1, 0}}}
	decision, err = Decide("local", []float64{1, 0}, 2_000_000, identical, 0.7, 0, 1_000_000)
	if err != nil || decision.State != "ambiguous" {
		t.Fatalf("equal scores at zero margin: %+v %v", decision, err)
	}
}

func TestHostedHandlesAreOpaqueStableIDs(t *testing.T) {
	adapter := Adapter{Mode: "hosted", Architecture: "custom", OutputKinds: []string{"weights"}}
	for _, handle := range []string{"https://host/model?token=secret", "//host/model", "provider\nsecret", " provider:17", "provider:model#token", "provider:run with spaces"} {
		output := Output{ContractVersion: "1", Kind: "weights", Architecture: "custom", SupportedOperations: []string{"custom-inference"}, HostedHandle: handle}
		if validateOutput(adapter, output) == nil {
			t.Fatal("transport or control handle accepted", handle)
		}
	}
	for _, handle := range []string{"provider:model/version-17", "namespace:model:17"} {
		if !opaqueHandle(handle) {
			t.Fatal("normal namespace handle rejected", handle)
		}
	}
}
