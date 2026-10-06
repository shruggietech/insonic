# S003 data model

A workspace-scoped publication journal is keyed by operation UUID and carries immutable intent digest, artifact/location UUIDs, profile ID/revision, SHA-256/size/kind, relative unique key, optional exact version, verification method and multipart upload/parts. Generation is positive; owner is UUID; lease expiry uses catalog time. Allowed states are pending, available, missing, retiring, retired, aborted. Available admission inserts immutable Artifact and ArtifactLocation together with its receipt.

Retention references have UUID identity and nonsecret purpose. Materialization leases have UUID owner/identity, positive generation and expiry; local paths are ephemeral and excluded from durable manifests. Retirement requires grace, zero live leases and zero explicit/structural references; it blocks new admission while uncertain deletion is reconciled. Historical transition evidence and verified admission receipt survive snapshot restore. Imported authority expires.
