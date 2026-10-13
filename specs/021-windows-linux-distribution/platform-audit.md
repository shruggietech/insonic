# S021 platform audit

Audited 2026-10-13 against the current owned runtime, package/source recipes, qualification pins, workflows, candidate schemas and Windows/Linux public help.

| Surface | Disposition |
| --- | --- |
| Owned Darwin Go files | Removed eight files for desktop linking, native credentials/probes, lifecycle fixtures, backup publication and voice-model retrieval. Generic Unix behavior remains for Linux. |
| Host integration | Shared host checks reject unsupported hosts before workspace/runtime effects. Qualification probes and processing worker entry points also reject them. Windows/Linux credential and profile behavior remains. |
| Owned package/media recipes | Removed Darwin targets, application bundle assembly/discovery, loader repair, signing/notarization, VideoToolbox and webview/worker workarounds. Current exact-source media configuration has Windows/Linux targets only. |
| CI/release | Removed macOS jobs, matrices, dependencies, inputs and signing variable references. Windows/Linux native, four extracted-package variants, source closure, frontend and live backend lanes remain. All job budgets remain ten minutes. |
| Repository gates | Read current main branch protection and active default-branch ruleset 20478126. Required checks are foundation and docs, with no macOS check or condition. No settings mutation was required. |
| Current authority | Constitution 3.0.0, active roadmap/bootstrap, README, 1.0.1 documentation/schema, generated installation text, downloads and future release generation use Windows/Linux. Product/docs/schema stay aligned at the unreleased 1.0.1 candidate. |
| Historical release parsing | release.py keeps historical platform/asset-name and notarization validation only for version-bound records. It reads immutable selected-version schema data under trusted current validation. Current 1.0.1 and future records require four variants/six content assets/eight publication assets; 1.0.0 retains six/eight/ten. |
| Historical bundle data | package_directories and inventory checks retain old MacOS/Resources layout data to validate immutable archives. They provide no current target, assembler or executable discovery route. |
| Frozen history | docs/v1.0.0, schemas/v1.0.0, v0.0.0 artifacts, completed earlier specs and changelog history remain unchanged. Version-specific changelog references are historical. |
| Upstream metadata | Third-party Go/npm module records, dependency license/source notices and lockfile optional Darwin binaries remain upstream provenance, not an owned support claim. |
| Time zones | Australia/Darwin in the Windows-zone mapping and embedded IANA archive is geographic data and remains intact. |
| Injection clearing | Clearing DYLD loader injection variables in tool/package environments remains defensive environment hygiene, not a macOS runtime dependency. |
| Withdrawal statements/tests | Explicit 1.0.1 withdrawal notes and negative/historical tests remain truthful dispositions. |

Current source searches found no remaining owned macOS build/runtime/support path. Historical documentation/schema directories pass an unchanged-byte Git comparison against the slice base. Package/publication verification retains exact revision, inventory, source, signing and qualification requirements.
