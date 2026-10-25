# Worker probe contract: S048

The existing workers.inspect result and environment receipt inspection gain an optional validated probe object. CLI JSON, processing/setup readiness and desktop consume that same object. Old receipts may omit it.

Fixed reasons distinguish success; unavailable interpreter/home; process launch/exit; deadline/cancellation; malformed/oversized output; missing package/version mismatch; import/CPU failure; inventory limit/read/hash failure; invalid checkpoint. Stage labels identify interpreter_hash, private_home, process_start, package_versions, library_import, cpu_prerequisites, cuda_prerequisites, dependency_inventory, dependency_hashing, output_validation, receipt_verification, complete and not_started as applicable.

All numeric bounds and clock origins follow [data-model.md](../data-model.md). A private checkpoint carries generated fixed facts only, is atomically replaced and removed with the probe home. Malformed progress cannot authorize compatibility or reuse. Safe fixed operator reasons survive failed setup/work inspection. Genuine mismatched bytes remain an integrity failure; interrupted verification does not claim changed bytes.

The original caller expiry remains authoritative. Client disconnect stops ephemeral inspection/readiness, while enqueued setup remains durable until explicit work cancellation. Partial readiness retains already observed facts and labels unattempted work unknown with a corrective action.

The serialized history field is `checkpoints`; the catalog continues to refuse raw `observations` in durable results.
