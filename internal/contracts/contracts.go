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
		"workspace_mismatch": "The request belongs to another workspace.", "not_found": "The item is unavailable in this runtime session.",
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
	Kind        string `json:"kind"`
	Version     string `json:"schema_version"`
	WorkspaceID string `json:"workspace_id"`
	RequestID   string `json:"request_id"`
	Operation   string `json:"operation"`
	JobID       string `json:"job_id,omitempty"`
	DurationMS  int    `json:"duration_ms,omitempty"`
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

// Domain interfaces reserve shared authority boundaries; backend SQL never escapes.
type Artifact struct {
	ID        string
	Digest    string
	Size      int64
	ProfileID string
	Version   string
}
type Revision struct {
	WorkspaceID string
	EntityID    string
	Expected    uint64
	OperationID string
	Document    json.RawMessage
}
type Claim struct {
	WorkspaceID string
	JobID       string
	OwnerID     string
	Generation  uint64
}
type OutboxEvent struct {
	WorkspaceID string
	Target      string
	Sequence    uint64
	Revision    uint64
	Document    json.RawMessage
}
type Catalog interface {
	CommitRevision(context.Context, Revision) (uint64, error)
	ClaimAttempt(context.Context, Claim) (Claim, error)
	CompleteAttempt(context.Context, Claim, Revision) error
	ClaimOutbox(context.Context, string, string) (OutboxEvent, error)
	AcknowledgeOutbox(context.Context, Claim, OutboxEvent) error
}
type ArtifactStore interface {
	Publish(context.Context, string, []byte) (Artifact, error)
	Materialize(context.Context, Artifact) (string, error)
	Verify(context.Context, Artifact) error
}
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
