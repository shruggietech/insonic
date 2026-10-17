// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func setupBytes() []byte {
	payload := []byte("exact complete MSI source")
	digest := sha256.Sum256(payload)
	p := testPackage
	p.Size = int64(len(payload))
	p.SHA256 = hex.EncodeToString(digest[:])
	metadata, _ := json.Marshal(p)
	var b bytes.Buffer
	b.Write(bytes.Repeat([]byte{0x4d}, 64))
	b.Write(payload)
	b.Write(metadata)
	b.WriteString(FooterMagic)
	binary.Write(&b, binary.LittleEndian, uint64(len(payload)))
	binary.Write(&b, binary.LittleEndian, uint32(len(metadata)))
	binary.Write(&b, binary.LittleEndian, uint32(0))
	return b.Bytes()
}
func TestAppendedPayloadIsBoundedAndHashValidatedWithoutLoadingWholePackage(t *testing.T) {
	data := setupBytes()
	p, section, err := ReadPayload(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := CopyPayload(&output, section, p); err != nil {
		t.Fatal(err)
	}
	if output.String() != "exact complete MSI source" {
		t.Fatal(output.String())
	}
	data[64] ^= 1
	_, section, err = ReadPayload(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if CopyPayload(io.Discard, section, p) == nil {
		t.Fatal("accepted corrupt embedded MSI")
	}
}
func TestMalformedFootersAndIdentityCannotSelectOutsideExecutable(t *testing.T) {
	for _, mutate := range []func([]byte) []byte{
		func(b []byte) []byte { return b[:31] },
		func(b []byte) []byte { b[len(b)-32] = 'x'; return b },
		func(b []byte) []byte { binary.LittleEndian.PutUint64(b[len(b)-16:], 1<<63); return b },
		func(b []byte) []byte { binary.LittleEndian.PutUint32(b[len(b)-8:], 65537); return b },
		func(b []byte) []byte {
			return bytes.Replace(b, []byte(Family), []byte(strings.Repeat("x", len(Family))), 1)
		},
	} {
		data := mutate(setupBytes())
		if _, _, err := ReadPayload(bytes.NewReader(data), int64(len(data))); err == nil {
			t.Fatal("accepted malformed payload")
		}
	}
}
