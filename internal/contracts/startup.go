// SPDX-License-Identifier: Apache-2.0
package contracts

// StartupReport contains generated conditions only. It never carries argv,
// environment, stderr, file paths, protected input or arbitrary error text.
type StartupReport struct {
	Kind        string `json:"kind"`
	Version     string `json:"schema_version"`
	StartupID   string `json:"startup_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	State       string `json:"state"`
	Stage       string `json:"stage"`
	Code        string `json:"code,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

func ValidOperatorReason(reason string) bool { _, ok := operatorReasons[reason]; return ok }
