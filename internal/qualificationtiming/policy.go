// SPDX-License-Identifier: Apache-2.0
// Package qualificationtiming contains execution allowances for controlled checks.
// Ordinary product operations must retain their own configured deadlines.
package qualificationtiming

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

//go:embed policy.json
var policyJSON []byte

var policy = func() map[string]int {
	value, err := decodePolicy(policyJSON)
	if err != nil {
		panic(err)
	}
	return value
}()

func decodePolicy(raw []byte) (map[string]int, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("qualification timing policy must be an object")
	}
	values := map[string]int{}
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || !strings.HasSuffix(key, "_seconds") {
			return nil, fmt.Errorf("qualification timing policy key has no seconds unit")
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate qualification timing key: %s", key)
		}
		var seconds int
		if decoder.Decode(&seconds) != nil || seconds < 1 || seconds > 86400 {
			return nil, fmt.Errorf("invalid qualification seconds: %s", key)
		}
		values[key] = seconds
	}
	if _, err = decoder.Token(); err != nil {
		return nil, err
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("extra qualification timing policy input")
	}
	for _, key := range []string{"cleanup_seconds", "readiness_seconds", "frontend_seconds", "playback_seconds", "help_seconds", "help_watchdog_seconds", "gui_seconds", "gui_watchdog_seconds", "exit_child_seconds", "exit_watchdog_seconds", "exit_matrix_seconds", "native_ui_seconds", "native_action_seconds", "admission_seconds", "durable_work_seconds", "fixture_idle_seconds", "request_seconds"} {
		if values[key] < 1 {
			return nil, fmt.Errorf("missing qualification timing key: %s", key)
		}
	}
	for _, pair := range [][2]string{{"help_watchdog_seconds", "help_seconds"}, {"gui_watchdog_seconds", "gui_seconds"}, {"exit_watchdog_seconds", "exit_child_seconds"}} {
		if values[pair[0]]+values["cleanup_seconds"] > values[pair[1]] {
			return nil, fmt.Errorf("qualification watchdog consumes cleanup headroom: %s", pair[0])
		}
	}
	if values["exit_matrix_seconds"] < 12*values["exit_child_seconds"]+240 {
		return nil, fmt.Errorf("native Exit matrix lacks repeated-case headroom")
	}
	return values, nil
}

func Seconds(key string) int {
	value, ok := policy[key]
	if !ok {
		panic("unknown qualification timing key: " + key)
	}
	return value
}

func Duration(key string) time.Duration { return time.Duration(Seconds(key)) * time.Second }

// Context preserves an earlier parent deadline and cancellation. Sequential
// operations must share their original parent instead of resetting its clock.
func Context(parent context.Context, key string) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, Duration(key))
}
