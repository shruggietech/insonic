# Research decisions

## Portable authority

Decision: catalog-native capture/restore plus streamed digest-addressed directory bundles. Rationale: existing snapshot validation binds immutable journals to receipts; editing exported JSON breaks authority. Retain historical receipt identity required for provenance, copy only current/durable byte authority. Alternatives: raw catalog export lacks bytes; backend-native DB backups do not support cross-backend restores; tar extraction adds avoidable path risk.

Decision: stop local owner, acquire workspace lock and fence catalog writes during transfer. Rationale: app startup resumes work and a generic lease currently does not fence writes. Reference-only pins are persistent until explicit release. Alternatives: read-only best-effort snapshots race cleanup; implicit expiration can invalidate a backup.

Decision: reset graph checkpoints and rebuild accepted evidence. Rationale: source graph checkpoints cannot establish destination graph state. Alternatives: native graph file copies are backend-specific.

## Source-complete packages

Decision: owned native source builds, exact source archives/build recipes/configuration and capability proof. Rationale: Windows upstream bundles about 40 external libraries; Linux corresponding sources do not match the pinned binary. FFmpeg core alone is insufficient. Preserve built-in decoders and explicitly account for retained external input capabilities.

Primary references: [FFmpeg redistribution checklist](https://ffmpeg.org/legal.html), [upstream static-build source policy](https://github.com/eugeneware/ffmpeg-static/blob/master/README.md).

## Release and tracking

Decision: baseline 0.0.0 remains until explicit version preparation; update owned version bindings, preserve immutable prior docs/schema, never globally replace upstream pin versions. Decision: complete draft-first verified release upload, publication readback, then configured exact-version docs promotion. Rationale: interruptions must not publish partial assets or promote unmatched docs.

Primary reference: [GitHub managing releases](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository).

Decision: canonical labels validate before API mutations; preserve existing issue body/title/state and project fields. An opened-only classifier maps allowlisted area selections to labels without changing edited historical issues. Compatibility labels remain.

## Integration decisions and dependency disposition

Decision: keep destination credential selection and vault/native credential namespace independent of the imported catalog workspace UUID. The optional namespace retains existing credentials without copying secret bytes into the bundle or rebinding the destination vault.

Decision: transform nonportable acquisition payloads only in a disposable validated catalog, then admit new proof receipts. The source catalog remains unchanged. Restored incomplete local acquisitions require a fresh source election; original local paths cannot become resumable work. Both captured-source and generated-portable catalog proofs are recorded.

Decision: keep a committed restore passive with a durable pending receipt until exact destination configuration activation completes. Expiring transfer ownership alone cannot make a failed restore writable. Recovery binds the original bundle and destination profiles.

Dependency PRs #17 and #23 are integrated into this slice: setup-go 7.0.0, Wails 2.16.0, x/crypto 0.57.0 and x/net 0.59.0. Full local Go/frontend checks pass; the shared all-platform native CI boundary qualifies the resulting combination. Separate PR merges are unnecessary after S014 reaches main.

Decision: share one reusable native source workflow between CI and release, with dependency and FFmpeg compilation stages. Compact same-run transfers prove exact revision, compiler, recipe, dependency install files, binary hashes and original archives before consumers reuse them. This preserves full decoder input support and removes recompilation from backend/native qualification. Measure actual workflow turnaround as well as per-job timeouts.

Decision: resolve bare Windows build executables against the selected child PATH before hidden process creation. Windows process startup otherwise searches the parent environment, which can bind a cache identity to a different compiler than the configured source toolchain.

Decision: use independent same-platform source chains and verify the complete cache before scheduling cold dependency/FFmpeg stages. Warm paths publish a hash-verified same-run transfer immediately; cold paths retain both source stages. Local cold compilation takes 504 seconds before downstream native checks, so full cold CI turnaround cannot be claimed compliant from per-job limits. Live cold/cached timing remains an explicit convergence task.

Decision: qualify shared runtime acceptance once per platform and run both package variants independently after source transfer. Every existing core, native, credential, frontend and bridge check remains in the runtime lane; each package lane exercises its complete extracted-package operations. Separate workspaces let package smoke tests isolate native libraries without racing another variant, and avoid serializing runtime acceptance with the package journeys.

Decision: retain original source archives in a separate exact-pin cache, rehash every archive before use, bound transient download retries and schedule dav1d first with the available cores. The correction run qualified macOS dependency/source closure and Windows core checks, but Linux acquisition and Windows one-worker compilation failed. Recipe changes require fresh all-platform source evidence.

Decision: release publication may exclude only the exact workflow-dispatch release run's own unfinished check suite when it qualifies the same candidate revision. Verify the run identity, repository, workflow, event and SHA before excluding it; every other required check remains mandatory. This avoids waiting for the publisher's own future completion without ignoring an unrelated pending release or CI run.

Decision: promotion-only downloads and completely revalidates the original published candidate rather than rebuilding the same revision. Archive timestamps and generated site build identifiers are not assumed reproducible. Fixed GitHub asset endpoints receive authorization; validated official CDN redirects receive no token. Fresh release desktop qualification runs the sibling CLI prerequisite first. CI permits each platform's qualification to consume its own verified same-run transfer despite another platform's source failure; missing or invalid transfers still fail.

Decision: share only portable original archives through the exact-pin cross-OS cache; compiler/install/binary caches remain platform-specific. Every consumer rehashes before use. The pinned [cache cross-OS guidance](https://github.com/actions/cache/blob/caa296126883cff596d87d8935842f9db880ef25/tips-and-workarounds.md#cross-os-cache) and [action inputs](https://github.com/actions/cache/blob/caa296126883cff596d87d8935842f9db880ef25/action.yml) support this reuse. Add Linux's explicit static math library for the retained game-music decoder; failure-only bounded diagnostics retain probe evidence without accepting mismatched source bytes or interpreting source logs as workflow commands.

Decision: privileged publication runs protected default-main tooling and validates exact trusted checkout plus candidate ancestry before checking out candidate data. Historical reviewed revisions remain valid. Documentation preflight and promotion run separately with read-only GitHub permission, a main-only deployment environment policy and an explicitly named dedicated host credential. Reserved automation names cannot enter the host command. Environment reviewers are optional; this integrity boundary does not impose a new attended approval ritual.

Decision: portable processing/training/matching/dependency payloads use native proof receipts and source-free tool tokens, preserving elected model/settings/routing authority. Recovery constructs ephemeral destination invocation paths, rebuilds accepted recognition context and leaves unavailable custom exact-file bindings queued for automatic retry. Standard tools rebind by role to validated destination platform binaries; custom adapter/support files retain exact content identity through destination-local processing configuration. No original or rebound absolute execution path becomes portable catalog data.

Decision: macOS app executables and native libraries remain in `Contents/MacOS`; companion tools, notices, sources, help and manifests belong in `Contents/Resources`. Runtime discovery resolves that resource root after relocation. Putting ordinary resources inside the app's code directory prevents valid macOS signing.

Decision: use one authoritative complete-cache probe, three independent compression/audio/AV1 dependency groups for cold builds, and a verified transactional merge before dependent formats and FFmpeg. Complete source receipts bind the carried group receipts and their timing/provenance. Merge restores original archives without reacquisition. Each platform's runtime/package jobs wait only for its own source workflow; Unix core checks run concurrently with source compilation. Missing or invalid same-run source authority remains a failure. These changes optimize the measured critical path without reducing source capabilities or test coverage; full cold/cached turnaround still requires live evidence.
