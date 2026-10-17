// SPDX-License-Identifier: Apache-2.0
package graph

import "testing"

func TestNativeTrailingTerminator(t *testing.T) {
	for _, dialect := range []string{"ladybug-cypher", "arcade-opencypher", "arcade-sql"} {
		base := "RETURN 1 AS value"
		if dialect == "arcade-sql" {
			base = "SELECT 1 AS value"
		}
		for _, tail := range []string{";", "; /* final */", "; -- final\n", "; // final\n"} {
			if e := ValidateNative(dialect, base+tail, nil); e != nil {
				t.Errorf("%s %q: %v", dialect, tail, e)
			}
		}
		for _, tail := range []string{";;", "; RETURN 2", "; /* unterminated", "; CREATE (n)"} {
			if e := ValidateNative(dialect, base+tail, nil); e == nil {
				t.Errorf("accepted %q", base+tail)
			}
		}
	}
}
