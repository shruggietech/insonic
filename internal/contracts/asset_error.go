// SPDX-License-Identifier: Apache-2.0
package contracts

import "strings"

// AssetObservation identifies an elected logical asset, never an owner path.
type AssetObservation struct {
	Role string `json:"role"`
	Name string `json:"name"`
}

var assetRoles = strings.Fields("worker_runtime worker_installer worker_wheel model_file trainer_executable trainer_support trainer_output artifact")
var assetReasons = map[string]struct{ code, message string }{
	"artifact_missing":   {"checksum_mismatch", "is missing from storage. Restore the recorded object from an intact source before retrying."},
	"artifact_identity":  {"checksum_mismatch", "has a different storage version. Restore the exact recorded object version before retrying."},
	"artifact_range":     {"checksum_mismatch", "does not match the requested byte range. Check the selected backend's ranged-read support before retrying."},
	"asset_missing":      {"unavailable", "is missing. Restore the exact elected file and retry."},
	"asset_unreadable":   {"unavailable", "cannot be read. Check local access permissions and retry."},
	"asset_not_regular":  {"unavailable", "is not a regular file. Supply the exact elected file and retry."},
	"asset_size":         {"checksum_mismatch", "has the wrong byte length. Replace it with the complete pinned file and retry."},
	"asset_hash":         {"checksum_mismatch", "does not match its pinned SHA-256. Replace the damaged or incorrect file and retry."},
	"asset_tool_missing": {"unavailable", "requires an unavailable local tool. Restore the elected tool and retry."},
}

// ValidAssetName bounds generated logical names across Windows and Unix hosts.
func ValidAssetName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if r < 33 || r > 126 || strings.ContainsRune(`\:"<>|?*`, r) {
			return false
		}
	}
	if strings.HasPrefix(name, "/") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") {
			return false
		}
		base := strings.ToUpper(strings.Split(part, ".")[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
			return false
		}
	}
	return true
}

func AssetFailure(reason, role, name string) *Error {
	rule, ok := assetReasons[reason]
	if !ok {
		return Fail("operation_failed")
	}
	e := Condition(rule.code, reason)
	validRole := false
	for _, allowed := range assetRoles {
		if role == allowed {
			validRole = true
			break
		}
	}
	if validRole && ValidAssetName(name) {
		e.Asset = &AssetObservation{Role: role, Name: name}
		e.Message = "The elected " + role + " asset " + name + " " + rule.message
	}
	return e
}
