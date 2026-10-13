// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RoleReport struct {
	Role       string            `json:"role"`
	Adapter    string            `json:"adapter_id"`
	Dimensions map[string]string `json:"dimensions"`
	Correction string            `json:"correction,omitempty"`
	Untested   []string          `json:"untested_capabilities"`
}
type Report struct {
	Version   string       `json:"schema_version"`
	CheckedAt string       `json:"checked_at"`
	Roles     []RoleReport `json:"roles"`
}

func Preflight(ctx context.Context, root string, c workspace.Config, p contracts.SecretProvider) (Report, error) {
	out := Report{contracts.Version, time.Now().UTC().Format(time.RFC3339Nano), []RoleReport{}}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return out, contracts.Fail("cancelled")
	}
	w, e := Draft(root, c)
	if e != nil {
		return out, e
	}
	raw, _ := json.Marshal(c)
	if _, e = Build(raw); e != nil {
		return out, e
	}
	for _, entry := range []struct {
		role    string
		profile workspace.Profile
	}{{"storage", c.Profiles.Storage}, {"catalog", c.Profiles.Catalog}, {"graph", c.Profiles.Graph}} {
		r := RoleReport{entry.role, entry.profile.Adapter, map[string]string{"syntax": "passed", "reachability": "untested", "transport": "untested", "authentication": "untested", "compatibility": "untested"}, "", []string{"write", "provisioning", "migration"}}
		switch entry.profile.Adapter {
		case "filesystem", "sqlite", "ladybugdb":
			key := "path"
			if entry.profile.Adapter == "filesystem" {
				key = "root"
			}
			target, _ := entry.profile.Configuration[key].(string)
			if !filepath.IsAbs(target) {
				base := w.Control
				if entry.profile.Adapter == "ladybugdb" {
					base = w.Root
				}
				target = filepath.Join(base, target)
			}
			st, err := os.Stat(target)
			if err == nil {
				r.Dimensions["reachability"] = "passed"
				if entry.profile.Adapter == "filesystem" && !st.IsDir() {
					r.Dimensions["syntax"] = "failed"
					r.Correction = "Choose an artifact directory."
				}
				if entry.profile.Adapter != "filesystem" && !st.Mode().IsRegular() {
					r.Dimensions["syntax"] = "failed"
					r.Correction = "Choose a regular embedded database file."
				}
			} else if os.IsNotExist(err) {
				r.Dimensions["reachability"] = "absent"
				r.Correction = "Target is absent. Explicit provisioning creates the selected target."
			} else {
				r.Dimensions["reachability"] = "failed"
				r.Correction = "Check target access permissions."
			}
			r.Dimensions["transport"] = "filesystem (locality and WAL support untested)"
			r.Dimensions["authentication"] = "not-applicable"
			r.Untested = append(r.Untested, "schema (path inspection does not open embedded databases)")
		case "s3":
			var cfg artifact.S3Config
			Decode(entry.profile.Configuration, &cfg)
			u, _ := url.Parse(cfg.Endpoint)
			if u != nil {
				r.Dimensions["transport"] = u.Scheme
			}
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.ResponseHeaderTimeout = 5 * time.Second
			defer transport.CloseIdleConnections()
			store, err := artifact.NewS3(ctx, cfg, p, transport)
			if err == nil {
				var exists bool
				exists, err = store.ProbeBucket(ctx)
				store.Close()
				if err == nil && exists {
					r.Dimensions["reachability"] = "passed"
					r.Dimensions["authentication"] = "bucket-access-observed"
					r.Dimensions["compatibility"] = "bucket-access-observed"
				} else if err == nil {
					r.Dimensions["reachability"] = "absent"
					r.Correction = "Verify the selected bucket exists; provisioning does not create S3 buckets."
				}
			}
			if err != nil {
				r.Dimensions["reachability"] = "indeterminate"
				r.Correction = "Verify endpoint, bucket access, selected credentials and TLS. HeadBucket may not distinguish denial from absence."
			}
		case "postgresql":
			var cfg catalog.PostgreSQLConfig
			Decode(entry.profile.Configuration, &cfg)
			r.Dimensions["transport"] = cfg.TLSMode
			pc, err := catalog.PostgreSQLProbeConfig(ctx, cfg, p)
			if err == nil {
				pc.RuntimeParams["default_transaction_read_only"] = "on"
				var conn *pgx.Conn
				conn, err = pgx.ConnectConfig(ctx, pc)
				pc.Password = ""
				if err == nil {
					r.Dimensions["reachability"] = "passed"
					r.Dimensions["authentication"] = "passed"
					if cfg.TLSMode == "verify-full" {
						r.Dimensions["transport"] = "verified-tls"
					}
					var exists bool
					err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname=$1)", cfg.Schema).Scan(&exists)
					if err == nil {
						if !exists {
							r.Dimensions["compatibility"] = "absent"
							r.Correction = "Catalog schema is absent; elect explicit provisioning."
						} else {
							var version int
							var digest string
							err = conn.QueryRow(ctx, "SELECT version,digest FROM "+pgx.Identifier{cfg.Schema, "catalog_schema"}.Sanitize()+" WHERE singleton=1").Scan(&version, &digest)
							if err == nil && catalog.CompatibleCatalogSchema(version, digest) {
								r.Dimensions["compatibility"] = "passed"
							} else {
								r.Dimensions["compatibility"] = "indeterminate"
								r.Correction = "Verify catalog schema version and read permission before migration."
							}
						}
					}
					conn.Close(ctx)
				}
			}
			if err != nil && r.Correction == "" {
				r.Dimensions["reachability"] = "failed"
				r.Correction = "Verify connection, TLS identity/CA and the selected catalog credential."
			}
		case "arcadedb":
			endpoint, _ := entry.profile.Configuration["endpoint"].(string)
			db, _ := entry.profile.Configuration["database"].(string)
			id, _ := entry.profile.Configuration["credential_id"].(string)
			req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(endpoint, "/")+"/api/v1/exists/"+url.PathEscape(db), nil)
			if err == nil && id != "" {
				if p == nil {
					err = contracts.Fail("unavailable")
				} else {
					var secret []byte
					secret, err = p.Resolve(ctx, id)
					if err == nil {
						var auth catalog.Credentials
						err = contracts.DecodeExplore(secret, &auth)
						clear(secret)
						if err == nil {
							req.SetBasicAuth(auth.Username, auth.Password)
						}
					}
				}
			}
			client := http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			defer client.CloseIdleConnections()
			if err == nil {
				var res *http.Response
				res, err = client.Do(req)
				if err == nil {
					data, readErr := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
					res.Body.Close()
					if readErr != nil || len(data) > 1<<20 {
						err = contracts.Fail("output_limit")
					} else if res.StatusCode == 200 {
						var v struct {
							Result *bool `json:"result"`
						}
						if catalog.ValidateJSON(data) != nil || json.NewDecoder(bytes.NewReader(data)).Decode(&v) != nil || v.Result == nil {
							err = contracts.Fail("operation_failed")
						} else {
							r.Dimensions["reachability"] = "passed"
							r.Dimensions["authentication"] = "database-access-observed"
							r.Dimensions["compatibility"] = "absent"
							if *v.Result {
								r.Dimensions["compatibility"] = "database-exists (schema untested)"
							}
						}
					} else {
						err = contracts.Fail("unavailable")
					}
				}
			}
			u, _ := url.Parse(endpoint)
			if u != nil {
				r.Dimensions["transport"] = u.Scheme
			}
			if err != nil {
				r.Dimensions["reachability"] = "failed"
				r.Correction = "Verify ArcadeDB endpoint/database, authenticated access and TLS."
			}
		default:
			r.Dimensions["syntax"] = "failed"
			r.Correction = "Select a supported backend."
		}
		if ctx.Err() != nil {
			return out, contracts.Fail("cancelled")
		}
		out.Roles = append(out.Roles, r)
	}
	return out, nil
}
