//go:build integration

package qualification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestPostgreSQLTransactionFeasibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, os.Getenv("INSONIC_FIXTURE_POSTGRES"))
	if err != nil {
		t.Fatal("fixture PostgreSQL connection failed")
	}
	defer conn.Close(ctx)
	table := fmt.Sprintf("insonic_probe_%d", time.Now().UnixNano())
	if _, err := conn.Exec(ctx, "CREATE TABLE "+table+" (id INTEGER PRIMARY KEY, unix_ns BIGINT NOT NULL)"); err != nil {
		t.Fatal("fixture table create failed")
	}
	defer conn.Exec(context.Background(), "DROP TABLE "+table)
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal("begin failed")
	}
	if _, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1,$2)", 1, int64(1780000000000000001)); err != nil {
		t.Fatal("write failed")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal("rollback failed")
	}
	var count int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
		t.Fatal("rollback not observed")
	}
	tx, err = conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal("begin failed")
	}
	if _, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1,$2)", 1, int64(1780000000000000001)); err != nil {
		t.Fatal("write failed")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal("commit failed")
	}
	var ns int64
	if err := conn.QueryRow(ctx, "SELECT unix_ns FROM "+table+" WHERE id=$1", 1).Scan(&ns); err != nil || ns != 1780000000000000001 {
		t.Fatal("exact committed timestamp differs")
	}
	t.Log("pinned PostgreSQL serializable commit/rollback and exact integer read passed")
}

func TestGenericS3FixtureFeasibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := minio.New(os.Getenv("INSONIC_FIXTURE_S3"), &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture", ""), Secure: false, Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if err != nil {
		t.Fatal("S3 client failed")
	}
	bucket := fmt.Sprintf("insonic-%d", time.Now().UnixNano())
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		t.Fatal("bucket create failed")
	}
	defer client.RemoveBucket(context.Background(), bucket)
	data := []byte("immutable-source-fixture")
	if _, err := client.PutObject(ctx, bucket, "source", bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream"}); err != nil {
		t.Fatal("object publication failed")
	}
	defer client.RemoveObject(context.Background(), bucket, "source", minio.RemoveObjectOptions{})
	stat, err := client.StatObject(ctx, bucket, "source", minio.StatObjectOptions{})
	if err != nil || stat.Size != int64(len(data)) {
		t.Fatal("completed object visibility failed")
	}
	object, err := client.GetObject(ctx, bucket, "source", minio.GetObjectOptions{})
	if err != nil {
		t.Fatal("get failed")
	}
	got, err := io.ReadAll(object)
	object.Close()
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("verified object bytes differ")
	}
	opts := minio.GetObjectOptions{}
	opts.SetRange(2, 7)
	object, err = client.GetObject(ctx, bucket, "source", opts)
	if err != nil {
		t.Fatal("range failed")
	}
	got, err = io.ReadAll(object)
	object.Close()
	if err != nil || !bytes.Equal(got, data[2:8]) {
		t.Fatal("range bytes differ")
	}
	if err := client.RemoveObject(ctx, bucket, "source", minio.RemoveObjectOptions{}); err != nil {
		t.Fatal("delete failed")
	}
	if _, err := client.StatObject(ctx, bucket, "source", minio.StatObjectOptions{}); err == nil {
		t.Fatal("deleted object remained visible")
	}
	t.Log("S3 path-style put/head/full-read/range/delete passed; this fixture does not qualify production authentication")
}

func TestArcadeNativeDialectAndTransactions(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}
	database := fmt.Sprintf("insonic%d", time.Now().UnixNano())
	call := func(route string, body any, session string) (map[string]any, string, error) {
		data, _ := json.Marshal(body)
		request, err := http.NewRequest("POST", os.Getenv("INSONIC_FIXTURE_ARCADE")+route, bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("invalid fixture request")
		}
		request.SetBasicAuth("root", "fixture-only-password")
		request.Header.Set("Content-Type", "application/json")
		if session != "" {
			request.Header.Set("arcadedb-session-id", session)
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, "", fmt.Errorf("fixture connection failed")
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, "", fmt.Errorf("fixture response status %d", response.StatusCode)
		}
		var result map[string]any
		if response.StatusCode != 204 {
			decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
			decoder.UseNumber()
			if err := decoder.Decode(&result); err != nil {
				return nil, "", fmt.Errorf("fixture response invalid")
			}
		}
		return result, response.Header.Get("arcadedb-session-id"), nil
	}
	if _, _, err := call("/api/v1/server", map[string]any{"command": "create database " + database}, ""); err != nil {
		t.Fatal(err)
	}
	defer call("/api/v1/server", map[string]any{"command": "drop database " + database}, "")
	command := func(language, sql string, params map[string]any) map[string]any {
		return map[string]any{"language": language, "command": sql, "params": params}
	}
	if _, _, err := call("/api/v1/command/"+database, command("sql", "CREATE VERTEX TYPE Fixture", nil), ""); err != nil {
		t.Fatal(err)
	}
	for _, end := range []string{"rollback", "commit"} {
		_, session, err := call("/api/v1/begin/"+database, map[string]any{}, "")
		if err != nil || session == "" {
			t.Fatal("fixture transaction session missing")
		}
		if _, _, err := call("/api/v1/command/"+database, command("sql", "CREATE VERTEX Fixture SET id = :id, unix_ns = :unix_ns", map[string]any{"id": "probe", "unix_ns": int64(1780000000000000001)}), session); err != nil {
			t.Fatal(err)
		}
		if _, _, err := call("/api/v1/"+end+"/"+database, map[string]any{}, session); err != nil {
			t.Fatal(err)
		}
		result, _, err := call("/api/v1/query/"+database, command("sql", "SELECT id, unix_ns FROM Fixture WHERE id = :id", map[string]any{"id": "probe"}), "")
		if err != nil {
			t.Fatal(err)
		}
		rows, ok := result["result"].([]any)
		if !ok {
			t.Fatal("fixture result missing")
		}
		if end == "rollback" && len(rows) != 0 {
			t.Fatal("rollback leaked row")
		}
		if end == "commit" {
			if len(rows) != 1 {
				t.Fatal("committed row missing")
			}
			row := rows[0].(map[string]any)
			if row["unix_ns"].(json.Number).String() != "1780000000000000001" {
				t.Fatal("timestamp precision lost")
			}
		}
	}
	result, _, err := call("/api/v1/command/"+database, command("opencypher", "MATCH (f:Fixture) RETURN f.id AS id", nil), "")
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := result["result"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatal("native OpenCypher result differs")
	}
	t.Log("pinned ArcadeDB session/commit/rollback/exact integers and native OpenCypher passed")
}
