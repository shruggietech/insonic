# S010 data model

## Current authorities

LibraryEntry retains its stable UUID and current canonical digest/size/publication. New canonical entries have no original local locator and no native-sidecar publication. Existing legacy entries remain diagnosed until an explicit migration slice.

Facts gains canonical policy, source digest/size, tool identity, ordered source/canonical stream mapping and rational timeline origin. Facts.Streams describes actual retained canonical bytes. Metadata reports describe captured original source facts after locator scrubbing.

Recording retains one embedded validated current Cue JSON. Imported local speaker IDs follow the upstream bounded string contract; recording/global speaker/mapping IDs remain UUIDs. Known-speaker correlation remains external.

## Admission

Resolve target -> stage stable input -> capture/discover -> convert/validate candidates -> verify artifact publication -> one fenced catalog acceptance -> recoverable retirements -> terminal receipt.

New audio with transcript commits library+recording together. Standalone transcript validates current source/document revisions and preserves stable audio identity. Default collisions produce successful skips, and explicit replacement invalidates dependent references under current authority.

Retries bind frozen source/document digests and elections. Failed/cancelled/stale work cannot publish current rows. Active leases and shared current publications delay retirement.

## Portability

Additive facts and bounded local IDs use current portable catalog records. Receipt/snapshot proofs must recognize the new atomic operation and preserve current-reference integrity across SQLite/PostgreSQL export/restore.