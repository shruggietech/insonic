# S012 interface contract

References: unprefixed UUID or `base:UUID`, `speaker:UUID`, lowercase 1-64 alias, and explicit configured `source:NAME/SELECTOR`. The source selector resolves a bounded catalog entry with exact complete manifest. Alias targets carry declared operation and kind; hosted target metadata is explicit and does not advertise local weights. Alias/source IDs are stable caller-supplied UUIDs; the unique name remains bound to its ID through tombstones.

Runtime model operations add resolve/discover, alias list/show/set/remove and source list/show/set/remove. Set includes expected_revision and target/configuration; remove retains a tombstone. Existing register/acquire/list/show/verify/materialize remain. CLI wraps those operations with versioned JSON and human summaries. Source mutation and discovery do not run engines. Saved pipeline inspection does not acquire models.

Processing/import model fields accept shared references. Submission returns exact selected model identities and acquisition IDs through work inspection. Work exposes acquisition versus processing phase and dependency failure; pending dependencies do not claim inference success. Retry preserves exact frozen selection.

Desktop model selectors label alias/exact version/capability/availability and show incompatible choices clearly. Settings provide alias/discovery actions and advanced configured sources using shared runtime contracts. Hosted and trained inspection remains truthful about adapter and retrieval availability.
