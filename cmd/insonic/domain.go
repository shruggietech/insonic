// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
)

type domainFlags struct {
	options                                         library.Options
	manifest, subtitle, title, credential, adapter  string
	record, transcriptCredential, transcriptAdapter string
	legacySidecar                                   bool
	revision                                        int64
	after                                           string
	afterOrdinal                                    int64
	limit                                           int
	newEntry                                        bool
	seen                                            map[string]bool
}

func parseFlags(args []string) ([]string, domainFlags, error) {
	f := domainFlags{seen: map[string]bool{}}
	positional := []string{}
	literal := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" && !literal {
			literal = true
			continue
		}
		if literal || !strings.HasPrefix(arg, "--") {
			positional = append(positional, arg)
			continue
		}
		key := arg
		if arg == "--transcript" || arg == "--subtitle" {
			key = "transcript-choice"
		}
		if arg == "--copy" || arg == "--reference" {
			key = "copy-choice"
		}
		if f.seen[key] && key != "--known-speaker" {
			return nil, f, contracts.Fail("invalid_request")
		}
		f.seen[key] = true
		switch arg {
		case "--replace-audio":
			yes := true
			f.options.ReplaceAudio = &yes
			continue
		case "--transcript-applies":
			yes := true
			f.options.TranscriptApplies = &yes
			continue
		case "--replace-transcript":
			yes := true
			f.options.ReplaceTranscript = &yes
			continue
		case "--legacy-sidecar":
			f.legacySidecar = true
			continue
		case "--copy", "--reference":
			copy := arg == "--copy"
			f.options.Copy = &copy
			continue
		case "--new-entry":
			f.newEntry = true
			continue
		case "--local-http":
			allow := true
			f.options.LocalHTTP = &allow
			continue
		}
		i++
		if i >= len(args) {
			return nil, f, contracts.Fail("invalid_request")
		}
		value := args[i]
		if value == "" {
			return nil, f, contracts.Fail("invalid_request")
		}
		switch arg {
		case "--known-speaker":
			f.options.KnownSpeakers = append(f.options.KnownSpeakers, value)
		case "--existing-transcript":
			f.options.ExistingTranscript = value
		case "--existing-roster":
			f.options.ExistingRoster = value
		case "--manifest":
			f.manifest = value
		case "--subtitle", "--transcript":
			f.subtitle = value
		case "--record":
			f.record = value
		case "--transcript-credential-id":
			if !contracts.ValidID(value) {
				return nil, f, contracts.Fail("invalid_request")
			}
			f.transcriptCredential = value
		case "--transcript-adapter":
			f.transcriptAdapter = value
		case "--attribution":
			f.options.Attribution = value
		case "--transcript-format":
			f.options.TranscriptFormat = value
		case "--subtitle-language":
			f.options.SubtitleLanguage = value
		case "--diarization-model-id":
			f.options.DiarizationModelID = value
		case "--subtitle-stream-index":
			n, e := strconv.Atoi(value)
			if e != nil || n < 0 {
				return nil, f, contracts.Fail("invalid_request")
			}
			f.options.SubtitleStreamIndex = &n
		case "--transcript-max-bytes", "--transcript-timeout-ms":
			n, e := strconv.ParseInt(value, 10, 64)
			if e != nil || n < 1 {
				return nil, f, contracts.Fail("invalid_request")
			}
			if arg == "--transcript-max-bytes" {
				f.options.TranscriptMaxBytes = &n
			} else {
				f.options.TranscriptTimeoutMS = &n
			}
		case "--title":
			f.title = value
		case "--credential-id":
			if !contracts.ValidID(value) {
				return nil, f, contracts.Fail("invalid_request")
			}
			f.credential = value
		case "--acquisition-adapter":
			f.adapter = value
		case "--acquisition-max-bytes", "--acquisition-timeout-ms":
			n, e := strconv.ParseInt(value, 10, 64)
			if e != nil || n <= 0 {
				return nil, f, contracts.Fail("invalid_request")
			}
			if arg == "--acquisition-max-bytes" {
				f.options.AcquisitionMaxBytes = &n
			} else {
				f.options.AcquisitionTimeoutMS = &n
			}
		case "--originated-at":
			f.options.OriginatedAt = value
		case "--originated-on":
			f.options.OriginatedOn = value
		case "--originated-earliest", "--originated-latest":
			instant, e := parseBound(value)
			if e != nil {
				return nil, f, e
			}
			if arg == "--originated-earliest" {
				f.options.OriginatedEarliest = instant
			} else {
				f.options.OriginatedLatest = instant
			}
		case "--timezone":
			f.options.Timezone = value
		case "--dst-fold":
			f.options.DSTFold = value
		case "--dst-gap":
			f.options.DSTGap = value
		case "--date-precedence":
			f.options.DatePrecedence = value
		case "--preset":
			f.options.Preset = value
		case "--revision":
			var e error
			f.revision, e = strconv.ParseInt(value, 10, 64)
			if e != nil || f.revision < 1 {
				return nil, f, contracts.Fail("invalid_request")
			}
		case "--after":
			if !contracts.ValidID(value) {
				return nil, f, contracts.Fail("invalid_request")
			}
			f.after = value
		case "--after-ordinal":
			var e error
			f.afterOrdinal, e = strconv.ParseInt(value, 10, 64)
			if e != nil || f.afterOrdinal < 0 {
				return nil, f, contracts.Fail("invalid_request")
			}
		case "--limit":
			var e error
			f.limit, e = strconv.Atoi(value)
			if e != nil || f.limit < 1 || f.limit > 100 {
				return nil, f, contracts.Fail("invalid_request")
			}
		default:
			return nil, f, contracts.Fail("invalid_request")
		}
	}
	if f.options.OriginatedAt != "" && f.options.OriginatedOn != "" {
		return nil, f, contracts.Fail("invalid_request")
	}
	if (f.options.OriginatedEarliest != nil || f.options.OriginatedLatest != nil) && (f.options.OriginatedAt != "" || f.options.OriginatedOn != "") {
		return nil, f, contracts.Fail("invalid_request")
	}
	return positional, f, nil
}
func flagsAllowed(f domainFlags, allowed ...string) bool {
	valid := map[string]bool{}
	for _, key := range allowed {
		valid[key] = true
	}
	for key := range f.seen {
		if !valid[key] {
			return false
		}
	}
	return true
}

var dateFlags = []string{"--originated-at", "--originated-on", "--originated-earliest", "--originated-latest", "--timezone", "--dst-fold", "--dst-gap", "--date-precedence"}

func parseBound(value string) (*catalog.Instant, error) {
	t, e := time.Parse(time.RFC3339Nano, value)
	if e != nil || t.Before(time.Unix(0, math.MinInt64)) || t.After(time.Unix(0, math.MaxInt64)) {
		return nil, contracts.Fail("invalid_request")
	}
	return &catalog.Instant{ISO: t.UTC().Format(time.RFC3339Nano), UnixNS: t.UnixNano()}, nil
}

func absoluteSource(source string) (string, error) {
	u, e := url.Parse(source)
	if e == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return source, nil
	}
	if e == nil && len(u.Scheme) > 1 {
		return "", contracts.Fail("invalid_request")
	}
	absolute, e := filepath.Abs(source)
	if e != nil {
		return "", contracts.Fail("invalid_request")
	}
	return absolute, nil
}
func payload(value any) (json.RawMessage, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return raw, nil
}

func parseDomain(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	group := args[0]
	if group == "models" && args[1] == "base" {
		args = append([]string{"models"}, args[2:]...)
		if len(args) < 2 {
			return fail()
		}
	}
	command := args[1]
	if group == "models" && command == "download" {
		command = "acquire"
	}
	positional, f, e := parseFlags(args[2:])
	if e != nil {
		return "", "", nil, e
	}
	operation := group + "." + command
	if group == "transcript" && command == "import" {
		if !flagsAllowed(f, transcriptFlags...) {
			return fail()
		}
		if f.manifest != "" {
			if len(positional) != 0 || f.record != "" || f.legacySidecar {
				return fail()
			}
			request, err := library.ReadManifest(f.manifest)
			if err != nil {
				return "", "", nil, err
			}
			overrideOptions(&request.Defaults, f.options)
			for i := range request.Items {
				item := &request.Items[i]
				if item.TranscriptCredentialID == "" {
					item.TranscriptCredentialID = f.transcriptCredential
				}
				if item.TranscriptAdapter == "" {
					item.TranscriptAdapter = f.transcriptAdapter
				}
				if item.Record == "" {
					return fail()
				}
			}
			data, err := payload(request)
			return "media.import", "", data, err
		}
		if f.record == "" || (!f.legacySidecar && len(positional) != 1) || (f.legacySidecar && len(positional) != 0) || !flagsAllowed(f, transcriptFlags...) {
			return fail()
		}
		source := "<managed>"
		if !f.legacySidecar {
			source, e = absoluteSource(positional[0])
			if e != nil {
				return fail()
			}
		}
		request, err := library.PrepareImport(library.ImportRequest{Defaults: f.options, Items: []library.Item{{Record: f.record, Transcript: source, TranscriptCredentialID: f.transcriptCredential, TranscriptAdapter: f.transcriptAdapter}}})
		if err != nil {
			return "", "", nil, err
		}
		data, err := payload(request)
		return "media.import", "", data, err
	}
	if (group == "media" || group == "models" || group == "work") && command == "list" && len(positional) == 0 && flagsAllowed(f, "--after", "--limit") {
		if len(f.seen) == 0 {
			return operation, "", nil, nil
		}
		data, e := payload(struct {
			AfterID string `json:"after_id,omitempty"`
			Limit   int    `json:"limit,omitempty"`
		}{f.after, f.limit})
		return operation, "", data, e
	}
	if group == "work" {
		if command == "results" && len(positional) == 1 && contracts.ValidID(positional[0]) && flagsAllowed(f, "--after-ordinal", "--limit") {
			data, e := payload(struct {
				AfterOrdinal int64 `json:"after_ordinal"`
				Limit        int   `json:"limit,omitempty"`
			}{f.afterOrdinal, f.limit})
			return operation, positional[0], data, e
		}
		if len(f.seen) > 0 {
			return fail()
		}
		if command == "list" && len(positional) == 0 {
			return operation, "", nil, nil
		}
		if (command == "show" || command == "cancel" || command == "retry") && len(positional) == 1 && contracts.ValidID(positional[0]) {
			return operation, positional[0], nil, nil
		}
		return fail()
	}
	if group == "models" {
		if len(f.seen) > 0 {
			return fail()
		}
		if command == "list" && len(positional) == 0 {
			return operation, "", nil, nil
		}
		if (command == "show" || command == "verify" || command == "materialize") && len(positional) == 1 && contracts.ValidID(positional[0]) {
			return operation, positional[0], nil, nil
		}
		if (command == "register" || command == "acquire") && len(positional) == 1 {
			file, e := os.Open(positional[0])
			if e != nil {
				return "", "", nil, contracts.Fail("unavailable")
			}
			defer file.Close()
			raw, e := io.ReadAll(io.LimitReader(file, (1<<20)+1))
			if e != nil || len(raw) > 1<<20 {
				return "", "", nil, contracts.Fail("output_limit")
			}
			var manifest models.Manifest
			if e = models.DecodeManifest(raw, &manifest); e != nil {
				return "", "", nil, e
			}
			data, e := payload(models.Request{Manifest: manifest})
			return operation, "", data, e
		}
		return fail()
	}
	if group != "media" {
		return fail()
	}
	if command == "list" && len(positional) == 0 && len(f.seen) == 0 {
		return operation, "", nil, nil
	}
	if (command == "show" || command == "metadata" || command == "raw") && len(positional) == 1 && contracts.ValidID(positional[0]) && len(f.seen) == 0 {
		return operation, positional[0], nil, nil
	}
	if command == "import" {
		allowed := append(append([]string{}, dateFlags...), "copy-choice", "--manifest", "transcript-choice", "--title", "--credential-id", "--acquisition-adapter", "--local-http", "--acquisition-max-bytes", "--acquisition-timeout-ms", "--new-entry", "--preset")
		allowed = append(allowed, transcriptFlags...)
		allowed = append(allowed, "--known-speaker", "--replace-audio", "--existing-transcript", "--transcript-applies", "--existing-roster")
		if !flagsAllowed(f, allowed...) || f.manifest != "" && len(positional) > 0 || f.manifest == "" && len(positional) == 0 {
			return fail()
		}
		req := library.ImportRequest{Kind: "import-manifest", Version: contracts.Version}
		if f.manifest != "" {
			req, e = library.ReadManifest(f.manifest)
			if e != nil {
				return "", "", nil, e
			}
		} else {
			for _, source := range positional {
				absolute, e := absoluteSource(source)
				if e != nil {
					return "", "", nil, e
				}
				req.Items = append(req.Items, library.Item{Source: absolute})
			}
		}
		overrideOptions(&req.Defaults, f.options)
		subtitle := f.subtitle
		if subtitle != "" {
			subtitle, e = absoluteSource(subtitle)
			if e != nil {
				return fail()
			}
		}
		for i := range req.Items {
			item := &req.Items[i]
			if subtitle != "" && item.Subtitle == "" && item.Transcript == "" {
				item.Transcript = subtitle
			}
			if item.TranscriptCredentialID == "" {
				item.TranscriptCredentialID = f.transcriptCredential
			}
			if item.TranscriptAdapter == "" {
				item.TranscriptAdapter = f.transcriptAdapter
			}
			if f.record != "" {
				if len(req.Items) != 1 {
					return fail()
				}
				item.Record = f.record
			}
			if item.Kind == "" && item.Source != "" {
				item.Kind = "media"
			}
			if f.legacySidecar {
				return fail()
			}
			if f.title != "" && item.Title == "" {
				item.Title = f.title
			}
			if f.credential != "" && item.CredentialID == "" {
				item.CredentialID = f.credential
			}
			if f.adapter != "" && item.AcquisitionAdapter == "" {
				item.AcquisitionAdapter = f.adapter
			}
			if f.newEntry {
				item.NewEntry = true
			}
		}
		req, e = library.PrepareImport(req)
		if e != nil {
			return "", "", nil, e
		}
		data, e := payload(req)
		return operation, "", data, e
	}
	if len(positional) == 0 || !contracts.ValidID(positional[0]) {
		return fail()
	}
	id := positional[0]
	if command == "refresh" && len(positional) == 1 && flagsAllowed(f, append(append([]string{}, dateFlags...), "--preset")...) {
		// Validate effective option enums using the same shared import contract.
		if _, e = library.PrepareImport(library.ImportRequest{Defaults: f.options, Items: []library.Item{{Source: "validation"}}}); e != nil {
			return "", "", nil, e
		}
		data, e := payload(f.options)
		return operation, id, data, e
	}
	if command == "set-origin" && len(positional) == 1 && f.revision > 0 && flagsAllowed(f, append(append([]string{}, dateFlags...), "--revision")...) && (f.options.OriginatedAt != "" || f.options.OriginatedOn != "" || f.options.OriginatedEarliest != nil || f.options.OriginatedLatest != nil) {
		if _, e = library.PrepareImport(library.ImportRequest{Defaults: f.options, Items: []library.Item{{Source: "validation"}}}); e != nil {
			return "", "", nil, e
		}
		data, e := payload(struct {
			Revision int64           `json:"revision"`
			Options  library.Options `json:"options"`
		}{f.revision, f.options})
		return operation, id, data, e
	}
	if command == "relocate" && len(positional) == 2 && f.revision > 0 && flagsAllowed(f, "--revision") {
		absolute, e := filepath.Abs(positional[1])
		if e != nil {
			return fail()
		}
		data, e := payload(struct {
			Revision int64  `json:"revision"`
			Path     string `json:"path"`
		}{f.revision, absolute})
		return operation, id, data, e
	}
	return fail()
}
func overrideOptions(target *library.Options, source library.Options) {
	*target = library.MergeOptions(*target, source)
}

var transcriptFlags = []string{"--record", "--legacy-sidecar", "--replace-transcript", "--attribution", "--transcript-format", "--transcript-credential-id", "--transcript-adapter", "--transcript-max-bytes", "--transcript-timeout-ms", "--subtitle-stream-index", "--subtitle-language", "--diarization-model-id", "--local-http", "--manifest"}
