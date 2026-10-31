// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
	"strings"
	"testing"
)

func TestResourcePolicyReducesBatchExactlyOnceBeforeCPU(t *testing.T) {
	var devices []string
	var batches []int
	_, got, err := runResourcePolicy(context.Background(), "auto", 8, func(_ context.Context, device string, batch int) ([]byte, error) {
		devices = append(devices, device)
		batches = append(batches, batch)
		if device == "cuda" {
			return nil, contracts.Fail("cuda_resource_exhausted")
		}
		return []byte("ok"), nil
	})
	if err != nil || !reflect.DeepEqual(devices, []string{"cuda", "cuda", "cpu"}) || !reflect.DeepEqual(batches, []int{8, 1, 1}) || !got.Valid() || len(got.ResourceAdjustments) != 1 {
		t.Fatalf("devices=%v batches=%v receipt=%+v error=%v", devices, batches, got, err)
	}
}
func TestResourcePolicyReducedCUDAAndCPUFailuresRetainTruth(t *testing.T) {
	for _, cpu := range []bool{false, true} {
		requested := "auto"
		if cpu {
			requested = "cpu"
		}
		calls := 0
		_, got, err := runResourcePolicy(context.Background(), requested, 4, func(_ context.Context, device string, batch int) ([]byte, error) {
			calls++
			if !cpu && batch == 1 {
				return []byte("ok"), nil
			}
			return nil, contracts.Fail("cuda_resource_exhausted")
		})
		if cpu {
			if err == nil || calls != 1 || got.Device != "" {
				t.Fatalf("CPU: %+v %v", got, err)
			}
		} else if err != nil || calls != 2 || got.Device != "cuda" || !got.Valid() {
			t.Fatalf("CUDA: %+v %v", got, err)
		}
	}
}
func TestResourcePolicyNeverChangesNonResourceFailureOrCancelledWork(t *testing.T) {
	calls := 0
	_, _, err := runResourcePolicy(context.Background(), "auto", 8, func(context.Context, string, int) ([]byte, error) {
		calls++
		return nil, contracts.Fail("checksum_mismatch")
	})
	if err == nil || calls != 1 {
		t.Fatal(calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	_, receipt, err := runResourcePolicy(ctx, "auto", 8, func(context.Context, string, int) ([]byte, error) {
		calls++
		cancel()
		return nil, contracts.Fail("cuda_resource_exhausted")
	})
	if err == nil || calls != 1 || len(receipt.ResourceAdjustments) != 0 {
		t.Fatal(calls, receipt, err)
	}
}

func TestResourcePolicyFailureSanitizesCompleteAdjustmentReceipt(t *testing.T) {
	_, receipt, err := runResourcePolicy(context.Background(), "auto", 4, func(context.Context, string, int) ([]byte, error) {
		return nil, contracts.Fail("cuda_resource_exhausted")
	})
	safe := contracts.SafeError(err)
	if safe.DeviceAttempt == nil || len(safe.DeviceAttempt.ResourceAdjustments) != 1 || len(safe.DeviceAttempt.AttemptedDevices) != 3 || receipt.Device != "" {
		t.Fatal(receipt, safe)
	}
	forged := receipt
	forged.Device = "cpu"
	forged.ResourceAdjustments[0].BatchSize = 2
	if forged.Valid() {
		t.Fatal("forged batch adjustment accepted")
	}
}

func TestResourcePolicyNoticesIncludeSuccessfulBatchReduction(t *testing.T) {
	execution := DeviceExecution{RequestedDevice: "auto", AttemptedDevices: []string{"cuda", "cuda"}, Device: "cuda", Count: 1, ResourceAdjustments: []contracts.ResourceAdjustment{{Attempt: 2, PreviousBatchSize: 8, BatchSize: 1, Reason: "cuda_resource_exhausted"}}}
	notices := DeviceNotices([]DeviceExecution{execution, execution})
	if len(notices) != 1 || !strings.Contains(notices[0], "8 to 1") {
		t.Fatal(notices)
	}
	execution.AttemptedDevices = append(execution.AttemptedDevices, "cpu")
	execution.Device = "cpu"
	execution.FallbackReason = "cuda_resource_exhausted"
	notices = DeviceNotices([]DeviceExecution{execution})
	if len(notices) != 2 || !strings.Contains(notices[1], "completed on CPU") {
		t.Fatal(notices)
	}
}
