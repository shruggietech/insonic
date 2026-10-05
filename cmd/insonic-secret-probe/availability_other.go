//go:build !darwin

package main

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/zalando/go-keyring"
)

func available() bool {
	// Unique missing lookup neither writes nor returns an existing credential.
	_, err := keyring.Get("insonic-qualification", contracts.ID())
	return err == keyring.ErrNotFound
}
