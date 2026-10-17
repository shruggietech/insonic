// SPDX-License-Identifier: Apache-2.0
// Package setup coordinates the per-user Windows MSI without owning user data.
package setup

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const Family = "{96D72489-3629-52FB-A30D-79CCD4CAE9BC}"
const FooterMagic = "INSONIC-SETUP-01"
const maxPackageSize = 8 << 30

var guidPattern = regexp.MustCompile(`^\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,2})\.(0|[1-9][0-9]{0,2})\.(0|[1-9][0-9]{0,4})$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Package struct {
	Product string `json:"product_code"`
	Upgrade string `json:"upgrade_code"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size_bytes"`
}

func (p Package) Validate() error {
	var major, minor, patch int
	_, err := fmt.Sscanf(p.Version, "%d.%d.%d", &major, &minor, &patch)
	if !guidPattern.MatchString(p.Product) || p.Upgrade != Family || !versionPattern.MatchString(p.Version) || err != nil || major > 255 || minor > 255 || patch > 65535 || !digestPattern.MatchString(p.SHA256) || p.Size <= 0 || p.Size > maxPackageSize {
		return errors.New("invalid insonic full-package identity")
	}
	return nil
}

// ReadPayload accepts only the bounded overlay authored by package_setup.py.
// The MSI is streamed; the executable and its multi-gigabyte payload are never
// read into one allocation. Package digests identify bytes, not a publisher.
func ReadPayload(file io.ReaderAt, size int64) (Package, *io.SectionReader, error) {
	var p Package
	if size < 96 {
		return p, nil, errors.New("setup payload footer is missing")
	}
	footer := make([]byte, 32)
	if _, err := file.ReadAt(footer, size-32); err != nil {
		return p, nil, err
	}
	length := binary.LittleEndian.Uint64(footer[16:24])
	metadataSize := int64(binary.LittleEndian.Uint32(footer[24:28]))
	if string(footer[:16]) != FooterMagic || binary.LittleEndian.Uint32(footer[28:]) != 0 || length == 0 || length > maxPackageSize || metadataSize <= 0 || metadataSize > 4096 {
		return p, nil, errors.New("invalid setup payload bounds")
	}
	start := size - 32 - metadataSize - int64(length)
	if start < 64 {
		return p, nil, errors.New("setup payload escapes executable")
	}
	metadata := io.NewSectionReader(file, size-32-metadataSize, metadataSize)
	d := json.NewDecoder(metadata)
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, nil, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return p, nil, errors.New("trailing setup metadata")
	}
	if err := p.Validate(); err != nil {
		return p, nil, err
	}
	if p.Size != int64(length) {
		return p, nil, errors.New("setup payload length differs")
	}
	return p, io.NewSectionReader(file, start, p.Size), nil
}
func CopyPayload(output io.Writer, source io.Reader, p Package) error {
	if err := p.Validate(); err != nil {
		return err
	}
	digest := sha256.New()
	n, err := io.Copy(io.MultiWriter(output, digest), io.LimitReader(source, p.Size+1))
	if err != nil {
		return err
	}
	if n != p.Size || hex.EncodeToString(digest.Sum(nil)) != p.SHA256 {
		return errors.New("full MSI source length or SHA256 differs; obtain the selected official package again")
	}
	return nil
}
