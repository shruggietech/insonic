//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"golang.org/x/sys/unix"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceFIFORefusedWithoutWaitingForWriter(t *testing.T) {
	name := filepath.Join(t.TempDir(), "fifo")
	if e := unix.Mkfifo(name, 0600); e != nil {
		t.Fatal(e)
	}
	done := make(chan bool, 1)
	go func() {
		f, e := openSource(name)
		if e == nil {
			f.Close()
		}
		done <- e != nil
	}()
	select {
	case refused := <-done:
		if !refused {
			t.Fatal("FIFO admitted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO open waited for a writer")
	}
}
