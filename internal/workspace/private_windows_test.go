package workspace

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateSecurityUsesBinaryIdentityAndAccess(t *testing.T) {
	// Administrator RIDs may render as SDDL aliases instead of their full SID.
	sid, err := windows.StringToSid("S-1-5-21-1-2-3-500")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		dacl  string
		valid bool
	}{
		{"D:P(A;OICI;FA;;;" + sid.String() + ")", true},
		{"D:PAI(A;OICI;0x001F01FF;;;" + sid.String() + ")", true},
		{"D:(A;OICI;FA;;;" + sid.String() + ")", false},
		{"D:P(A;OICI;FA;;;" + sid.String() + ")(A;OICI;FA;;;WD)", false},
		{"D:P(A;OICI;FR;;;" + sid.String() + ")", false},
		{"D:P(A;OICI;FA;;;WD)", false},
	} {
		sd, err := windows.SecurityDescriptorFromString("O:" + sid.String() + scenario.dacl)
		if err != nil {
			t.Fatal(err)
		}
		if privateSecurity(sd, sid) != scenario.valid {
			t.Fatal("binary access boundary")
		}
	}
}

func TestSecureAlreadyOwnedDirectoryWithoutOwnershipTransfer(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "owned-modify")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		t.Fatal(e)
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;0x001301bf;;;" + user.User.Sid.String() + ")")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
	if e = SecureDirectory(dir, false); e != nil {
		t.Fatal("owned writable directory should need DACL normalization, not an owner transfer", e)
	}
	if e = PrivateDirectory(dir); e != nil {
		t.Fatal("private owner/ACL invariant lost", e)
	}
}

func TestSecurePrivateDirectoryDoesNotReapplyPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(dir, true); err != nil {
		t.Fatal(err)
	}
	writes := 0
	err := secureDirectory(dir, false, func(windows.SECURITY_INFORMATION, *windows.SID, *windows.ACL) error {
		writes++
		return errors.New("unexpected permission rewrite")
	})
	if err != nil || writes != 0 {
		t.Fatalf("already-private directory requires validation only: writes=%d error=%v", writes, err)
	}
	if err := PrivateDirectory(dir); err != nil {
		t.Fatal("private authority changed", err)
	}
}

func TestSecureDirectoryRechecksChangedPermissionsAndRetainsWriteFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "changed")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := SecureDirectory(dir, true); err != nil {
				t.Fatal(err)
			}
			user, err := windows.GetCurrentProcessToken().GetTokenUser()
			if err != nil {
				t.Fatal(err)
			}
			descriptor, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FR;;;WD)")
			if err != nil {
				t.Fatal(err)
			}
			acl, _, err := descriptor.DACL()
			if err != nil {
				t.Fatal(err)
			}
			if err = windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
				t.Fatal(err)
			}
			if PrivateDirectory(dir) == nil {
				t.Fatal("changed access unexpectedly remained private")
			}
			sentinel := errors.New("controlled normalization failure")
			writes := 0
			err = secureDirectory(dir, false, func(flags windows.SECURITY_INFORMATION, owner *windows.SID, normalized *windows.ACL) error {
				writes++
				if flags&windows.OWNER_SECURITY_INFORMATION != 0 || owner != nil {
					t.Fatal("already-owned directory requested ownership transfer")
				}
				if fail {
					return sentinel
				}
				return windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, flags, owner, nil, normalized, nil)
			})
			if writes != 1 {
				t.Fatal("changed ACL did not require exactly one normalization", writes)
			}
			if fail {
				if err == nil || PrivateDirectory(dir) == nil {
					t.Fatal("failed normalization became accepted authority", err)
				}
			} else if err != nil || PrivateDirectory(dir) != nil {
				t.Fatal("changed ACL did not recover private authority", err)
			}
		})
	}
}
