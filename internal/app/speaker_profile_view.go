// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
)

// Profile election and current evidence usability are separate observations.
func (a *App) speakerProfileView(p catalog.SpeakerProfile) map[string]any {
	v := map[string]any{"id": p.ID, "revision": p.Revision, "version_id": p.VersionID, "state": p.State, "usable": false, "usability": "no-active-profile"}
	if p.State != "active" {
		return v
	}
	v["usability"] = "output-unavailable"
	output, err := a.Catalog.SpeakerOutput(a.ctx, p.VersionID)
	if err != nil {
		return v
	}
	var m struct {
		State string `json:"state"`
	}
	if json.Unmarshal(output.Metadata, &m) != nil {
		return v
	}
	v["usability"] = "evidence-invalidated"
	if m.State == "current" && output.Kind != "voice-embedding" {
		v["usability"] = "output-kind-not-supported-by-matcher"
	}
	if m.State == "current" && output.Kind == "voice-embedding" {
		v["usable"] = true
		v["usability"] = "current-independent-evidence"
	}
	return v
}
