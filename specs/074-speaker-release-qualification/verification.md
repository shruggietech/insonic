# S074 verification

## 2026-11-03 specification and design

Base main ad0fb3534f88b8065361967b3c34640e44d762e1 is clean and synchronized. Branch codex/074-speaker-release-qualification, feature directory and ignored selector all identify S074. Existing #233/#211 are assigned S074/In progress; no new issue.

Installed Spec Kit specify/clarify/checklist/plan/tasks workflows completed with their Python helpers and resolved templates. No extension hooks exist. Built-in requirements checklist passes 11/11. Reviewer-owned qualification checklist remains 0/10, unchanged; explicit owner autopilot authorization allows implementation without a new approval pause.

Read-only analyze covered 12 functional requirements, five success criteria, eight acceptance scenarios, 21 tasks and all five constitution principles. Coverage is complete; no critical/high findings, unmapped task or unresolved scope ambiguity. FR-001/002/003 map T003-T009; FR-004/005 map T006-T012; FR-006/007 map T002/T003/T006/T010-T012; FR-008/009 map T013-T017; FR-010/011/012 map T009/T015/T017-T021. SC-001/002 map T004-T009; SC-003 maps T010-T012; SC-004 maps T013-T017; SC-005 maps T020-T021. Governance supplies T001/T002/T019. Real final measurement will follow version preparation so receipts identify 1.3.0.

Preparation research found compatible acoustic libraries, CUDA availability and an NVIDIA RTX 5090, hash-verified pinned community-1/whisper snapshots and the frozen licensed acoustic corpus. A cached qwen2.5:14b model is available; its owned loopback service has not started. These are readiness observations, not inference acceptance. Raw logs, model paths and commands remain under ignored .local/s074; public receipts must not contain credentials or machine paths.

.gitignore already excludes .local, build outputs, models, credentials and runtime state. No ignore changes are needed. The prior local qualification evidence remains intact.

## 2026-11-03 failing-first implementation

Corpus/scoring tests failed before implementation, then passed 10/10. Election/device/identity tests failed before implementation, then passed with the scorer tests 18/18. The native development CLI built successfully. First installed readiness stopped on the DLL search path; the hidden launcher now supplies the pinned native library directory.

The first complete deterministic campaign accounted for all eight cases and nine voices, with one oracle mismatch: imported supplied transcripts have an unknown stream route, so contradictory names share one evidence root and correctly remain ambiguous. Catalog source-root and resolver contracts independently establish this behavior. The corpus annotation was corrected before real-model evaluation; no resolver behavior, acceptance threshold or independence guard was weakened. Old failed receipts are retained in .local/s074. The corrected frozen corpus SHA-256 is c24a3b86629c159960b4a6e5b41823d3ed4eb27814e3164d3af522b2470388b4.

## 2026-11-03 candidate preparation

The corrected deterministic installed campaign passed all eight cases/nine outcomes; identity precision and known-identity coverage are each 3/3. These are controlled-provider contract results, not model quality. Aligned product/docs/master-schema version 1.3.0 was prepared; published docs/v1.2.0 and schemas/v1.2.0 exactly match tag v1.2.0. The schema-12 catalog upgrade and pre-upgrade-backup rollback boundary are documented.

Frontend check/test/build passed. Foundation/schema checks passed after adding the required highlights heading. Root tests identified new timing owners; the campaign now consumes the shared phase policy and all three owners are classified, and the root tests pass. The initial site export check correctly rejected stale v1.2.0 output; fresh build/verification follows. The first Go suite lacked the Windows USERPROFILE environment inherited by the Node launcher; failures consistently reported the unavailable OS profile. A bounded rerun supplies the verified current profile and preserves the original logs.

The hosted-capture bootstrap temporarily omits only new-version walkthrough sections. Complete new-version text is retained locally and will be restored with the original hosted PNGs/receipts before the official PR. No local GUI capture, release tag or publication is performed.

Launcher readiness diagnosis: Go platform paths require USERPROFILE, APPDATA and LOCALAPPDATA. Supplying all verified directories makes the focused workspace suite pass. The failed environment-only Go tree was stopped before rerunning. Python discovery (771 tests) exposed the inherited Unix locale and a bare compiler resolving to the wrong version under GOTOOLCHAIN=local; the verified launcher now places Go 1.27.1 first on PATH and uses the Windows-supported C locale. These corrections are ignored operator configuration, not product behavior changes. Fresh complete suites are in progress.
