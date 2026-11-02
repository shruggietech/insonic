// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type corruptAnswerRead struct {
	artifact.Store
	id string
}

func (s corruptAnswerRead) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	if p.ID == s.id {
		return io.NopCloser(bytes.NewReader(bytes.Repeat([]byte{'x'}, int(length)))), nil
	}
	return s.Store.OpenRange(ctx, p, offset, length)
}

type uncertainAnswerSetting struct {
	catalog.Catalog
	lost bool
}

func (s *uncertainAnswerSetting) PutNamedSetting(ctx context.Context, op, name string, expected int64, value json.RawMessage) (catalog.Receipt, error) {
	r, e := s.Catalog.PutNamedSetting(ctx, op, name, expected, value)
	if e == nil && strings.HasPrefix(name, "answer-context:") && !s.lost {
		s.lost = true
		return catalog.Receipt{}, contracts.Fail("unavailable")
	}
	return r, e
}

func TestAnswerCacheCorruptionLostOutcomeAndRestart(t *testing.T) {
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	entry := desktopMedia(t, a, false)
	desktopRecording(t, a, entry, 0, doc)
	r := answers.Request{Version: contracts.Version, Question: "Research"}
	uncertain := &uncertainAnswerSetting{Catalog: a.Catalog}
	a.Catalog = uncertain
	first, err := a.prepareAnswer(context.Background(), r)
	if err != nil || !uncertain.lost {
		t.Fatal("lost committed cache outcome", first, err)
	}
	service, err := a.artifactService()
	if err != nil {
		t.Fatal(err)
	}
	original := service.Store
	service.Store = corruptAnswerRead{original, first.Package.ID}
	repaired, err := a.prepareAnswer(context.Background(), r)
	service.Store = original
	if err != nil || repaired.CacheHit || repaired.Package.ID == first.Package.ID || !slices.Contains(repaired.Diagnostics, "cache_content_rejected") {
		t.Fatal("corrupt bytes reused", repaired, err)
	}
	if repaired.Usage.SourceBytes > repaired.Package.Request.Budget.SourceBytes {
		t.Fatal("corrupt fetch escaped budget", repaired.Usage)
	}
	if _, err = a.Catalog.ClaimRetirement(context.Background(), first.Package.ID, contracts.ID(), 0, time.Minute); err == nil {
		t.Fatal("historical answer package was not retained")
	}
	w := a.Workspace
	a.Close()
	w.Config.Profiles.Graph.Adapter = "ladybugdb"
	b, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	after, err := b.prepareAnswer(context.Background(), r)
	if err != nil || !after.CacheHit || after.Package.ID != repaired.Package.ID {
		t.Fatal("restart lost compatible retained package", after, err)
	}
	old, err := b.inspectAnswer(context.Background(), first.Package.ID)
	if err != nil || old.Package.ID != first.Package.ID {
		t.Fatal("historical artifact was overwritten", old, err)
	}
}
