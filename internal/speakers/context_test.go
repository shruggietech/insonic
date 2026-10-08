// SPDX-License-Identifier: Apache-2.0
package speakers

import (
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"testing"
)

func TestContextDeterministicUTF8BudgetAndScopes(t *testing.T) {
	s := catalog.ContextSnapshot{Revision: 7, Filter: catalog.ContextFilter{Language: "en"}, ExtraHints: []string{"éé", "Jane"}, Speakers: []catalog.SpeakerIdentity{{Speaker: catalog.Speaker{ID: "b", Name: "Jane", State: "active"}, Aliases: []catalog.SpeakerAlias{{ID: "c", Text: "Janie", State: "active", Language: "en"}, {ID: "d", Text: "Wrong", State: "active", Language: "fr"}}}}, Terms: []catalog.Term{{ID: "a", Canonical: "spectrogram", Variants: json.RawMessage(`["plural"]`), State: "active"}, {ID: "z", Canonical: "inactive", Variants: json.RawMessage(`[]`), State: "inactive"}}}
	first, e := CompileContext(s, 13, true)
	if e != nil {
		t.Fatal(e)
	}
	second, _ := CompileContext(s, 13, true)
	if first.Digest != second.Digest || first.Joined != "éé, Jane" || first.Bytes > 13 {
		t.Fatal(first)
	}
	if len(first.Omitted) == 0 {
		t.Fatal("omissions hidden")
	}
	unsupported, e := CompileContext(s, 200, false)
	if e != nil || len(unsupported.Hints) != 0 || len(unsupported.Omitted) == 0 {
		t.Fatal(unsupported, e)
	}
	if _, e = CompileContext(s, 8193, true); e == nil {
		t.Fatal("unsafe local byte budget accepted")
	}
	s.Terms[0].Canonical = "changed"
	changed, _ := CompileContext(s, 13, true)
	if changed.SnapshotDigest == first.SnapshotDigest {
		t.Fatal("input changes not recorded")
	}
}
func TestContextOmissionDetailsBoundedWithExactTotals(t *testing.T) {
	snapshot := catalog.ContextSnapshot{}
	for i := 0; i < 3000; i++ {
		snapshot.Speakers = append(snapshot.Speakers, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: fmt.Sprintf("speaker-%04d", i), Name: "name", State: "active"}})
	}
	compiled, e := CompileContext(snapshot, 0, false)
	if e != nil {
		t.Fatal(e)
	}
	if len(compiled.Omitted) != 2048 || compiled.OmittedTotal != 3000 || !compiled.OmittedTruncated {
		t.Fatal("omission accounting", compiled.OmittedTotal, len(compiled.Omitted))
	}
	if len(compiled.Diagnostics) != 1 || compiled.Diagnostics[0].Count != 3000 {
		t.Fatal(compiled.Diagnostics)
	}
}
