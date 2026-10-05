// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
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
	if !privateSecurity(sd, user.User.Sid) {
		return contracts.Fail("unavailable")
	}
	return nil
}

func privateSecurity(sd *windows.SECURITY_DESCRIPTOR, user *windows.SID) bool {
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || !owner.Equals(user) {
		return false
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return false
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		return false
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if windows.GetAce(dacl, 0, &ace) != nil || ace == nil {
		return false
	}
	// FILE_ALL_ACCESS with inheritable current-user access, no inherited ACEs.
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE ||
		ace.Header.AceFlags != windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE ||
		ace.Mask != 0x001F01FF {
		return false
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	return sid.Equals(user)
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
