// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
)

func (a *App) catalogQuery(q contracts.QueryInput, cause error) (any, error) {
	typed, ok := cause.(*contracts.Error)
	if !ok || (typed.Code != "unavailable" && typed.Code != "operation_failed" && typed.Code != "unsupported_capability" && typed.Code != "input_limit" && typed.Code != "output_limit") {
		return nil, cause
	}
	snap, e := a.Catalog.Export(a.ctx)
	if e != nil {
		return nil, e
	}
	c, e := explore.Build(snap)
	if e != nil {
		return nil, e
	}
	out, e := explore.Query(c, c.Refs, q)
	if e != nil {
		return nil, e
	}
	out.Basis = "current-catalog-fallback"
	out.GraphError = typed.Code
	return out, nil
}
