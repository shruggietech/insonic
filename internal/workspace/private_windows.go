// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/sys/windows"
	"os"
	"strings"
)

func PrivateDirectory(dir string) error {
	stat, err := os.Lstat(dir)
	if err != nil || !stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
		return contracts.Fail("unavailable")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return contracts.Fail("unavailable")
	}
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	owner, _, err := sd.Owner()
	if err != nil || owner.String() != user.User.Sid.String() {
		return contracts.Fail("unavailable")
	}
	security := sd.String()
	if !strings.Contains(security, "D:P") || !strings.HasSuffix(security, "(A;OICI;FA;;;"+owner.String()+")") || strings.Count(security, "(") != 1 {
		return contracts.Fail("unavailable")
	}
	return nil
}

func SecureDirectory(dir string, newlyCreated ...bool) error {
	stat, err := os.Lstat(dir)
	if err != nil || !stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
		return contracts.Fail("unavailable")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return contracts.Fail("unavailable")
	}
	existing, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	owner, _, err := existing.Owner()
	created := len(newlyCreated) == 1 && newlyCreated[0]
	if err != nil || (!created && owner.String() != user.User.Sid.String()) {
		return contracts.Fail("unavailable")
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return contracts.Fail("unavailable")
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return contracts.Fail("unavailable")
	}
	// Elevated tokens can default new directories to Administrators ownership.
	// Only the successful creator may normalize that owner to the current user.
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, user.User.Sid, nil, dacl, nil); err != nil {
		return contracts.Fail("unavailable")
	}
	return PrivateDirectory(dir)
}
