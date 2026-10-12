// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"golang.org/x/sys/windows"
	"os"
)

func privateFile(file *os.File, created bool) error {
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		return logError()
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	original := windows.Handle(file.Fd())
	name, err := windows.UTF16PtrFromString(file.Name())
	if err != nil {
		return err
	}
	access := uint32(windows.READ_CONTROL | windows.WRITE_DAC | windows.FILE_READ_ATTRIBUTES)
	if created {
		access |= windows.WRITE_OWNER
	}
	handle, err := windows.CreateFile(name, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	var first, second windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(original, &first) != nil || windows.GetFileInformationByHandle(handle, &second) != nil || first.VolumeSerialNumber != second.VolumeSerialNumber || first.FileIndexHigh != second.FileIndexHigh || first.FileIndexLow != second.FileIndexLow || second.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return logError()
	}
	existing, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := existing.Owner()
	if err != nil || owner == nil || !created && !owner.Equals(user.User.Sid) {
		return logError()
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	flags := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION)
	var selectedOwner *windows.SID
	if created {
		flags |= windows.OWNER_SECURITY_INFORMATION
		selectedOwner = user.User.Sid
	}
	return windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, flags, selectedOwner, nil, dacl, nil)
}
