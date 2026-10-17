//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var installer = windows.NewLazySystemDLL("msi.dll")

// Keep UTF-16 allocations and typed output pointers alive across each syscall.
func native(name string, args ...any) (uint32, error) {
	proc := installer.NewProc(name)
	if err := proc.Find(); err != nil {
		return 0, err
	}
	values := make([]uintptr, len(args))
	var strings16 []*uint16
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			p, err := windows.UTF16PtrFromString(v)
			if err != nil {
				return 0, err
			}
			strings16 = append(strings16, p)
			values[i] = uintptr(unsafe.Pointer(p))
		case uint32:
			values[i] = uintptr(v)
		case int:
			values[i] = uintptr(v)
		case *uint32:
			values[i] = uintptr(unsafe.Pointer(v))
		case *uint16:
			values[i] = uintptr(unsafe.Pointer(v))
		default:
			return 0, errors.New("invalid native MSI argument")
		}
	}
	r, _, _ := proc.Call(values...)
	runtime.KeepAlive(strings16)
	runtime.KeepAlive(args)
	return uint32(r), nil
}
func checked(name string, args ...any) error {
	code, err := native(name, args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("%s returned %d", name, code)
	}
	return nil
}
func closeMSI(handle uint32) {
	if handle != 0 {
		_, _ = native("MsiCloseHandle", handle)
	}
}
func openDatabase(source string) (uint32, error) {
	var db uint32
	err := checked("MsiOpenDatabaseW", source, 0, &db)
	return db, err
}
func databaseValue(db uint32, query string) (string, error) {
	var view, record uint32
	if err := checked("MsiDatabaseOpenViewW", db, query, &view); err != nil {
		return "", err
	}
	defer closeMSI(view)
	if err := checked("MsiViewExecute", view, 0); err != nil {
		return "", err
	}
	if err := checked("MsiViewFetch", view, &record); err != nil {
		return "", err
	}
	defer closeMSI(record)
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := checked("MsiRecordGetStringW", record, 1, &buffer[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buffer[:size]), nil
}
func inspectMSI(source string) (Package, error) {
	p := Package{}
	db, err := openDatabase(source)
	if err != nil {
		return p, err
	}
	defer closeMSI(db)
	values := map[string]*string{"ProductCode": &p.Product, "UpgradeCode": &p.Upgrade, "ProductVersion": &p.Version}
	for name, target := range values {
		value, err := databaseValue(db, "SELECT `Value` FROM `Property` WHERE `Property`='"+name+"'")
		if err != nil {
			return p, err
		}
		*target = value
	}
	for name, expected := range map[string]string{"ProductName": "insonic", "Manufacturer": "ShruggieTech", "ProductLanguage": "1033"} {
		value, err := databaseValue(db, "SELECT `Value` FROM `Property` WHERE `Property`='"+name+"'")
		if err != nil || value != expected {
			return p, errors.New("source is not the selected per-user insonic MSI")
		}
	}
	if value, err := databaseValue(db, "SELECT `Cabinet` FROM `Media`"); err != nil || value != "#payload.cab" {
		return p, errors.New("recovery requires a complete embedded MSI cabinet")
	}
	if _, err := databaseValue(db, "SELECT `Name` FROM `_Streams` WHERE `Name`='payload.cab'"); err != nil {
		return p, errors.New("full MSI recovery cabinet is missing")
	}
	return p, nil
}
func productInfo(product, property string) (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := checked("MsiGetProductInfoExW", product, user.User.Sid.String(), 2, property, &buffer[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buffer[:size]), nil
}
func installedProduct() (string, error) {
	var selected string
	for index := uint32(0); index < 64; index++ {
		buffer := make([]uint16, 39)
		code, err := native("MsiEnumRelatedProductsW", Family, 0, index, &buffer[0])
		if err != nil {
			return "", err
		}
		if code == 259 {
			return selected, nil
		}
		if code != 0 {
			return "", fmt.Errorf("related product enumeration returned %d", code)
		}
		product := windows.UTF16ToString(buffer)
		if !guidPattern.MatchString(product) {
			return "", errors.New("invalid related product identity")
		}
		if _, err := productInfo(product, "VersionString"); err != nil {
			continue
		} // Only this user's unmanaged per-user registration.
		if selected != "" {
			return "", errors.New("multiple insonic registrations require correction before setup")
		}
		selected = product
	}
	return "", errors.New("related product enumeration exceeds bound")
}

type nativeEngine struct {
	pinned       *os.File
	logDirectory string
}

func (e *nativeEngine) State(product string) int32 {
	code, err := native("MsiQueryProductStateW", product)
	if err != nil {
		return -2
	}
	return int32(code)
}
func (e *nativeEngine) ValidateSource(source Source) error {
	if e.pinned != nil {
		_ = e.pinned.Close()
		e.pinned = nil
	}
	f, err := pinFile(source.Path)
	if err != nil {
		return err
	}
	if err = CopyPayload(io.Discard, f, source.Package); err != nil {
		f.Close()
		return err
	}
	actual, err := inspectMSI(source.Path)
	if err != nil || actual.Product != source.Package.Product || actual.Version != source.Package.Version || actual.Upgrade != source.Package.Upgrade {
		f.Close()
		return errors.New("local full MSI identity differs from retained source record")
	}
	e.pinned = f
	return nil
}
func (e *nativeEngine) close() {
	if e.pinned != nil {
		_ = e.pinned.Close()
		e.pinned = nil
	}
}
func (e *nativeEngine) logging(name string) error {
	target := filepath.Join(e.logDirectory, name+".log")
	if err := noReparse(e.logDirectory); err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		if err := noReparse(target); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return checked("MsiEnableLogW", 0x1000, target, 0)
}
func (e *nativeEngine) Remove(product string) uint32 {
	if e.logging("recovery-remove") != nil {
		return 1603
	}
	code, err := native("MsiConfigureProductExW", product, 0, 2, "REMOVE=ALL LAUNCH=0 OPEN_DOCUMENTATION=0 REBOOT=ReallySuppress")
	if err != nil {
		return 1603
	}
	return code
}
func (e *nativeEngine) Install(source Source, properties string) uint32 {
	if e.logging("recovery-install") != nil {
		return 1603
	}
	code, err := native("MsiInstallProductW", source.Path, properties)
	if err != nil {
		return 1603
	}
	return code
}
func snapshot(product string) (Choices, error) {
	c := Choices{Product: product}
	var err error
	if (&nativeEngine{}).State(product) != 5 {
		return c, errors.New("previous product is not fully installed; repair it before upgrading")
	}
	c.Version, err = productInfo(product, "VersionString")
	if err != nil {
		return c, err
	}
	for _, feature := range []string{"Core", "Desktop", "CLI"} {
		state, err := native("MsiQueryFeatureStateW", product, feature)
		if err != nil || ((state != 3 && state != 2) || feature == "Core" && state != 3) {
			return c, errors.New("previous feature registration is incomplete")
		}
		if feature == "Desktop" {
			c.Desktop = state == 3
		}
		if feature == "CLI" {
			c.CLI = state == 3
		}
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\ShruggieTech\insonic`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return c, err
	}
	defer key.Close()
	c.Directory, _, err = key.GetStringValue("InstallDir")
	if err != nil {
		return c, err
	}
	c.Directory = filepath.Clean(c.Directory)
	if !filepath.IsAbs(c.Directory) || len(filepath.VolumeName(c.Directory)) != 2 || noReparse(c.Directory) != nil {
		return c, errors.New("previous program directory is unavailable or redirected")
	}
	for name, target := range map[string]*bool{"StartMenu": &c.StartMenu, "DesktopShortcut": &c.DesktopShortcut} {
		v, _, err := key.GetStringValue(name)
		if err != nil || (v != "0" && v != "1") {
			return c, errors.New("previous shortcut choices are incomplete")
		}
		*target = v == "1"
	}
	if _, err := choiceProperties(c); err != nil {
		return c, err
	}
	marker, err := readBounded(filepath.Join(c.Directory, "insonic-msi-ownership.json"), 1024)
	if err != nil || string(marker) != "{\"family\":\"insonic\",\"kind\":\"windows-msi-installation\"}\n" {
		return c, errors.New("previous program ownership marker differs")
	}
	data, err := readBounded(filepath.Join(c.Directory, "package-inventory.json"), 32<<20)
	if err != nil {
		return c, err
	}
	var inv struct {
		Kind     string `json:"kind"`
		Version  string `json:"schema_version"`
		Platform string `json:"platform"`
		Files    []File `json:"files"`
	}
	if json.Unmarshal(data, &inv) != nil || inv.Kind != "native-package-inventory" || inv.Version != c.Version || inv.Platform != "windows_amd64" || len(inv.Files) == 0 || len(inv.Files) > 100000 {
		return c, errors.New("previous installed inventory differs")
	}
	for _, f := range inv.Files {
		if f.Path == "insonic-desktop.exe" && !c.Desktop {
			continue
		}
		c.Files = append(c.Files, f)
	}
	for name, bytes := range map[string][]byte{"package-inventory.json": data, "insonic-msi-ownership.json": marker} {
		sum := sha256.Sum256(bytes)
		c.Files = append(c.Files, File{Path: name, Size: int64(len(bytes)), SHA256: hex.EncodeToString(sum[:])})
	}
	return c, verifyFiles(c)
}
func filePath(c Choices, f File) (string, error) {
	if f.Path == "" || strings.ContainsAny(f.Path, "\\:\x00") || strings.HasPrefix(f.Path, "/") || !digestPattern.MatchString(f.SHA256) || f.Size < 0 {
		return "", errors.New("invalid installed inventory path")
	}
	rel := filepath.FromSlash(f.Path)
	if filepath.Clean(rel) != rel || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("installed inventory escapes program directory")
	}
	target := filepath.Join(c.Directory, rel)
	if err := noReparse(target); err != nil {
		return "", err
	}
	return target, nil
}
func verifyFiles(c Choices) error {
	if !filepath.IsAbs(c.Directory) || len(c.Files) == 0 {
		return errors.New("previous byte inventory is incomplete")
	}
	seen := map[string]bool{}
	for _, f := range c.Files {
		target, err := filePath(c, f)
		if err != nil {
			return err
		}
		key := strings.ToLower(target)
		if seen[key] {
			return errors.New("duplicate previous inventory path")
		}
		seen[key] = true
		input, err := pinFile(target)
		if err != nil {
			return err
		}
		digest := sha256.New()
		n, copyErr := io.Copy(digest, input)
		closeErr := input.Close()
		if copyErr != nil || closeErr != nil || n != f.Size || hex.EncodeToString(digest.Sum(nil)) != f.SHA256 {
			return errors.New("previous owned program bytes differ; repair the previous version before upgrading")
		}
	}
	return nil
}
func (e *nativeEngine) Verify(before Choices) error {
	after, err := snapshot(before.Product)
	if err != nil {
		return err
	}
	if after.Version != before.Version || !strings.EqualFold(after.Directory, before.Directory) || after.CLI != before.CLI || after.Desktop != before.Desktop || after.StartMenu != before.StartMenu || after.DesktopShortcut != before.DesktopShortcut {
		return errors.New("previous installation choices were not restored")
	}
	return verifyFiles(before)
}
func apartment() (func(), error) {
	runtime.LockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	initialize := ole.NewProc("CoInitializeEx")
	uninitialize := ole.NewProc("CoUninitialize")
	if initialize.Find() != nil || uninitialize.Find() != nil {
		runtime.UnlockOSThread()
		return nil, errors.New("Windows setup COM initialization is unavailable")
	}
	hr, _, _ := initialize.Call(0, 2)
	if int32(hr) < 0 {
		runtime.UnlockOSThread()
		return nil, errors.New("Windows setup COM initialization failed")
	}
	return func() { _, _ = native("MsiEnableLogW", 0, 0, 0); uninitialize.Call(); runtime.UnlockOSThread() }, nil
}
