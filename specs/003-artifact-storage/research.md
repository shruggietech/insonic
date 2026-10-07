# S003 research

## Storage protocol

Decision: reuse pinned minio Core and explicit static credentials, endpoint/region/addressing configuration. Persist multipart identity, resume accepted parts, verify exact readable final bytes independently.
Rationale: high-level upload hides upload IDs; multipart ETags are not whole-content digests. Validate range headers/length, including providers returning an unexpected full body.
Alternatives: high-level opaque upload and universal conditional writes would hide recovery state or assume unqualified capabilities.

Primary references: [S3 conditional writes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html), [integrity](https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html), [multipart completion](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html), [abort](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html), [pinned fixture](https://raw.githubusercontent.com/adobe/S3Mock/5.2.3/README.md). Research agent s3_research inspected the exact pinned SDK source.

## Catalog authority

Decision: add an operational lifecycle table without changing immutable domain records. Preserve legacy migration digest and implement v1->v2 upgrade and snapshot compatibility.
Rationale: modifying v1 DDL digest would prevent opening existing workspaces; timer jobs cannot serve upload authority.
Alternatives: mutable location history and disconnected filesystem checks violate existing evidence/fencing contracts.
Research agent artifact_research inspected mutation, migration, snapshot, job and runtime boundaries.

## Clarification coverage

All functional/data/interaction/quality/dependency/edge/constraint/terminology/completion categories clear. Owner scope and storage docs answer provider policy and lifecycle semantics. No formal owner questions required.
