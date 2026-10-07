// SPDX-License-Identifier: Apache-2.0
package contracts

import "testing"

func TestConfigurationEnvelopePlacement(t *testing.T) {
	id := ID()
	for _, operation := range []string{"pipelines.list", "speakers.list", "terms.list", "terms.compile"} {
		if !ConfigurationRequestValid(Request{Operation: operation}) {
			t.Fatal(operation)
		}
		if ConfigurationRequestValid(Request{Operation: operation, ItemID: id}) {
			t.Fatal("list or compile accepted item", operation)
		}
	}
	for _, operation := range []string{"pipelines.show", "speakers.show", "terms.show"} {
		if !ConfigurationRequestValid(Request{Operation: operation, ItemID: id}) {
			t.Fatal(operation)
		}
		if ConfigurationRequestValid(Request{Operation: operation, ItemID: id, Data: []byte(`{}`)}) {
			t.Fatal("show accepted unused data", operation)
		}
	}
	for _, operation := range []string{"pipelines.set", "speakers.set", "terms.set"} {
		if ConfigurationRequestValid(Request{Operation: operation, ItemID: id}) {
			t.Fatal("missing mutation input", operation)
		}
		if !ConfigurationRequestValid(Request{Operation: operation, ItemID: id, Data: []byte(`{}`)}) {
			t.Fatal(operation)
		}
	}
	for _, operation := range []string{"pipelines.inspect", "speakers.aliases", "speakers.select", "speakers.diagnostics"} {
		if !ConfigurationRequestValid(Request{Operation: operation, ItemID: id}) {
			t.Fatal(operation)
		}
	}
	if ConfigurationRequestValid(Request{Operation: "pipelines.unknown", ItemID: id}) || ConfigurationRequestValid(Request{Operation: "terms.list", PublicationID: id}) {
		t.Fatal("unknown or unused authority admitted")
	}
}
