// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
)

const Version = "0.0.0"

// MaxWorkPayload bounds normalized durable input independently of the smaller
// response frame budget. Import manifests and runtime requests share this limit.
const MaxWorkPayload = 8 << 20

func ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("secure randomness unavailable")
	}
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidID(s string) bool { return uuid.MatchString(s) }

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func Fail(code string) *Error {
	message := map[string]string{
		"invalid_request": "The request is invalid.", "incompatible_version": "The contract version is unsupported.",
		"workspace_mismatch": "The request belongs to another workspace.", "not_found": "The item is unavailable in this workspace.",
		"conflict": "Another owner or attempt already controls this operation.", "unavailable": "The selected dependency is unavailable.",
		"cancelled": "The operation was cancelled.", "output_limit": "The child output limit was exceeded.", "operation_failed": "The operation failed.",
	}[code]
	if message == "" {
		code = "operation_failed"
		message = "The operation failed."
	}
	return &Error{code, message}
}

type Request struct {
	Kind            string          `json:"kind"`
	Version         string          `json:"schema_version"`
	WorkspaceID     string          `json:"workspace_id"`
	RequestID       string          `json:"request_id"`
	Operation       string          `json:"operation"`
	JobID           string          `json:"job_id,omitempty"`
	DurationMS      int             `json:"duration_ms,omitempty"`
	AfterGeneration int64           `json:"after_generation,omitempty"`
	PublicationID   string          `json:"publication_id,omitempty"`
	SourcePath      string          `json:"source_path,omitempty"`
	ArtifactKind    string          `json:"artifact_kind,omitempty"`
	LeaseID         string          `json:"lease_id,omitempty"`
	ReferenceID     string          `json:"reference_id,omitempty"`
	MaxBytes        int64           `json:"max_bytes,omitempty"`
	ItemID          string          `json:"item_id,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
}
type Response struct {
	Kind        string `json:"kind"`
	Version     string `json:"schema_version"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	SessionID   string `json:"runtime_session_id,omitempty"`
	Result      any    `json:"result,omitempty"`
	Error       *Error `json:"error,omitempty"`
}

// Implemented streaming storage and typed catalog interfaces live in their
// domain packages; secret values remain behind this separate private boundary.
type SecretProvider interface {
	Resolve(context.Context, string) ([]byte, error)
	Status(context.Context, string) (string, error)
}
type Capability struct {
	AdapterID       string   `json:"adapter_id"`
	ContractVersion string   `json:"contract_version"`
	State           string   `json:"state"`
	Operations      []string `json:"operations"`
}
