// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"testing"
)

func TestProcessingConfigurationUsesRegisteredContract(t *testing.T) {
	raw, err := registry.ReadFile("v0.0.0/master.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var master struct {
		Definitions map[string]struct {
			Reference string `json:"$ref"`
		} `json:"$defs"`
	}
	if json.Unmarshal(raw, &master) != nil || master.Definitions["processing-tools"].Reference != "processing-tools.schema.json" {
		t.Fatal("selected processing contract is not registered in the packaged master")
	}
	valid := []byte(`{"kind":"processing-tools","schema_version":"0.0.0","cueson":{"executable":"/opt/insonic/cueson","executable_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"processing":{}}`)
	if err := ValidateDocument(valid); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{
		`{"kind":"processing-tools","schema_version":"0.0.0","cueson":{},"processing":{}}`,
		`{"kind":"processing-tools","schema_version":"0.0.0","cueson":{"executable":"/opt/cueson","executable_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"processing":{"threads":0}}`,
		`{"kind":"processing-tools","schema_version":"0.0.0","cueson":{"executable":"/opt/cueson","executable_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"processing":{"worker":{"path":"/opt/worker"}}}`,
	} {
		if err := ValidateDocument([]byte(invalid)); err == nil {
			t.Fatal("invalid processing configuration admitted")
		}
	}
}
