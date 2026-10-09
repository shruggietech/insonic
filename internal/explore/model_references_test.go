// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func TestModelReferenceProjectionPreservesExactTargets(t *testing.T) {
	base, alias, source, removed := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	target, _ := json.Marshal(catalog.ModelTarget{Kind: "base", ID: base, Operation: "transcription"})
	s := catalog.Snapshot{Revision: 4, Records: catalog.Records{
		BaseModels:   []catalog.BaseModelInstall{{ID: base, Name: "Selected", Version: "1", Revision: 1}},
		ModelAliases: []catalog.ModelAlias{{ID: alias, Name: "speech-main", Revision: 3, State: "active", Target: target}, {ID: removed, Name: "removed", Revision: 4, State: "deleted", Target: target}},
		ModelSources: []catalog.ModelSource{{ID: source, Name: "custom", Revision: 2, State: "active", URL: "https://example.com/catalog", CredentialID: contracts.ID()}},
	}}
	c, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, edge := range c.Refs.Edges {
		if edge.Kind == "resolves-to" && edge.From == "model-alias:"+alias && edge.To == "base-model:"+base {
			found = true
		}
	}
	if !found || c.Rows["model-alias:"+alias].Label != "speech-main" || c.Rows["model-source:"+source].Label != "custom" {
		t.Fatal("reference projection missing")
	}
	if _, ok := c.Rows["model-alias:"+removed]; ok {
		t.Fatal("deleted convenience alias presented active")
	}
	for _, node := range c.Refs.Nodes {
		if node.ID == "model-source:"+source {
			encoded, _ := json.Marshal(node)
			if bytes.Contains(encoded, []byte("credential_id")) || bytes.Contains(encoded, []byte("https://")) {
				t.Fatal("source routing configuration leaked into graph")
			}
		}
	}
}
