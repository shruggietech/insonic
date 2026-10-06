// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const partSize int64 = 8 << 20

type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	Region          string `json:"region"`
	AddressingStyle string `json:"addressing_style"`
	Authentication  string `json:"authentication"`
	CredentialID    string `json:"credential_id"`
}
type S3Credentials struct {
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	SessionToken string `json:"session_token"`
}
type S3 struct {
	core           minio.Core
	bucket, prefix string
}

func NewS3(ctx context.Context, c S3Config, secrets contracts.SecretProvider, transport http.RoundTripper) (*S3, error) {
	endpoint, e := url.Parse(c.Endpoint)
	if e != nil || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || (endpoint.Path != "" && endpoint.Path != "/") || c.Bucket == "" || c.Region == "" {
		return nil, contracts.Fail("invalid_request")
	}
	prefix := strings.TrimSuffix(c.Prefix, "/")
	if prefix != "" && !validKey(prefix) {
		return nil, contracts.Fail("invalid_request")
	}
	var creds *credentials.Credentials
	switch c.Authentication {
	case "credential":
		if secrets == nil || c.CredentialID == "" {
			return nil, contracts.Fail("unavailable")
		}
		raw, e := secrets.Resolve(ctx, c.CredentialID)
		if e != nil {
			return nil, contracts.Fail("unavailable")
		}
		var values S3Credentials
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&values) != nil || values.AccessKey == "" || values.SecretKey == "" {
			return nil, contracts.Fail("unavailable")
		}
		creds = credentials.NewStaticV4(values.AccessKey, values.SecretKey, values.SessionToken)
	case "environment":
		values := S3Credentials{os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN")}
		if values.AccessKey == "" || values.SecretKey == "" {
			return nil, contracts.Fail("unavailable")
		}
		creds = credentials.NewStaticV4(values.AccessKey, values.SecretKey, values.SessionToken)
	case "anonymous":
		creds = credentials.NewStaticV4("", "", "")
	default:
		return nil, contracts.Fail("invalid_request")
	}
	lookup := minio.BucketLookupAuto
	switch c.AddressingStyle {
	case "path":
		lookup = minio.BucketLookupPath
	case "virtual":
		lookup = minio.BucketLookupDNS
	case "auto":
	default:
		return nil, contracts.Fail("invalid_request")
	}
	client, e := minio.New(endpoint.Host, &minio.Options{Creds: creds, Secure: endpoint.Scheme == "https", Region: c.Region, BucketLookup: lookup, Transport: transport})
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	return &S3{minio.Core{Client: client}, c.Bucket, prefix}, nil
}
func (s *S3) Close() error { return nil }
func (s *S3) Capabilities() Capabilities {
	return Capabilities{Adapter: "s3", RangedReads: true, MultipartResume: true, VersionIDs: true, Verification: "sha256-readback"}
}
func (s *S3) key(p catalog.Publication) (string, error) {
	if !validKey(p.Key) {
		return "", contracts.Fail("invalid_request")
	}
	if s.prefix == "" {
		return p.Key, nil
	}
	return s.prefix + "/" + p.Key, nil
}
func (s *S3) Stat(ctx context.Context, p catalog.Publication) (Object, error) {
	k, e := s.key(p)
	if e != nil {
		return Object{}, e
	}
	o, e := s.core.StatObject(ctx, s.bucket, k, minio.StatObjectOptions{VersionID: p.Version})
	if e != nil {
		return Object{}, redact(ctx, e)
	}
	return Object{o.Size, o.VersionID, o.ETag}, nil
}
func (s *S3) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	if !validRange(p, offset, length) {
		return nil, contracts.Fail("invalid_request")
	}
	k, e := s.key(p)
	if e != nil {
		return nil, e
	}
	if length == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	opts := minio.GetObjectOptions{VersionID: p.Version}
	partial := offset != 0 || length != p.Size
	if partial {
		if e = opts.SetRange(offset, offset+length-1); e != nil {
			return nil, contracts.Fail("invalid_request")
		}
	}
	body, info, header, e := s.core.GetObject(ctx, s.bucket, k, opts)
	if e != nil {
		return nil, redact(ctx, e)
	}
	if info.Size != length || p.Version != "" && info.VersionID != p.Version || partial && header.Get("Content-Range") != fmt.Sprintf("bytes %d-%d/%d", offset, offset+length-1, p.Size) {
		body.Close()
		return nil, contracts.Fail("conflict")
	}
	return sectionCloser{contextReader{ctx, io.LimitReader(body, length)}, body}, nil
}
func (s *S3) PublishImmutable(ctx context.Context, p catalog.Publication, source *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	key, e := s.key(p)
	if e != nil {
		return p, e
	}
	if st, e := s.core.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{VersionID: p.Version}); e == nil {
		p.Version = st.VersionID
		return p, Verify(ctx, s, p)
	} else if ctx.Err() != nil {
		return p, contracts.Fail("cancelled")
	} else {
		code := minio.ToErrorResponse(e).Code
		if code != "NoSuchKey" && code != "NoSuchVersion" && code != "NotFound" {
			return p, contracts.Fail("unavailable")
		}
	}
	opts := minio.PutObjectOptions{ContentType: "application/octet-stream", DisableMultipart: true}
	if p.Size <= partSize {
		p.CompletionRequested = true
		p, e = save(p)
		if e != nil {
			return p, e
		}
		source.Seek(0, 0)
		info, e := s.core.Client.PutObject(ctx, s.bucket, key, contextReader{ctx, source}, p.Size, opts)
		if e != nil {
			return p, redact(ctx, e)
		}
		p.Version = info.VersionID
		return p, Verify(ctx, s, p)
	}
	// Size parts to fit the protocol's 10,000-part limit, but stream each part
	// from the staged file so larger objects do not increase buffer memory.
	if p.PartSize == 0 {
		p.PartSize = max(partSize, (p.Size+9999)/10000)
	}
	if p.UploadID == "" {
		p.UploadID, e = s.core.NewMultipartUpload(ctx, s.bucket, key, opts)
		if e != nil {
			return p, redact(ctx, e)
		}
		p, e = save(p)
		if e != nil {
			return p, e
		}
	}
	known := map[int]minio.ObjectPart{}
	if len(p.Parts) > 0 {
		for marker := 0; ; {
			listed, err := s.core.ListObjectParts(ctx, s.bucket, key, p.UploadID, marker, 1000)
			if err != nil {
				return p, redact(ctx, err)
			}
			for _, part := range listed.ObjectParts {
				known[part.PartNumber] = part
			}
			if !listed.IsTruncated {
				break
			}
			if listed.NextPartNumberMarker <= marker {
				return p, contracts.Fail("unavailable")
			}
			marker = listed.NextPartNumberMarker
		}
	}
	var complete []minio.CompletePart
	for off, number := int64(0), 1; off < p.Size; off, number = off+p.PartSize, number+1 {
		count := min(p.PartSize, p.Size-off)
		if len(p.Parts) >= number {
			saved := p.Parts[number-1]
			remote, exists := known[number]
			if exists && strings.Trim(remote.ETag, "\"") == strings.Trim(saved.ETag, "\"") && remote.Size == count && saved.Size == count {
				complete = append(complete, minio.CompletePart{PartNumber: number, ETag: saved.ETag})
				continue
			}
		}
		md := md5.New()
		sh := sha256.New()
		if _, e = copyContext(ctx, io.MultiWriter(md, sh), io.NewSectionReader(source, off, count)); e != nil {
			return p, redact(ctx, e)
		}
		part, e := s.core.PutObjectPart(ctx, s.bucket, key, p.UploadID, number, contextReader{ctx, io.NewSectionReader(source, off, count)}, count, minio.PutObjectPartOptions{Md5Base64: base64.StdEncoding.EncodeToString(md.Sum(nil)), Sha256Hex: hex.EncodeToString(sh.Sum(nil))})
		if e != nil {
			return p, redact(ctx, e)
		}
		record := catalog.UploadPart{Number: number, ETag: part.ETag, Size: count}
		if len(p.Parts) >= number {
			p.Parts[number-1] = record
		} else {
			p.Parts = append(p.Parts, record)
		}
		p, e = save(p)
		if e != nil {
			return p, e
		}
		complete = append(complete, minio.CompletePart{PartNumber: number, ETag: part.ETag})
	}
	opts.DisableMultipart = false
	p.CompletionRequested = true
	p, e = save(p)
	if e != nil {
		return p, e
	}
	info, e := s.core.CompleteMultipartUpload(ctx, s.bucket, key, p.UploadID, complete, opts)
	if e != nil {
		return p, redact(ctx, e)
	}
	p.Version = info.VersionID
	return p, Verify(ctx, s, p)
}
func (s *S3) Abort(ctx context.Context, p catalog.Publication) error {
	key, e := s.key(p)
	if e != nil {
		return e
	}
	if _, e = s.core.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{VersionID: p.Version}); e == nil {
		return contracts.Fail("conflict")
	} else if code := minio.ToErrorResponse(e).Code; code != "NoSuchKey" && code != "NoSuchVersion" && code != "NotFound" {
		return redact(ctx, e)
	}
	if p.CompletionRequested {
		return s.uncertainAbort(ctx, p)
	}
	if p.UploadID == "" {
		return nil
	}
	e = s.core.AbortMultipartUpload(ctx, s.bucket, key, p.UploadID)
	if code := minio.ToErrorResponse(e).Code; code == "NoSuchUpload" {
		return s.uncertainAbort(ctx, p)
	}
	return redact(ctx, e)
}

// An absent upload plus an absent object is not evidence of abort when a
// provider delays completed-object visibility. Bound observation, then retain
// the pending journal unless the original abort received an affirmative reply.
func (s *S3) uncertainAbort(ctx context.Context, p catalog.Publication) error {
	key, e := s.key(p)
	if e != nil {
		return e
	}
	for _, delay := range []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond} {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return contracts.Fail("cancelled")
		case <-timer.C:
		}
		if _, e = s.core.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{VersionID: p.Version}); e == nil {
			return contracts.Fail("conflict")
		}
		code := minio.ToErrorResponse(e).Code
		if code != "NoSuchKey" && code != "NoSuchVersion" && code != "NotFound" {
			return redact(ctx, e)
		}
	}
	return contracts.Fail("unavailable")
}
func (s *S3) DeleteUnreferenced(ctx context.Context, p catalog.Publication) error {
	if p.State != "retiring" {
		return contracts.Fail("conflict")
	}
	key, e := s.key(p)
	if e != nil {
		return e
	}
	if e = s.core.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{VersionID: p.Version}); e != nil {
		return redact(ctx, e)
	}
	_, e = s.core.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{VersionID: p.Version})
	if code := minio.ToErrorResponse(e).Code; code == "NoSuchKey" || code == "NoSuchVersion" || code == "NotFound" {
		return nil
	}
	return contracts.Fail("unavailable")
}
