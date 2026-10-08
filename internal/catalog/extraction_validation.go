// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
)

func validateExtractionCues(v Extraction, r Recording) error {
	cues, e := evidence.ParseCues(r.Document)
	if e != nil {
		return e
	}
	var assertions []evidence.Assertion
	if strict(v.Assertions, &assertions) != nil || len(assertions) > 100000 {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, a := range assertions {
		if a.ID == "" || seen[a.ID] || evidence.ValidateAssertion(a, cues) != nil {
			return contracts.Fail("invalid_request")
		}
		canonical := evidence.Deduplicate([]evidence.Assertion{a})
		if len(canonical) != 1 || canonical[0].ID != a.ID {
			return contracts.Fail("invalid_request")
		}
		seen[a.ID] = true
	}
	return nil
}
