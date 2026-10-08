// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
)

type failureEngine struct {
	engine
	uncertain bool
	failFacts bool
}

func (e *failureEngine) Begin(ctx context.Context, read bool) (session, error) {
	s, err := e.engine.Begin(ctx, read)
	if err != nil {
		return nil, err
	}
	return failureSession{s, e}, nil
}

type failureSession struct {
	session
	e *failureEngine
}

func (s failureSession) Query(ctx context.Context, write bool, q string, p map[string]any) (Rows, error) {
	if write && s.e.failFacts && p["entity_id"] != nil {
		return nil, contracts.Fail("operation_failed")
	}
	return s.session.Query(ctx, write, q, p)
}
func (s failureSession) Commit(ctx context.Context) error {
	e := s.session.Commit(ctx)
	if e == nil && s.e.uncertain {
		s.e.uncertain = false
		return contracts.Fail("operation_failed")
	}
	return e
}
