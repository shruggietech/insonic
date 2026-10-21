// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateBoundedLogsAndWhitelistedContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.jsonl")
	log, err := Open(path, "debug")
	if err != nil {
		t.Fatal(err)
	}
	if second, err := Open(path, "debug"); err == nil {
		second.Close()
		t.Fatal("concurrent log owner admitted")
	}
	for i := 0; i < 14000; i++ {
		if err = log.Write("debug", Event{Operation: "work.wait", RequestID: contracts.ID(), WorkspaceID: contracts.ID(), Code: "unavailable", Reason: "wait_timeout", State: "running"}); err != nil {
			t.Fatal(err)
		}
	}
	if err = log.Close(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", ".1", ".2"} {
		bytes, err := os.ReadFile(path + suffix)
		if err != nil || len(bytes) > MaxBytes {
			t.Fatal(suffix, len(bytes), err)
		}
		if strings.Contains(string(bytes), "passphrase") || strings.Contains(string(bytes), "payload") {
			t.Fatal("protected context logged")
		}
	}
	log, err = Open(path, "info")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err = log.Write("debug", Event{Operation: "work.show"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err = log.Write("error", Event{Operation: "secret=untrusted", RequestID: "credential-secret", Code: "untrusted-provider-body", Reason: "private-input"}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	tail := string(after[len(before):])
	if strings.Contains(tail, "secret") || strings.Contains(tail, "private-input") || strings.Contains(tail, "provider-body") {
		t.Fatal("untrusted context persisted", tail)
	}
}
func TestLogPathFailureIsReported(t *testing.T) {
	if log, err := Open(filepath.Join(t.TempDir(), "missing", "log.jsonl"), "info"); err == nil {
		log.Close()
		t.Fatal("missing directory admitted")
	}
}

func TestExistingOversizedLogIsRefusedBeforeRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.jsonl")
	for _, suffix := range []string{"", ".1", ".2"} {
		if err := os.WriteFile(path+suffix, make([]byte, MaxBytes+1), 0600); err != nil {
			t.Fatal(err)
		}
		if log, err := Open(path, "info"); err == nil {
			log.Close()
			t.Fatal("oversized elected log admitted", suffix)
		}
		if err := os.Remove(path + suffix); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStartupMetadataIsAllowlistedAndNumeric(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup.jsonl")
	log, err := Open(path, "error")
	if err != nil {
		t.Fatal(err)
	}
	code := uint32(5)
	for _, event := range []Event{
		{Operation: "desktop.startup", FailureStage: "desktop-window", SystemErrorCode: &code},
		{Operation: "work.show", FailureStage: "desktop-window", SystemErrorCode: &code},
		{Operation: "desktop.startup", FailureStage: "private-path-secret", SystemErrorCode: &code},
	} {
		if err = log.Write("error", event); err != nil {
			t.Fatal(err)
		}
	}
	if err = log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(rows) != 3 || !strings.Contains(rows[0], `"failure_stage":"desktop-window"`) || !strings.Contains(rows[0], `"system_error_code":5`) {
		t.Fatal(string(raw))
	}
	for _, row := range rows[1:] {
		if strings.Contains(row, "failure_stage") || strings.Contains(row, "system_error_code") || strings.Contains(row, "secret") {
			t.Fatal("untrusted startup metadata persisted", row)
		}
	}
}
