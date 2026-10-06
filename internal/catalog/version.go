// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"regexp"
	"strconv"
	"strings"
)

var versionRule = regexp.MustCompile(`^\s*(>=|<=|>|<|=)?\s*([0-9]+(?:\.[0-9]+){0,2})\s*$`)

func versionParts(value string) ([3]int64, error) {
	var out [3]int64
	parts := strings.Split(value, ".")
	if len(parts) > 3 {
		return out, contracts.Fail("invalid_request")
	}
	for i, p := range parts {
		n, e := strconv.ParseInt(p, 10, 64)
		if e != nil || n < 0 {
			return out, contracts.Fail("invalid_request")
		}
		out[i] = n
	}
	return out, nil
}
func versionMatches(actual, required string) (bool, error) {
	a, e := versionParts(actual)
	if e != nil {
		return false, e
	}
	rules := strings.Split(required, ",")
	if len(rules) > 8 {
		return false, contracts.Fail("invalid_request")
	}
	match := true
	for _, rule := range rules {
		parts := versionRule.FindStringSubmatch(rule)
		if parts == nil {
			return false, contracts.Fail("invalid_request")
		}
		b, e := versionParts(parts[2])
		if e != nil {
			return false, e
		}
		compare := 0
		for i := range a {
			if a[i] < b[i] {
				compare = -1
				break
			}
			if a[i] > b[i] {
				compare = 1
				break
			}
		}
		switch parts[1] {
		case "", "=":
			match = match && compare == 0
		case ">=":
			match = match && compare >= 0
		case "<=":
			match = match && compare <= 0
		case ">":
			match = match && compare > 0
		case "<":
			match = match && compare < 0
		}
	}
	return match, nil
}
