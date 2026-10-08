// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestDesktopOperationSchemas(t *testing.T) {
	for _, test := range []struct {
		op   string
		item bool
		data string
	}{
		{"settings.show", false, ""},
		{"settings.set", false, `{"section":"appearance","revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","value":{"theme":"system","reduced_motion":false}}`},
		{"recordings.cues", true, `{"limit":100,"revision":1}`},
		{"media.playback", true, `{"revision":1}`},
		{"media.capture", true, `{"section":"metadata","limit":65536}`},
		{"media.playback-check", true, `{"playback_id":"11111111-1111-4111-8111-111111111111"}`},
		{"media.playback-close", true, `{"playback_id":"11111111-1111-4111-8111-111111111111"}`},
	} {
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: test.op, Data: json.RawMessage(test.data)}
		if test.item {
			req.ItemID = contracts.ID()
		}
		raw, _ := json.Marshal(req)
		if e := ValidateRequest(raw); e != nil {
			t.Fatal(test.op, e)
		}
		req.SourcePath = "arbitrary"
		raw, _ = json.Marshal(req)
		if ValidateRequest(raw) == nil {
			t.Fatal(test.op, "unrelated path accepted")
		}
	}
}
