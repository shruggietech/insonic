# S061 data model

No new persisted backup or catalog entity is introduced. Existing manifest, backup pins, publication progress and pending activation remain authoritative.

- **Backup request**: UUID, source workspace and capture revision/digest, mode and output directory. A completed identical retry verifies the bundle and repeats durability before reconciling pins. Different requests cannot replace it.
- **Restore attempt**: Bundle digest plus exact destination profile/root digest. Pending publication progress remains private to that destination. Committed catalog authority remains fenced until configuration activation completes.
- **Failure**: Existing classified error code; optional generated reason `backup_cleanup` indicates a secondary owned-cleanup failure. Internal joined causes are never serialized as raw text.
- **Qualification receipt**: Format/platform/executable digest, explicit actual-filesystem-obstruction origin, false physical-exhaustion claim, exact required case set, bundle and accepted object digest identities, failure classification, integrity checks, retry identity and owned cleanup.

Transitions: backup staging -> verified staged bundle -> nonreplacing publication -> durability -> capture-pin reconciliation. Restore validation -> destination publication/progress -> fenced catalog commit -> configuration activation -> completed progress cleanup. Any failure refuses success; existing authority and exact retry bindings remain required.
