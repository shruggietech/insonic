// SPDX-License-Identifier: Apache-2.0
package process

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestStreamHelper(t *testing.T) {
	if os.Getenv("INSONIC_STREAM_HELPER") != "1" {
		return
	}
	io.Copy(os.Stdout, os.Stdin)
	os.Stderr.WriteString("stream diagnostic")
	os.Exit(0)
}

func TestStreamWritesBeyondCaptureLimitWithoutBuffering(t *testing.T) {
	exe, _ := os.Executable()
	input := strings.Repeat("abcd", 1<<18)
	var output bytes.Buffer
	result, err := Stream(processTestContext(t), Spec{Executable: exe, Args: []string{"-test.run=^TestStreamHelper$"}, Env: []string{"INSONIC_STREAM_HELPER=1"}, Input: strings.NewReader(input), MaxOutput: 1024}, &output)
	if err != nil || output.String() != input || len(result.Stdout) != 0 || string(result.Stderr) != "stream diagnostic" {
		t.Fatalf("stream result %+v: %v (%d bytes)", result, err, output.Len())
	}
}

type failedStreamWriter struct{}

func (failedStreamWriter) Write([]byte) (int, error) { return 0, errors.New("injected writer failure") }

func TestStreamRejectsOutputFailureAndCancellation(t *testing.T) {
	exe, _ := os.Executable()
	spec := Spec{Executable: exe, Args: []string{"-test.run=^TestStreamHelper$"}, Env: []string{"INSONIC_STREAM_HELPER=1"}, Input: strings.NewReader("data"), MaxOutput: 1024}
	if _, err := Stream(processTestContext(t), spec, failedStreamWriter{}); err == nil {
		t.Fatal("writer failure accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := Stream(ctx, spec, io.Discard)
	if err == nil || result.Started {
		t.Fatal("cancelled stream launched")
	}
}
