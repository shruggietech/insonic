//go:build windows

// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/sys/windows"
)

func modifyOnlyDirectory(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;OICI;0x1301bf;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestPrivateFileInInheritedModifyDirectory(t *testing.T) {
	directory := modifyOnlyDirectory(t)
	file, err := os.OpenFile(filepath.Join(directory, "created.jsonl"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err = privateFile(file, true); err != nil {
		t.Fatalf("securing a newly created current-user file under inherited Modify failed: %T: %v", err, err)
	}
	log, err := Open(filepath.Join(directory, "operator.jsonl"), "info")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
}

func assertPrivateACL(t *testing.T, path string) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || !owner.Equals(user.User.Sid) {
		t.Fatal("unexpected owner", err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("ACL inheritance remains enabled", err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount != 1 {
		t.Fatal("unexpected private ACL", err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err = windows.GetAce(acl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask != 0x1f01ff || !sid.Equals(user.User.Sid) {
		t.Fatal("private ACL grants unexpected access")
	}
}

func TestModifyDirectoryExistingRotationAndContention(t *testing.T) {
	directory := modifyOnlyDirectory(t)
	path := filepath.Join(directory, "rotating.jsonl")
	if err := os.WriteFile(path, make([]byte, MaxBytes-1), 0600); err != nil {
		t.Fatal(err)
	}
	log, err := Open(path, "info")
	if err != nil {
		t.Fatal(err)
	}
	if second, err := Open(path, "info"); err == nil {
		second.Close()
		t.Fatal("concurrent writer admitted")
	} else if !strings.Contains(err.Error(), "already in use") {
		t.Fatal("contention guidance lost", err)
	}
	if err = log.Write("error", StartupEvent("desktop-window", nil)); err != nil {
		t.Fatal(err)
	}
	if err = log.Close(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", ".1", ".lock"} {
		assertPrivateACL(t, path+suffix)
	}
	log, err = Open(path, "info")
	if err != nil {
		t.Fatal("existing private full-control files refused", err)
	}
	if err = log.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateFileRejectsForeignSelectedOwnerWithoutChangingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.jsonl")
	if err := os.WriteFile(path, []byte("retained"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	other, err := windows.StringToSid("S-1-1-0")
	if err != nil {
		t.Fatal(err)
	}
	if err = securePrivateFile(file, false, other); err == nil {
		t.Fatal("foreign existing owner admitted")
	}
	after, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil || before.String() != after.String() {
		t.Fatal("foreign ACL changed", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "retained" {
		t.Fatal("foreign bytes changed", err)
	}
}

func TestPrivateFileRejectsReplacedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.jsonl")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	file := os.NewFile(uintptr(handle), path)
	defer file.Close()
	if err = os.Rename(path, path+".original"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if err = privateFile(file, true); err == nil {
		t.Fatal("replaced file identity admitted")
	}
	after, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || before.String() != after.String() {
		t.Fatal("replacement ACL changed", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "replacement" {
		t.Fatal("replacement bytes changed", err)
	}
}

func TestPrivateIdentityRejectsReparsePoint(t *testing.T) {
	original := windows.ByHandleFileInformation{VolumeSerialNumber: 1, FileIndexHigh: 2, FileIndexLow: 3}
	reopened := original
	reopened.FileAttributes = windows.FILE_ATTRIBUTE_REPARSE_POINT
	if samePrivateFile(original, reopened) {
		t.Fatal("reparse point admitted as regular original file")
	}
}

func TestDiagnosticCreationDeniedInReadOnlyDirectory(t *testing.T) {
	directory := t.TempDir()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(D;OICI;0x6;;;" + user.User.Sid.String() + ")(A;OICI;FR;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		restore, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
		if e != nil {
			t.Error(e)
			return
		}
		dacl, _, e := restore.DACL()
		if e == nil {
			e = windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
		}
		if e != nil {
			t.Error(e)
		}
	})
	withoutFilePrivileges(t, func() {
		if probe, err := os.OpenFile(filepath.Join(directory, "must-not-create"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600); err == nil {
			probe.Close()
			t.Fatal("denied-creation fixture does not deny ordinary creation")
		}
		if log, err := Open(filepath.Join(directory, "denied.jsonl"), "info"); err == nil {
			log.Close()
			t.Fatal("read-only directory admitted")
		} else {
			message := err.Error()
			// Rooted native I/O and ordinary CreateFile can reject different
			// access boundaries under elevated Windows CI tokens. Every route
			// must refuse the log and identify a concrete corrective boundary.
			if contracts.SafeError(err).Code != "unavailable" ||
				!(strings.Contains(message, "opened or created") || strings.Contains(message, "could not be inspected") || strings.Contains(message, "could not be made private")) {
				t.Fatal("actionable access guidance lost", err)
			}
		}
	})
}

// Elevated CI accounts may carry backup/restore privileges. Test an ordinary
// access check on one locked OS thread without changing the process token.
func withoutFilePrivileges(t *testing.T, check func()) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var source, restricted windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &source); err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := windows.DuplicateTokenEx(source, windows.TOKEN_IMPERSONATE|windows.TOKEN_QUERY|windows.TOKEN_ADJUST_PRIVILEGES, nil, windows.SecurityImpersonation, windows.TokenImpersonation, &restricted); err != nil {
		t.Fatal(err)
	}
	defer restricted.Close()
	if err := windows.AdjustTokenPrivileges(restricted, true, nil, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := windows.SetThreadToken(nil, restricted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := windows.RevertToSelf(); err != nil {
			t.Error(err)
		}
	}()
	check()
}

func TestReadOnlyExistingLockReportsCreationAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.jsonl")
	if err := os.WriteFile(path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path+".lock", 0400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path+".lock", 0600); err != nil {
			t.Error(err)
		}
	})
	if log, err := Open(path, "info"); err == nil {
		log.Close()
		t.Fatal("read-only lock admitted")
	} else if !strings.Contains(err.Error(), "opened or created") {
		t.Fatal("creation/open guidance lost", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("log created despite inaccessible lock", err)
	}
}
