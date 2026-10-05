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

2026-10-05: The one authorized second-round request is issuecomment-6001512602 (code and security requested together). Codex reports completed Code Review of d072205, with three P2 findings (4187929755, 4187929765, 4187929776). Runtime and qualification supervisors now own Unix groups or pre-resume Windows jobs, including descendant cancellation and timeout cleanup. Local Windows descendant writer regressions pass with Go race checks and five Python checks. The macOS probe now disables UI, checks unlocked/readable state and performs a unique non-writing missing-item lookup; its isolated locked-keychain fixture awaits hosted execution. No separate security-review outcome has been returned. No third request will be made.

2026-10-05: Windows CI continued rejecting workspace privacy at b5962e9. Directory verification now compares binary SID, owner, protected DACL, one allowed inheritable FILE_ALL_ACCESS ACE and the exact user SID. Equivalent text forms and foreign/extra/unprotected grants have focused passing regression coverage. This removes SDDL rendering aliases from the authorization decision; hosted acceptance remains pending.

2026-10-05: Run 37365552723 passes Windows core acceptance and actual Cueson/Ladybug/SQLite operations; all Linux checks and alternative-backend fixtures pass. Windows CLI qualification exposed the supervisor closing a deliberately detached runtime with its client. The owned first-client bootstrap now explicitly permits requested job breakaway, while ordinary supervised jobs retain strict descendant ownership. Local cross-process CLI reuse/cancel/retry and ordinary timeout writer cleanup both pass. macOS locked-keychain execution remains queued.

2026-10-05: Windows full native qualification passes run 37366431231 in 4m21s, including the process-tree regressions, native dependency pair, cross-process CLI, credential availability and actual WebView2 bridge. Acceptance audit detected omitted _next documentation assets under Go's default directory embed. The embedded local-asset graph regression fails before all:assets and passes afterward; actual Windows desktop qualification and static/integrity checks pass. Hosted execution of this corrected packaging remains pending.
