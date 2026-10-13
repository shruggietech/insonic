package workspace

import (
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
