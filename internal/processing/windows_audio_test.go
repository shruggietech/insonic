// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

func TestWindowSourceVerifiesChunksBeforeDecoderReads(t *testing.T) {
	raw := bytes.Repeat([]byte("canonical"), 200000)
	h := sha256.Sum256(raw)
	s := &verifiedWindowSource{size: int64(len(raw)), digest: hex.EncodeToString(h[:]), open: func(context.Context) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(raw)), nil }}
	if err := s.verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	r, err := s.reader(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	r.Close()
	if err != nil || !bytes.Equal(raw, got) {
		t.Fatal(err)
	}
	raw[0] ^= 1
	r, err = s.reader(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b := make([]byte, 1)
	n, err := r.Read(b)
	if err == nil || n != 0 {
		t.Fatal("changed chunk reached decoder", n, err)
	}
}

func TestWindowSourceRejectsInitialMismatchAndCancellation(t *testing.T) {
	s := &verifiedWindowSource{size: 3, digest: hex.EncodeToString(make([]byte, 32)), open: func(context.Context) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader([]byte("abc"))), nil }}
	if err := s.verify(context.Background()); err == nil {
		t.Fatal("bad initial digest accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.verify(ctx); err == nil {
		t.Fatal("cancelled verify accepted")
	}
}
