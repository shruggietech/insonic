// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func strict(data []byte, value any) error {
	if catalog.ValidateJSON(data) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	d.UseNumber()
	if d.Decode(value) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func ReadManifest(path string) (ImportRequest, error) {
	var r ImportRequest
	absolute, e := filepath.Abs(path)
	if e != nil {
		return r, contracts.Fail("invalid_request")
	}
	f, e := os.Open(absolute)
	if e != nil {
		return r, contracts.Fail("unavailable")
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if e != nil || len(data) > 8<<20 {
		return r, contracts.Fail("output_limit")
	}
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		r, e = readCSV(data)
	} else {
		e = strict(data, &r)
		if e == nil && (r.Kind != "import-manifest" || r.Version != contracts.Version) {
			e = contracts.Fail("invalid_request")
		}
	}
	if e != nil {
		return r, e
	}
	for i := range r.Items {
		if r.Items[i].Source != "" {
			r.Items[i].Source = resolvePath(filepath.Dir(absolute), r.Items[i].Source)
		}
		if r.Items[i].Transcript != "" {
			r.Items[i].Transcript = resolvePath(filepath.Dir(absolute), r.Items[i].Transcript)
		}
		if r.Items[i].Subtitle != "" {
			r.Items[i].Subtitle = resolvePath(filepath.Dir(absolute), r.Items[i].Subtitle)
		}
	}
	return PrepareImport(r)
}
func resolvePath(base, source string) string {
	if hasRemoteScheme(source) {
		return source
	}
	if filepath.IsAbs(source) {
		return source
	}
	return filepath.Join(base, source)
}
func readCSV(data []byte) (ImportRequest, error) {
	r := ImportRequest{Kind: "import-manifest", Version: contracts.Version}
	reader := csv.NewReader(bytes.NewReader(data))
	headers, e := reader.Read()
	if e != nil {
		return r, contracts.Fail("invalid_request")
	}
	allowed := map[string]bool{"source": true, "subtitle": true, "title": true, "copy": true, "originated_at": true, "originated_on": true, "originated_earliest": true, "originated_latest": true, "timezone": true, "dst_fold": true, "dst_gap": true, "preset": true, "date_precedence": true, "credential_id": true, "acquisition_adapter": true, "new_entry": true}
	for _, field := range []string{"local_http", "acquisition_max_bytes", "acquisition_timeout_ms", "transcript", "record", "transcript_credential_id", "transcript_adapter", "transcript_max_bytes", "transcript_timeout_ms", "replace_transcript", "attribution", "transcript_format", "subtitle_stream_index", "subtitle_language", "diarization_model_id", "expected_revision", "expected_recording_revision", "kind", "replace_audio", "existing_transcript", "transcript_applies", "existing_roster", "known_speakers", "expected_roster_revision"} {
		allowed[field] = true
	}
	seen := map[string]bool{}
	for _, h := range headers {
		if !allowed[h] || seen[h] {
			return r, contracts.Fail("invalid_request")
		}
		seen[h] = true
	}
	if !seen["source"] && !(seen["record"] && (seen["transcript"] || seen["subtitle"])) {
		return r, contracts.Fail("invalid_request")
	}
	for {
		row, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return r, contracts.Fail("invalid_request")
		}
		doc := map[string]any{}
		for i, value := range row {
			if value == "" {
				continue
			}
			h := headers[i]
			if h == "copy" || h == "new_entry" || h == "local_http" || h == "replace_transcript" || h == "replace_audio" || h == "transcript_applies" {
				v, e := strconv.ParseBool(value)
				if e != nil {
					return r, contracts.Fail("invalid_request")
				}
				doc[h] = v
			} else if h == "acquisition_max_bytes" || h == "acquisition_timeout_ms" || h == "transcript_max_bytes" || h == "transcript_timeout_ms" || h == "subtitle_stream_index" || h == "expected_revision" || h == "expected_recording_revision" || h == "expected_roster_revision" {
				v, e := strconv.ParseInt(value, 10, 64)
				if e != nil {
					return r, contracts.Fail("invalid_request")
				}
				doc[h] = v
			} else if h == "known_speakers" {
				var refs []string
				if json.Unmarshal([]byte(value), &refs) != nil || refs == nil {
					return r, contracts.Fail("invalid_request")
				}
				doc[h] = refs
			} else if h == "originated_earliest" || h == "originated_latest" {
				bound, e := BoundInstant(value)
				if e != nil {
					return r, contracts.Fail("invalid_request")
				}
				doc[h] = bound
			} else {
				doc[h] = value
			}
		}
		var item Item
		if strict(marshal(doc), &item) != nil {
			return r, contracts.Fail("invalid_request")
		}
		r.Items = append(r.Items, item)
		if len(r.Items) > 10000 {
			return r, contracts.Fail("output_limit")
		}
	}
	return r, nil
}
