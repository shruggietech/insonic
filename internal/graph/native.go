// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"unicode"
)

func nativeTokens(text string) ([]string, error) {
	tokens, _, e := nativeScan(text)
	return tokens, e
}

func nativeScan(text string) ([]string, int, error) {
	if len(text) == 0 || len(text) > 65536 {
		return nil, -1, contracts.Condition("invalid_request", "native_syntax")
	}
	tokens := []string{}
	terminator := -1
	var lastSignificant byte
	for i := 0; i < len(text); {
		c := text[i]
		if c == ';' {
			if terminator >= 0 {
				return nil, -1, contracts.Condition("unsupported_capability", "native_statement")
			}
			terminator = i
			i++
			continue
		}
		if c == 0 {
			return nil, -1, contracts.Condition("invalid_request", "native_syntax")
		}
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if terminator >= 0 && !strings.HasPrefix(text[i:], "/*") && !strings.HasPrefix(text[i:], "//") && !strings.HasPrefix(text[i:], "--") {
			return nil, -1, contracts.Condition("unsupported_capability", "native_statement")
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
				return nil, -1, contracts.Condition("unsupported_capability", "native_function")
			}
			if !closed {
				return nil, -1, contracts.Condition("invalid_request", "native_syntax")
			}
			lastSignificant = quote
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return nil, -1, contracts.Condition("invalid_request", "native_syntax")
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
			if k < len(text) && text[k] == '(' && lastSignificant == '.' {
				return nil, -1, contracts.Condition("unsupported_capability", "native_function")
			}
			if k < len(text) && text[k] == '(' && token != "MATCH" && token != "OPTIONAL" && token != "WHERE" && token != "IN" && token != "NOT" && token != "EXISTS" && token != "RETURN" && token != "WITH" && token != "UNWIND" && token != "SELECT" && !strings.Contains("|COUNT|SUM|MIN|MAX|AVG|COALESCE|TOSTRING|TOINTEGER|LOWER|UPPER|ABS|SIZE|LENGTH|COLLECT|", "|"+token+"|") {
				return nil, -1, contracts.Condition("unsupported_capability", "native_function")
			}
			i = j
			lastSignificant = 'w'
			continue
		}
		lastSignificant = c
		i++
	}
	return tokens, terminator, nil
}
func ValidateNative(dialect, text string, p map[string]contracts.QueryParameter) error {
	_, e := validatedNative(dialect, text, p)
	return e
}

// validatedNative returns the same read with its optional terminator removed.
// Engine parsers differ on terminators, so normalize only after the shared guard.
func validatedNative(dialect, text string, p map[string]contracts.QueryParameter) (string, error) {
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "native", Dialect: dialect, Text: text}, Parameters: p}
	if e := q.Validate(); e != nil {
		return "", e
	}
	for name := range p {
		if strings.Contains(strings.ToUpper(name), "PROFILEEXECUTION") {
			return "", contracts.Condition("unsupported_capability", "native_read_only")
		}
	}
	tokens, terminator, e := nativeScan(text)
	if e != nil {
		return "", e
	}
	if len(tokens) == 0 {
		return "", contracts.Condition("invalid_request", "native_syntax")
	}
	first := tokens[0]
	if first == "EXPLAIN" && len(tokens) > 1 {
		first = tokens[1]
	}
	if dialect == "arcade-sql" {
		if first != "SELECT" {
			return "", contracts.Condition("unsupported_capability", "native_read_only")
		}
	} else if first != "MATCH" && first != "OPTIONAL" && first != "RETURN" && first != "WITH" && first != "UNWIND" {
		return "", contracts.Condition("unsupported_capability", "native_read_only")
	}
	forbidden := "|CREATE|MERGE|DELETE|DETACH|SET|REMOVE|DROP|ALTER|COPY|LOAD|INSTALL|ATTACH|DETACH|CALL|PROFILE|BEGIN|COMMIT|ROLLBACK|TRANSACTION|UPDATE|INSERT|INTO|LET|TRAVERSE|EVAL|EXECUTE|EXPORT|IMPORT|PRAGMA|"
	for _, token := range tokens {
		if strings.Contains(forbidden, "|"+token+"|") || strings.Contains(token, "PROFILEEXECUTION") {
			return "", contracts.Condition("unsupported_capability", "native_read_only")
		}
	}
	if terminator >= 0 {
		text = text[:terminator]
	}
	return strings.TrimSpace(text), nil
}
