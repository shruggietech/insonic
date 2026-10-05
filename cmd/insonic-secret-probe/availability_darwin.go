package main

import (
	"context"
	"github.com/shruggietech/insonic/internal/process"
	"time"
)

func available() bool {
	// Listing services does not resolve an item or prompt to unlock a keychain.
	// Supervise the actual child rather than a library's unbounded security command.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := process.Run(ctx, process.Spec{Executable: "/usr/bin/security", Args: []string{"list-keychains"}, MaxOutput: 1 << 20})
	return err == nil
}
