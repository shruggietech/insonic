// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func genericOptions(datasetID string) TrainOptions {
	return TrainOptions{DatasetID: datasetID, Name: "Generic", Kind: "weights", Adapter: Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "custom", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Executable: libraryPin()}}
}
func weightOutput(raw []byte) Output {
	return Output{ContractVersion: "1", Kind: "weights", Architecture: "custom", Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Artifacts: []Artifact{{Role: "weights.bin", Format: "binary", Data: raw, Size: int64(len(raw)), SHA256: hash(raw)}}}
}

type brokenReadStore struct {
	artifact.Store
	id      string
	calls   int
	failAt  int
	corrupt bool
}

func (s *brokenReadStore) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	if p.ID == s.id {
		s.calls++
		if s.calls >= s.failAt {
			if s.corrupt {
				return io.NopCloser(strings.NewReader(strings.Repeat("x", int(length)))), nil
			}
			return nil, contracts.Fail("unavailable")
		}
	}
	return s.Store.OpenRange(ctx, p, offset, length)
}

func TestExactFetchCorruptionAndMaterializationFailureLeaveNoDestinationOrLease(t *testing.T) {
	for _, corrupt := range []bool{true, false} {
		t.Run(strconv.FormatBool(corrupt), func(t *testing.T) {
			s, db := fixture(t)
			dataset, _ := datasetFixture(t, s, db, 1)
			raw := []byte("exact trained bytes")
			s.LocalRunner = func(context.Context, Adapter, AdapterRequest) (Output, error) { return weightOutput(raw), nil }
			result, err := s.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: genericOptions(dataset.Dataset.ID)}))
			if err != nil {
				t.Fatal(err)
			}
			version := result.(map[string]any)["version_id"].(string)
			output, err := s.Show(context.Background(), version)
			if err != nil {
				t.Fatal(err)
			}
			descriptors, err := ManifestArtifactDescriptors(output.Metadata)
			if err != nil || len(descriptors) != 1 {
				t.Fatal(descriptors, err)
			}
			failAt := 2
			if corrupt {
				failAt = 1
			}
			s.Artifacts.Store = &brokenReadStore{Store: s.Artifacts.Store, id: descriptors[0].PublicationID, failAt: failAt, corrupt: corrupt}
			destination := filepath.Join(t.TempDir(), "exact")
			if _, err = s.Fetch(context.Background(), version, destination); err == nil {
				t.Fatal("broken exact artifact fetched")
			}
			if _, err = os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatal("partial destination visible", err)
			}
			publication, err := db.Publication(context.Background(), descriptors[0].PublicationID)
			if err != nil || len(publication.Leases) != 0 {
				t.Fatal("failed retrieval retained held lease", publication.Leases, err)
			}
		})
	}
}

func TestHostedHandleOnlyOutputExplainsUnsupportedFileFetch(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := weightOutput([]byte("unused"))
		o.Artifacts = nil
		o.HostedHandle = "selected-provider-version-17"
		json.NewEncoder(w).Encode(o)
	}))
	defer server.Close()
	s.Client = server.Client()
	options := genericOptions(dataset.Dataset.ID)
	options.Adapter.Mode = "hosted"
	options.Adapter.Executable = library.PinnedFile{}
	options.Adapter.Endpoint = server.URL
	options.Adapter.RemoteModel = "trainer"
	options.Adapter.UpstreamRevision = "immutable"
	result, err := s.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: options}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Fetch(context.Background(), result.(map[string]any)["version_id"].(string), filepath.Join(t.TempDir(), "hosted"))
	if failure, ok := err.(*contracts.Error); !ok || failure.Code != "unsupported_retrieval" {
		t.Fatal("hosted-only retrieval misreported", err)
	}
}

func TestHostedSecretURLCannotBecomePortableModelHandle(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		output := weightOutput([]byte("unused"))
		output.Artifacts = nil
		output.HostedHandle = "https://provider.invalid/download?token=secret"
		json.NewEncoder(w).Encode(output)
	}))
	defer server.Close()
	s.Client = server.Client()
	options := genericOptions(dataset.Dataset.ID)
	options.Adapter.Mode = "hosted"
	options.Adapter.Executable = library.PinnedFile{}
	options.Adapter.Endpoint = server.URL
	options.Adapter.RemoteModel = "trainer"
	options.Adapter.UpstreamRevision = "immutable"
	if _, err := s.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: options})); err == nil {
		t.Fatal("temporary secret URL accepted as model")
	}
	if outputs, err := s.List(context.Background(), dataset.Dataset.SpeakerID); err != nil || len(outputs) != 0 {
		t.Fatal("invalid hosted model published", outputs, err)
	}
}

func TestSourceCorrectionDuringTrainingRejectsAndRetiresOwnedPublications(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 1)
	raw := []byte("stale attempted weights")
	s.LocalRunner = func(ctx context.Context, _ Adapter, _ AdapterRequest) (Output, error) {
		mappings, err := db.SpeakerMappings(ctx, dataset.References[0].RecordingID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.SetSpeakerMapping(ctx, contracts.ID(), dataset.References[0].RecordingRevision, mappings[0]); err != nil {
			t.Fatal(err)
		}
		return weightOutput(raw), nil
	}
	claim := work(t, db, "models.train", TrainPayload{Options: genericOptions(dataset.Dataset.ID)})
	if _, err := s.ExecuteTrain(context.Background(), claim); err == nil {
		t.Fatal("stale corpus accepted")
	}
	if outputs, err := s.List(context.Background(), dataset.Dataset.SpeakerID); err != nil || len(outputs) != 0 {
		t.Fatal("stale model published", outputs, err)
	}
	ids := []string{models.StableID("speaker-artifact:" + claim.ID + ":" + strconv.FormatInt(claim.Generation, 10) + ":0:" + hash(raw)), models.StableID("speaker-preparation:" + claim.ID + ":" + strconv.FormatInt(claim.Generation, 10)), models.StableID("speaker-output-manifest:" + claim.ID + ":" + strconv.FormatInt(claim.Generation, 10))}
	for _, id := range ids {
		if publication, err := db.Publication(context.Background(), id); err == nil && publication.State == "available" {
			t.Fatal("unaccepted managed output not retired", id, publication.State)
		}
	}
}

func TestMaliciousOutputPathAndFalseContentHashAreRejected(t *testing.T) {
	s, db := fixture(t)
	directory, err := s.scratch()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	claim := work(t, db, "models.train", map[string]any{})
	for _, a := range []Artifact{{Role: "../escape", Format: "binary", Data: []byte("x"), Size: 1, SHA256: hash([]byte("x"))}, {Role: "weights.bin", Path: "../escape", Format: "binary", Size: 1, SHA256: hash([]byte("x"))}, {Role: "weights.bin", Format: "binary", Data: []byte("x"), Size: 1, SHA256: hash([]byte("y"))}} {
		if _, err = s.publishOutput(context.Background(), claim, a, directory, 0, "speaker-model"); err == nil {
			t.Fatal("malicious output accepted", a.Role, a.Path)
		}
	}
}
