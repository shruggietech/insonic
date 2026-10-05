//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net"
	"os"
	"path/filepath"
)

func endpoint(w *workspace.Workspace) (string, error) {
	directory, err := workspace.RuntimeDirectory()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(w.Root))
	file := filepath.Join(directory, fmt.Sprintf("%x.sock", hash[:16]))
	if len(file) > 100 {
		return "", contracts.Fail("unavailable")
	}
	return file, nil
}
func listen(w *workspace.Workspace) (net.Listener, func(), error) {
	file, err := endpoint(w)
	if err != nil {
		return nil, nil, err
	}
	// Only the held OS owner lock authorizes stale endpoint cleanup.
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return nil, nil, contracts.Fail("unavailable")
	}
	listener, err := net.Listen("unix", file)
	if err != nil {
		return nil, nil, contracts.Fail("unavailable")
	}
	if err := os.Chmod(file, 0600); err != nil {
		listener.Close()
		os.Remove(file)
		return nil, nil, contracts.Fail("unavailable")
	}
	return listener, func() { listener.Close(); os.Remove(file) }, nil
}
func dial(ctx context.Context, w *workspace.Workspace) (net.Conn, error) {
	file, err := endpoint(w)
	if err != nil {
		return nil, err
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", file)
}
