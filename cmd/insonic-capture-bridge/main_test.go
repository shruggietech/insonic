// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixtureCall(method string, args ...any) call {
	c := call{Method: method}
	for _, a := range args {
		raw, _ := json.Marshal(a)
		c.Args = append(c.Args, raw)
	}
	return c
}

func TestFixtureBoundaryRequiresOwnedProfileIsolation(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "My voice archive")
	if fixtureBoundary(library) == nil {
		t.Fatal("accepted unowned fixture")
	}
	if e := os.WriteFile(filepath.Join(root, "capture-owner.json"), []byte(`{"fixture_id":"11111111111111111111111111111111","kind":"insonic-ui-capture"}`), 0600); e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "TEMP", "TMP"} {
		t.Setenv(key, filepath.Join(root, key))
	}
	if e := fixtureBoundary(library); e != nil {
		t.Fatal(e)
	}
	t.Setenv("APPDATA", filepath.Dir(root))
	if fixtureBoundary(library) == nil {
		t.Fatal("accepted owner profile outside fixture")
	}
}

func TestCaptureExcludesOwnerPathsSecretsAndExternalBackends(t *testing.T) {
	library := "C:/insonic-capture-fixture/library"
	id := "11111111-1111-4111-8111-111111111111"
	local := `{"kind":"backend-plan","schema_version":"1.0.2","display_name":"My voice archive"}`
	good := []call{fixtureCall("Show"), fixtureCall("SetupDraft", "show", "{}", id), fixtureCall("SetupDraft", "location", `{"path":"`+library+`"}`, id), fixtureCall("BackendSetup", "plan", library, local, "{}", "native", id)}
	good = append(good, fixtureCall("SetupDraft", "save", `{"expected_revision":0,"path":"`+library+`","plan":`+local+`,"stage":"backends","status":"active","credential_backend":"native"}`, id))
	for _, c := range good {
		if e := authorize(c, library); e != nil {
			t.Fatal(c.Method, e)
		}
	}
	bad := []call{fixtureCall("Choose", "media"), fixtureCall("SetupDraft", "location", `{"path":"C:/Users/Owner"}`, id), fixtureCall("BackendSetup", "plan", library, local, `{"session":{"secret":"value"}}`, "native", id), fixtureCall("BackendSetup", "plan", library, local, `{"passphrase":"value"}`, "vault", id), fixtureCall("BackendSetup", "plan", library, `{"kind":"backend-plan","schema_version":"1.0.2","display_name":"My voice archive","profiles":{"storage":{"adapter_id":"filesystem","configuration":{"root":"C:/Users/Owner"}}}}`, "{}", "native", id), fixtureCall("SelectWorkspace", library, "My voice archive", true), fixtureCall("Credential", "add"), fixtureCall("Operate", map[string]any{"request_id": id, "operation": "media.import", "source_path": "C:/Users/Owner/audio.wav"}), fixtureCall("Operate", map[string]any{"request_id": id, "operation": "settings.set", "data": map[string]any{"endpoint": "https://outside.example"}})}
	for _, c := range bad {
		if authorize(c, library) == nil {
			t.Fatalf("accepted boundary %s", c.Method)
		}
	}
}

func TestCaptureRejectsUnknownAndTrailingFields(t *testing.T) {
	for _, raw := range []string{`{"method":"Show","args":[],"command":"anything"}`, `{"method":"Show","args":[]} {}`} {
		var c call
		if decode([]byte(raw), &c) == nil {
			t.Fatal("accepted malformed transport")
		}
	}
}
