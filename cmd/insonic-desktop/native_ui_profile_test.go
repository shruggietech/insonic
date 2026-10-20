//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
)

//go:noinline
func nativeProfileDeepCall(depth int) ([][]string, error) {
	if depth == 0 {
		return captureNativeFunctionNames()
	}
	return nativeProfileDeepCall(depth - 1)
}

//go:noinline
func nativeProfileOuterMarker() ([][]string, error) { return nativeProfileDeepCall(48) }

func TestNativeFunctionProfileKeepsOuterCallerBeyond32Frames(t *testing.T) {
	stacks, err := nativeProfileOuterMarker()
	if err != nil {
		t.Fatal(err)
	}
	for _, stack := range stacks {
		for _, name := range stack {
			if strings.HasSuffix(name, ".nativeProfileOuterMarker") {
				if len(stack) <= 32 {
					t.Fatal("outer caller did not require a deep profile")
				}
				return
			}
		}
	}
	t.Fatal("profile hides outer caller beyond 32 frames")
}

func TestNativeFunctionProfileDiscardsAddressesPathsLabelsAndArguments(t *testing.T) {
	raw := "goroutine profile: total 1\n1 @ 0xfeed\n# labels: {secret: protected-value}\n#\t0xfeed\tgithub.com/wailsapp/example.(*Window).WndProc+0x1\tC:/private/fixture.go:42\n#\t0xbeef\tmain.callback(protected-value)+0x2\tC:/private/fixture.go:43\n\n"
	stacks, err := parseNativeFunctionProfile(raw)
	if err != nil || len(stacks) != 1 || len(stacks[0]) != 1 || stacks[0][0] != "github.com/wailsapp/example.(*Window).WndProc" {
		t.Fatal(stacks, err)
	}
	if _, err = parseNativeFunctionProfile(strings.Repeat("x", nativeProfileByteLimit+1)); err == nil {
		t.Fatal("accepted unbounded raw profile")
	}
	var buf nativeProfileBuffer
	if _, err = buf.Write([]byte(strings.Repeat("x", nativeProfileByteLimit))); err != nil {
		t.Fatal(err)
	}
	if _, err = buf.Write([]byte("x")); err == nil || buf.Len() != nativeProfileByteLimit {
		t.Fatal("profile writer exceeded its memory bound")
	}
}
