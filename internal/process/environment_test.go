// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestEnvironmentHelper(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "--environment-helper") {
		return
	}
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	json.NewEncoder(os.Stdout).Encode(values)
	os.Exit(0)
}

func TestChildEnvironmentIsolation(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"INSONIC_SESSION_CREDENTIALS", "OPENAI_API_KEY", "ARBITRARY_PROVIDER_TOKEN", "HF_TOKEN", "PYTHONPATH"} {
		t.Setenv(key, "fake-test-credential")
	}
	t.Setenv("LANG", "C")
	for _, launch := range []string{"Capture", "Run"} {
		t.Run(launch, func(t *testing.T) {
			spec := Spec{Executable: executable, Args: []string{"-test.run=^TestEnvironmentHelper$", "--", "--environment-helper"}, Env: append(LocalEnvironment(), "LANG=C", "HF_HUB_OFFLINE=1", "INSONIC_EXPLICIT_FLAG=selected"), CleanEnv: true, MaxOutput: 1 << 20}
			var output []byte
			if launch == "Capture" {
				result, e := Capture(context.Background(), spec)
				err, output = e, result.Stdout
			} else {
				result, e := Run(context.Background(), spec)
				err, output = e, result.Output
			}
			if err != nil {
				t.Fatal(err)
			}
			var values map[string]string
			if json.Unmarshal(output, &values) != nil {
				t.Fatal("invalid environment helper response")
			}
			for _, key := range []string{"INSONIC_SESSION_CREDENTIALS", "OPENAI_API_KEY", "ARBITRARY_PROVIDER_TOKEN", "HF_TOKEN", "PYTHONPATH"} {
				if _, exists := values[key]; exists {
					t.Fatalf("unexpected inherited credential/configuration key: %s", key)
				}
			}
			if values["LANG"] != "C" || values["HF_HUB_OFFLINE"] != "1" || values["INSONIC_EXPLICIT_FLAG"] != "selected" {
				t.Fatal("explicit selected environment was lost")
			}
			// An explicitly empty environment must remain nonnil at exec.Cmd,
			// otherwise Go silently restores the complete parent environment.
			spec.Env = nil
			if launch == "Capture" {
				result, e := Capture(context.Background(), spec)
				err, output = e, result.Stdout
			} else {
				result, e := Run(context.Background(), spec)
				err, output = e, result.Output
			}
			values = nil
			if err != nil || json.Unmarshal(output, &values) != nil {
				t.Fatal("empty-environment launch failed", err)
			}
			if _, inherited := values["INSONIC_SESSION_CREDENTIALS"]; inherited {
				t.Fatal("empty environment inherited parent credentials")
			}
			for key := range values {
				if runtime.GOOS != "windows" || !strings.EqualFold(key, "SystemRoot") {
					t.Fatalf("empty environment gained unexpected key: %s", key)
				}
			}
			// Existing process callers keep their default inherited environment.
			spec.CleanEnv = false
			if launch == "Capture" {
				result, e := Capture(context.Background(), spec)
				err, output = e, result.Stdout
			} else {
				result, e := Run(context.Background(), spec)
				err, output = e, result.Output
			}
			values = nil
			if err != nil || json.Unmarshal(output, &values) != nil || values["INSONIC_SESSION_CREDENTIALS"] != "fake-test-credential" {
				t.Fatal("default inherited environment semantics changed")
			}
		})
	}
}

func TestLocalEnvironmentKeepsOnlySelectedExecutionKeys(t *testing.T) {
	t.Setenv("LANG", "C")
	t.Setenv("HOME", "fake-unselected-home")
	t.Setenv("APPDATA", "fake-unselected-appdata")
	t.Setenv("INSONIC_SESSION_CREDENTIALS", "fake-test-credential")
	t.Setenv("ARBITRARY_PROVIDER_TOKEN", "fake-test-credential")
	var loader string
	switch runtime.GOOS {
	case "windows":
		loader = "CUDA_PATH"
	case "linux":
		loader = "LD_LIBRARY_PATH"
	case "darwin":
		loader = "DYLD_LIBRARY_PATH"
	}
	if loader != "" {
		t.Setenv(loader, t.TempDir())
	}
	values := map[string]string{}
	for _, entry := range LocalEnvironment() {
		key, value, _ := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			key = strings.ToUpper(key)
		}
		values[key] = value
	}
	for _, key := range []string{"HOME", "APPDATA", "INSONIC_SESSION_CREDENTIALS", "ARBITRARY_PROVIDER_TOKEN"} {
		if _, inherited := values[key]; inherited {
			t.Fatalf("unselected key survived allowlist: %s", key)
		}
	}
	if values["LANG"] != "C" || loader != "" && values[loader] != os.Getenv(loader) {
		t.Fatal("required execution or selected loader setting lost")
	}
}
