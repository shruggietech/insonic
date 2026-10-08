// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"strings"
	"unicode"
)

func skipNativeSpace(text string, i int) int {
	for i < len(text) {
		if unicode.IsSpace(rune(text[i])) {
			i++
			continue
		}
		if strings.HasPrefix(text[i:], "/*") {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return len(text)
			}
			i += end + 4
			continue
		}
		if strings.HasPrefix(text[i:], "//") || strings.HasPrefix(text[i:], "--") {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		break
	}
	return i
}
