// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf16"
)

var normalLaunchAction = errors.New("cannot launch at normal user privilege; finish installation, then open insonic normally from the Start menu")

func limitedAccount(currentSID, limitedSID string, limitedElevated bool) error {
	if currentSID == "" || limitedSID == "" || currentSID != limitedSID || limitedElevated {
		return normalLaunchAction
	}
	return nil
}

// unicodeEnvironment rebuilds the limited user's environment instead of
// forwarding the installer's elevated environment. CLI path variables override
// same-named values exactly once, with Windows case-insensitive identity.
func unicodeEnvironment(base []string, private map[string]string) ([]uint16, error) {
	values := map[string]string{}
	nameOf := func(entry string) (string, error) {
		if strings.ContainsRune(entry, 0) {
			return "", normalLaunchAction
		}
		start := 0
		if strings.HasPrefix(entry, "=") {
			start = 1
		}
		i := strings.IndexByte(entry[start:], '=')
		if i < 0 || start+i == 0 {
			return "", normalLaunchAction
		}
		return entry[:start+i], nil
	}
	for _, entry := range base {
		name, e := nameOf(entry)
		if e != nil {
			return nil, e
		}
		values[strings.ToUpper(name)] = entry
	}
	for name, value := range private {
		if name != "INSONIC_LAUNCH_SCRIPT" && name != "INSONIC_LAUNCH_CLI" && name != "INSONIC_PROGRAM_DIRECTORY" || strings.ContainsRune(value, 0) {
			return nil, normalLaunchAction
		}
		values[strings.ToUpper(name)] = name + "=" + value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var text strings.Builder
	for _, key := range keys {
		text.WriteString(values[key])
		text.WriteByte(0)
	}
	text.WriteByte(0)
	if len(keys) == 0 {
		text.WriteByte(0)
	}
	return utf16.Encode([]rune(text.String())), nil
}
