// SPDX-License-Identifier: Apache-2.0
// A capture-only transport for the production desktop bridge and shared runtime.
// It never opens native windows or accepts owner data, credentials or remote routes.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/shruggietech/insonic/desktop"
	"github.com/shruggietech/insonic/internal/backendsetup"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
)

type call struct {
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

func decode(raw json.RawMessage, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func stringsFrom(c call, count int) ([]string, error) {
	if len(c.Args) != count {
		return nil, fmt.Errorf("argument count")
	}
	out := make([]string, count)
	for i, raw := range c.Args {
		if decode(raw, &out[i]) != nil {
			return nil, fmt.Errorf("string argument")
		}
	}
	return out, nil
}

func localPlan(raw string) error {
	c, e := backendsetup.Build([]byte(raw))
	if e != nil {
		return e
	}
	if c.DisplayName != "My voice archive" {
		return fmt.Errorf("fixture display name")
	}
	for i, p := range []workspace.Profile{c.Profiles.Storage, c.Profiles.Catalog, c.Profiles.Graph} {
		expected := []string{"filesystem", "sqlite", "ladybugdb"}[i]
		key := []string{"root", "path", "path"}[i]
		target := []string{"artifacts", "catalog.sqlite", "graph/library.lbug"}[i]
		if p.Adapter != expected || len(p.Configuration) != 1 || p.Configuration[key] != target {
			return fmt.Errorf("capture allows fixture local defaults only")
		}
	}
	return nil
}

func fixtureBoundary(library string) error {
	root := filepath.Dir(library)
	if filepath.Base(library) != "My voice archive" {
		return fmt.Errorf("fixture library")
	}
	var identity struct {
		FixtureID string `json:"fixture_id"`
		Kind      string `json:"kind"`
	}
	raw, e := os.ReadFile(filepath.Join(root, "capture-owner.json"))
	if e != nil || decode(raw, &identity) != nil || identity.Kind != "insonic-ui-capture" || len(identity.FixtureID) != 32 {
		return fmt.Errorf("fixture ownership")
	}
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "TEMP", "TMP"} {
		value := os.Getenv(key)
		rel, e := filepath.Rel(root, value)
		if !filepath.IsAbs(value) || e != nil || rel == ".." || len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator) {
			return fmt.Errorf("profile isolation")
		}
	}
	return nil
}

func authorize(c call, library string) error {
	switch c.Method {
	case "Show":
		if len(c.Args) != 0 {
			return fmt.Errorf("argument count")
		}
		return nil
	case "Choose":
		a, e := stringsFrom(c, 1)
		if e != nil || a[0] != "workspace" {
			return fmt.Errorf("picker boundary")
		}
		return nil
	case "SetupDraft":
		a, e := stringsFrom(c, 3)
		if e != nil || !contracts.ValidID(a[2]) {
			return fmt.Errorf("request identity")
		}
		switch a[0] {
		case "show":
			if a[1] != "{}" {
				return fmt.Errorf("show input")
			}
		case "location":
			var input struct {
				Path string `json:"path"`
			}
			if decode([]byte(a[1]), &input) != nil || input.Path != library {
				return fmt.Errorf("location boundary")
			}
		case "save":
			var input struct {
				ExpectedRevision  int64           `json:"expected_revision"`
				Path              string          `json:"path"`
				Plan              json.RawMessage `json:"plan"`
				Stage             string          `json:"stage"`
				Status            string          `json:"status"`
				CredentialBackend string          `json:"credential_backend"`
			}
			if decode([]byte(a[1]), &input) != nil {
				return fmt.Errorf("draft input shape")
			}
			if input.Path != library {
				return fmt.Errorf("draft fixture path")
			}
			// The shared draft store normalizes the unsaved empty election to native.
			if input.CredentialBackend != "" && input.CredentialBackend != "native" {
				return fmt.Errorf("draft credential mode")
			}
			if e := localPlan(string(input.Plan)); e != nil {
				return fmt.Errorf("draft plan: %w", e)
			}
		default:
			return fmt.Errorf("draft action")
		}
		return nil
	case "BackendSetup":
		a, e := stringsFrom(c, 6)
		if e != nil || !contracts.ValidID(a[5]) || a[1] != library || a[4] != "native" {
			return fmt.Errorf("backend boundary")
		}
		if a[0] != "plan" && a[0] != "preflight" && a[0] != "provision" {
			return fmt.Errorf("backend action")
		}
		if localPlan(a[2]) != nil {
			return fmt.Errorf("backend configuration")
		}
		var protected struct {
			Session map[string]json.RawMessage `json:"session"`
		}
		if decode([]byte(a[3]), &protected) != nil || len(protected.Session) != 0 {
			return fmt.Errorf("protected input excluded")
		}
		return nil
	case "SelectWorkspace":
		if len(c.Args) != 3 {
			return fmt.Errorf("argument count")
		}
		var path, name string
		var create bool
		if decode(c.Args[0], &path) != nil || decode(c.Args[1], &name) != nil || decode(c.Args[2], &create) != nil || path != library || name != "My voice archive" || create {
			return fmt.Errorf("selection boundary")
		}
		return nil
	case "Operate":
		if len(c.Args) != 1 {
			return fmt.Errorf("argument count")
		}
		var req contracts.Request
		if decode(c.Args[0], &req) != nil || !contracts.ValidID(req.RequestID) {
			return fmt.Errorf("request boundary")
		}
		allowed := map[string]bool{"workspace.show": true, "settings.show": true, "setup.show": true, "setup.save": true, "pipelines.list": true, "media.list": true, "work.list": true, "import-presets.list": true, "models.list": true, "models.alias.list": true, "models.speaker.list": true, "models.source.list": true, "workers.catalog": true, "models.defaults": true}
		if !allowed[req.Operation] {
			return fmt.Errorf("operation is not in capture inventory: %.60s", req.Operation)
		}
		if req.ItemID != "" || req.SourcePath != "" || req.PublicationID != "" {
			return fmt.Errorf("operation boundary")
		}
		return nil
	default:
		return fmt.Errorf("method boundary")
	}
}

func main() {
	library := flag.String("library", "", "exact isolated fixture library")
	cli := flag.String("cli", "", "current CLI executable")
	flag.Parse()
	if flag.NArg() != 0 || !filepath.IsAbs(*library) || !filepath.IsAbs(*cli) || fixtureBoundary(*library) != nil {
		fmt.Fprintln(os.Stderr, "owned isolated fixture and absolute CLI paths required")
		os.Exit(2)
	}
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		fmt.Fprintln(os.Stderr, "capture listener unavailable")
		os.Exit(1)
	}
	origin := "http://" + listener.Addr().String()
	tokenBytes := make([]byte, 32)
	if _, e = rand.Read(tokenBytes); e != nil {
		os.Exit(1)
	}
	token := hex.EncodeToString(tokenBytes)
	bridge := &desktop.Bridge{CLIExecutable: *cli}
	desktop.SetPicker(bridge, func(string) (string, error) { return *library, nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ownerStarted bool
	var ownerDone chan error
	var mu sync.Mutex
	var calls []string
	ensureOwner := func() error {
		if ownerStarted {
			return nil
		}
		w := desktop.SelectedWorkspace(bridge)
		if w == nil {
			return nil
		}
		ready := make(chan struct{})
		ownerDone = make(chan error, 1)
		go func() { ownerDone <- local.Serve(ctx, w, local.Options{Ready: ready, Idle: 10 * time.Minute}) }()
		select {
		case <-ready:
			ownerStarted = true
			return nil
		case e := <-ownerDone:
			return e
		case <-time.After(25 * time.Second):
			return fmt.Errorf("fixture owner timeout")
		}
	}
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 70 * time.Second}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Capture-Token") != token || r.Header.Get("Origin") != origin {
			http.Error(w, "capture authorization", 403)
			return
		}
		if r.URL.Path == "/receipt" && r.Method == "GET" {
			mu.Lock()
			defer mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"calls": calls, "runtime": "shared-production-runtime", "bridge": "desktop.Bridge", "platform": runtime.GOOS, "architecture": runtime.GOARCH})
			return
		}
		if r.URL.Path == "/shutdown" && r.Method == "POST" {
			w.WriteHeader(204)
			go func() { cancel(); server.Close() }()
			return
		}
		if r.URL.Path != "/bridge" || r.Method != "POST" {
			http.Error(w, "capture route", 404)
			return
		}
		raw, e := io.ReadAll(io.LimitReader(r.Body, (128<<10)+1))
		if e != nil || len(raw) > 128<<10 {
			http.Error(w, "capture limit", 413)
			return
		}
		var c call
		if decode(raw, &c) != nil {
			http.Error(w, "capture boundary", 400)
			return
		}
		if e := authorize(c, *library); e != nil {
			fmt.Fprintln(os.Stderr, "capture boundary:", c.Method, e)
			http.Error(w, "capture boundary", 400)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if len(calls) >= 512 {
			http.Error(w, "capture call limit", 429)
			return
		}
		calls = append(calls, c.Method)
		var out any
		switch c.Method {
		case "Show":
			out = bridge.Show()
		case "Choose":
			out = *library
		case "SetupDraft":
			a, _ := stringsFrom(c, 3)
			out = bridge.SetupDraft(a[0], a[1], a[2])
		case "BackendSetup":
			a, _ := stringsFrom(c, 6)
			response := bridge.BackendSetup(a[0], a[1], a[2], a[3], a[4], a[5])
			if response.Error == nil && a[0] == "provision" {
				if e := ensureOwner(); e != nil {
					http.Error(w, "fixture runtime unavailable", 500)
					return
				}
			}
			out = response
		case "SelectWorkspace":
			var path, name string
			var create bool
			decode(c.Args[0], &path)
			decode(c.Args[1], &name)
			decode(c.Args[2], &create)
			out = bridge.SelectWorkspace(path, name, create)
		case "Operate":
			var req contracts.Request
			decode(c.Args[0], &req)
			out = bridge.Operate(req)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})
	json.NewEncoder(os.Stdout).Encode(map[string]any{"endpoint": "http://" + listener.Addr().String(), "token": token, "version": contracts.Version})
	e = server.Serve(listener)
	cancel()
	if ownerStarted {
		select {
		case <-ownerDone:
		case <-time.After(10 * time.Second):
			fmt.Fprintln(os.Stderr, "fixture owner did not stop")
			os.Exit(1)
		}
	}
	if e != nil && e != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "capture transport failed")
		os.Exit(1)
	}
}
