//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/workspace"
)

func inventoryCabinet(payload []byte) []byte {
	sum := sha256.Sum256([]byte("package-inventory.json"))
	member := "f" + hex.EncodeToString(sum[:])[:28]
	var entries, raw bytes.Buffer
	for _, file := range []struct {
		name string
		data []byte
	}{{"unrelated.txt", []byte("must not extract")}, {member, payload}} {
		_ = binary.Write(&entries, binary.LittleEndian, uint32(len(file.data)))
		_ = binary.Write(&entries, binary.LittleEndian, uint32(raw.Len()))
		for _, value := range []uint16{0, 0, 0, 32} {
			_ = binary.Write(&entries, binary.LittleEndian, value)
		}
		entries.WriteString(file.name)
		entries.WriteByte(0)
		raw.Write(file.data)
	}
	start := 44 + entries.Len()
	data := make([]byte, 44)
	copy(data, "MSCF")
	binary.LittleEndian.PutUint32(data[8:], uint32(start+8+raw.Len()))
	binary.LittleEndian.PutUint32(data[16:], 44)
	data[24] = 3
	data[25] = 1
	binary.LittleEndian.PutUint16(data[26:], 1)
	binary.LittleEndian.PutUint16(data[28:], 2)
	binary.LittleEndian.PutUint32(data[36:], uint32(start))
	binary.LittleEndian.PutUint16(data[40:], 1)
	data = append(data, entries.Bytes()...)
	block := make([]byte, 8)
	binary.LittleEndian.PutUint16(block[4:], uint16(raw.Len()))
	binary.LittleEndian.PutUint16(block[6:], uint16(raw.Len()))
	data = append(data, block...)
	return append(data, raw.Bytes()...)
}

// Read-only MSI database/cabinet verification. This never installs a product,
// changes registration/policy, creates an account or opens installer UI.
func TestLegacySourceInventoryIsReadFromCabinetWithoutExtractingOtherFiles(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	root := t.TempDir()
	scratch := filepath.Join(root, "private-scratch")
	if err := os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	if err := workspace.SecureDirectory(scratch, true); err != nil {
		t.Fatal(err)
	}
	payload := []byte("{\"kind\":\"native-package-inventory\",\"version\":\"1.0.0\"}\n")
	sum := sha256.Sum256(payload)
	cab := filepath.Join(root, "source.cab")
	if err := os.WriteFile(cab, inventoryCabinet(payload), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "read-only-database.msi")
	var db uint32
	if err := checked("MsiOpenDatabaseW", source, 3, &db); err != nil {
		t.Fatal(err)
	}
	execute := func(query string, record uint32) {
		t.Helper()
		var view uint32
		if err := checked("MsiDatabaseOpenViewW", db, query, &view); err != nil {
			t.Fatal(err)
		}
		defer closeMSI(view)
		if err := checked("MsiViewExecute", view, record); err != nil {
			t.Fatal(err)
		}
	}
	execute("CREATE TABLE `Property` (`Property` CHAR(72) NOT NULL, `Value` CHAR(0) PRIMARY KEY `Property`)", 0)
	record, err := native("MsiCreateRecord", 2)
	if err != nil || record == 0 {
		t.Fatal("create source stream record", err)
	}
	if err := checked("MsiRecordSetStringW", record, 1, "payload.cab"); err != nil {
		t.Fatal(err)
	}
	if err := checked("MsiRecordSetStreamW", record, 2, cab); err != nil {
		t.Fatal(err)
	}
	execute("INSERT INTO `_Streams` (`Name`,`Data`) VALUES (?,?)", record)
	closeMSI(record)
	if err := checked("MsiDatabaseCommit", db); err != nil {
		t.Fatal(err)
	}
	closeMSI(db)
	before := testChoices
	before.Files = []File{{Path: "package-inventory.json", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}}
	e := nativeEngine{logDirectory: scratch}
	if err := e.ValidateInventory(Source{Path: source}, before); err != nil {
		t.Fatal(err)
	}
	before.Files[0].SHA256 = strings.Repeat("0", 64)
	if err := e.ValidateInventory(Source{Path: source}, before); err == nil {
		t.Fatal("accepted another installed inventory")
	}
	entries, err := os.ReadDir(scratch)
	if err != nil || len(entries) != 0 {
		t.Fatal("inspection leaked or extracted unrelated files", entries, err)
	}
}
