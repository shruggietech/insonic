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

2026-10-05: All three native jobs pass at implementation commit 3fcce18 in [run 37367868022](https://github.com/shruggietech/insonic/actions/runs/37367868022). Linux amd64 finishes in 3m3s, Windows amd64 in 2m17s and macOS arm64 in 1m46s. Each executes core acceptance/vet, descendant cleanup, checksum-verified Cueson source round-trip, real Ladybug/SQLite operations, cross-process owner reuse/cancel/retry, complete offline-help assets and a real native webview frontend/bridge/IPC round-trip. Artifact receipts retain exact OS/architecture, dependency identities and digests. Foundation checks pass; documentation and alternative-fixture jobs are still queued at this entry.

2026-10-05: macOS core acceptance includes TestMissingLookupDistinguishesLockedKeychain using an isolated native Security.framework fixture. Windows and macOS missing-item probes report available without returning credentials; Linux's absent Secret Service reports unavailable. No production credentials or persistent credential operations are exercised. All five external review findings have evidence replies and resolved threads. Exactly two Code Review rounds occurred; the final request also asked for security review, but no separate security-review result was returned.

2026-10-05: Receipt audit found native Linux loader warnings interleaved with the JSON webview result. Qualification now extracts exactly one JSON result, requires the expected passed statuses and shared version, and rejects missing, ambiguous or failed results. The regression fails before the writer exists and passes afterward. All six Python checks and actual local hidden Windows desktop execution pass; emitted desktop/webview receipts parse as standalone JSON.

2026-10-05: Run 37367868022 initially cancels documentation and adapter jobs without executing a step. GitHub's failure annotation states: "The job was not acquired by Runner of type hosted even after multiple attempts". Only these infrastructure failures are retried; the native/foundation results remain passing. The final local integrity check covers 176 project text files, 14 schemas, 12 contracts and 13 examples; all 41 maintainer tests and both 25-page documentation exports pass.

2026-10-05: Final code publication includes the validated receipt writer and completed implementation evidence. T016 remains open only for definitive green PR-head checks and owner handoff. Superseded queued checks do not block pushing an already-verified correction; owner merge remains unauthorized for the agent.

2026-10-05: The owner explicitly removes mandatory manual PR reviews. Repository main protection now requires zero approving reviews, retaining the review settings object, strict foundation/docs checks, resolved conversations, linear history and force-push/deletion restrictions. The inherited organization PR rules already require zero approvals and are unchanged. Bootstrap and development policy are aligned under constitution 2.1.2, with a regression for preserving zero approvals and detecting the obsolete one-approval gate. The owner's final merge remains the handoff; no additional bot-review round is requested.

2026-10-05: The approval-gate regression fails against the previous bootstrap (one required approval), then passes with zero required approvals and code-owner/last-push approval disabled. All 42 maintainer tests, project/schema integrity and both 25-page documentation exports pass. Live API readback confirms the narrow approval-count update retains every other existing branch protection.

## Issue acceptance mapping

| Issue | Delivered acceptance | Evidence |
| --- | --- | --- |
| #1, native platform feasibility | Exact Go/Wails/Cueson/Ladybug/SQLite pins and verified native assets; actual CLI, desktop, IPC, offline help and secret capability checks on three OSes; PostgreSQL/S3/ArcadeDB feasibility | Three native jobs and uploaded receipts in run 37367868022; successful alternative-fixture operations in run 37366431231 |
| #2, shared-runtime scaffolding | Canonical discovery/private platform paths, atomic initialization, stable one-owner lock, user-only bounded IPC, shared CLI/Wails dispatch, session cancellation/retry/generation fencing, redacted errors and whole-tree hidden supervision | Core Go/vet and Python acceptance plus separate CLI/desktop subprocess execution on all three OSes in run 37367868022; local Windows Go race checks |

Only Windows amd64, Linux amd64 and macOS arm64 have executed evidence. Windows arm64, Linux arm64 and macOS amd64 are locked upstream artifacts, not qualified execution targets. The Wails shell establishes the shared boundary; full desktop workflows remain #10. Attempts remain session-scoped; durable catalog/jobs, production storage and credential/model implementations remain #3-#5. S3Mock establishes protocol feasibility, not production endpoint authentication. No installable release, owner merge or issue closure has occurred. Final PR-head CI must pass before handoff.
