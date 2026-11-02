# S067 Data model

Promotion receipt format 1: kind pages-promotion, repository/domain/branch, version/revision, archive SHA256/size, previous_commit/previous_tree, pages_commit/pages_tree when staged, phase, destination_changed (false, true or unconfirmed), observations (ordered sanitized phase/event/head/error category), failure with stage and exception category. Atomic persistence precedes the branch update. Initial receipt creation refuses an existing path.

Transitions: qualified -> staged -> update-attempted -> updated -> build-requested -> deployed -> public-verified; any stage can fail while retaining the last identity and observations. Uncertain mutation records unconfirmed until exact readback. No-op stages retain the original head and require its original tree without a self-parent relationship.

Read-only reconciliation consumes the original attempted receipt and original candidate; produces a distinct result linked by receipt SHA256. It re-reads original and staged trees, checks parent relation, re-derives original prepared files and verifies the exact current built deployment. The original receipt remains byte-for-byte unchanged. Malformed identity, unrelated head, altered archive, missing build or changed destination fails.
