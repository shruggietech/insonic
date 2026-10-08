// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestPipelineElectionCannotDisappearOrDivergeInDurableWork(t *testing.T) {
	a := configuredApp(t)
	id := contracts.ID()
	configuredResult(t, realRequest(a, "pipelines.set", id, map[string]any{"expected_revision": 0, "pipeline": map[string]any{"id": id, "name": "bound", "preset": "local", "configuration": localDefinition()}}))
	o, election, e := a.electRecordingOptions(RecordingOptions{PipelineID: id, Transcription: "generate", Diarization: "run"})
	if e != nil {
		t.Fatal(e)
	}
	p := recordingPayload{Options: o, Election: election}
	if validateElection(p) != nil {
		t.Fatal("valid election rejected")
	}
	p.Election = nil
	if validateElection(p) == nil {
		t.Fatal("missing elected routing accepted")
	}
	p.Election = election
	p.Options.RecognitionModelID = contracts.ID()
	if validateElection(p) == nil {
		t.Fatal("different execution model accepted")
	}
	p.Options = o
	p.Options.Recognition.ContextDigest = "changed"
	if validateElection(p) == nil {
		t.Fatal("different hints digest accepted")
	}
}
