// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
	"strings"
)

// Table metadata is compiled from typed records, never accepted from callers.
type domainTable struct {
	name, field, primary, checks string
	foreign                      []string
}

var domains = []domainTable{
	{"profile_revision", "Profiles", "id,revision", "CHECK (revision>0), CHECK (role IN ('storage','catalog','graph'))", nil},
	{"artifact", "Artifacts", "id", "CHECK (size>=0 AND length(digest)=64)", nil},
	{"artifact_location", "Locations", "id", "CHECK (profile_revision>0), CHECK (state IN ('available','missing','retired','pending'))", []string{"artifact_id:artifact:id", "profile_id,profile_revision:profile_revision:id,revision"}},
	{"media_entry", "Media", "id", "", nil},
	{"media_asset", "Assets", "id", "CHECK (duration_us>=0)", []string{"artifact_id:artifact:id"}},
	{"media_entry_asset", "MediaAssets", "id", "", []string{"media_id:media_entry:id", "asset_id:media_asset:id"}},
	{"metadata_snapshot", "Metadata", "id", "CHECK (state IN ('captured','no-embedded-metadata','partial','unsupported','failed'))", []string{"asset_id:media_asset:id", "report_artifact_id:artifact:id"}},
	{"metadata_observation", "Observations", "id", "CHECK (instance>=0)", []string{"snapshot_id:metadata_snapshot:id"}},
	{"date_observation", "Dates", "id", "", []string{"media_id:media_entry:id", "observation_id:metadata_observation:id"}},
	{"date_selection", "DateSelections", "id", "CHECK (revision>0), UNIQUE (workspace_id,media_id,revision)", []string{"media_id:media_entry:id", "date_id:date_observation:id"}},
	{"speaker", "Speakers", "id", "", nil},
	{"speaker_segment", "Segments", "id,revision", "CHECK (revision>0 AND start_us>=0 AND end_us>start_us AND channel>=0)", []string{"asset_id:media_asset:id", "speaker_id:speaker:id", "clip_artifact_id:artifact:id"}},
	{"training_dataset", "Datasets", "id", "", []string{"speaker_id:speaker:id", "manifest_artifact_id:artifact:id"}},
	{"dataset_member", "Members", "id", "CHECK (ordinal>=0 AND segment_revision>0), UNIQUE (workspace_id,dataset_id,ordinal)", []string{"dataset_id:training_dataset:id", "segment_id,segment_revision:speaker_segment:id,revision"}},
	{"training_run", "Runs", "id", "", []string{"dataset_id:training_dataset:id", "speaker_id:speaker:id", "job_id:job:id", "preparation_artifact_id:artifact:id"}},
	{"speaker_model", "Models", "id", "", []string{"speaker_id:speaker:id"}},
	{"model_version", "Versions", "id", "", []string{"model_id:speaker_model:id", "run_id:training_run:id", "dataset_id:training_dataset:id", "manifest_artifact_id:artifact:id"}},
	{"model_artifact", "ModelArtifacts", "id", "", []string{"version_id:model_version:id", "artifact_id:artifact:id"}},
	{"model_association", "ModelAssociations", "id", "CHECK (revision>0), UNIQUE (workspace_id,model_id,speaker_id,revision)", []string{"model_id:speaker_model:id", "speaker_id:speaker:id"}},
}

type column struct {
	name string
	typ  reflect.Type
	path []int
}

var instantType = reflect.TypeOf(Instant{})
var rawType = reflect.TypeOf(json.RawMessage{})

func columns(typ reflect.Type) []column {
	var out []column
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		name := f.Tag.Get("json")
		ft := f.Type
		base := ft
		if base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		if base == instantType {
			out = append(out, column{name + "_iso", reflect.TypeOf(""), []int{i, 0}}, column{name + "_unix_ns", reflect.TypeOf(int64(0)), []int{i, 1}})
		} else {
			out = append(out, column{name, ft, []int{i}})
		}
	}
	return out
}
func (d domainTable) typ() reflect.Type {
	f, _ := reflect.TypeOf(Records{}).FieldByName(d.field)
	return f.Type.Elem()
}
func names(cols []column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.name
	}
	return out
}
func fieldValue(v reflect.Value, path []int) any {
	for _, i := range path {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Type() == rawType {
		return string(v.Bytes())
	}
	return v.Interface()
}
func (d domainTable) ddl() string {
	parts := []string{"workspace_id TEXT NOT NULL"}
	for _, c := range columns(d.typ()) {
		typ := c.typ
		nullable := typ.Kind() == reflect.Pointer
		if nullable {
			typ = typ.Elem()
		}
		if len(c.path) == 2 && d.typ().Field(c.path[0]).Type.Kind() == reflect.Pointer {
			nullable = true
		}
		sqlType := "TEXT"
		if typ.Kind() == reflect.Int64 {
			sqlType = "BIGINT"
		}
		null := " NOT NULL"
		if nullable {
			null = ""
		}
		parts = append(parts, c.name+" "+sqlType+null)
	}
	parts = append(parts, "PRIMARY KEY (workspace_id,"+d.primary+")", "FOREIGN KEY (workspace_id) REFERENCES workspace(id)")
	// Profile revisions are referenced by exact revision, not just profile identity.
	for _, fk := range d.foreign {
		v := strings.Split(fk, ":")
		parts = append(parts, "FOREIGN KEY (workspace_id,"+v[0]+") REFERENCES "+v[1]+"(workspace_id,"+v[2]+")")
	}
	if d.checks != "" {
		parts = append(parts, d.checks)
	}
	return "CREATE TABLE IF NOT EXISTS " + d.name + " (" + strings.Join(parts, ",") + ")"
}

var operationalDDL = []string{
	`CREATE TABLE IF NOT EXISTS workspace (id TEXT PRIMARY KEY, revision BIGINT NOT NULL CHECK (revision>=0), schema_version BIGINT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS operation_receipt (workspace_id TEXT NOT NULL, id TEXT NOT NULL, digest TEXT NOT NULL, revision BIGINT NOT NULL CHECK (revision>0), result TEXT NOT NULL, PRIMARY KEY(workspace_id,id), UNIQUE(workspace_id,revision), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`,
	`CREATE TABLE IF NOT EXISTS setting_revision (workspace_id TEXT NOT NULL, name TEXT NOT NULL, revision BIGINT NOT NULL CHECK (revision>0), value TEXT NOT NULL, PRIMARY KEY(workspace_id,name,revision), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`,
	`CREATE TABLE IF NOT EXISTS job (workspace_id TEXT NOT NULL, id TEXT NOT NULL, operation_id TEXT NOT NULL, duration_ms BIGINT NOT NULL CHECK(duration_ms BETWEEN 1 AND 60000), generation BIGINT NOT NULL CHECK(generation>0), state TEXT NOT NULL CHECK(state IN ('running','succeeded','failed','cancelled','interrupted')), PRIMARY KEY(workspace_id,id), UNIQUE(workspace_id,operation_id), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`,
	`CREATE TABLE IF NOT EXISTS job_attempt (workspace_id TEXT NOT NULL, id TEXT NOT NULL, job_id TEXT NOT NULL, owner_id TEXT NOT NULL, generation BIGINT NOT NULL CHECK(generation>0), state TEXT NOT NULL CHECK(state IN ('running','succeeded','failed','cancelled','interrupted')), lease_until BIGINT NOT NULL, started_ns BIGINT NOT NULL, ended_ns BIGINT, reason TEXT NOT NULL, PRIMARY KEY(workspace_id,id), UNIQUE(workspace_id,job_id,generation), FOREIGN KEY(workspace_id,job_id) REFERENCES job(workspace_id,id))`,
	`CREATE TABLE IF NOT EXISTS workspace_lease (workspace_id TEXT NOT NULL, role TEXT NOT NULL, owner_id TEXT NOT NULL, generation BIGINT NOT NULL CHECK(generation>0), lease_until BIGINT NOT NULL, PRIMARY KEY(workspace_id,role), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`,
	`CREATE TABLE IF NOT EXISTS graph_target (workspace_id TEXT NOT NULL, id TEXT NOT NULL, next_sequence BIGINT NOT NULL CHECK(next_sequence>=0), checkpoint BIGINT NOT NULL CHECK(checkpoint>=0), owner_id TEXT, generation BIGINT NOT NULL CHECK(generation>=0), lease_until BIGINT NOT NULL, PRIMARY KEY(workspace_id,id), FOREIGN KEY(workspace_id) REFERENCES workspace(id))`,
	`CREATE TABLE IF NOT EXISTS graph_event (workspace_id TEXT NOT NULL, target_id TEXT NOT NULL, sequence BIGINT NOT NULL CHECK(sequence>0), predecessor BIGINT NOT NULL CHECK(predecessor>=0), revision BIGINT NOT NULL CHECK(revision>0), operation_id TEXT NOT NULL, document TEXT NOT NULL, digest TEXT NOT NULL, PRIMARY KEY(workspace_id,target_id,sequence), FOREIGN KEY(workspace_id,target_id) REFERENCES graph_target(workspace_id,id), FOREIGN KEY(workspace_id,operation_id) REFERENCES operation_receipt(workspace_id,id))`,
}

func migrationStatements() []string {
	out := append([]string{}, operationalDDL...)
	for _, d := range domains {
		out = append(out, d.ddl())
	}
	return out
}
func migrationDigest() string {
	h := sha256.Sum256([]byte(strings.Join(migrationStatements(), "\n")))
	return hex.EncodeToString(h[:])
}
func (s *Store) checkVersion(ctx context.Context) error {
	var v int
	var digest string
	e := s.db.QueryRowContext(ctx, "SELECT version,digest FROM catalog_schema WHERE singleton=1").Scan(&v, &digest)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	if v != SchemaVersion || digest != migrationDigest() {
		return contracts.Fail("incompatible_version")
	}
	return nil
}
func (s *Store) migrate(ctx context.Context) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return sanitize(e)
	}
	defer tx.Rollback()
	if s.backend == "postgresql" {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext(current_schema()))"); e != nil {
			return sanitize(e)
		}
	}
	if _, e = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS catalog_schema (singleton BIGINT PRIMARY KEY CHECK(singleton=1), version BIGINT NOT NULL, digest TEXT NOT NULL)"); e != nil {
		return sanitize(e)
	}
	var version int
	var digest string
	e = tx.QueryRowContext(ctx, "SELECT version,digest FROM catalog_schema WHERE singleton=1").Scan(&version, &digest)
	if e == nil {
		if version != SchemaVersion || digest != migrationDigest() {
			return contracts.Fail("incompatible_version")
		}
		return sanitize(tx.Commit())
	}
	if e != sql.ErrNoRows {
		return sanitize(e)
	}
	for _, q := range migrationStatements() {
		if _, e = tx.ExecContext(ctx, q); e != nil {
			return sanitize(e)
		}
	}
	if _, e = s.exec(ctx, tx, "INSERT INTO catalog_schema(singleton,version,digest) VALUES(1,?,?)", SchemaVersion, migrationDigest()); e != nil {
		return sanitize(e)
	}
	return sanitize(tx.Commit())
}
func (s *Store) ensureWorkspace(ctx context.Context) error {
	_, e := s.db.ExecContext(ctx, s.query("INSERT INTO workspace(id,revision,schema_version) VALUES(?,0,?) ON CONFLICT(id) DO NOTHING"), s.workspace, SchemaVersion)
	return sanitize(e)
}
func validateRecord(value any) error {
	v := reflect.ValueOf(value)
	for _, c := range columns(v.Type()) {
		x := fieldValue(v, c.path)
		if x == nil {
			continue
		}
		if c.name == "id" || (strings.HasSuffix(c.name, "_id") && c.name != "credential_id") {
			if id, ok := x.(string); !ok || !contracts.ValidID(id) {
				return contracts.Fail("invalid_request")
			}
		}
	}
	switch r := value.(type) {
	case Profile:
		if !positive(r.Revision) || r.Version != contracts.Version || !nonsecret(r.Configuration) {
			return contracts.Fail("invalid_request")
		}
	case Artifact:
		if !digestPattern.MatchString(r.Digest) || r.Size < 0 || r.Kind == "" {
			return contracts.Fail("invalid_request")
		}
	case Asset:
		if r.DurationUS < 0 || r.Role == "" {
			return contracts.Fail("invalid_request")
		}
	case MetadataSnapshot:
		if !r.Captured.valid() || r.Extractor == "" || r.Version == "" {
			return contracts.Fail("invalid_request")
		}
	case Observation:
		if r.Instance < 0 || r.Family == "" || r.Tag == "" || !validJSON(r.Raw) || !validJSON(r.Value) {
			return contracts.Fail("invalid_request")
		}
	case DateObservation:
		if r.Resolved != nil && !r.Resolved.valid() {
			return contracts.Fail("invalid_request")
		}
		if r.Basis == "" || r.Precision == "" {
			return contracts.Fail("invalid_request")
		}
	case DateSelection:
		if !positive(r.Revision) || r.Policy == "" {
			return contracts.Fail("invalid_request")
		}
	case Segment:
		if !positive(r.Revision) || r.StartUS < 0 || r.EndUS <= r.StartUS || r.Channel < 0 || !validJSON(r.Attribution) {
			return contracts.Fail("invalid_request")
		}
	case Dataset:
		if !nonsecret(r.Options) {
			return contracts.Fail("invalid_request")
		}
	case DatasetMember:
		if r.Ordinal < 0 || !positive(r.SegmentRevision) {
			return contracts.Fail("invalid_request")
		}
	case TrainingRun:
		if r.Adapter == "" || !nonsecret(r.Options) {
			return contracts.Fail("invalid_request")
		}
	case ModelAssociation:
		if !positive(r.Revision) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func (s *Store) insertRecords(ctx context.Context, tx *sql.Tx, records Records) error {
	// Frozen datasets admit ordered membership only in their creation transaction.
	created := map[string]bool{}
	for _, d := range records.Datasets {
		created[d.ID] = true
	}
	for _, m := range records.Members {
		if !created[m.DatasetID] {
			return contracts.Fail("conflict")
		}
	}
	v := reflect.ValueOf(records)
	for _, d := range domains {
		items := v.FieldByName(d.field)
		cols := columns(d.typ())
		list := append([]string{"workspace_id"}, names(cols)...)
		marks := strings.TrimSuffix(strings.Repeat("?,", len(list)), ",")
		q := "INSERT INTO " + d.name + "(" + strings.Join(list, ",") + ") VALUES(" + marks + ")"
		for i := 0; i < items.Len(); i++ {
			item := items.Index(i)
			if e := validateRecord(item.Interface()); e != nil {
				return e
			}
			args := []any{s.workspace}
			for _, c := range cols {
				args = append(args, fieldValue(item, c.path))
			}
			if _, e := s.exec(ctx, tx, q, args...); e != nil {
				return e
			}
		}
	}
	// Segment bounds depend on the immutable source clock, not only local ordering.
	var invalid int
	e := s.row(ctx, tx, "SELECT count(*) FROM speaker_segment s JOIN media_asset a ON a.workspace_id=s.workspace_id AND a.id=s.asset_id WHERE s.workspace_id=? AND s.end_us>a.duration_us", s.workspace).Scan(&invalid)
	if e != nil {
		return e
	}
	if invalid > 0 {
		return contracts.Fail("invalid_request")
	}
	for _, q := range []string{
		"SELECT count(*) FROM date_selection s JOIN date_observation o ON o.workspace_id=s.workspace_id AND o.id=s.date_id WHERE s.workspace_id=? AND s.media_id<>o.media_id",
		"SELECT count(*) FROM training_run r JOIN training_dataset d ON d.workspace_id=r.workspace_id AND d.id=r.dataset_id WHERE r.workspace_id=? AND r.speaker_id<>d.speaker_id",
		"SELECT count(*) FROM model_version v JOIN training_run r ON r.workspace_id=v.workspace_id AND r.id=v.run_id JOIN speaker_model m ON m.workspace_id=v.workspace_id AND m.id=v.model_id WHERE v.workspace_id=? AND (v.dataset_id<>r.dataset_id OR m.speaker_id<>r.speaker_id)",
	} {
		if e = s.row(ctx, tx, q, s.workspace).Scan(&invalid); e != nil {
			return e
		}
		if invalid > 0 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func (s *Store) readRecords(ctx context.Context, tx *sql.Tx) (Records, error) {
	var out Records
	v := reflect.ValueOf(&out).Elem()
	for _, d := range domains {
		cols := columns(d.typ())
		rows, e := tx.QueryContext(ctx, s.query("SELECT "+strings.Join(names(cols), ",")+" FROM "+d.name+" WHERE workspace_id=? ORDER BY "+d.primary), s.workspace)
		if e != nil {
			return out, e
		}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptr := make([]any, len(cols))
			for i := range vals {
				ptr[i] = &vals[i]
			}
			if e = rows.Scan(ptr...); e != nil {
				rows.Close()
				return out, e
			}
			doc := map[string]any{}
			for i, c := range cols {
				val := vals[i]
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
				field := d.typ().Field(c.path[0])
				key := field.Tag.Get("json")
				if len(c.path) == 2 {
					if val == nil {
						doc[key] = nil
						continue
					}
					nested, _ := doc[key].(map[string]any)
					if nested == nil {
						nested = map[string]any{}
						doc[key] = nested
					}
					sub := instantType.Field(c.path[1]).Tag.Get("json")
					nested[sub] = val
				} else {
					if c.typ == rawType {
						var x any
						if e = strict([]byte(fmt.Sprint(val)), &x); e != nil {
							rows.Close()
							return out, e
						}
						val = x
					}
					doc[key] = val
				}
			}
			data, e := json.Marshal(doc)
			if e != nil {
				rows.Close()
				return out, e
			}
			item := reflect.New(d.typ())
			if e = strict(data, item.Interface()); e != nil {
				rows.Close()
				return out, e
			}
			v.FieldByName(d.field).Set(reflect.Append(v.FieldByName(d.field), item.Elem()))
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
	}
	return out, nil
}
