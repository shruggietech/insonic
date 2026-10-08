// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

type forbiddenFunctionEngine struct{ called bool }

func (e *forbiddenFunctionEngine) Begin(context.Context, bool) (session, error) {
	e.called = true
	return nil, contracts.Fail("operation_failed")
}
func (e *forbiddenFunctionEngine) Close() {}

func TestNativeFunctionAllowlistAcrossCommentWhitespace(t *testing.T) {
	for _, dialect := range []string{"ladybug-cypher", "arcade-opencypher", "arcade-sql"} {
		for _, gap := range []string{" ", "/**/", " /* gap */ ", "// gap\n", "-- gap\n", "/*one*/ -- two\n /*three*/\t"} {
			for _, name := range []string{"custom", "`custom`", "\"custom\""} {
				text := "MATCH (n) RETURN " + name + gap + "()"
				if dialect == "arcade-sql" {
					text = "SELECT " + name + gap + "() FROM Entity"
				}
				t.Run(dialect+"/"+name+"/"+gap, func(t *testing.T) {
					engine := &forbiddenFunctionEngine{}
					backend := "ladybugdb"
					if dialect != "ladybug-cypher" {
						backend = "arcadedb"
					}
					a := &adapter{engine: engine, backend: backend}
					_, err := a.Query(context.Background(), dialect, text, nil)
					if e, ok := err.(*contracts.Error); !ok || e.Code != "unsupported_capability" {
						t.Fatalf("%q: %v", text, err)
					}
					if engine.called {
						t.Fatal("disallowed function reached engine")
					}
				})
			}
			text := "MATCH (n) RETURN count" + gap + "(n)"
			if dialect == "arcade-sql" {
				text = "SELECT count" + gap + "(*) FROM Entity"
			}
			if err := ValidateNative(dialect, text, nil); err != nil {
				t.Fatalf("allowed function %q: %v", text, err)
			}
		}
		text := "MATCH (n) RETURN custom /* unterminated"
		if dialect == "arcade-sql" {
			text = "SELECT custom /* unterminated"
		}
		if err := ValidateNative(dialect, text, nil); err == nil {
			t.Fatal("unterminated comment accepted")
		}
	}
}
