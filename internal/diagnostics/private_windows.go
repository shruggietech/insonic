// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"os"

	"golang.org/x/sys/windows"
)

func privateFile(file *os.File, created bool) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	return securePrivateFile(file, created, user.User.Sid)
}

func privateHandle(file *os.File, access uint32) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(file.Name())
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateFile(name, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, err
	}
	var original, reopened windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &original) != nil ||
		windows.GetFileInformationByHandle(handle, &reopened) != nil ||
		!samePrivateFile(original, reopened) {
		windows.CloseHandle(handle)
		return 0, logError()
	}
	return handle, nil
}

func samePrivateFile(original, reopened windows.ByHandleFileInformation) bool {
	return original.VolumeSerialNumber == reopened.VolumeSerialNumber &&
		original.FileIndexHigh == reopened.FileIndexHigh && original.FileIndexLow == reopened.FileIndexLow &&
		reopened.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0
}

func securePrivateFile(file *os.File, created bool, user *windows.SID) error {
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() || user == nil {
		return logError()
	}
	access := uint32(windows.READ_CONTROL | windows.WRITE_DAC | windows.FILE_READ_ATTRIBUTES)
	handle, err := privateHandle(file, access)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	existing, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := existing.Owner()
	if err != nil || owner == nil || !created && !owner.Equals(user) {
		return logError()
	}

	// Most new files already have the selected owner. Inherited Modify access
	// does not grant WRITE_OWNER, and no ownership change is needed in that case.
	changeOwner := created && !owner.Equals(user)
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return securePrivateOwner(file, handle, access, changeOwner, user, dacl)
}

func securePrivateOwner(file *os.File, handle windows.Handle, access uint32, changeOwner bool, user *windows.SID, dacl *windows.ACL) error {
	// An ordinary creator can update this new file's DACL while inherited Modify
	// omits WRITE_OWNER. Install the required caller-only ACL before reopening
	// for necessary owner normalization. No diagnostic data is written until both
	// steps succeed, and existing foreign-owned files were already refused.
	flags := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION)
	if err := windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, flags, nil, nil, dacl, nil); err != nil {
		return err
	}
	if !changeOwner {
		return nil
	}
	selected, err := privateHandle(file, access|windows.WRITE_OWNER)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(selected)
	return windows.SetSecurityInfo(selected, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION, user, nil, nil, nil)
}
