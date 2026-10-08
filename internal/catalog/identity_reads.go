// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
	"strings"
)

// Load bounded selected domains in one query, rather than a query per term or
// alias. Fields and table names come only from compiled catalog metadata.
func (s *Store) identityRecords(ctx context.Context, tx *sql.Tx, field, where string, args []any, limit int) (out Records, e error) {
	d := domainNamed(field)
	cols := columns(d.typ())
	q := "SELECT " + joinNames(d) + " FROM " + d.name + " WHERE workspace_id=?"
	parameters := []any{s.workspace}
	if where != "" {
		q += " AND (" + where + ")"
		parameters = append(parameters, args...)
	}
	q += " ORDER BY id LIMIT ?"
	parameters = append(parameters, limit)
	rows, e := tx.QueryContext(ctx, s.query(q), parameters...)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	target := reflect.ValueOf(&out).Elem().FieldByName(field)
	bytes := 0
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if e = rows.Scan(ptrs...); e != nil {
			return out, e
		}
		doc := map[string]any{}
		for i, c := range cols {
			value := values[i]
			if raw, ok := value.([]byte); ok {
				value = string(raw)
			}
			if c.typ == rawType {
				text, ok := value.(string)
				if !ok || ValidateJSON([]byte(text)) != nil {
					return out, contracts.Fail("invalid_request")
				}
				value = json.RawMessage(text)
			}
			doc[c.name] = value
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			return out, err
		}
		bytes += len(raw)
		if bytes > 4<<20 {
			return out, contracts.Fail("input_limit")
		}
		item := reflect.New(d.typ())
		if e = strict(raw, item.Interface()); e != nil {
			return out, e
		}
		target.Set(reflect.Append(target, item.Elem()))
	}
	return out, rows.Err()
}
func boundedPage(value any) bool { raw, e := json.Marshal(value); return e == nil && len(raw) <= 1<<20 }
func selectedIDs(ids []string) (string, []any) {
	marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return marks, args
}
