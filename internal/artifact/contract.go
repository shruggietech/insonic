// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"strings"
)

type Capabilities struct {
	Adapter           string `json:"adapter"`
	ConditionalCreate bool   `json:"conditional_create"`
	RangedReads       bool   `json:"ranged_reads"`
	MultipartResume   bool   `json:"multipart_resume"`
	VersionIDs        bool   `json:"version_ids"`
	Verification      string `json:"verification"`
}
type Object struct {
	Size    int64  `json:"size"`
	Version string `json:"version"`
	ETag    string `json:"etag"`
}
type Store interface {
	Capabilities() Capabilities
	Stat(context.Context, catalog.Publication) (Object, error)
	OpenRange(context.Context, catalog.Publication, int64, int64) (io.ReadCloser, error)
	PublishImmutable(context.Context, catalog.Publication, *os.File, func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error)
	Abort(context.Context, catalog.Publication) error
	DeleteUnreferenced(context.Context, catalog.Publication) error
	Close() error
}

func validKey(k string) bool {
	if len(k) == 0 || len(k) > 1024 || strings.ContainsAny(k, "\\\x00\r\n:") || strings.HasPrefix(k, "/") {
		return false
	}
	for _, s := range strings.Split(k, "/") {
		if s == "" || s == "." || s == ".." || strings.TrimSpace(s) != s || strings.HasSuffix(s, ".") {
			return false
		}
		base := strings.ToUpper(strings.Split(s, ".")[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
			return false
		}
	}
	return true
}
func validRange(p catalog.Publication, offset, length int64) bool {
	return offset >= 0 && length >= 0 && offset <= p.Size && length <= p.Size-offset
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, contracts.Fail("cancelled")
	}
	return r.reader.Read(b)
}
func copyContext(ctx context.Context, w io.Writer, r io.Reader) (int64, error) {
	return io.CopyBuffer(w, contextReader{ctx, r}, make([]byte, 128<<10))
}
func Verify(ctx context.Context, store Store, p catalog.Publication) error {
	stat, e := store.Stat(ctx, p)
	if e != nil {
		return e
	}
	if stat.Size != p.Size {
		return contracts.AssetFailure("asset_size", "artifact", "object")
	}
	if p.Version != "" && stat.Version != p.Version {
		return contracts.AssetFailure("artifact_identity", "artifact", "object")
	}
	body, e := store.OpenRange(ctx, p, 0, p.Size)
	if e != nil {
		return e
	}
	defer body.Close()
	h := sha256.New()
	n, e := copyContext(ctx, h, body)
	if e != nil {
		return redact(ctx, e)
	}
	if n != p.Size {
		return contracts.AssetFailure("asset_size", "artifact", "object")
	}
	if hex.EncodeToString(h.Sum(nil)) != p.Digest {
		return contracts.AssetFailure("asset_hash", "artifact", "object")
	}
	return nil
}
func redact(ctx context.Context, e error) error {
	if e == nil {
		return nil
	}
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	var original *contracts.Error
	if errors.As(e, &original) {
		return original
	}
	if capacityFailure(e) {
		return contracts.Fail("insufficient_space")
	}
	return contracts.Fail("unavailable")
}
