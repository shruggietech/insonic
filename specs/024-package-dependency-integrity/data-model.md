# S024 data model

## Executable dependency identity (internal)

Each selected product binary supplies GoVersion, its command Path, Main module and Deps. Each dependency has Path, Version, Sum and an optional versioned Replace with the same fields. Require elected Go compiler and the correct product command. Reject incomplete/unversioned identity, checksum disagreement, duplicate/conflicting logical modules and inconsistent shared replacements. CLI selects one binary; desktop selects both. Effective sources are the replacement if present, otherwise the original dependency. Deduplicate shared effective identities; different versions/checksums for one effective path fail.

## Resolved source evidence (internal)

A selective Go module record must match the binary logical and effective identity exactly. Dir must be a nonempty absolute existing directory, resolving to the Go-escaped effective module path and version under GOMODCACHE. Reject symlinked directories/files outside that identity, project-root substitution and missing/empty notices. Require at least one license/copying file; preserve any additional NOTICE files. Modules without source go.mod remain supported. Hash the effective selected source files in Go h1 format against the linked binary Sum. Capture notice bytes while computing that hash and copy that verified snapshot, so altered cache contents or later source changes cannot substitute notice bytes. No unrelated tooling lookup or download is performed.

## Existing package inventory (unchanged external shape)

`dependencies.modules` stays an array of `{module, version, notices}` records, now listing effective linked sources only. Each notice is relative to `notices/` and canonically stored at `go/<escaped-module>@<escaped-version>/<upstream-name>`. No absolute source paths or credentials are persisted. Go-LICENSE is included separately. Package verification compares exact binary membership and requires every reference to name an inventoried nonempty file; native assembly additionally compares source bytes. Older published inventory data remains readable; current candidate qualification requires current evidence.

JavaScript records keep their existing shape; the elected Mermaid runtime closure includes actual resolved KaTeX 0.18.2 notices. No catalog, API, schema, runtime state or migration changes.

## Delivery states

Specified -> designed -> tasks/analyzed -> implemented/locally verified -> official PR/review -> final-head CI green -> owner review. At most two bot review rounds; source #87 remains open until consolidation merge. Owner retains merge/release authority.
