// SPDX-License-Identifier: Apache-2.0
package contracts

// DeviceAttempt records only attempted work. A failed operation cannot claim an
// actual successful device, and child-provided text never belongs in this field.
type ResourceAdjustment struct {
	Attempt           int    `json:"attempt"`
	PreviousBatchSize int    `json:"previous_batch_size"`
	BatchSize         int    `json:"batch_size"`
	Reason            string `json:"reason"`
}

type DeviceAttempt struct {
	ResourceAdjustments []ResourceAdjustment `json:"resource_adjustments,omitempty"`
	RequestedDevice     string               `json:"requested_device"`
	AttemptedDevices    []string             `json:"attempted_devices"`
	FallbackReason      string               `json:"fallback_reason,omitempty"`
}

func (a DeviceAttempt) Valid() bool {
	if len(a.ResourceAdjustments) > 0 {
		if len(a.ResourceAdjustments) != 1 || len(a.AttemptedDevices) < 2 || len(a.AttemptedDevices) > 3 || a.AttemptedDevices[0] != "cuda" || a.AttemptedDevices[1] != "cuda" || (a.RequestedDevice != "auto" && a.RequestedDevice != "cuda") {
			return false
		}
		x := a.ResourceAdjustments[0]
		if x.Attempt != 2 || x.PreviousBatchSize < 2 || x.PreviousBatchSize > 64 || x.BatchSize != 1 || x.Reason != "cuda_resource_exhausted" {
			return false
		}
		a.AttemptedDevices = append([]string(nil), a.AttemptedDevices[1:]...)
		a.ResourceAdjustments = nil
		return a.Valid()
	}
	fallback := false
	switch a.FallbackReason {
	case "cuda_no_device", "cuda_library_missing", "cuda_library_incompatible", "cuda_compute_unsupported", "cuda_resource_exhausted":
		fallback = true
	case "":
	default:
		return false
	}
	if a.RequestedDevice == "cpu" {
		return len(a.AttemptedDevices) == 1 && a.AttemptedDevices[0] == "cpu" && !fallback
	}
	if a.RequestedDevice != "auto" && a.RequestedDevice != "cuda" {
		return false
	}
	if len(a.AttemptedDevices) == 1 {
		// Cancellation can follow a classified CUDA failure before CPU starts.
		return a.AttemptedDevices[0] == "cuda"
	}
	return len(a.AttemptedDevices) == 2 && a.AttemptedDevices[0] == "cuda" && a.AttemptedDevices[1] == "cpu" && fallback
}

func safeDeviceAttempt(a *DeviceAttempt) *DeviceAttempt {
	if a == nil || !a.Valid() {
		return nil
	}
	return &DeviceAttempt{ResourceAdjustments: append([]ResourceAdjustment(nil), a.ResourceAdjustments...), RequestedDevice: a.RequestedDevice, AttemptedDevices: append([]string(nil), a.AttemptedDevices...), FallbackReason: a.FallbackReason}
}

// WithDeviceAttempt preserves the sanitized cause and copies a bounded receipt.
// Empty or inconsistent observations are omitted instead of becoming authority.
func WithDeviceAttempt(err error, attempt DeviceAttempt) *Error {
	if err == nil {
		return nil
	}
	out := SafeError(err)
	out.DeviceAttempt = safeDeviceAttempt(&attempt)
	return out
}
