//go:build system_ladybug

// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"encoding/json"
	lbug "github.com/LadybugDB/go-ladybug"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"time"
)

type ladybugEngine struct{ db *lbug.Database }
type ladybugSession struct {
	conn  *lbug.Connection
	ended bool
}

func OpenLadybug(path string) (Adapter, error) {
	if path == "" {
		return nil, contracts.Fail("invalid_request")
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	config := lbug.DefaultSystemConfig()
	config.BufferPoolSize = 128 << 20
	config.MaxNumThreads = 2
	db, e := lbug.OpenDatabase(path, config)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	return &adapter{engine: &ladybugEngine{db: db}, backend: "ladybugdb", version: ladybugVersion()}, nil
}
func (e *ladybugEngine) Close() { e.db.Close() }
func (e *ladybugEngine) Begin(ctx context.Context, read bool) (session, error) {
	c, err := lbug.OpenConnection(e.db)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	s := &ladybugSession{conn: c}
	q := "BEGIN TRANSACTION"
	if read {
		q += " READ ONLY"
	}
	if _, err = s.Query(ctx, true, q, nil); err != nil {
		c.Close()
		return nil, err
	}
	return s, nil
}
func (s *ladybugSession) Query(ctx context.Context, _ bool, q string, p map[string]any) (Rows, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	timeout := 20 * time.Second
	if end, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(end))
	}
	if timeout <= 0 {
		return nil, contracts.Fail("cancelled")
	}
	s.conn.SetTimeout(uint64(max(1, timeout.Milliseconds())))
	done := make(chan struct{})
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		select {
		case <-ctx.Done():
			s.conn.Interrupt()
		case <-done:
		}
	}()
	defer func() { close(done); <-joined }()
	var r *lbug.QueryResult
	var err error
	if len(p) == 0 {
		r, err = s.conn.Query(q)
	} else {
		stmt, e := s.conn.Prepare(q)
		if e != nil {
			return nil, contracts.Fail("operation_failed")
		}
		defer stmt.Close()
		r, err = s.conn.Execute(stmt, p)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, contracts.Fail("cancelled")
		}
		return nil, contracts.Fail("operation_failed")
	}
	defer r.Close()
	columns := r.GetColumnNames()
	out := Rows{}
	size := 0
	for r.HasNext() {
		if len(out) >= 200000 {
			return nil, contracts.Fail("output_limit")
		}
		tuple, e := r.Next()
		if e != nil {
			return nil, contracts.Fail("operation_failed")
		}
		row := map[string]any{}
		for i, k := range columns {
			v, e := tuple.GetValue(uint64(i))
			if e != nil {
				tuple.Close()
				return nil, contracts.Fail("operation_failed")
			}
			row[k] = v
		}
		tuple.Close()
		raw, e := json.Marshal(row)
		if e != nil {
			return nil, contracts.Fail("operation_failed")
		}
		size += len(raw)
		if size > 8<<20 {
			return nil, contracts.Fail("output_limit")
		}
		out = append(out, row)
	}
	return out, nil
}
func (s *ladybugSession) Commit(ctx context.Context) error {
	_, e := s.Query(ctx, true, "COMMIT", nil)
	if e == nil {
		s.ended = true
		s.conn.Close()
	}
	return e
}
func (s *ladybugSession) Rollback(ctx context.Context) {
	if s.ended {
		return
	}
	if ctx.Err() != nil {
		ctx = context.Background()
	}
	s.Query(ctx, true, "ROLLBACK", nil)
	s.ended = true
	s.conn.Close()
}
