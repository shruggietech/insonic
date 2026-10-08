// SPDX-License-Identifier: Apache-2.0
package schemas

import "testing"

func TestCorrelationQualityUsesExplicitTypedSetting(t *testing.T) {
	prefix := `{"kind":"runtime-request","schema_version":"0.0.0","workspace_id":"11111111-1111-4111-8111-111111111111","request_id":"22222222-2222-4222-8222-222222222222","item_id":"33333333-3333-4333-8333-333333333333","operation":"`
	for _, operation := range []string{"speakers.select", "speakers.diagnostics"} {
		for _, quality := range []string{`{}`, `{"enabled":true}`, `{"enabled":false}`} {
			if err := ValidateRequest([]byte(prefix + operation + `","data":{"quality":` + quality + `}}`)); err != nil {
				t.Fatal(err)
			}
		}
		for _, quality := range []string{`{"enabled":null}`, `{"enabled":"false"}`, `{"threshold":0.5}`} {
			if err := ValidateRequest([]byte(prefix + operation + `","data":{"quality":` + quality + `}}`)); err == nil {
				t.Fatal("invalid correlation quality admitted", quality)
			}
		}
	}
}
