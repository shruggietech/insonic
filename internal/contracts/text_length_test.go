// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"strings"
	"testing"
)

func TestQueryUnicodeCharacterLimits(t *testing.T) {
	for _, alphabet := range []string{"a", "é", "漢", "😀", "e\u0301"} {
		unit := len([]rune(alphabet))
		q := QueryInput{Title: strings.Repeat(alphabet, 512/unit), Definition: QueryDefinition{Mode: "normalized", Operation: "text-search", Filters: QueryFilter{Text: strings.Repeat(alphabet, 4096/unit)}}}
		if err := q.Validate(); err != nil {
			t.Errorf("%q at scalar limit: %v", alphabet, err)
		}
		q.Title += "a"
		if q.Validate() == nil {
			t.Fatal("513 title characters accepted")
		}
	}
	q := QueryInput{Title: string([]byte{0xff}), Definition: QueryDefinition{Mode: "normalized", Operation: "media-list"}}
	if q.Validate() == nil {
		t.Fatal("damaged UTF-8 accepted")
	}
}
