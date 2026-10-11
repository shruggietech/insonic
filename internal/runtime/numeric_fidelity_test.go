// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestResponseDecoderPreservesExactNumericTokens(t *testing.T) {
	raw := []byte(`{"kind":"runtime-response","schema_version":"1.0.1","result":{"unix_ns":1791720000123456789,"nested":[9007199254740993,-9007199254740993,9223372036854775807,-9223372036854775808,1.25]}}`)
	var response contracts.Response
	if err := decode(raw, &response); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"1791720000123456789", "9007199254740993", "-9007199254740993", "9223372036854775807", "-9223372036854775808", "1.25"} {
		if !bytes.Contains(encoded, []byte(token)) {
			t.Fatalf("lost token %s: %s", token, encoded)
		}
	}
}
