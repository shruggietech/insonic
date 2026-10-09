// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/shruggietech/insonic/internal/library"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHostedResultCannotPublishPreparedScratchThroughArtifactOrCheckpointPaths(t *testing.T) {
	for _, checkpoint := range []bool{false, true} {
		t.Run(map[bool]string{false: "artifact", true: "checkpoint"}[checkpoint], func(t *testing.T) {
			service, db := fixture(t)
			dataset, _ := datasetFixture(t, service, db, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Uploads []struct {
						ID     string `json:"id"`
						SHA256 string `json:"sha256"`
						Size   int64  `json:"size"`
					} `json:"uploads"`
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Uploads) != 1 {
					t.Error("elected upload unavailable")
					return
				}
				input := request.Uploads[0]
				output := weightOutput([]byte("unused"))
				output.Artifacts = nil
				stolen := Artifact{Role: "weights.bin", Format: "binary", Path: input.ID + ".wav", SHA256: input.SHA256, Size: input.Size}
				if checkpoint {
					output.State = "checkpointed"
					output.Checkpoints = []Checkpoint{{Step: 1, Compatibility: "optimizer-v1", Artifact: stolen}}
				} else {
					output.Artifacts = []Artifact{stolen}
				}
				json.NewEncoder(w).Encode(output)
			}))
			defer server.Close()
			service.Client = server.Client()
			options := hostedOptions(dataset.Dataset.ID, server.URL)
			options.Adapter.SupportsResume = checkpoint
			claim := work(t, db, "models.train", TrainPayload{Options: options})
			if _, err := service.ExecuteTrain(context.Background(), claim); err == nil {
				t.Fatal("remote selected caller-owned scratch bytes")
			}
			if outputs, err := service.List(context.Background(), dataset.Dataset.SpeakerID); err != nil || len(outputs) != 0 {
				t.Fatal("remote scratch published", outputs, err)
			}
			if checkpoint {
				current, err := db.Work(context.Background(), claim.ID)
				if err != nil || current.Phase == "checkpointed" {
					t.Fatal("scratch checkpoint accepted", current.Phase, err)
				}
			}
		})
	}
}

func TestHostedPrivateTimeoutSendsFreshBoundedCancellation(t *testing.T) {
	for _, bodyTimeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting_response", true: "reading_body"}[bodyTimeout], func(t *testing.T) {
			service, db := fixture(t)
			dataset, _ := datasetFixture(t, service, db, 1)
			remoteJob := make(chan struct{})
			cancelled := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Operation string            `json:"operation"`
					WorkID    string            `json:"work_id"`
					Uploads   []json.RawMessage `json:"uploads"`
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil {
					t.Error("bad hosted request")
					return
				}
				if request.Operation == "train" {
					if bodyTimeout {
						w.WriteHeader(http.StatusOK)
						w.(http.Flusher).Flush()
					}
					<-remoteJob
					return
				}
				if request.Operation != "cancel" || request.WorkID == "" || len(request.Uploads) != 0 {
					t.Error("invalid selected cancellation", request)
				}
				// The remote task survives disconnection and stops only upon this explicit request.
				time.Sleep(75 * time.Millisecond)
				cancelled <- struct{}{}
				close(remoteJob)
				w.Write([]byte(`{}`))
			}))
			defer func() {
				select {
				case <-remoteJob:
				default:
					close(remoteJob)
				}
				server.Close()
			}()
			service.Client = server.Client()
			options := hostedOptions(dataset.Dataset.ID, server.URL)
			options.Adapter.SupportsCancel = true
			options.Adapter.Limits.TimeoutMS = 25
			_, err := service.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: options}))
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("private adapter deadline lost", err)
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("remote task survives timeout without declared cancel")
			}
		})
	}
}

func hostedOptions(datasetID, endpoint string) TrainOptions {
	options := genericOptions(datasetID)
	options.Adapter.Mode = "hosted"
	options.Adapter.Executable = library.PinnedFile{}
	options.Adapter.Endpoint = endpoint
	options.Adapter.RemoteModel = "trainer"
	options.Adapter.UpstreamRevision = "immutable"
	return options
}
