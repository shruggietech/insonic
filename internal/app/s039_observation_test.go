// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestS039UnobservedFailuresOmitZeroReceipts(t *testing.T) {
	a := configuredApp(t)
	for _, op := range []string{"work.show", "media.show", "pipelines.show", "speakers.show", "terms.show", "artifacts.show", "artifacts.materialize", "artifacts.lease-renew", "jobs.show", "models.alias.show", "models.source.show", "graph.layout-show"} {
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op, ItemID: contracts.ID()}
		if strings.HasPrefix(op, "artifacts.") {
			req.PublicationID = req.ItemID
			req.ItemID = ""
			if op == "artifacts.lease-renew" {
				req.LeaseID = contracts.ID()
			}
		}
		if op == "jobs.show" {
			req.JobID = req.ItemID
			req.ItemID = ""
		}
		response := a.Dispatch(req)
		raw, _ := json.Marshal(response)
		if response.Error == nil || response.Result != nil || strings.Contains(string(raw), "\"result\"") {
			t.Errorf("%s invented a result: %s", op, raw)
		}
	}
}
