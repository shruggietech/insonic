# Backup recovery acceptance contract

Public create/verify/restore syntax and persisted backup format stay compatible. Recognized wrapped disk/quota errors report `insufficient_space`. Normal occupied-target errors remain `conflict`; short writes and unrecognized I/O failures remain failures. A generated `backup_cleanup` reason reports residual owned cleanup while retaining the primary classified code.

Native regression cases cover open/write/flush/close, incomplete writes, creation publication, progress replacement, activation replacement and secondary cleanup. They check no false completion, unchanged accepted bytes, retained request/pin/fence authority, and identical retry after each recoverable fault clears. Native platform mappings are tested separately from physical exhaustion.

CLI source/package receipts require actual confined backup-parent, restore-publication and restore-progress obstructions, unchanged blocker removal, exact request retry, source-independent restored byte verification and no owned fixture leaks. Validation rejects missing cases, unknown fields, wrong platform/executable, false physical-exhaustion claims, non-true checks and inconsistent identities. The maintained coverage ledger binds the receipt to its source/package context.

No physical disk-full, live provider quota, power-loss or new inference acceptance is implied.
