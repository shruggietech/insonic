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
