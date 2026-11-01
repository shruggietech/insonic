# S065 data model

No catalog migration or new product data entity is introduced. Catalog schema remains 11.

| Entity | Identity and constraints | Lifecycle |
| --- | --- | --- |
| Release version | Canonical semantic version 1.1.0 across software/docs/master schema; previous 1.0.3 remains explicitly supported | Prepared candidate, qualified candidate, later separately published |
| Candidate manifest | Exact source revision, four product archives, Windows installer assets, docs/schema versions, current CLI/GUI capabilities, original digests/receipts | Collection, verification, retention; publication separate |
| Historical authority | Exact published tag and original versioned docs/schemas/assets; upstream identities independent | Immutable source; released rendering elected by manifest |
| Capture record | Exact source/version/platform and original enumerated image bytes, dimensions and SHA256 | Isolated capture, archive verification, unchanged import, validation |
| Acceptance record | Run/attempt/head/revision, readiness and terminal timestamps, checks and separate phase times | Provisional reserve, terminal audit, retained immutable receipt |
| Review ledger | One automatic and at most one manual round, finding/thread identities and observed result | Requested, completed, fixed/responded/resolved; absent security receipt remains absent |

Qualification failure and cancellation never become accepted publication authority. Rebuilding produces a new candidate and does not inherit acceptance of previous bytes.
