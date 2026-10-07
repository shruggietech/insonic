// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestLargeNormalizedManifestReachesDurableRuntime(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "large ingress")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready, done := make(chan struct{}), make(chan error, 1)
	go func() { done <- Serve(ctx, w, Options{Idle: time.Minute, Ready: ready}) }()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("startup")
	}
	input := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version, Defaults: library.Options{Timezone: "UTC"}}
	for i := 0; i < 4000; i++ {
		input.Items = append(input.Items, library.Item{Source: "missing.wav", Title: strings.Repeat("x", 300)})
	}
	input, e = library.PrepareImport(input)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(input)
	if len(raw) <= MaxFrame {
		t.Fatal("fixture did not exercise large ingress")
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "media.import", Data: raw}
	response, e := Call(ctx, w, req)
	if e != nil || response.Error != nil {
		t.Fatal("valid batch rejected", response.Error, e)
	}
	cancel()
	select {
	case e = <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown")
	}
	db, e := catalog.OpenWorkspace(context.Background(), w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	work, e := db.Work(context.Background(), req.RequestID)
	if e != nil || len(work.Payload) <= MaxFrame {
		t.Fatal("batch was not durable", e)
	}
	var recovered library.ImportRequest
	if json.Unmarshal(work.Payload, &recovered) != nil || len(recovered.Items) != 4000 || recovered.Items[3999].Title != input.Items[3999].Title {
		t.Fatal("batch content changed")
	}
}

func TestRequestAndResponseFramesHaveSeparateBudgets(t *testing.T) {
	large := strings.Repeat("x", MaxFrame+10) + "\n"
	if _, e := frameLimit(strings.NewReader(large), MaxRequestFrame); e != nil {
		t.Fatal("large request frame rejected")
	}
	if _, e := frame(strings.NewReader(large)); e == nil {
		t.Fatal("large response frame accepted")
	}
	if _, e := frameLimit(strings.NewReader(strings.Repeat("x", MaxRequestFrame+1)+"\n"), MaxRequestFrame); e == nil {
		t.Fatal("oversized request frame accepted")
	}
}
