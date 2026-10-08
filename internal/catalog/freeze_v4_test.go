package catalog

import (
	"strings"
	"testing"
)

func TestFrozenV4DDL(t *testing.T) {
	if hash([]byte(strings.Join(historicalV4DDL, "\n"))) != historicalV4Digest() {
		t.Fatal("historical schema4 changed")
	}
}
