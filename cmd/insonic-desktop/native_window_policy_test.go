//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeWindowCalibrationSelectionAndFixtureValidation(t *testing.T) {
	for _, sample := range []struct {
		input   string
		seconds int
	}{{"2", 2}, {"10", 10}} {
		got, err := selectedNativeWindowJoinSeconds(sample.input)
		if err != nil || got != sample.seconds {
			t.Fatal("declared calibration selection rejected", sample, got, err)
		}
		parent, cancel := context.WithCancel(context.Background())
		ctx := nativeWindowJoinContext(parent, got)
		if nativeWindowJoinBudget(ctx) != time.Duration(sample.seconds)*time.Second {
			t.Fatal("fixture allowance lost")
		}
		cancel()
		if ctx.Err() != context.Canceled {
			t.Fatal("fixture allowance discarded parent cancellation")
		}
	}
	for _, input := range []string{"0", "1", "3", "9", "11", "-1", "2s", "02", " 10", "10 "} {
		if _, err := selectedNativeWindowJoinSeconds(input); err == nil {
			t.Fatal("undeclared calibration arm accepted", input)
		}
	}
}

func TestNativeWindowFixtureRejectsUndeclaredAllowance(t *testing.T) {
	for _, seconds := range []int{-1, 1, 3, 11} {
		path := filepath.Join(t.TempDir(), "fixture.json")
		raw := fmt.Sprintf("{\"action\":\"menu\",\"window_join_seconds\":%d}", seconds)
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readExitFixture(path); err == nil {
			t.Fatal("undeclared child allowance accepted", seconds)
		}
	}
}
