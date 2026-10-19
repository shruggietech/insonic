// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"golang.org/x/sys/windows"
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
		err := windows.QueryInformationJobObject(tree.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil)
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
			return report, ctx.Err()
		case <-ticker.C:
		}
	}
}
