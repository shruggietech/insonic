# Shared artifact operations

Store operations: Capabilities, Stat, OpenRange (offset/length), PublishImmutable with durable nonsecret upload journal, Verify and DeleteUnreferenced under service-issued retirement claims. Adapters are byte transports; catalog controls admission.

Runtime operations: artifacts.publish source path/kind, show/verify/reconcile/abort/retire publication UUID, materialize publication UUID, lease renew/release publication UUID+lease UUID, retain/release-reference publication UUID+reference UUID. Request IDs reconcile publications; changed source content under the same request ID conflicts. No file bytes or credentials cross JSON IPC.

Errors use existing redacted invalid_request/conflict/unavailable/cancelled vocabulary. Provider choices never fall back. Materialization/retirement barriers are identical across catalogs.
