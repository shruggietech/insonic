// SPDX-License-Identifier: Apache-2.0
package qualificationtiming

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEmbeddedPolicyAndParentRelationships(t *testing.T) {
	if Seconds("help_seconds") < 180 || Seconds("gui_seconds") < 600 || Seconds("exit_child_seconds") < 180 || Seconds("native_ui_seconds") < 60 {
		t.Fatal("successful execution allowance regressed")
	}
	if Duration("exit_matrix_seconds") < 12*Duration("exit_child_seconds")+240*time.Second {
		t.Fatal("repeated Windows launches lost setup and cleanup headroom")
	}
	for _, pair := range [][2]string{{"help_watchdog_seconds", "help_seconds"}, {"gui_watchdog_seconds", "gui_seconds"}, {"exit_watchdog_seconds", "exit_child_seconds"}} {
		if Duration(pair[0])+Duration("cleanup_seconds") > Duration(pair[1]) {
			t.Fatal("watchdog consumes cleanup headroom", pair)
		}
	}
}

func TestPolicyRejectsInvalidAndInconsistentSeconds(t *testing.T) {
	changed := func(key string, value any) []byte {
		var values map[string]any
		if err := json.Unmarshal(policyJSON, &values); err != nil {
			t.Fatal(err)
		}
		values[key] = value
		raw, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, raw := range [][]byte{
		[]byte(`{}`),
		changed("help_seconds", 0),
		changed("help_seconds", 1.5),
		changed("help_seconds", Seconds("help_watchdog_seconds")),
		append(append([]byte{}, policyJSON...), []byte(` {}`)...),
	} {
		if _, err := decodePolicy(raw); err == nil {
			t.Fatalf("invalid policy accepted: %s", raw)
		}
	}
}

func TestQualificationContextPreservesEarlierCancellationAndDeadline(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	child, stop := Context(parent, "readiness_seconds")
	defer stop()
	cancel()
	if child.Err() != context.Canceled {
		t.Fatal("earlier cancellation was discarded")
	}
	parent, cancel = context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	defer cancel()
	child, stop = Context(parent, "frontend_seconds")
	defer stop()
	parentEnd, _ := parent.Deadline()
	childEnd, _ := child.Deadline()
	if !parentEnd.Equal(childEnd) {
		t.Fatal("parent deadline extended")
	}
}
