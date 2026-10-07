// SPDX-License-Identifier: Apache-2.0
package process

import (
	"os"
	"runtime"
	"strings"
)

// LocalEnvironment carries only execution essentials and selected native-library
// or CUDA device paths. Application credentials and provider configuration are
// never part of a local tool's ambient environment.
func LocalEnvironment() []string {
	allowed := map[string]bool{"PATH": true, "SystemRoot": true, "WINDIR": true, "TEMP": true, "TMP": true, "TMPDIR": true, "LANG": true, "LC_ALL": true}
	switch runtime.GOOS {
	case "windows":
		delete(allowed, "SystemRoot")
		allowed["SYSTEMROOT"] = true
		allowed["CUDA_PATH"] = true
		allowed["CUDA_VISIBLE_DEVICES"] = true
	case "linux":
		allowed["LD_LIBRARY_PATH"] = true
		allowed["CUDA_VISIBLE_DEVICES"] = true
	case "darwin":
		allowed["DYLD_LIBRARY_PATH"] = true
		allowed["DYLD_FALLBACK_LIBRARY_PATH"] = true
	}
	selected := []string{}
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			key = strings.ToUpper(key)
		}
		if found && allowed[key] {
			selected = append(selected, entry)
		}
	}
	return selected
}

func childEnvironment(spec Spec) []string {
	if spec.CleanEnv {
		// A nonnil empty slice is deliberate: nil means inherit to os/exec.
		return append([]string{}, spec.Env...)
	}
	return append(os.Environ(), spec.Env...)
}
