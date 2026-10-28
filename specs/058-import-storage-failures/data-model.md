# Data model: Local import storage-failure recovery

No catalog schema or product-data migration.

- Import authority remains source digest/size, durable work/attempt receipt, canonical/report publications and accepted recording.
- Native fault observations identify injected write/flush/rewind/cleanup/report/persistence boundaries and independently checked bytes/state.
- The maintainer storage receipt binds format, platform, executable digest, a canonical-publication filesystem obstruction case, failed and successful work/item observations, source/sentinel/candidate identities, explicit actual-filesystem origin, and final cleanup.
- The failed batch retains its partial result and failed durable-work state. A distinct new request succeeds after obstruction removal; idempotent replay returns that same successful work without another recording.
- Physical exhaustion is false/unexecuted in this fixture receipt. Native injected capacity failures remain separate evidence.
