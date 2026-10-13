# S021 research

## Current and historical matrices

Decision: Generate Windows/Linux only; derive historical validation cardinality from the selected immutable version schema, read as data by trusted current validators. Rationale: Current six-package constants otherwise strand historical 1.0.0 records. Alternatives: deleting historical validators or accepting arbitrary declared matrices would weaken provenance. Current packages=4, candidate content assets=6, published assets=8; historical 1.0.0 retains 6/8/10.

## Native removal

Decision: Remove Darwin credentials/backup/probe implementations, package bundle/runtime paths, loader variables, VideoToolbox and hosted-webview workarounds. Preserve generic Unix implementations for Linux and defensive injected-environment clearing. Reject unsupported host/target before workspace/native side effects. Rationale: Owned support removal must not weaken Linux process/credential behavior.

## Installation ownership

Decision: Validate physical source/destination ancestors, copy into a uniquely owned attempt directory and atomically promote to the elected absent destination. Physical directory identity, rather than a copyable marker alone, governs failure cleanup. Rationale: Lexical GetFullPath alone misses junction/SUBST aliases; pre-existing/substituted directories never acquire cleanup ownership. Forced failure and alias/substitution tests use disposable fixtures.

## User-facing guidance

Decision: Use the current desktop page for online/offline installation authority, variant-specific generated INSTALL and consistent download/release highlights. Current CLI version/help/workspace init/doctor syntax is reused. Runtime has no stop command: finish/cancel work, close clients and allow the existing 30-second idle owner to exit (at least 31 seconds without further requests), checking only relevant old-path processes if needed. Configuration/model locations are taken from actual workspace/profile choices, not invented global folder defaults.

## Prerequisites

Windows packages require Microsoft Visual C++ x64 runtime; desktop additionally requires WebView2. Official architecture-matched guidance and online/offline installer routes belong before installation. Local inference environments and weights remain separately configured. MSI/security-dialog screenshot/setup flows remain their separate issues.

## Repository gate audit

Main branch protection requires foundation/docs only. Active ruleset 20478126 requires PR/non-fast-forward/deletion rules and no macOS status context. No repository settings mutation is necessary. Research was delegated read-only under speckit-plan; production edits remain coordinated.
