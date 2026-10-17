//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"unsafe"

	"github.com/shruggietech/insonic/internal/workspace"
	"golang.org/x/sys/windows"
)

const inventoryProperty = "INSONIC_INVENTORY_SHA256"

func inventoryDigest(before Choices) (string, error) {
	var digest string
	for _, file := range before.Files {
		if file.Path == "package-inventory.json" {
			if digest != "" || !digestPattern.MatchString(file.SHA256) {
				return "", errors.New("previous inventory identity is ambiguous")
			}
			digest = file.SHA256
		}
	}
	if digest == "" {
		return "", errors.New("previous inventory identity is missing")
	}
	return digest, nil
}

// ValidateInventory binds the full source to the installed bytes before any
// advertised-product removal. Legacy packages lack the new fingerprint, so
// inspect only their embedded inventory through read-only MSI/archive APIs.
func (e *nativeEngine) ValidateInventory(source Source, before Choices) error {
	expected, err := inventoryDigest(before)
	if err != nil {
		return err
	}
	db, err := openDatabase(source.Path)
	if err != nil {
		return err
	}
	defer closeMSI(db)
	value, present, err := optionalDatabaseValue(db, "SELECT `Value` FROM `Property` WHERE `Property`='"+inventoryProperty+"'")
	if err != nil {
		return err
	}
	if present {
		if value != expected {
			return errors.New("previous MSI inventory differs from the installed release; obtain its exact original full MSI")
		}
		return nil
	}
	return legacyInventory(db, e.logDirectory, expected)
}

func optionalDatabaseValue(db uint32, query string) (string, bool, error) {
	var view, record uint32
	if err := checked("MsiDatabaseOpenViewW", db, query, &view); err != nil {
		return "", false, err
	}
	defer closeMSI(view)
	if err := checked("MsiViewExecute", view, 0); err != nil {
		return "", false, err
	}
	code, err := native("MsiViewFetch", view, &record)
	if err != nil {
		return "", false, err
	}
	if code == 259 {
		return "", false, nil
	}
	if code != 0 {
		return "", false, fmt.Errorf("source inventory query returned %d", code)
	}
	defer closeMSI(record)
	buffer := make([]uint16, 128)
	size := uint32(len(buffer))
	if err := checked("MsiRecordGetStringW", record, 1, &buffer[0], &size); err != nil {
		return "", false, err
	}
	return windows.UTF16ToString(buffer[:size]), true, nil
}

func exportCabinet(db uint32, output *os.File) error {
	var view, record uint32
	if err := checked("MsiDatabaseOpenViewW", db, "SELECT `Data` FROM `_Streams` WHERE `Name`='payload.cab'", &view); err != nil {
		return err
	}
	defer closeMSI(view)
	if err := checked("MsiViewExecute", view, 0); err != nil {
		return err
	}
	if err := checked("MsiViewFetch", view, &record); err != nil {
		return err
	}
	defer closeMSI(record)
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		size := uint32(len(buffer))
		if err := checked("MsiRecordReadStream", record, 1, &buffer[0], &size); err != nil {
			return err
		}
		if size > uint32(len(buffer)) {
			return errors.New("source cabinet stream exceeds its buffer")
		}
		if size == 0 {
			break
		}
		total += int64(size)
		if total > maxPackageSize {
			return errors.New("source cabinet exceeds setup package bound")
		}
		if _, err := output.Write(buffer[:size]); err != nil {
			return err
		}
	}
	if total == 0 {
		return errors.New("source cabinet is empty")
	}
	return output.Sync()
}

type cabinetFile struct {
	Name                   *uint16
	Size                   uint32
	Error                  uint32
	Date, Time, Attributes uint16
	Target                 [260]uint16
}

func legacyInventory(db uint32, directory, expected string) error {
	// A selected legacy download may be read-only or on removable storage.
	// Scratch data always belongs in the private application cache, not beside it.
	if directory == "" {
		return errors.New("private source-inspection scratch directory is unavailable")
	}
	if err := noReparse(directory); err != nil {
		return err
	}
	if err := workspace.PrivateDirectory(directory); err != nil {
		return err
	}
	cab, err := os.CreateTemp(directory, "cabinet-*.cab")
	if err != nil {
		return err
	}
	defer os.Remove(cab.Name())
	err = exportCabinet(db, cab)
	closeErr := cab.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	pinned, err := pinFile(cab.Name())
	if err != nil {
		return err
	}
	defer pinned.Close()
	output, err := os.CreateTemp(directory, "inventory-*.json")
	if err != nil {
		return err
	}
	targetName := output.Name()
	if err := output.Close(); err != nil {
		return err
	}
	defer os.Remove(targetName)
	target, err := windows.UTF16FromString(targetName)
	if err != nil || len(target) > 260 {
		return errors.New("legacy inventory scratch path exceeds the Windows cabinet API bound")
	}
	sum := sha256.Sum256([]byte("package-inventory.json"))
	member := "f" + hex.EncodeToString(sum[:])[:28]
	found := false
	var callbackErr error
	var count uint32
	var expanded uint64
	callback := windows.NewCallback(func(_ uintptr, notification uint32, param1 unsafe.Pointer, param2 uintptr) uintptr {
		switch notification {
		case 0x11: // SPFILENOTIFY_FILEINCABINET
			info := (*cabinetFile)(param1)
			count++
			expanded += uint64(info.Size)
			if count > 100000 || expanded > uint64(maxPackageSize)*4 {
				callbackErr = errors.New("legacy cabinet inventory exceeds package bounds")
				return 0
			}
			if windows.UTF16PtrToString(info.Name) != member {
				return 2
			} // FILEOP_SKIP
			if found || info.Size == 0 || info.Size > 32<<20 {
				callbackErr = errors.New("legacy inventory member is duplicate or exceeds its bound")
				return 0
			}
			found = true
			copy(info.Target[:], target)
			return 1 // FILEOP_DOIT, exact private target only
		case 0x12: // SPFILENOTIFY_NEEDNEWCABINET: never follow another cabinet.
			callbackErr = errors.New("legacy full MSI unexpectedly requires another cabinet")
			return 1
		case 0x13: // SPFILENOTIFY_FILEEXTRACTED, FILEPATHS.Win32Error follows two pointers.
			type filePaths struct {
				Target, Source *uint16
				Error, Flags   uint32
			}
			if code := (*filePaths)(param1).Error; code != 0 {
				callbackErr = fmt.Errorf("legacy inventory extraction returned %d", code)
				return uintptr(code)
			}
		}
		return 0
	})
	proc := windows.NewLazySystemDLL("setupapi.dll").NewProc("SetupIterateCabinetW")
	if err := proc.Find(); err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(cab.Name())
	if err != nil {
		return err
	}
	ok, _, callErr := proc.Call(uintptr(unsafe.Pointer(name)), 0, callback, 0)
	runtime.KeepAlive(name)
	runtime.KeepAlive(target)
	if callbackErr != nil {
		return callbackErr
	}
	if ok == 0 {
		return fmt.Errorf("legacy inventory cabinet inspection failed: %w", callErr)
	}
	if !found {
		return errors.New("legacy full MSI inventory is missing")
	}
	f, err := pinFile(targetName)
	if err != nil {
		return err
	}
	defer f.Close()
	digest := sha256.New()
	n, err := io.Copy(digest, io.LimitReader(f, (32<<20)+1))
	if err != nil {
		return err
	}
	if n > 32<<20 || hex.EncodeToString(digest.Sum(nil)) != expected {
		return errors.New("legacy full MSI inventory differs from the installed release")
	}
	return nil
}
