// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestPublicationExpiryAndReplay(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	profile := Profile{ID: contracts.ID(), Revision: 1, Role: "storage", Adapter: "filesystem", Version: contracts.Version, Configuration: []byte(`{"root":"artifacts"}`)}
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: 0, Records: Records{Profiles: []Profile{profile}}}); e != nil {
		t.Fatal(e)
	}
	p := Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: profile.ID, ProfileRevision: 1, Digest: strings.Repeat("c", 64), Size: 0, Kind: "other", Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
	p, e := s.BeginPublication(ctx, p, time.Millisecond)
	if e != nil {
		t.Fatal(e)
	}
	time.Sleep(3 * time.Millisecond)
	old := p
	old.Verification = "sha256-readback"
	if _, e = s.AdmitPublication(ctx, old); e == nil {
		t.Fatal("expired admission")
	}
	p.Owner = contracts.ID()
	p, e = s.BeginPublication(ctx, p, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SavePublication(ctx, old, time.Second); e == nil {
		t.Fatal("stale replacement")
	}
	p.Verification = "sha256-readback"
	p, e = s.AdmitPublication(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	p.Owner = contracts.ID()
	again, e := s.BeginPublication(ctx, p, time.Second)
	if e != nil || again.AdmissionID != p.AdmissionID {
		t.Fatal("accepted replay failed")
	}
}
