// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

type rosterMutation struct {
	ExpectedRevision *int64   `json:"expected_revision"`
	Speakers         []string `json:"speakers"`
}

func (a *App) rosterOperation(req contracts.Request) (any, error) {
	mode := strings.TrimPrefix(req.Operation, "recordings.roster.")
	if mode == "show" {
		return a.Catalog.Roster(a.ctx, req.ItemID)
	}
	var input rosterMutation
	if strictPayload(req.Data, &input) != nil || input.ExpectedRevision == nil || *input.ExpectedRevision < 0 || input.Speakers == nil {
		return nil, contracts.Fail("invalid_request")
	}
	return a.Catalog.MutateRoster(a.ctx, req.RequestID, req.ItemID, *input.ExpectedRevision, mode, input.Speakers)
}
