// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestDeclaredRosterProjectionIsIndependent(t *testing.T) {
	id, sp := contracts.ID(), contracts.ID()
	s := catalog.Snapshot{Revision: 4, Records: catalog.Records{Library: []catalog.LibraryEntry{{ID: id, Revision: 2, Title: "Audio only"}}, Speakers: []catalog.Speaker{{ID: sp, Name: "Declared", Revision: 3, State: "active"}}, Rosters: []catalog.RosterHeader{{ID: id, Revision: 4}}, RosterMembers: []catalog.RosterMember{{RecordingID: id, SpeakerID: sp, Revision: 4}}}}
	c, e := Build(s)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, edge := range c.Refs.Edges {
		if edge.Kind == "declared-speaker" && edge.From == "media:"+id && edge.To == "speaker:"+sp {
			found = true
		}
		if edge.Kind == "maps-to" {
			t.Fatal("roster inferred mapping")
		}
	}
	if !found || c.Rows["roster:"+id].Kind != "declared-roster" {
		t.Fatal("roster context missing")
	}
}

func TestSelectedSpeakerProfileLinksExactOutputWithoutReplacingOrigin(t *testing.T) {
	sp, family, version := contracts.ID(), contracts.ID(), contracts.ID()
	s := catalog.Snapshot{Records: catalog.Records{Speakers: []catalog.Speaker{{ID: sp, Name: "Known", Revision: 1, State: "active"}}, Models: []catalog.Model{{ID: family, SpeakerID: sp, Name: "Profile"}}, Versions: []catalog.ModelVersion{{ID: version, ModelID: family, Kind: "voice-embedding", State: "invalidated"}}, SpeakerOutputs: []catalog.SpeakerOutput{{ID: version, ModelID: family, SpeakerID: sp, Kind: "voice-embedding", Name: "Profile"}}, SpeakerProfiles: []catalog.SpeakerProfile{{ID: sp, Revision: 3, VersionID: version, State: "active"}}}}
	c, e := Build(s)
	if e != nil {
		t.Fatal(e)
	}
	selected := false
	for _, edge := range c.Refs.Edges {
		if edge.Kind == "uses-profile" && edge.From == "speaker:"+sp && edge.To == "version:"+version {
			selected = true
		}
	}
	if !selected {
		t.Fatal("exact selected profile missing from graph")
	}
	s.Records.SpeakerProfiles[0].State = "cleared"
	s.Records.SpeakerProfiles[0].VersionID = ""
	c, e = Build(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, edge := range c.Refs.Edges {
		if edge.Kind == "uses-profile" {
			t.Fatal("cleared profile projected as active")
		}
	}
	if _, ok := c.Rows["version:"+version]; !ok {
		t.Fatal("profile clear removed immutable version")
	}
}
