# Security policy

## Reporting

Report vulnerabilities through [GitHub private vulnerability reporting](https://github.com/shruggietech/insonic/security/advisories/new). Do not post credentials, private media or exploit details in a public issue. If GitHub reporting is unavailable, use a private contact route listed on the [ShruggieTech website](https://shruggie.tech/).

Include the affected version, a minimal reproduction, impact and any relevant dependency version. Remove credentials and private content. Maintainers assess the report and coordinate a fix and disclosure; this baseline does not promise a response-time SLA.

## Supported versions

v1.0.0 is an unpublished release candidate; v0.0.0 remains a specification and maintainer-tooling baseline. No official product binaries are supported yet. Once application releases exist, the latest stable release receives security fixes, with exceptions documented per advisory.

## Credential and dependency handling

The intended product [credential contract](docs/v1.0.0/security.md) requires encrypted secret storage, status-only display and redacted diagnostics. Dependencies and release artifacts are pinned and checked. Security fixes remain tracked work with truthful verification, not an implied guarantee from a passing CI badge.
