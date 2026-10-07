// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"unicode/utf8"
)

// checkedUTF8 validates input across read boundaries before JSON can replace
// damaged encoding. An incomplete rune is carried into the next chunk.
type checkedUTF8 struct {
	reader  io.Reader
	carry   []byte
	failure error
}

func (r *checkedUTF8) Read(p []byte) (int, error) {
	if r.failure != nil {
		return 0, r.failure
	}
	n, e := r.reader.Read(p)
	data := p[:n]
	if len(r.carry) > 0 {
		data = append(append([]byte{}, r.carry...), data...)
		r.carry = nil
	}
	for len(data) > 0 {
		if !utf8.FullRune(data) {
			r.carry = append(r.carry, data...)
			break
		}
		runeValue, width := utf8.DecodeRune(data)
		if runeValue == utf8.RuneError && width == 1 {
			r.failure = contracts.Fail("invalid_request")
			return n, r.failure
		}
		data = data[width:]
	}
	if e == io.EOF && len(r.carry) > 0 {
		r.failure = contracts.Fail("invalid_request")
		return n, r.failure
	}
	return n, e
}

// ReadSnapshotReader validates a file stream into a private disposable spool.
// Typed decoding then uses that same validated input, avoiding an extra whole
// input allocation and a race if the original source changes between passes.
// Decoded records remain in memory in proportion to catalog size.
func ReadSnapshotReader(input io.Reader) (Snapshot, error) {
	var snap Snapshot
	spool, e := os.CreateTemp("", "insonic-catalog-*.json")
	if e != nil {
		return snap, contracts.Fail("unavailable")
	}
	defer func() { spool.Close(); os.Remove(spool.Name()) }()
	stream := &checkedUTF8{reader: io.TeeReader(input, spool)}
	e = validateJSONDecoder(json.NewDecoder(stream))
	if stream.failure != nil {
		return snap, stream.failure
	}
	if e != nil {
		return snap, e
	}
	if _, e = spool.Seek(0, io.SeekStart); e != nil {
		return snap, contracts.Fail("unavailable")
	}
	decoder := json.NewDecoder(spool)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var envelope snapshotEnvelope
	if e = decoder.Decode(&envelope); e != nil {
		return snap, contracts.Fail("invalid_request")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return snap, contracts.Fail("invalid_request")
	}
	return decodedSnapshot(envelope)
}
