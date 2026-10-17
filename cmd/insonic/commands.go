// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"sort"
	"strconv"
	"strings"
)

type commandDefinition struct {
	Path     []string
	Operands string
	Min, Max int
	Options  []string
	Required []string
}
type invocation struct {
	Args, Path            []string
	Root, RequestID       string
	StartupID             string
	Machine, Help         bool
	Quiet, Verbose, Debug bool
	LogLevel, LogFile     string
}
type cliOption struct {
	Value    bool
	Values   string
	Min, Max int64
	Numeric  bool
}

var optionDefinitions = func() map[string]cliOption {
	out := map[string]cliOption{}
	for _, key := range strings.Fields("--workspace --request-id --startup-id --log-level --log-file --input --manifest --title --record --transcript --subtitle --credential-id --acquisition-adapter --acquisition-max-bytes --acquisition-timeout-ms --originated-at --originated-on --originated-earliest --originated-latest --timezone --dst-fold --dst-gap --date-precedence --preset --pipeline-id --destination-profile --attribution --transcript-format --transcript-credential-id --transcript-adapter --transcript-max-bytes --transcript-timeout-ms --subtitle-stream-index --subtitle-language --diarization-model-id --known-speaker --existing-transcript --existing-roster --revision --after --after-ordinal --limit --speaker --media --dialect --file --parameters --saved --operation --expected-revision --destination --mode --credential-backend --timeout-ms") {
		out[key] = cliOption{Value: true}
	}
	for _, key := range strings.Fields("--no-pipeline --json --verbose --debug --quiet --copy --reference --local-http --new-entry --replace-audio --transcript-applies --replace-transcript --legacy-sidecar --secret-input --help -h --version") {
		out[key] = cliOption{}
	}
	enums := map[string]string{"--dst-fold": "earlier|later", "--dst-gap": "shift-forward", "--date-precedence": "owner-first|embedded-first|filesystem-fallback", "--attribution": "auto|native|off|diarize", "--transcript-format": "cueson|srt|vtt|ass|ssa", "--existing-transcript": "keep|clear|replace", "--existing-roster": "retain|clear", "--mode": "self-contained|reference-only", "--operation": "transcription|diarization|voice-matching|speaker-model-training", "--dialect": "ladybug-cypher|arcade-opencypher|arcade-sql"}
	for key, values := range enums {
		o := out[key]
		o.Values = values
		out[key] = o
	}
	for key, limits := range map[string][2]int64{"--limit": {1, 100}, "--timeout-ms": {1, 600000}, "--revision": {1, 9223372036854775807}, "--expected-revision": {0, 9223372036854775807}, "--after-ordinal": {0, 9223372036854775807}, "--subtitle-stream-index": {0, 9223372036854775807}, "--transcript-max-bytes": {1, 16 << 20}, "--transcript-timeout-ms": {1, 600000}, "--acquisition-max-bytes": {1, 9223372036854775807}, "--acquisition-timeout-ms": {1, 9223372036854}} {
		o := out[key]
		o.Numeric = true
		o.Min = limits[0]
		o.Max = limits[1]
		out[key] = o
	}
	return out
}()

var commandDefinitions = func() []commandDefinition {
	var out []commandDefinition
	add := func(path, operands string, min, max int, options, required string) {
		out = append(out, commandDefinition{strings.Fields(path), operands, min, max, strings.Fields(options), strings.Fields(required)})
	}
	add("version", "", 0, 0, "", "")
	add("doctor", "", 0, 0, "", "")
	add("workspace init", "DIRECTORY", 1, 1, "", "")
	add("workspace plan", "", 0, 0, "--input", "--input")
	add("workspace preflight", "DIRECTORY", 1, 1, "--input --secret-input", "--input")
	add("workspace provision", "DIRECTORY", 1, 1, "--input --secret-input --credential-backend", "--input")
	add("query services", "", 0, 0, "--input", "")
	add("query assistance-test", "", 0, 0, "--input", "--input")
	add("workspace show", "", 0, 0, "", "")
	add("runtime serve", "", 0, 0, "--secret-input --startup-id", "")
	for _, a := range strings.Fields("select add replace delete status unlock load") {
		min, max, operand := 1, 1, "UUID"
		if a == "unlock" || a == "load" {
			min, max, operand = 0, 0, ""
		}
		if a == "select" {
			operand = "native|vault|session"
		}
		add("credentials "+a, operand, min, max, "", "")
	}
	for _, a := range strings.Fields("show migrate export restore") {
		n, operand := 0, ""
		if a == "export" || a == "restore" {
			n, operand = 1, "FILE"
		}
		add("catalog "+a, operand, n, n, "", "")
	}
	for _, a := range strings.Fields("start show history cancel retry") {
		max, operand := 1, "JOB_UUID"
		if a == "start" {
			operand = "MILLISECONDS (1..600000)"
		}
		if a == "history" {
			max = 2
			operand += " [AFTER_GENERATION]"
		}
		add("jobs "+a, operand, 1, max, "", "")
	}
	for _, a := range strings.Fields("publish show verify materialize reconcile abort retire cache-prune lease-renew lease-release retain release-reference") {
		n, max, operand := 1, 1, "PUBLICATION_UUID"
		if a == "publish" {
			n, max, operand = 2, 2, "FILE KIND"
		}
		if a == "materialize" {
			max = 2
			operand += " [MAX_BYTES]"
		}
		if a == "lease-renew" || a == "lease-release" || a == "retain" || a == "release-reference" {
			n, max = 2, 2
			operand += " REFERENCE_OR_LEASE_UUID"
		}
		add("artifacts "+a, operand, n, max, "", "")
	}
	for _, a := range strings.Fields("create show verify restore release") {
		opts := ""
		if a == "create" {
			opts = "--mode"
		}
		add("backup "+a, "DIRECTORY_OR_BACKUP_UUID", 1, 1, opts, "")
	}
	for _, g := range strings.Fields("pipelines speakers terms") {
		actions := "list show set"
		if g == "pipelines" {
			actions += " inspect"
		}
		if g == "speakers" {
			actions += " aliases select diagnostics"
		}
		if g == "terms" {
			actions += " compile"
		}
		for _, a := range strings.Fields(actions) {
			n, operand := 1, "UUID"
			if a == "list" || a == "compile" {
				n, operand = 0, ""
			}
			opts, required := "--input", ""
			if a == "show" {
				opts = ""
			}
			if a == "set" {
				required = "--input"
			}
			add(g+" "+a, operand, n, n, opts, required)
		}
	}
	for _, a := range strings.Fields("create list show update delete") {
		n, operand, opts, required := 0, "", "", ""
		if a == "show" || a == "update" || a == "delete" {
			n = 1
			operand = "UUID_OR_EXACT_NAME"
		}
		if a == "list" {
			opts = "--input"
		}
		if a == "update" || a == "delete" {
			operand = "UUID"
		}
		if a == "create" || a == "update" || a == "delete" {
			opts = "--input"
			required = "--input"
		}
		add("import-presets "+a, operand, n, n, opts, required)
	}
	add("setup location", "[DIRECTORY]", 0, 1, "", "")
	add("setup draft show", "", 0, 0, "", "")
	add("setup draft save", "", 0, 0, "--input", "--input")
	add("setup provision", "", 0, 0, "--secret-input --credential-backend", "")
	add("setup show", "", 0, 0, "", "")
	add("setup save", "", 0, 0, "--input", "--input")
	add("setup readiness", "", 0, 0, "", "")
	add("settings setup", "", 0, 0, "", "")
	add("settings show", "", 0, 0, "", "")
	add("settings set", "", 0, 0, "--input", "--input")
	add("media tools", "FILE", 1, 1, "", "")
	add("processing tools", "FILE", 1, 1, "", "")
	add("workers catalog", "", 0, 0, "", "")
	add("workers inspect", "", 0, 0, "--input", "--input")
	add("workers plan", "", 0, 0, "--input", "--input")
	add("workers setup", "", 0, 0, "--input", "--input")
	add("processing readiness", "", 0, 0, "--input", "")
	add("models defaults", "", 0, 0, "", "")
	add("models prepare", "", 0, 0, "--input", "--input")
	dates := "--originated-at --originated-on --originated-earliest --originated-latest --timezone --dst-fold --dst-gap --date-precedence"
	transcript := "--record --legacy-sidecar --replace-transcript --attribution --transcript-format --transcript-credential-id --transcript-adapter --transcript-max-bytes --transcript-timeout-ms --subtitle-stream-index --subtitle-language --diarization-model-id --local-http --manifest"
	add("media import", "[FILES...] or --manifest FILE", 0, 10000, dates+" "+transcript+" --copy --reference --transcript --subtitle --title --credential-id --acquisition-adapter --acquisition-max-bytes --acquisition-timeout-ms --new-entry --preset --pipeline-id --no-pipeline --destination-profile --known-speaker --replace-audio --existing-transcript --transcript-applies --existing-roster", "")
	add("transcript import", "[PATH_OR_URL --record UUID_OR_TITLE] or --manifest FILE", 0, 1, transcript, "")
	for _, a := range strings.Fields("list show metadata raw refresh set-origin relocate capture playback playback-check playback-close") {
		n, max, operand, opts, required := 1, 1, "MEDIA_UUID", "", ""
		switch a {
		case "list":
			n, max, operand, opts = 0, 0, "", "--after --limit"
		case "refresh":
			opts = dates + " --preset --pipeline-id --no-pipeline --destination-profile"
		case "set-origin":
			opts = dates + " --revision"
			required = "--revision"
		case "relocate":
			n, max, operand, opts, required = 2, 2, "MEDIA_UUID PATH", "--revision", "--revision"
		case "capture", "playback", "playback-check", "playback-close":
			opts, required = "--input", "--input"
		}
		add("media "+a, operand, n, max, opts, required)
	}
	for _, a := range strings.Fields("show document mappings process assemble map-speaker match export resolve-segment cues") {
		opts, required := "--input", "--input"
		if a == "show" {
			opts, required = "", ""
		}
		if a == "document" || a == "mappings" || a == "cues" {
			required = ""
		}
		add("recordings "+a, "MEDIA_UUID", 1, 1, opts, required)
	}
	for _, a := range strings.Fields("show add remove replace clear") {
		opts, required := "--speaker --expected-revision", "--expected-revision"
		if a == "show" {
			opts, required = "", ""
		}
		if a == "clear" {
			opts = "--expected-revision"
		}
		if a == "add" || a == "remove" {
			required += " --speaker"
		}
		add("recordings roster "+a, "MEDIA_UUID", 1, 1, opts, required)
	}
	for _, a := range strings.Fields("list show results wait cancel retry") {
		n, operand, opts := 1, "WORK_UUID", ""
		if a == "list" {
			n, operand, opts = 0, "", "--after --limit"
		}
		if a == "results" {
			opts = "--after-ordinal --limit"
		}
		if a == "wait" {
			opts = "--timeout-ms"
		}
		add("work "+a, operand, n, n, opts, "")
	}
	for _, prefix := range []string{"models", "models base"} {
		for _, a := range strings.Fields("register acquire download list show verify materialize") {
			n, operand, opts := 1, "MODEL_UUID", ""
			if a == "register" || a == "acquire" || a == "download" {
				operand = "MANIFEST_FILE"
			}
			if a == "list" {
				n, operand, opts = 0, "", "--after --limit"
				if prefix == "models" {
					opts += " --speaker"
				}
			}
			add(prefix+" "+a, operand, n, n, opts, "")
		}
	}
	add("models resolve", "REFERENCE", 1, 1, "--operation", "--operation")
	add("models discover", "SOURCE_UUID", 1, 1, "", "")
	for _, g := range strings.Fields("alias source") {
		for _, a := range strings.Fields("list show set remove") {
			n, operand, opts, required := 1, "UUID", "", ""
			if a == "list" {
				n, operand, opts = 0, "", "--input"
			}
			if a == "set" {
				opts, required = "--input", "--input"
			}
			if a == "remove" {
				opts, required = "--expected-revision", "--expected-revision"
			}
			add("models "+g+" "+a, operand, n, n, opts, required)
		}
	}
	for _, g := range strings.Fields("dataset speaker") {
		for _, a := range strings.Fields("list show") {
			n, operand, opts := 1, "UUID", ""
			if a == "list" {
				n, operand, opts = 0, "", "--input"
			}
			add("models "+g+" "+a, operand, n, n, opts, "")
		}
	}
	add("models dataset create", "", 0, 0, "--input", "--input")
	add("models train", "", 0, 0, "--input", "--input")
	add("models speaker fetch", "VERSION_UUID", 1, 1, "--destination", "--destination")
	add("models profile set", "SPEAKER_UUID", 1, 1, "--input", "--input")
	for _, a := range strings.Fields("capabilities status publish rebuild") {
		add("graph "+a, "", 0, 0, "", "")
	}
	for _, a := range strings.Fields("extract show") {
		req := ""
		if a == "extract" {
			req = "--input"
		}
		add("evidence "+a, "MEDIA_UUID", 1, 1, "--input", req)
	}
	for _, a := range strings.Fields("assist assistance-show assistance-set run explain list show save validate") {
		n, operand, opts, required := 0, "", "--input", ""
		if a == "show" || a == "save" {
			n, operand = 1, "QUERY_UUID"
		}
		if a == "assistance-show" {
			opts = ""
		}
		if a == "run" || a == "explain" {
			opts += " --saved --revision"
		}
		if a == "assist" || a == "assistance-set" || a == "save" || a == "validate" {
			required = "--input"
		}
		add("query "+a, operand, n, n, opts, required)
	}
	add("query native", "", 0, 0, "--dialect --file --parameters", "--dialect --file")
	add("search", "TEXT", 1, 1, "--speaker --media --limit --after", "")
	add("timeline calendar", "", 0, 0, "--input", "")
	add("timeline recording", "MEDIA_UUID", 1, 1, "--input", "")
	add("views show", "VIEW_UUID", 1, 1, "--input", "")
	add("views save", "VIEW_UUID", 1, 1, "--input", "--input")
	return out
}()

func containsOption(options []string, key string) bool {
	for _, s := range options {
		if s == key {
			return true
		}
	}
	return false
}
func argumentError(message string) error {
	return &contracts.Error{Code: "invalid_request", Message: message}
}

func parseInvocation(args []string) (invocation, error) {
	out := invocation{RequestID: contracts.ID()}
	var words, options, literals []string
	seen := map[string]int{}
	values := map[string]string{}
	var first error
	literal := false
	fail := func(e error) {
		if first == nil {
			first = e
		}
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if literal {
			words = append(words, arg)
			literals = append(literals, arg)
			continue
		}
		if arg == "--" {
			literal = true
			continue
		}
		key, value, equals := strings.Cut(arg, "=")
		if !strings.HasPrefix(key, "-") {
			words = append(words, arg)
			continue
		}
		spec, known := optionDefinitions[key]
		if !known {
			fail(argumentError("Unknown option " + key + ". Use command --help for valid options."))
			continue
		}
		seen[key]++
		if spec.Value {
			if !equals {
				if i+1 >= len(args) {
					fail(argumentError(key + " requires a value."))
					continue
				}
				i++
				value = args[i]
			}
			values[key] = value
		} else if equals {
			fail(argumentError(key + " is a switch and takes no value."))
		}
		switch key {
		case "--verbose":
			out.Verbose = true
		case "--debug":
			out.Debug = true
		case "--quiet":
			out.Quiet = true
		case "--log-level":
			out.LogLevel = value
		case "--log-file":
			out.LogFile = value
		case "--json":
			out.Machine = true
		case "--help", "-h":
			if !equals {
				out.Help = true
			}
		case "--version":
			words = append(words, "version")
		case "--workspace":
			out.Root = value
		case "--request-id":
			out.RequestID = value
		case "--startup-id":
			out.StartupID = value
		default:
			options = append(options, key)
			if spec.Value {
				options = append(options, value)
			}
		}
	}
	if len(words) > len(literals) && words[0] == "help" {
		out.Help = true
		words = words[1:]
	}
	var selected *commandDefinition
	depth := 0
	for i := range commandDefinitions {
		c := &commandDefinitions[i]
		if len(c.Path) > len(words)-len(literals) {
			continue
		}
		match := true
		for j, s := range c.Path {
			if words[j] != s {
				match = false
				break
			}
		}
		if match && len(c.Path) > depth {
			selected = c
			depth = len(c.Path)
		}
	}
	group := strings.Join(words, " ")
	isGroup := len(words) == 0
	for _, c := range commandDefinitions {
		if strings.HasPrefix(strings.Join(c.Path, " "), group+" ") {
			isGroup = true
		}
	}
	if isGroup && selected == nil {
		out.Path = words
		if first != nil && !out.Help {
			return out, first
		}
		out.Help = true
		return out, nil
	}
	if selected == nil {
		return out, argumentError("Unknown command. Use insonic --help for available commands.")
	}
	out.Path = selected.Path
	if out.Help {
		return out, nil
	}
	if first != nil {
		return out, first
	}
	for key, count := range seen {
		repeat := key == "--known-speaker" || key == "--speaker" && strings.HasPrefix(strings.Join(out.Path, " "), "recordings roster ")
		if count > 1 && !repeat {
			return out, argumentError(key + " may be supplied only once.")
		}
	}
	if out.Quiet && (out.Verbose || out.Debug) {
		return out, argumentError("--quiet cannot be combined with --verbose or --debug.")
	}
	if out.LogLevel == "" {
		out.LogLevel = "info"
		if out.Debug {
			out.LogLevel = "debug"
		}
	}
	if !containsOption([]string{"error", "warn", "info", "debug"}, out.LogLevel) {
		return out, argumentError("--log-level accepts error|warn|info|debug.")
	}
	if seen["--log-file"] > 0 && out.LogFile == "" {
		return out, argumentError("--log-file requires a nonempty file path.")
	}
	if seen["--log-level"] > 0 && values["--log-level"] == "" {
		return out, argumentError("--log-level requires a value.")
	}
	if seen["--startup-id"] > 0 && (!contracts.ValidID(out.StartupID) || strings.Join(out.Path, " ") != "runtime serve") {
		return out, argumentError("--startup-id is reserved for runtime serve and requires a UUID.")
	}
	if seen["--request-id"] > 0 && !contracts.ValidID(out.RequestID) {
		return out, argumentError("--request-id requires a UUID.")
	}
	if seen["--workspace"] > 0 && out.Root == "" {
		return out, argumentError("--workspace requires a nonempty directory.")
	}
	for i := 0; i < len(options); i++ {
		key := options[i]
		spec := optionDefinitions[key]
		if !containsOption(selected.Options, key) {
			return out, argumentError(key + " is not accepted by " + strings.Join(out.Path, " ") + ".")
		}
		if spec.Value {
			i++
			value := options[i]
			if value == "" {
				return out, argumentError(key + " requires a nonempty value.")
			}
			if spec.Values != "" && !strings.Contains("|"+spec.Values+"|", "|"+value+"|") {
				return out, argumentError(key + " accepts " + spec.Values + ".")
			}
			if spec.Numeric {
				n, e := strconv.ParseInt(value, 10, 64)
				max := spec.Max
				if key == "--limit" && out.Path[0] == "search" {
					max = 500
				}
				if e != nil || n < spec.Min || n > max {
					return out, argumentError(fmt.Sprintf("%s requires an integer from %d to %d.", key, spec.Min, max))
				}
			}
		}
	}
	for _, set := range [][]string{{"--copy", "--reference"}, {"--transcript", "--subtitle"}, {"--originated-at", "--originated-on"}, {"--input", "--saved"}} {
		if seen[set[0]] > 0 && seen[set[1]] > 0 {
			return out, argumentError(strings.Join(set, " and ") + " are mutually exclusive.")
		}
	}
	if (seen["--originated-earliest"] > 0 || seen["--originated-latest"] > 0) && (seen["--originated-at"] > 0 || seen["--originated-on"] > 0) {
		return out, argumentError("Date bounds cannot be combined with --originated-at or --originated-on.")
	}
	operands := words[depth:]
	if len(operands) < selected.Min || len(operands) > selected.Max {
		return out, argumentError("Usage: insonic " + strings.Join(selected.Path, " ") + " " + selected.Operands + ". Use --help for details.")
	}
	for _, key := range strings.Fields("--after --media --credential-id --transcript-credential-id --saved") {
		if key == "--after" && out.Path[0] == "search" {
			if len(values[key]) > 2048 {
				return out, argumentError("--after requires an opaque cursor of at most 2048 bytes.")
			}
			continue
		}
		if seen[key] > 0 && !contracts.ValidID(values[key]) {
			return out, argumentError(key + " requires a UUID.")
		}
	}
	if seen["--speaker"] > 0 && !strings.HasPrefix(strings.Join(out.Path, " "), "recordings roster ") && !contracts.ValidID(values["--speaker"]) {
		return out, argumentError("--speaker requires a UUID.")
	}
	operandHints := strings.Fields(selected.Operands)
	for i, value := range operands {
		if i < len(operandHints) && strings.Contains(operandHints[i], "UUID") && !strings.Contains(operandHints[i], "_OR_") && !contracts.ValidID(value) {
			return out, argumentError(operandHints[i] + " requires a UUID.")
		}
	}
	if out.Path[0] == "credentials" && out.Path[1] == "select" && !containsOption([]string{"native", "vault", "session"}, operands[0]) {
		return out, argumentError("credentials select accepts native, vault or session.")
	}
	if out.Path[0] == "jobs" && out.Path[1] == "start" {
		n, e := strconv.ParseInt(operands[0], 10, 64)
		if e != nil || n < 1 || n > 600000 {
			return out, argumentError("jobs start requires milliseconds from 1 to 600000.")
		}
	}
	for _, key := range selected.Required {
		if seen[key] == 0 {
			return out, argumentError(key + " is required. Use command --help for its input contract.")
		}
	}
	if strings.Join(out.Path, " ") == "media import" && len(operands) == 0 && seen["--manifest"] == 0 {
		return out, argumentError("media import requires files or --manifest FILE.")
	}
	if strings.Join(out.Path, " ") == "transcript import" {
		if seen["--manifest"] > 0 {
			if len(operands) != 0 || seen["--record"] > 0 || seen["--legacy-sidecar"] > 0 {
				return out, argumentError("--manifest uses per-item recording targets and cannot be combined with a positional transcript, --record or --legacy-sidecar.")
			}
		} else if seen["--record"] == 0 || seen["--legacy-sidecar"] == 0 && len(operands) != 1 || seen["--legacy-sidecar"] > 0 && len(operands) != 0 {
			return out, argumentError("A single transcript requires PATH_OR_URL and --record UUID_OR_TITLE; --legacy-sidecar uses --record without a path. Use --manifest FILE for per-item targets.")
		}
	}
	out.Args = append([]string{}, selected.Path...)
	domain := out.Path[0] == "media" && len(out.Path) > 1 && !containsOption([]string{"tools", "capture", "playback", "playback-check", "playback-close"}, out.Path[1]) || out.Path[0] == "transcript" || out.Path[0] == "work" && out.Path[1] != "wait" || out.Path[0] == "models" && (len(out.Path) == 2 && out.Path[1] != "resolve" && out.Path[1] != "discover" && out.Path[1] != "train" && seen["--speaker"] == 0 || len(out.Path) > 2 && out.Path[1] == "base")
	if domain {
		out.Args = append(out.Args, options...)
		if len(literals) > 0 {
			normal := len(operands) - len(literals)
			out.Args = append(out.Args, operands[:normal]...)
			out.Args = append(out.Args, "--")
			out.Args = append(out.Args, literals...)
		} else {
			out.Args = append(out.Args, operands...)
		}
	} else {
		out.Args = append(out.Args, operands...)
		out.Args = append(out.Args, options...)
	}
	// Existing saved-query parser expects its selection before optional revision.
	if seen["--saved"] > 0 && seen["--revision"] > 0 {
		out.Args = append(append([]string{}, selected.Path...), "--saved", values["--saved"], "--revision", values["--revision"])
	}
	return out, nil
}

func commandHelp(path []string) string {
	prefix := strings.Join(path, " ")
	var b strings.Builder
	fmt.Fprintf(&b, "insonic %s\n\n", prefix)
	for _, c := range commandDefinitions {
		p := strings.Join(c.Path, " ")
		if p == prefix {
			fmt.Fprintf(&b, "Usage: insonic %s %s [--workspace DIRECTORY] [--request-id UUID] [--json]\n", p, c.Operands)
			for _, key := range c.Options {
				o := optionDefinitions[key]
				suffix := ""
				if o.Value {
					suffix = " VALUE"
				}
				if key == "--input" {
					suffix = " FILE (JSON file, not inline JSON)"
				}
				fmt.Fprintf(&b, "  %s%s", key, suffix)
				if o.Values != "" {
					fmt.Fprintf(&b, "; values: %s", o.Values)
				}
				if o.Numeric {
					max := o.Max
					if p == "search" && key == "--limit" {
						max = 500
					}
					fmt.Fprintf(&b, "; range: %d..%d", o.Min, max)
				}
				if containsOption(c.Required, key) {
					b.WriteString("; required")
				}
				if key == "--known-speaker" || key == "--speaker" && strings.HasPrefix(p, "recordings roster ") {
					b.WriteString("; repeatable")
				}
				b.WriteByte('\n')
			}
			if containsOption(c.Options, "--limit") {
				b.WriteString("Default page limit: 100. Omit --after for the initial page.\n")
			}
			if p == "search" {
				b.WriteString("Continue with the same search text and filters plus --after NEXT_CURSOR from the preceding page. A page can contain only relationships; follow next_cursor until absent.\n")
			}
			if p == "query native" {
				b.WriteString("Read-only subset: one statement, optional trailing semicolon and comments. Cypher starts MATCH, OPTIONAL MATCH, RETURN, WITH or UNWIND; SQL starts SELECT; EXPLAIN may prefix a read. No mutation, transaction control, procedures or PROFILE. Pure functions: COUNT, SUM, MIN, MAX, AVG, COALESCE, TOSTRING, TOINTEGER, LOWER, UPPER, ABS, SIZE, LENGTH, COLLECT. Parenthesized read expressions are accepted; function names must be unqualified. Other functions (including range) are rejected; use literal inputs or supported functions. Parameter JSON must have unique object members at every level.\n")
			}
			if p == "work wait" {
				b.WriteString("Default timeout: 30000 ms. A timeout leaves work running; use work show, wait or cancel.\n")
			}
			if strings.HasPrefix(p, "credentials ") {
				b.WriteString("Vault add/replace/delete/unlock/load use bounded protected stdin JSON; never pass secrets in arguments. Native/session delete needs no stdin.\n")
			}
			if strings.HasPrefix(p, "backup ") || strings.HasPrefix(p, "catalog ") {
				b.WriteString("Offline maintenance: restore/migrate requires exclusive ownership.\n")
			}
			if p == "runtime serve" {
				b.WriteString("Internal shared-runtime host. Noninteractive; normally started by clients.\n")
			}
			if strings.HasPrefix(p, "setup ") || p == "settings setup" {
				b.WriteString("Optional setup metadata never gates ordinary commands. Save with the displayed revision; preserve the request UUID and immutable input when retrying an unknown outcome. Reopen with setup show or settings setup. Readiness is a separate current task probe.\n")
			}
			operation := strings.ReplaceAll(p, " ", ".")
			if strings.HasPrefix(p, "models base ") {
				operation = "models." + c.Path[2]
			}
			if strings.HasSuffix(operation, ".download") {
				operation = strings.TrimSuffix(operation, ".download") + ".acquire"
			}
			if p == "transcript import" {
				operation = "media.import"
				b.WriteString("Single transcript: PATH_OR_URL --record UUID_OR_TITLE. Batch: --manifest FILE uses per-item recording targets without --record, a positional path or --legacy-sidecar.\n")
			}
			if containsOption(c.Options, "--input") {
				template, schema := schemas.InputHelp(operation)
				if template != nil {
					fmt.Fprintf(&b, "JSON file template (replace illustrative IDs/revisions with current records):\n%s\nInput contract, including optional fields, defaults and enums:\n%s\n", template, schema)
				}
			}
			if p == "credentials unlock" || p == "credentials add" || p == "credentials replace" || p == "credentials delete" {
				b.WriteString("Protected stdin template for vault: {\"passphrase\":\"...\"}; add/replace also require \"value\":\"...\".\n")
			}
			b.WriteString("Diagnostic controls: --verbose adds request/workspace identifiers; --debug adds generated progress detail; --quiet suppresses informational human output and progress. Warnings/errors remain visible. --log-level error|warn|info|debug selects diagnostic severity (default info, debug with --debug). --log-file FILE elects private rotating JSONL logs (1 MiB plus two predecessors); its directory must exist. No arguments, input documents, credentials, endpoints or environment values are logged. These controls do not change --json result envelopes.\n")
			b.WriteString("Global options may precede or follow command options. Scalar options appear once; value options accept --name=value. -- ends option and help recognition. Machine success uses stdout; machine failure uses stderr.\n")
			example := c.Operands
			if p == "transcript import" {
				example = "--manifest transcripts.json"
			}
			for _, key := range c.Required {
				value := "input.json"
				if key == "--revision" || key == "--expected-revision" {
					value = "1"
				}
				if key == "--record" {
					value = "11111111-1111-4111-8111-111111111111"
				}
				example += " " + key + " " + value
			}
			fmt.Fprintf(&b, "Example: insonic %s %s\n", p, example)
			return b.String()
		}
	}
	b.WriteString("Commands:\n")
	var paths []string
	for _, c := range commandDefinitions {
		p := strings.Join(c.Path, " ")
		if prefix == "" || strings.HasPrefix(p, prefix+" ") {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintf(&b, "  %s\n", p)
	}
	b.WriteString("Use help COMMAND, COMMAND --help or -h for syntax, defaults and input contracts.\nAliases: --version/version; models base; models download/acquire; models list --speaker/models speaker list; search/query run; query native/query run.\n")
	return b.String()
}
