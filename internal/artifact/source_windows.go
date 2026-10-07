// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
)

func openNoFollow(source string) (*os.File, error) {
	namePath := filepath.Clean(source)
	if !strings.HasPrefix(namePath, `\\?\`) {
		if strings.HasPrefix(namePath, `\\`) {
			namePath = `\\?\UNC\` + strings.TrimPrefix(namePath, `\\`)
		} else {
			namePath = `\\?\` + namePath
		}
	}
	name, e := windows.UTF16PtrFromString(namePath)
	if e != nil {
		return nil, e
	}
	handle, e := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_SEQUENTIAL_SCAN, 0)
	if e != nil {
		return nil, e
	}
	var info windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(handle, &info); e != nil {
		windows.CloseHandle(handle)
		return nil, e
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return nil, windows.ERROR_CANT_ACCESS_FILE
	}
	return os.NewFile(uintptr(handle), source), nil
}
