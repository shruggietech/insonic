# Runtime foundation evidence

## Spec Kit execution

2026-10-05: Specify and clarify produced the approved two-issue spec and reviewed requirements checklist. Installed setup_plan.py/setup_tasks.py helpers resolved the active feature; analysis prerequisites passed. Twelve functional requirements map to T001-T016 with no uncovered requirement or constitution conflict. No extension hooks exist. Explicit push/PR authorization overrides the default pre-push halt.

## Local execution

2026-10-05: Windows amd64 executes pinned Cueson/schema encode/validate/render and byte-identical source restore. Native Ladybug v0.21.2, pinned Go binding and SQLite execute creation/read/rollback. Asset digests are in internal/qualification/dependencies.json. Go resolves the binding revision to v0.17.1-0.20260804043248-42bbf464c74c.

2026-10-05: Windows native Wails webview completes frontend -> Go bridge -> user-only IPC -> runtime -> frontend. Headless bridge qualification confirms packaged offline-help/tokens. Separate CLI subprocesses reuse one session and exercise cancel/retry. A unique missing Credential Manager lookup reports available without creating or returning credentials.

## Acceptance boundaries

Workspace initialization/discovery, validation, ownership/IPC, idle lifecycle and session attempts are implemented. Catalog/artifact/secret interfaces preserve supported alternatives; persistence implementations remain #3-#5. Wails is a qualification shell. No product release or installable package is claimed.

The lock inventories six OS/architecture artifacts; passing evidence requires actual execution. macOS/Linux native checks and PostgreSQL/S3/ArcadeDB fixture execution remain pending hosted CI at initial publication. S3Mock does not prove production authentication. Locked/missing secret services report unavailable; encrypted fallback is #5.

## Review protocol

Official publication triggers initial review. At most one additional @Codex request is authorized. Every comment needs disposition/correction, all threads must resolve, and all CI must pass before owner merge handoff. No merge, release or third manual review is authorized.

2026-10-05: Local convergence registered runtime envelopes in the master schema, validates wire arguments before dispatch, bounds the macOS service subprocess and packages exact upstream typography. Go acceptance, vet and race detector pass; 41 maintainer tests, four archive tests and 14-schema/12-contract/13-example checks pass. Documentation and offline export resolve all links across 25 pages. Hosted native and alternative backend execution remains outstanding.

2026-10-05: Hosted run 37361902552 proves Linux native/webview, PostgreSQL serializable commit/rollback, S3 full/range visibility and ArcadeDB transactions/native dialect. Windows core acceptance exposed elevated-token default Administrators ownership; creator-only owner normalization corrects this without accepting another owner on existing directories. macOS native dependencies passed; Wails needs UniformTypeIdentifiers explicitly linked. Both corrections await a fresh hosted run.

2026-10-05: Initial Codex review submitted two P2 findings (4187825487, 4187825496). Owner locks now use the private per-user runtime path keyed by canonical root; read-only metadata works and a control-directory edit cannot bypass a live owner. Bounded history evicts the oldest terminal group when full and never evicts active work. New regression cases and Go race/vet checks pass. No manual second review has been requested yet.
