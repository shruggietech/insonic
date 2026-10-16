# Data Model: Media lifecycle correctness

No catalog migration is required.

- Current recording facts retain canonical format and current stream identity separately from original Metadata.MIME. Playback descriptors select the returned artifact/preview type without rewriting source observations.
- Accepted refresh advances the existing recording revision/report pointers once. Removed reports retain existing pending cleanup rows. Live leases/references defer retirement; released/expired retention permits normal retirement and done state. Retry uses the accepted receipt.
- Cleanup errors prioritize genuine storage failures over deferred conflicts. Current artifacts and owner inputs cannot be retired by obsolete-report cleanup.
- Transcript manifests retain per-item record selectors. Single positional input requires one target; manifest input forbids a conflicting global target.
- Acquisition work retains existing durable item/work error categories and safe reasons. Deadline, byte limit, explicit cancellation and content failure remain distinct; credentials/provider bodies are excluded.
