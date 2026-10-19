// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

// JOB_OBJECT_BASIC_ACCOUNTING_INFORMATION from winnt.h (four LARGE_INTEGER
// counters followed by four DWORD counters). x/sys does not expose this type.
type jobAccounting struct {
	TotalUser, TotalKernel, PeriodUser, PeriodKernel int64
	PageFaults, Total, ActiveProcesses, Terminated   uint32
}

// Keep the job open while descendants exit normally. Closing KILL_ON_JOB_CLOSE
// is cleanup, not an observation that a browser released its profile.
func waitDescendants(ctx context.Context, tree *processTree) (DescendantReport, error) {
	var report DescendantReport
	first := true
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var info jobAccounting
		err := queryJobInformation(tree.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
		if err != nil {
			return report, err
		}
		report.Remaining = info.ActiveProcesses
		if first {
			report.Initial = report.Remaining
			first = false
		}
		if report.Remaining == 0 {
			report.Released = true
			return report, nil
		}
		select {
		case <-ctx.Done():
			report.Processes = inspectDescendants(tree)
			return report, ctx.Err()
		case <-ticker.C:
		}
	}
}

func inspectDescendants(tree *processTree) []DescendantProcess {
	type processList struct {
		Assigned, Listed uint32
		IDs              [128]uintptr
	}
	var list processList
	if queryJobInformation(tree.job, windows.JobObjectBasicProcessIdList, uintptr(unsafe.Pointer(&list)), uint32(unsafe.Sizeof(list))) != nil || list.Listed > 128 {
		return nil
	}
	values := make([]DescendantProcess, 0, list.Listed)
	isInJob := windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")
	for _, id := range list.IDs[:list.Listed] {
		entry := DescendantProcess{PID: uint32(id), Image: "unknown"}
		handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(id))
		if err == nil {
			var owned uint32
			ok, _, _ := isInJob.Call(uintptr(handle), uintptr(tree.job), uintptr(unsafe.Pointer(&owned)))
			if ok != 0 && owned != 0 {
				state, waitErr := windows.WaitForSingleObject(handle, 0)
				entry.Observed = waitErr == nil
				entry.Running = waitErr == nil && state == uint32(windows.WAIT_TIMEOUT)
				var name [1024]uint16
				length := uint32(len(name))
				if windows.QueryFullProcessImageName(handle, 0, &name[0], &length) == nil {
					image := strings.ToLower(filepath.Base(windows.UTF16ToString(name[:length])))
					switch image {
					case "msedgewebview2.exe":
						entry.Image = "webview"
					case "insonic.exe":
						entry.Image = "cli"
					case "insonic-desktop.exe":
						entry.Image = "desktop"
					case "werfault.exe", "wermgr.exe":
						entry.Image = "windows-error-reporting"
					case "conhost.exe":
						entry.Image = "console-host"
					default:
						if strings.HasSuffix(image, ".test.exe") {
							entry.Image = "test"
						} else {
							entry.Image = "other"
						}
					}
				}
			}
			windows.CloseHandle(handle)
		}
		values = append(values, entry)
	}
	return values
}
