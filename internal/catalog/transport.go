// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/jackc/pgx/v5"
	"github.com/shruggietech/insonic/internal/contracts"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PostgreSQLBaseConfig contains no ambient authentication or insecure fallback.
func PostgreSQLBaseConfig(c PostgreSQLConfig) (*pgx.ConnConfig, error) {
	if !identifier.MatchString(c.Schema) || c.Host == "" || c.Database == "" || c.Port == 0 {
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
				return nil, contracts.Condition("invalid_request", "backend_local_transport")
			}
			tc.RootCAs = pool
		}
		cfg.TLSConfig = tc
	case "local":
		if strings.EqualFold(c.Host, "localhost") {
			cfg.Host = "127.0.0.1"
		} else if ip := net.ParseIP(c.Host); ip == nil || !ip.IsLoopback() {
			if !filepath.IsAbs(c.Host) || !strings.HasPrefix(c.Host, "/") {
				return nil, contracts.Fail("invalid_request")
			}
		}
	case "insecure":
	default:
		return nil, contracts.Fail("invalid_request")
	}
	return cfg, nil
}

// PostgreSQLProbeConfig resolves the elected credential before a read-only connection.
func PostgreSQLProbeConfig(ctx context.Context, c PostgreSQLConfig, p contracts.SecretProvider) (*pgx.ConnConfig, error) {
	cfg, e := PostgreSQLBaseConfig(c)
	if e != nil {
		return nil, e
	}
	if c.CredentialID != "" {
		if p == nil {
			return nil, contracts.Fail("unavailable")
		}
		if e = referencedPostgresAuthentication(p, c.CredentialID)(ctx, cfg); e != nil {
			return nil, e
		}
	}
	return cfg, nil
}

// CompatibleCatalogSchema checks the immutable migration identity without provisioning.
func CompatibleCatalogSchema(version int, digest string) bool {
	return version == SchemaVersion && digest == migrationDigest()
}
