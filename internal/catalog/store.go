// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const SchemaVersion = 2

type Store struct {
	db        *sql.DB
	workspace string
	backend   string
	schema    string
}

func (s *Store) Close() error    { return s.db.Close() }
func (s *Store) Backend() string { return s.backend }
func (s *Store) query(q string) string {
	if s.backend != "postgresql" {
		return q
	}
	n := 0
	var b strings.Builder
	for _, c := range q {
		if c == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
func (s *Store) exec(ctx context.Context, tx *sql.Tx, q string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, s.query(q), args...)
}
func (s *Store) row(ctx context.Context, tx *sql.Tx, q string, args ...any) *sql.Row {
	return tx.QueryRowContext(ctx, s.query(q), args...)
}

func OpenSQLite(ctx context.Context, path, id string) (*Store, error) {
	if !contracts.ValidID(id) || path == "" {
		return nil, contracts.Fail("invalid_request")
	}
	path, e := filepath.Abs(path)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	// Create privately before SQLite opens it; reject links/nonregular selected files.
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e == nil {
		f.Close()
	} else if !os.IsExist(e) {
		return nil, contracts.Fail("unavailable")
	}
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() {
		return nil, contracts.Fail("invalid_request")
	}
	u := sqliteURL(path)
	q := u.Query()
	q.Set("_foreign_keys", "on")
	q.Set("_journal_mode", "WAL")
	q.Set("_synchronous", "FULL")
	q.Set("_txlock", "immediate")
	q.Set("_busy_timeout", "5000")
	u.RawQuery = q.Encode()
	db, e := sql.Open("sqlite3", u.String())
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	db.SetMaxOpenConns(4)
	s := &Store{db: db, workspace: id, backend: "sqlite"}
	if e = s.migrate(ctx); e != nil {
		db.Close()
		return nil, e
	}
	if e = s.ensureWorkspace(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func sqliteURL(path string) url.URL {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return url.URL{Scheme: "file", Path: p}
}

type PostgreSQLConfig struct {
	Host         string `json:"host"`
	Port         uint16 `json:"port"`
	Database     string `json:"database"`
	Schema       string `json:"schema"`
	TLSMode      string `json:"tls_mode"`
	CAFile       string `json:"ca_file"`
	CredentialID string `json:"credential_id"`
}
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

func OpenPostgreSQL(ctx context.Context, c PostgreSQLConfig, id string, secrets contracts.SecretProvider, migrate bool) (*Store, error) {
	if !contracts.ValidID(id) || !identifier.MatchString(c.Schema) || c.Host == "" || c.Database == "" || c.Port == 0 {
		return nil, contracts.Fail("invalid_request")
	}
	// ParseConfig is required by pgx. Explicit assignments remove ambient routing/auth.
	cfg, e := pgx.ParseConfig("postgres://localhost/placeholder?sslmode=disable")
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	cfg.Host = c.Host
	cfg.Port = c.Port
	cfg.Database = c.Database
	cfg.User = ""
	cfg.Password = ""
	cfg.Fallbacks = nil
	cfg.RuntimeParams = map[string]string{"search_path": "\"" + c.Schema + "\",pg_catalog"}
	cfg.ConnectTimeout = 5 * time.Second
	cfg.TLSConfig = nil
	cfg.LookupFunc = net.DefaultResolver.LookupHost
	if c.CredentialID != "" {
		if secrets == nil {
			return nil, contracts.Fail("unavailable")
		}
		raw, e := secrets.Resolve(ctx, c.CredentialID)
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
		var auth Credentials
		e = strict(raw, &auth)
		for i := range raw {
			raw[i] = 0
		}
		if e != nil || auth.Username == "" {
			return nil, contracts.Fail("invalid_request")
		}
		cfg.User = auth.Username
		cfg.Password = auth.Password
	}
	if cfg.User == "" {
		cfg.User = "insonic"
	}
	switch c.TLSMode {
	case "", "verify-full":
		tc := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.Host}
		if c.CAFile != "" {
			raw, e := os.ReadFile(c.CAFile)
			if e != nil {
				return nil, contracts.Fail("unavailable")
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(raw) {
				return nil, contracts.Fail("invalid_request")
			}
			tc.RootCAs = pool
		}
		cfg.TLSConfig = tc
	case "local":
	default:
		return nil, contracts.Fail("invalid_request")
	}
	db := stdlib.OpenDB(*cfg)
	db.SetMaxOpenConns(8)
	s := &Store{db: db, workspace: id, backend: "postgresql", schema: c.Schema}
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, contracts.Fail("unavailable")
	}
	if migrate {
		if _, e = db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS \""+c.Schema+"\""); e == nil {
			e = s.migrate(ctx)
		}
	} else {
		e = s.checkVersion(ctx)
	}
	if e == nil {
		e = s.ensureWorkspace(ctx)
	}
	if e != nil {
		db.Close()
		return nil, sanitize(e)
	}
	return s, nil
}
func OpenWorkspace(ctx context.Context, w *workspace.Workspace, secrets contracts.SecretProvider, migrate bool) (*Store, error) {
	p := w.Config.Profiles.Catalog
	if p.Version != contracts.Version {
		return nil, contracts.Fail("incompatible_version")
	}
	raw, e := json.Marshal(p.Configuration)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	checked := func(store *Store, err error) (*Store, error) {
		if err != nil {
			return nil, err
		}
		if p.ExpectedBackendVersion != "" {
			q := "SELECT sqlite_version()"
			if store.backend == "postgresql" {
				q = "SHOW server_version"
			}
			var actual string
			if err = store.db.QueryRowContext(ctx, q).Scan(&actual); err == nil {
				var match bool
				match, err = versionMatches(strings.Fields(actual)[0], p.ExpectedBackendVersion)
				if err == nil && !match {
					err = contracts.Fail("incompatible_version")
				}
			}
			if err != nil {
				store.Close()
				return nil, sanitize(err)
			}
		}
		return store, nil
	}
	switch p.Adapter {
	case "sqlite":
		var c struct {
			Path string `json:"path"`
		}
		if strict(raw, &c) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if !filepath.IsAbs(c.Path) {
			c.Path = filepath.Join(w.Control, c.Path)
		}
		return checked(OpenSQLite(ctx, c.Path, w.Config.WorkspaceID))
	case "postgresql":
		var c PostgreSQLConfig
		if strict(raw, &c) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return checked(OpenPostgreSQL(ctx, c, w.Config.WorkspaceID, secrets, migrate))
	default:
		return nil, contracts.Fail("invalid_request")
	}
}
func sanitize(e error) error {
	if e == nil {
		return nil
	}
	var typed *contracts.Error
	if errors.As(e, &typed) {
		return typed
	}
	if errors.Is(e, sql.ErrNoRows) {
		return contracts.Fail("not_found")
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && (strings.HasPrefix(pg.Code, "23") || pg.Code == "22003") {
		return contracts.Fail("conflict")
	}
	var sq sqlite3.Error
	if errors.As(e, &sq) && sq.Code == sqlite3.ErrConstraint {
		return contracts.Fail("conflict")
	}
	return contracts.Fail("unavailable")
}
func retryable(e error) bool {
	var pg *pgconn.PgError
	if errors.As(e, &pg) {
		return pg.Code == "40001" || pg.Code == "40P01"
	}
	var sq sqlite3.Error
	return errors.As(e, &sq) && (sq.Code == sqlite3.ErrBusy || sq.Code == sqlite3.ErrLocked)
}
func (s *Store) write(ctx context.Context, f func(*sql.Tx, int64) error) error {
	for n := 0; n < 3; n++ {
		tx, e := s.db.BeginTx(ctx, nil)
		if e == nil {
			q := "SELECT revision FROM workspace WHERE id=?"
			if s.backend == "postgresql" {
				q += " FOR UPDATE"
			}
			var revision int64
			e = s.row(ctx, tx, q, s.workspace).Scan(&revision)
			if e == nil {
				e = f(tx, revision)
			}
			if e == nil {
				e = tx.Commit()
			} else {
				tx.Rollback()
			}
		}
		if e == nil {
			return nil
		}
		if !retryable(e) || n == 2 {
			return sanitize(e)
		}
		select {
		case <-ctx.Done():
			return contracts.Fail("cancelled")
		case <-time.After(time.Duration(n+1) * 10 * time.Millisecond):
		}
	}
	return contracts.Fail("unavailable")
}
func (s *Store) now(ctx context.Context, tx *sql.Tx) (int64, error) {
	if s.backend == "postgresql" {
		var n int64
		e := s.row(ctx, tx, "SELECT (extract(epoch from clock_timestamp()) * 1000000000)::bigint").Scan(&n)
		return n, e
	}
	return time.Now().UnixNano(), nil
}
func (s *Store) Revision(ctx context.Context) (int64, error) {
	var n int64
	e := s.db.QueryRowContext(ctx, s.query("SELECT revision FROM workspace WHERE id=?"), s.workspace).Scan(&n)
	return n, sanitize(e)
}
