// SPDX-License-Identifier: Apache-2.0
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Reviewed documentation inventory is shipped with its parser.
//
//go:embed command_guidance.json
var guidanceInventory []byte

type optionGuidance struct{ Meaning, Default, Conditions string }
type helpInventory struct {
	Commands, Groups map[string]string
	Options          map[string]optionGuidance
}

var guidance = func() helpInventory {
	var value helpInventory
	if err := json.Unmarshal(guidanceInventory, &value); err != nil {
		panic(err)
	}
	return value
}()

func commandScope(p string) string {
	if p == "version" || p == "doctor" || strings.HasPrefix(p, "workspace init") || strings.HasPrefix(p, "workspace plan") || strings.HasPrefix(p, "workspace preflight") || strings.HasPrefix(p, "workspace provision") || p == "setup location" || strings.HasPrefix(p, "setup draft ") || p == "setup provision" || p == "query services" {
		return "pre-library operation; no selected library is required. Setup drafts use the private user profile."
	}
	if strings.HasPrefix(p, "backup ") || strings.HasPrefix(p, "catalog ") {
		return "selected library; backup show only inspects its directory. Other backup actions and catalog migrate/restore require exclusive ownership; stop clients and wait for the owner to exit."
	}
	return "selected library, discovered from the working directory or elected with --workspace. Clients start or reuse its shared runtime."
}

func operandGuidance(c commandDefinition) string {
	p := strings.Join(c.Path, " ")
	switch {
	case p == "setup location":
		return "optional absolute directory outside installed application bytes; omitted previews the suggested default."
	case strings.HasPrefix(p, "backup "):
		if p == "backup release" {
			return "backup directory (absolute or relative), or the backup UUID returned by create. Release removes retention, not the source backup bytes."
		}
		return "backup directory (absolute or relative); create needs a fresh target, other actions read the existing backup."
	case p == "jobs start":
		return "required duration in milliseconds, 1..600000 inclusive; cancel does not require waiting for the duration."
	case p == "jobs history":
		return "required job UUID; optional nonnegative generation number returned by the previous page."
	case p == "artifacts publish":
		return "required source file and nonempty artifact kind (up to 128 Unicode characters), a caller classification such as derived or model. No fixed kind enum is imposed. File bytes remain immutable."
	case p == "artifacts materialize":
		return "required publication UUID; optional positive maximum byte count (default 10737418240 bytes). Reuse the returned lease_id for lease-renew/release."
	case strings.HasPrefix(p, "artifacts lease-"):
		return "required publication UUID and its returned lease UUID. A materialization lease lasts 30 seconds by default; renew before expiry and release when finished."
	case p == "artifacts retain" || p == "artifacts release-reference":
		return "required publication UUID and caller-owned reference UUID; retain keeps bytes live until the matching reference is released."
	case p == "credentials select":
		return "required backend: native, vault or session. Selection is persisted per library; secret values never belong in arguments."
	case strings.HasPrefix(p, "credentials "):
		return "required credential UUID, a library-local lookup identity; supply credential values only through protected stdin."
	case p == "media import":
		return "one or more source paths/URLs, or --manifest FILE; sources are not recording UUIDs. New admission retains managed canonical audio."
	case p == "transcript import":
		return "single transcript path/URL plus --record UUID_OR_TITLE, or --manifest FILE for per-item targets. --legacy-sidecar uses --record without a path."
	case p == "media tools" || p == "processing tools":
		return "required JSON configuration file, containing kind, schema_version and exact absolute executable paths/digests; see the complete positional document contract."
	case strings.Contains(p, "models") && (strings.HasSuffix(p, " register") || strings.HasSuffix(p, " acquire") || strings.HasSuffix(p, " download")):
		return "required immutable base-model manifest JSON file with kind, schema_version, name, model_version, upstream_revision and exact file digests. Acceptance returns work_id; use work wait WORK_UUID for completion."
	case p == "models resolve":
		return "required model reference: registered UUID, immutable name/version or configured alias. --operation selects the intended capability."
	case p == "search":
		return "required search text as one shell argument; quote spaces. Continue with unchanged text/filters and the opaque next_cursor."
	case strings.HasPrefix(p, "import-presets "):
		return "required preset UUID, or exact saved name for show only. Updates and deletion use the current revision from show."
	case p == "media relocate":
		return "required media UUID and new legacy referenced path; only legacy referenced entries can relocate. New admission owns canonical bytes."
	case c.Operands == "FILE":
		return "required JSON/file path; relative paths resolve from the invoking working directory. Exports require a new destination and never overwrite an existing file."
	case strings.Contains(c.Operands, "DIRECTORY"):
		return "required library directory; relative paths resolve from the invoking working directory. Provisioning requires a fresh target."
	default:
		return "required current library identity: " + c.Operands + ". Reuse the exact identity returned by its list/create/show operation, not a source filename."
	}
}

// Executable argv is independent of usage grammar and optional syntax.
func commandExample(c commandDefinition) []string {
	p := strings.Join(c.Path, " ")
	out := append([]string{}, c.Path...)
	if c.Min > 0 {
		operands := map[string][]string{
			"DIRECTORY": {"DIRECTORY"}, "ABSOLUTE_DIRECTORY": {"ABSOLUTE_DIRECTORY"}, "FILE": {"FILE"},
			"UUID": {"UUID"}, "UUID_OR_EXACT_NAME": {"UUID"}, "MEDIA_UUID": {"MEDIA_UUID"},
			"JOB_UUID": {"JOB_UUID"}, "JOB_UUID [AFTER_GENERATION]": {"JOB_UUID"},
			"MILLISECONDS (1..600000)": {"1"}, "PUBLICATION_UUID": {"PUBLICATION_UUID"},
			"FILE KIND": {"FILE", "derived"}, "PUBLICATION_UUID [MAX_BYTES]": {"PUBLICATION_UUID"},
			"PUBLICATION_UUID REFERENCE_OR_LEASE_UUID": {"PUBLICATION_UUID", "REFERENCE_UUID"},
			"DIRECTORY_OR_BACKUP_UUID":                 {"DIRECTORY"}, "WORK_UUID": {"WORK_UUID"},
			"MODEL_UUID": {"MODEL_UUID"}, "MANIFEST_FILE": {"MANIFEST_FILE"}, "REFERENCE": {"REFERENCE"},
			"SOURCE_UUID": {"SOURCE_UUID"}, "VERSION_UUID": {"VERSION_UUID"}, "SPEAKER_UUID": {"SPEAKER_UUID"},
			"QUERY_UUID": {"QUERY_UUID"}, "VIEW_UUID": {"VIEW_UUID"}, "MEDIA_UUID PATH": {"MEDIA_UUID", "FILE"},
			"TEXT": {"fixture"}, "native|vault|session": {"session"},
		}
		out = append(out, operands[c.Operands]...)
	}
	switch p {
	case "media import":
		out = append(out, "FILE", "--no-pipeline")
	case "transcript import":
		out = append(out, "FILE", "--record", "MEDIA_UUID")
	case "setup location":
		out = append(out, "ABSOLUTE_DIRECTORY")
	case "media set-origin":
		out = append(out, "--originated-on", "2026-10-04")
	case "query run", "query explain":
		out = append(out, "--input", "INPUT_FILE")
	}
	for _, key := range c.Required {
		value := "INPUT_FILE"
		switch key {
		case "--revision", "--expected-revision":
			value = "CURRENT_REVISION"
		case "--speaker":
			value = "SPEAKER_UUID"
		case "--destination":
			value = "DIRECTORY"
		case "--file":
			value = "FILE"
		default:
			if o := optionDefinitions[key]; o.Values != "" {
				value = strings.Split(o.Values, "|")[0]
			}
		}
		out = append(out, key, value)
	}
	return out
}

func writeTaskGuidance(b *strings.Builder, c commandDefinition) {
	p := strings.Join(c.Path, " ")
	fmt.Fprintf(b, "Purpose: %s\nWorkspace: %s\n", guidance.Commands[p], commandScope(p))
	b.WriteString("Paths: relative source, input and destination paths resolve from the invoking working directory; setup location requires an absolute directory. Tool configuration embeds absolute pinned executable paths.\n")
	if c.Operands != "" {
		fmt.Fprintf(b, "Operands: %s\n", operandGuidance(c))
	}
	if containsOption(c.Options, "--input") {
		b.WriteString("Input workflow: copy the JSON template to INPUT_FILE, replace documented identities/revisions/paths and elected fixture selections with current values, then run the example. Retain the same request UUID and immutable input when retrying an unknown mutation outcome.\n")
	}
	if p == "jobs cancel" || p == "work cancel" {
		b.WriteString("Lifecycle: cancellation applies to active work; a successful terminal item cannot be cancelled. Inspect its current state with jobs show JOB_UUID or work show WORK_UUID.\n")
	}
	if p == "jobs retry" || p == "work retry" {
		b.WriteString("Lifecycle: retry requires a failed, cancelled or interrupted item; successful terminal work is not retryable. Preserve the original immutable inputs.\n")
	}
	if p == "models train" {
		b.WriteString("Prerequisites: an elected nonempty speaker dataset and a configured speaker-training pipeline at its current revision; training is optional and its accepted work still requires completion.\n")
	}
	if strings.Contains(p, "import") || strings.Contains(p, "models") || strings.HasPrefix(p, "work ") || p == "recordings process" || p == "workers setup" {
		b.WriteString("Completion: accepted durable work is not completion. Reuse returned work_id with insonic work wait WORK_UUID; inspect work show/results and cancel or retry the same identity as appropriate.\n")
	}
	if p == "media tools" || p == "processing tools" {
		b.WriteString("Effective time: the shared runtime accepts the override now; subsequent operations use the saved exact pins. Already running operations keep their prior configuration.\n")
	}
	if p == "setup provision" || p == "workspace provision" || p == "workspace preflight" || p == "runtime serve" {
		b.WriteString("Protected stdin: --secret-input reads {\"passphrase\":\"...\",\"session\":{\"11111111-1111-4111-8111-111111111111\":\"...\"}}. Include only fields needed by the selected vault/session backend and plan. Values may be strings or provider credential objects; never put secrets in files or arguments. Native/local-default provisioning needs no protected stdin.\n")
	}
	if strings.HasPrefix(p, "credentials ") {
		b.WriteString("Protected stdin: add/replace use {\"value\":\"...\"} for native/session, or {\"passphrase\":\"...\",\"value\":\"...\"} for vault. Vault unlock/load/delete use {\"passphrase\":\"...\"}; session load uses {\"session\":{\"11111111-1111-4111-8111-111111111111\":\"...\"}}. Native/session delete and status need no stdin. Values may be provider objects. The object is bounded to 1 MiB; each secret is byte-bounded and is never echoed.\n")
	}
	fmt.Fprintf(b, "Example: insonic %s\n", strings.Join(commandExample(c), " "))
	b.WriteString("Placeholders: uppercase *_UUID and UUID are identities returned by this library; CURRENT_REVISION is its current revision; REFERENCE is an elected model reference; FILE, INPUT_FILE, MANIFEST_FILE, DIRECTORY and ABSOLUTE_DIRECTORY are owned fixture/user paths. INPUT_FILE and positional documents use the displayed contracts; replace only documented identities, revisions, paths and elected fixture selections. Synthetic qualification owns those fixtures and records each actual emitted invocation.\n")
}
