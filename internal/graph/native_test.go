package graph

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestNativeReadBoundary(t *testing.T) {
	for _, s := range []string{"MATCH (n:Entity) RETURN n.id", "MATCH (n) WHERE n.ref = 'CREATE; PROFILE' RETURN count(n)", "SELECT id FROM Entity WHERE id = :id", "EXPLAIN MATCH (n) RETURN n.id"} {
		dialect := "ladybug-cypher"
		if s[0:6] == "SELECT" {
			dialect = "arcade-sql"
		}
		if e := ValidateNative(dialect, s, nil); e != nil {
			t.Fatal(s, e)
		}
	}
	for _, s := range []string{"PROFILE MATCH (n) DELETE n", "MATCH (n) RETURN n; CREATE (:Entity)", "CALL evil()", "SELECT evil() FROM Entity", "MATCH (n) RETURN n UNION CREATE (:Entity)", "MATCH (n) SET n.id='x' RETURN n", "COMMIT", "SELECT evil /*comment*/ () FROM Entity", "SELECT \"evil\" /*comment*/ () FROM Entity", "MATCH (n) RETURN n /*"} {
		dialect := "ladybug-cypher"
		if strings.HasPrefix(s, "SELECT") {
			dialect = "arcade-sql"
		}
		if e := ValidateNative(dialect, s, nil); e == nil {
			t.Fatal("accepted effects", s)
		}
	}
	params := map[string]contracts.QueryParameter{"$profileExecution": {Type: "boolean", Value: json.RawMessage("true")}}
	if e := ValidateNative("arcade-opencypher", "MATCH (n) RETURN n", params); e == nil {
		t.Fatal("accepted reserved control")
	}
}
