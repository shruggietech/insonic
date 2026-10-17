//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shruggietech/insonic/internal/workspace"
	"golang.org/x/sys/windows"
)

type cache struct {
	root      string
	directory windows.Handle
	lock      *os.File
}

func noReparse(name string) error {
	name = filepath.Clean(name)
	if !filepath.IsAbs(name) || filepath.VolumeName(name) == "" {
		return errors.New("setup requires an absolute Windows path")
	}
	for current := name; ; current = filepath.Dir(current) {
		p, err := windows.UTF16PtrFromString(current)
		if err != nil {
			return err
		}
		attributes, err := windows.GetFileAttributes(p)
		if err != nil {
			return err
		}
		if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return errors.New("setup source or state path is redirected")
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}
func pinFile(name string) (*os.File, error) {
	if err := noReparse(name); err != nil {
		return nil, err
	}
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), name)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("setup source is not a regular local file")
	}
	return f, nil
}
func readBounded(name string, limit int64) ([]byte, error) {
	f, err := pinFile(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("setup state exceeds its bound")
	}
	return data, nil
}
func decodeFile(name string, output any) error {
	data, err := readBounded(name, 32<<20)
	if err != nil {
		return err
	}
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if err := d.Decode(output); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing setup state")
	}
	return nil
}
func randomID() (string, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce[:]), nil
}
func atomicJSON(name string, value any) error {
	if err := noReparse(filepath.Dir(name)); err != nil {
		return err
	}
	if _, err := os.Lstat(name); err == nil {
		if err := noReparse(name); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	id, err := randomID()
	if err != nil {
		return err
	}
	temporary := filepath.Join(filepath.Dir(name), "state-"+id+".tmp")
	f, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(value)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	from, err := windows.UTF16PtrFromString(temporary)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func openCache(takeLock bool) (*cache, error) {
	if windows.GetCurrentProcessToken().IsElevated() {
		return nil, errors.New("setup recovery sources must be processed by the ordinary invoking user")
	}
	base, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return nil, err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return nil, err
	}
	if err := noReparse(base); err != nil {
		return nil, err
	}
	c := &cache{root: filepath.Join(base, "insonic-setup")}
	created := false
	if err := os.Mkdir(c.root, 0700); err == nil {
		created = true
	} else if !os.IsExist(err) {
		return nil, err
	}
	if created {
		if err := workspace.SecureDirectory(c.root, true); err != nil {
			return nil, err
		}
	}
	if err := noReparse(c.root); err != nil {
		return nil, err
	}
	if err := workspace.PrivateDirectory(c.root); err != nil {
		return nil, errors.New("setup state directory is not privately owned by this account")
	}
	p, _ := windows.UTF16PtrFromString(c.root)
	c.directory, err = windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*cache, error) { c.close(); return nil, err }
	if takeLock {
		name := filepath.Join(c.root, "setup.lock")
		if _, err := os.Lstat(name); err == nil {
			if err := noReparse(name); err != nil {
				return fail(err)
			}
		} else if !os.IsNotExist(err) {
			return fail(err)
		}
		p, _ := windows.UTF16PtrFromString(name)
		handle, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return fail(errors.New("another setup operation is active; allow it to finish"))
		}
		c.lock = os.NewFile(uintptr(handle), name)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fail(err)
	}
	marker := struct {
		Kind string `json:"kind"`
		SID  string `json:"sid"`
	}{"insonic-setup-cache", user.User.Sid.String()}
	if created {
		if err := atomicJSON(filepath.Join(c.root, "owner.json"), marker); err != nil {
			return fail(err)
		}
	}
	var existing struct {
		Kind string `json:"kind"`
		SID  string `json:"sid"`
	}
	if err := decodeFile(filepath.Join(c.root, "owner.json"), &existing); err != nil || existing != marker {
		return fail(errors.New("setup state ownership marker differs"))
	}
	return c, nil
}
func (c *cache) close() {
	if c.lock != nil {
		_ = c.lock.Close()
		c.lock = nil
	}
	if c.directory != 0 {
		_ = windows.CloseHandle(c.directory)
		c.directory = 0
	}
}
func (c *cache) source(product string) (Source, error) {
	var p Package
	if !guidPattern.MatchString(product) {
		return Source{}, errors.New("invalid retained source identity")
	}
	dir := filepath.Join(c.root, product)
	if err := noReparse(dir); err != nil {
		return Source{}, err
	}
	if err := workspace.PrivateDirectory(dir); err != nil {
		return Source{}, errors.New("retained source directory is not privately owned")
	}
	if err := decodeFile(filepath.Join(dir, "source.json"), &p); err != nil {
		return Source{}, err
	}
	if err := p.Validate(); err != nil {
		return Source{}, err
	}
	if p.Product != product {
		return Source{}, errors.New("retained source belongs to another product")
	}
	return Source{Package: p, Path: filepath.Join(dir, "source.msi")}, nil
}
func (c *cache) retain(p Package, input io.Reader, repair ...bool) (Source, error) {
	if err := p.Validate(); err != nil {
		return Source{}, err
	}
	final := filepath.Join(c.root, p.Product)
	existing := false
	if info, err := os.Lstat(final); err == nil {
		existing = true
		if !info.IsDir() || noReparse(final) != nil {
			return Source{}, errors.New("retained source directory is redirected or unavailable")
		}
		source, readErr := c.source(p.Product)
		if readErr == nil {
			e := &nativeEngine{}
			verifyErr := e.ValidateSource(source)
			e.close()
			if verifyErr == nil {
				if source.Package != p {
					return Source{}, errors.New("a different valid full source is already retained for this product version")
				}
				return source, nil
			}
		}
		if len(repair) != 1 || !repair[0] {
			if readErr != nil {
				return Source{}, readErr
			}
			return Source{}, errors.New("retained source is corrupt; retry with the exact previous MSI and its official SHA256")
		}
		// Explicit original-source repair changes only this account's two owned
		// source files, after a complete replacement has been validated and flushed.
		if err := workspace.SecureDirectory(final); err != nil {
			return Source{}, err
		}
	} else if !os.IsNotExist(err) {
		return Source{}, err
	}

	id, err := randomID()
	if err != nil {
		return Source{}, err
	}
	stage := filepath.Join(c.root, "source-"+id)
	if err := os.Mkdir(stage, 0700); err != nil {
		return Source{}, err
	}
	if err := workspace.SecureDirectory(stage, true); err != nil {
		return Source{}, err
	}
	defer func() {
		_ = os.Remove(filepath.Join(stage, "source.msi"))
		_ = os.Remove(filepath.Join(stage, "source.json"))
		_ = os.Remove(stage)
	}()
	output, err := os.OpenFile(filepath.Join(stage, "source.msi"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Source{}, err
	}
	err = CopyPayload(output, input, p)
	if err == nil {
		err = output.Sync()
	}
	closeErr := output.Close()
	if err != nil {
		return Source{}, err
	}
	if closeErr != nil {
		return Source{}, closeErr
	}
	actual, err := inspectMSI(filepath.Join(stage, "source.msi"))
	if err != nil || actual.Product != p.Product || actual.Upgrade != p.Upgrade || actual.Version != p.Version {
		return Source{}, errors.New("embedded full MSI identity differs")
	}
	if err := atomicJSON(filepath.Join(stage, "source.json"), p); err != nil {
		return Source{}, err
	}
	if existing {
		from, err := windows.UTF16PtrFromString(filepath.Join(stage, "source.msi"))
		if err != nil {
			return Source{}, err
		}
		to, err := windows.UTF16PtrFromString(filepath.Join(final, "source.msi"))
		if err != nil {
			return Source{}, err
		}
		if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
			return Source{}, err
		}
		if err := atomicJSON(filepath.Join(final, "source.json"), p); err != nil {
			return Source{}, err
		}
	} else if err := os.Rename(stage, final); err != nil {
		return Source{}, err
	}
	return c.source(p.Product)
}

// Remove only the two known owned source files. Extra files and user data are
// never recursively removed; redirected or foreign state is left for correction.
func (c *cache) clean(product string) error {
	source, err := c.source(product)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := noReparse(source.Path); err != nil {
		return err
	}
	if err := os.Remove(source.Path); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(filepath.Dir(source.Path), "source.json")); err != nil {
		return err
	}
	if err := os.Remove(filepath.Dir(source.Path)); err != nil {
		return err
	}
	return nil
}
