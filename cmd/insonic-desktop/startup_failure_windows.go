//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import "golang.org/x/sys/windows"

func presentStartupFailure(message string) error {
	text, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return err
	}
	title, err := windows.UTF16PtrFromString("insonic could not start")
	if err != nil {
		return err
	}
	_, err = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
	return err
}
