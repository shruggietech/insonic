# Foundation data model

## Workspace

Existing workspace-config schema: stable UUID, display name, schema version, control directory and independently revisioned storage/catalog/graph profiles. Canonical locations are not source identity. Validate offline before state creation; initialization cannot overwrite.

## Operation

Request: protocol, workspace UUID, request UUID, operation and typed arguments. Response echoes identity plus session UUID and result or fixed error. Decode bounded input, reject unknown fields/trailing values and incompatible versions.

## Attempt

Workspace/job/attempt/session UUIDs, generation, duration and state. Running becomes cancelling/cancelled, failed or succeeded. Retry after failure/cancellation increments generation. Stale completion cannot alter current state. Caller disconnect does not cancel work; runtime restart reports old session jobs absent, with durable recovery reserved for #3.

## Adapter contracts

Versioned identity, capabilities and health; catalog typed transactions/revision/attempt/outbox; artifact immutable references/publication/materialization; separate private secret resolution and public credential status. Downstream implementations must preserve full backend parity.
