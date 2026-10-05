package workspace

import (
	"golang.org/x/sys/windows"
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
