// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"unicode"
)

func nativeTokens(text string) ([]string, error) {
	if len(text) == 0 || len(text) > 65536 {
		return nil, contracts.Fail("invalid_request")
	}
	tokens := []string{}
	for i := 0; i < len(text); {
		c := text[i]
		if c == ';' || c == 0 {
			return nil, contracts.Fail("unsupported_capability")
		}
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote := c
			i++
			closed := false
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == quote {
					if i+1 < len(text) && text[i+1] == quote {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			k := skipNativeSpace(text, i)
			if k < len(text) && text[k] == '(' {
				return nil, contracts.Fail("unsupported_capability")
			}
			if !closed {
				return nil, contracts.Fail("invalid_request")
			}
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return nil, contracts.Fail("invalid_request")
			}
			i += end + 4
			continue
		}
		if (c == '/' && i+1 < len(text) && text[i+1] == '/') || (c == '-' && i+1 < len(text) && text[i+1] == '-') {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		if unicode.IsLetter(rune(c)) || c == '_' || c == '$' {
			j := i + 1
			for j < len(text) && (unicode.IsLetter(rune(text[j])) || unicode.IsDigit(rune(text[j])) || text[j] == '_' || text[j] == '$') {
				j++
			}
			token := strings.ToUpper(text[i:j])
			tokens = append(tokens, token)
			k := skipNativeSpace(text, j)
			if k < len(text) && text[k] == '(' && token != "MATCH" && token != "OPTIONAL" && token != "WHERE" && token != "IN" && token != "NOT" && token != "EXISTS" && !strings.Contains("|COUNT|SUM|MIN|MAX|AVG|COALESCE|TOSTRING|TOINTEGER|LOWER|UPPER|ABS|SIZE|LENGTH|COLLECT|", "|"+token+"|") {
				return nil, contracts.Fail("unsupported_capability")
			}
			i = j
			continue
		}
		i++
	}
	return tokens, nil
}
func ValidateNative(dialect, text string, p map[string]contracts.QueryParameter) error {
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "native", Dialect: dialect, Text: text}, Parameters: p}
	if e := q.Validate(); e != nil {
		return e
	}
	for name := range p {
		if strings.Contains(strings.ToUpper(name), "PROFILEEXECUTION") {
			return contracts.Fail("unsupported_capability")
		}
	}
	tokens, e := nativeTokens(text)
	if e != nil {
		return e
	}
	if len(tokens) == 0 {
		return contracts.Fail("invalid_request")
	}
	first := tokens[0]
	if first == "EXPLAIN" && len(tokens) > 1 {
		first = tokens[1]
	}
	if dialect == "arcade-sql" {
		if first != "SELECT" {
			return contracts.Fail("unsupported_capability")
		}
	} else if first != "MATCH" && first != "OPTIONAL" && first != "RETURN" && first != "WITH" && first != "UNWIND" {
		return contracts.Fail("unsupported_capability")
	}
	forbidden := "|CREATE|MERGE|DELETE|DETACH|SET|REMOVE|DROP|ALTER|COPY|LOAD|INSTALL|ATTACH|DETACH|CALL|PROFILE|BEGIN|COMMIT|ROLLBACK|TRANSACTION|UPDATE|INSERT|INTO|LET|TRAVERSE|EVAL|EXECUTE|EXPORT|IMPORT|PRAGMA|"
	for _, token := range tokens {
		if strings.Contains(forbidden, "|"+token+"|") || strings.Contains(token, "PROFILEEXECUTION") {
			return contracts.Fail("unsupported_capability")
		}
	}
	return nil
}
