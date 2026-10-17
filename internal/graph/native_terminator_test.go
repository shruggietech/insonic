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

func TestNativeTerminatorNormalizedForEveryBackend(t *testing.T) {
	for _, dialect := range []string{"ladybug-cypher", "arcade-opencypher", "arcade-sql"} {
		base := "RETURN 'literal; value' AS value"
		if dialect == "arcade-sql" {
			base = "SELECT 'literal; value' AS value"
		}
		for _, tail := range []string{";", "; /* trailing */", "; -- trailing\n", "; // trailing\n"} {
			text, e := validatedNative(dialect, base+tail, nil)
			if e != nil || text != base {
				t.Fatalf("backend received %q: %v", text, e)
			}
		}
	}
}

func TestNativeParenthesizedReadClausesAreNotFunctionCalls(t *testing.T) {
	for _, tc := range []struct{ dialect, text string }{{"ladybug-cypher", "RETURN (1) AS value;"}, {"arcade-opencypher", "WITH (1) AS x RETURN x;"}, {"arcade-sql", "SELECT (1) AS value;"}} {
		if e := ValidateNative(tc.dialect, tc.text, nil); e != nil {
			t.Fatal(tc, e)
		}
	}
	for _, text := range []string{"RETURN custom(1)", "RETURN custom.count(1)", "RETURN custom /* gap */ . /* gap */ lower(1)", "WITH custom(1) AS x RETURN x", "UNWIND custom() AS x RETURN x", "RETURN (1); CALL effect()"} {
		if e := ValidateNative("ladybug-cypher", text, nil); e == nil {
			t.Fatal("accepted function/effect", text)
		}
	}
}
