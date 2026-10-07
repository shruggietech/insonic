// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"os"
	"testing"
)

func TestCaptureHelper(t *testing.T) {
	if os.Getenv("INSONIC_CAPTURE_HELPER") != "1" {
		return
	}
	os.Stdout.WriteString(`{"retained":true}`)
	os.Stderr.WriteString("extractor diagnostic")
	os.Exit(7)
}

func TestCapturePreservesSeparateFailureReports(t *testing.T) {
	exe, _ := os.Executable()
	result, err := Capture(context.Background(), Spec{Executable: exe, Args: []string{"-test.run=TestCaptureHelper"}, Env: []string{"INSONIC_CAPTURE_HELPER=1"}, MaxOutput: 1024})
	if err == nil || string(result.Stdout) != `{"retained":true}` || string(result.Stderr) != "extractor diagnostic" || result.ExitCode != 7 {
		t.Fatalf("capture outcome: %#v %v", result, err)
	}
}

func TestCaptureMarksTruncation(t *testing.T) {
	exe, _ := os.Executable()
	result, err := Capture(context.Background(), Spec{Executable: exe, Args: []string{"-test.run=TestCaptureHelper"}, Env: []string{"INSONIC_CAPTURE_HELPER=1"}, MaxOutput: 4})
	if err == nil || !result.Truncated || len(result.Stdout) > 4 || len(result.Stderr) > 4 {
		t.Fatalf("limit outcome: %#v %v", result, err)
	}
}
