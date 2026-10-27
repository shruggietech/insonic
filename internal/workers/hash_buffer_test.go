// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestReusedHashBufferKeepsCompleteIndependentFileDigests(t *testing.T) {
	buffer := make([]byte, 3)
	for _, data := range [][]byte{[]byte("longer than the buffer"), {}, []byte("x")} {
		path := filepath.Join(t.TempDir(), "dependency")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		got, err := hashWithBuffer(context.Background(), path, buffer)
		expected := sha256.Sum256(data)
		if err != nil || got.Path != path || got.SHA256 != hex.EncodeToString(expected[:]) {
			t.Fatal("reused buffer changed byte identity", got, err)
		}
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := hashWithBuffer(cancelled, path, buffer); err == nil {
			t.Fatal("cancelled hashing was accepted")
		}
	}
}
