// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"testing"
)

// The hashed Go test executable doubles as an actual Cueson-shaped child.
// Register only its fixed CLI flag so Driver.run uses its normal argument path.
func init() { flag.Bool("no-color", false, "Disable color in the Cueson environment test helper.") }

func TestCuesonChildEnvironmentHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "cueson-environment-helper" {
		return
	}
	observed := map[string]bool{}
	for _, key := range []string{"INSONIC_SESSION_CREDENTIALS", "OPENAI_API_KEY", "HF_TOKEN"} {
		_, observed[key] = os.LookupEnv(key)
	}
	result := struct {
		Secrets map[string]bool `json:"secrets_present"`
		Locale  string          `json:"locale"`
	}{observed, os.Getenv("LANG")}
	if json.NewEncoder(os.Stdout).Encode(result) != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestDriverChildEnvironmentDoesNotInheritCredentials(t *testing.T) {
	t.Setenv("INSONIC_SESSION_CREDENTIALS", "session-test-marker")
	t.Setenv("OPENAI_API_KEY", "provider-test-marker")
	t.Setenv("HF_TOKEN", "model-test-marker")
	t.Setenv("LANG", "C")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	driver, err := New(Tool{Executable: executable, ExecutableSHA256: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	result, err := driver.run(context.Background(), t.TempDir(), "-test.run=^TestCuesonChildEnvironmentHelper$", "--", "cueson-environment-helper")
	if err != nil {
		t.Fatal(err)
	}
	var observed struct {
		Secrets map[string]bool `json:"secrets_present"`
		Locale  string          `json:"locale"`
	}
	if json.Unmarshal(result.Stdout, &observed) != nil || len(observed.Secrets) != 3 {
		t.Fatal("invalid child environment report")
	}
	for key, present := range observed.Secrets {
		if present {
			t.Fatalf("Cueson child inherited credential environment key %s", key)
		}
	}
	if observed.Locale != "C" {
		t.Fatal("allowed locale was not preserved")
	}
}
