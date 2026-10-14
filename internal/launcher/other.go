//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package launcher

import "errors"

func SystemDirectory() (string, error) { return "", errors.New("Windows launcher requires Windows") }
func Launch(Plan) error                { return errors.New("Windows launcher requires Windows") }
func NoActiveClients(string) error     { return errors.New("Windows installer preflight requires Windows") }

func registeredInstallation(string) bool { return false }
func CheckShortcut(string, string) error {
	return errors.New("Windows installer shortcut checks require Windows")
}
