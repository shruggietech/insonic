# S027 distribution and capture contracts

One Windows x64 per-user MSI adds Desktop and CLI feature selection, both default. Core runtime remains mandatory. Exposed CLI shortcut is independent of required runtime bytes. MSI performs no network acquisition and no global PATH mutation.

A separate installers list complements exactly four archive package entries. Installer verification rejects wrong format/platform/version/source, duplicate identity, missing lifecycle qualification, and payload/source/help/license mismatch. scripts/windows_msi.py exports trusted verifier verify_installer(path, receipt, source_root=None); receipt identity remains separate from archive variant.

Capture tooling accepts only isolated fixture data and the current built frontend/runtime. Requests are bounded and allowlisted; no arbitrary external endpoint or owner profile is served. Native MSI capture/installation is disposable-CI-only. Cleanup is by exact owned process tree; no desktop-wide enumeration/close commands.

Launcher supports noninteractive describe/preflight for qualification. Actual CLI mode opens a persistent standard terminal/help session only on explicit user action. MSI launch occurs only after successful full interactive fresh installation and the checked Launch control.
