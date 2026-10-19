//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"fmt"
	"io"
	"runtime/pprof"
	"strconv"
	"strings"
)

const nativeProfileByteLimit = 4 << 20

// Keep raw profiler text in bounded memory only. Only parsed function symbols
// reach the existing failure receipt; addresses, paths and labels never do.
type nativeProfileBuffer struct{ bytes.Buffer }

func (b *nativeProfileBuffer) Write(p []byte) (int, error) {
	if len(p) > nativeProfileByteLimit-b.Len() {
		return 0, io.ErrShortWrite
	}
	return b.Buffer.Write(p)
}

func captureNativeFunctionNames() ([][]string, error) {
	profile := pprof.Lookup("goroutine")
	if profile.Count() > 256 {
		return nil, fmt.Errorf("native function profile exceeds limit")
	}
	var raw nativeProfileBuffer
	if err := profile.WriteTo(&raw, 1); err != nil {
		return nil, err
	}
	return parseNativeFunctionProfile(raw.String())
}

func parseNativeFunctionProfile(raw string) ([][]string, error) {
	if len(raw) > nativeProfileByteLimit {
		return nil, fmt.Errorf("native function profile exceeds limit")
	}
	var stacks [][]string
	var functions []string
	flush := func() {
		if len(functions) > 0 {
			stacks = append(stacks, functions)
			functions = nil
		}
	}
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "#" || !strings.HasPrefix(fields[1], "0x") {
			continue
		}
		symbol := fields[2]
		offset := strings.LastIndex(symbol, "+0x")
		if offset < 1 {
			continue
		}
		if _, err := strconv.ParseUint(symbol[offset+3:], 16, 64); err != nil {
			continue
		}
		name := symbol[:offset]
		valid := len(name) <= 240
		// Parentheses are valid only for a method receiver, never call arguments.
		if open := strings.IndexByte(name, '('); open >= 0 {
			close := strings.IndexByte(name, ')')
			if open == 0 || name[open-1] != '.' || close <= open+1 || close+1 >= len(name) || name[close+1] != '.' || strings.ContainsAny(name[close+1:], "()") || strings.Contains(name[open+1:close], "(") {
				valid = false
			}
		} else if strings.Contains(name, ")") {
			valid = false
		}
		for _, c := range name {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_./()*[],-", c)) {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if len(functions) >= 128 {
			return nil, fmt.Errorf("native profile depth exceeds limit")
		}
		functions = append(functions, name)
	}
	flush()
	if len(stacks) > 256 {
		return nil, fmt.Errorf("native function profile exceeds limit")
	}
	return stacks, nil
}
