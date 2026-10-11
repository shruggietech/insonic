// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestGeneratedOperatorReasonEnvelope(t *testing.T) {
	raw := []byte(fmt.Sprintf("{\"kind\":\"runtime-response\",\"schema_version\":\"%s\",\"error\":{\"code\":\"conflict\",\"message\":\"The credential ID is already stored.\",\"reason\":\"credential_duplicate\"}}", contracts.Version))
	if e := ValidateResponse(raw); e != nil {
		t.Fatal(e)
	}
}
