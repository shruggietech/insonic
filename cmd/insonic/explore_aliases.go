// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"strconv"
)

func parseExploreAlias(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	if args[0] == "search" {
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "text-search", Filters: contracts.QueryFilter{Text: args[1]}, Pagination: &contracts.QueryPagination{Limit: 100}}}
		for i := 2; i < len(args); i += 2 {
			if i+1 >= len(args) {
				return fail()
			}
			switch args[i] {
			case "--speaker":
				q.Definition.Filters.SpeakerIDs = []string{args[i+1]}
			case "--media":
				q.Definition.Filters.MediaIDs = []string{args[i+1]}
			case "--limit":
				n, e := strconv.Atoi(args[i+1])
				if e != nil {
					return fail()
				}
				q.Definition.Pagination.Limit = n
			default:
				return fail()
			}
		}
		if q.Validate() != nil {
			return fail()
		}
		raw, _ := json.Marshal(q)
		return "query.run", "", raw, nil
	}
	if args[0] == "query" && args[1] == "native" {
		dialect, file, paramFile := "", "", ""
		for i := 2; i < len(args); i += 2 {
			if i+1 >= len(args) {
				return fail()
			}
			switch args[i] {
			case "--dialect":
				dialect = args[i+1]
			case "--file":
				file = args[i+1]
			case "--parameters":
				paramFile = args[i+1]
			default:
				return fail()
			}
		}
		read := func(path string, limit int64) ([]byte, error) {
			f, e := os.Open(path)
			if e != nil {
				return nil, contracts.Fail("unavailable")
			}
			defer f.Close()
			raw, e := io.ReadAll(io.LimitReader(f, limit+1))
			if e != nil || int64(len(raw)) > limit {
				return nil, contracts.Fail("input_limit")
			}
			return raw, nil
		}
		text, e := read(file, 65536)
		if e != nil {
			return "", "", nil, e
		}
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "native", Dialect: dialect, Text: string(text)}}
		if paramFile != "" {
			raw, e := read(paramFile, 1<<20)
			if e != nil {
				return "", "", nil, e
			}
			if contracts.DecodeExplore(raw, &q.Parameters) != nil {
				return fail()
			}
		}
		if q.Validate() != nil {
			return fail()
		}
		raw, _ := json.Marshal(q)
		return "query.run", "", raw, nil
	}
	return fail()
}
