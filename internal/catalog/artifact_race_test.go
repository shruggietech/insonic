// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func availableArtifact(t *testing.T, s *Store) Publication {
	t.Helper()
	ctx := context.Background()
	rev, e := s.Revision(ctx)
	if e != nil {
		t.Fatal(e)
	}
	profile := Profile{ID: contracts.ID(), Revision: 1, Role: "storage", Adapter: "filesystem", Version: contracts.Version, Configuration: []byte(`{"root":"artifacts"}`)}
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Profiles: []Profile{profile}}}); e != nil {
		t.Fatal(e)
	}
	p := Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: profile.ID, ProfileRevision: 1, Digest: strings.Repeat("b", 64), Size: 1, Kind: "other", Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
	p, e = s.BeginPublication(ctx, p, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	p.Verification = "sha256-readback"
	p, e = s.AdmitPublication(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func artifactRaceSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	for i := 0; i < 16; i++ {
		p := availableArtifact(t, s)
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() { <-start; results <- s.ArtifactReference(ctx, p.ID, contracts.ID(), false) }()
		go func() { <-start; _, e := s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second); results <- e }()
		close(start)
		a, b := <-results, <-results
		if (a == nil) == (b == nil) {
			t.Fatalf("race did not have exactly one winner: %v %v", a, b)
		}
	}
	p := availableArtifact(t, s)
	rev, _ := s.Revision(ctx)
	asset := Asset{ID: contracts.ID(), ArtifactID: p.ArtifactID, Role: "original"}
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Assets: []Asset{asset}}}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second); e == nil {
		t.Fatal("structural source reference retired")
	}
	p = availableArtifact(t, s)
	if _, e := s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Second); e != nil {
		t.Fatal(e)
	}
	rev, _ = s.Revision(ctx)
	asset = Asset{ID: contracts.ID(), ArtifactID: p.ArtifactID, Role: "original"}
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Assets: []Asset{asset}}}); e == nil {
		t.Fatal("generic commit bypassed retirement")
	}
}
func TestArtifactRetirementRaces(t *testing.T) {
	s := localStore(t, contracts.ID())
	artifactRaceSuite(t, s)
}
