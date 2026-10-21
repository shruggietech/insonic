//go:build desktop && !windows

// SPDX-License-Identifier: Apache-2.0
package main

func startupRecoveryRunner(runChild func(exitFixture, string) (exitObservation, error)) (func(exitFixture, string) (exitObservation, error), func() bool) {
	return runChild, func() bool { return true }
}
func qualifyStartupFailureChild(string) int { return 1 }
