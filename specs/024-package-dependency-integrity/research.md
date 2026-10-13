# S024 research

Research date: 2026-10-13. Read-only agents investigated notice identity, renderer compatibility and captured runtime update. No implementation evidence is claimed here.

## Shipped Go membership and source identity

Decision: read `go version -m -json` from actual product executables, selecting CLI alone or CLI plus desktop. Deduplicate the union and reject conflicts. Selectively resolve `go list -mod=readonly -m -json <paths>`, checking Path/Version/Sum and replacements against each binary; validate the effective source's nonempty absolute Dir against canonical escaped GOMODCACHE identity. Copy only its upstream notice files, in collision-free module/version subdirectories, and require the elected compiler version and Go license.

Rationale: binary metadata describes linked modules; the current all-module graph includes unused tooling. Missing Dir currently becomes the checkout root. A module without a source-tree go.mod is valid (flatbuffers +incompatible), so a go.mod declaration alone cannot be the identity proof. Versioned replacements contribute their effective source identity; unversioned local replacements cannot establish a release notice identity and fail explicitly.

Alternatives rejected: full graph downloads add unrelated tooling; omitting missing evidence hides failures; root/LICENSE fallback substitutes project bytes; old slash-to-underscore paths can collide. Native assembly verifies copied bytes. Candidate collection independently compares inventory membership with binaries and requires inventoried notice paths, without depending on absent source caches. Add elected setup-go to the candidate job for cross-platform metadata reads; it never executes those binaries.

References: [Go build information](https://pkg.go.dev/runtime/debug#BuildInfo), [Go commands](https://pkg.go.dev/cmd/go), [module cache and replacement identity](https://go.dev/ref/mod).

## Elected runtime update

Decision: import Dependabot #87 at `576ef85aacbf1bcac1682ec898e56f29624d9869`, x/net 0.61.0 only. Its two checksum changes and go.mod version change are the entire captured diff. Original PR has no comments/reviews/threads at research time. Existing Go 1.27.1 and native pins suffice.

Rationale: Insonic uses x/net's HTML tokenizer. Upstream's QUIC frame handling change is not exercised by ordinary HTTP/1 fixtures; qualify existing offline/tokenizer/backend/credential and native lanes without claiming QUIC proof.

Alternative rejected: merging the source bot independently would split one approved slice. Keep it open until owner merges consolidation. [Upstream comparison](https://github.com/golang/net/compare/v0.60.0...v0.61.0).

## Mermaid and KaTeX

Decision: exact Mermaid 12.2.0 direct election and resolved KaTeX 0.18.2 lock, preserving Next 16.4.0/React 19.3.0. Mermaid's KaTeX range is ^0.18.0, so inspect the actual lock rather than assuming its minimum is patched. Add a behavior regression using inherited trust options without mutating global prototypes; inherited trust must be ignored and an explicit own true remains functional. Retain strict rendering configuration, local runtime bundling and upstream notice closure.

Rationale: GHSA-238p-pmpm-9mq7 is one low advisory reported through KaTeX and Mermaid. KaTeX does not itself pollute prototypes; the issue needs already-polluted or attacker-controlled option prototypes. Installed 0.16.47 accepts Object.create({trust:true}), while ordinary options refuse. Do not invent product policy or claim a demonstrated application exploit.

Add bounded hidden headless browser qualification of real exported documentation diagram SVGs and light/dark rerendering using the built local runtime and CSS for both public and offline exports. Require a browser; fail rather than silently skip. Reuse Chrome/Edge on Windows and installed Chrome/Chromium on hosted Linux, with private temporary profile, redirected I/O and ProcessTree cleanup. Keep existing page/link/schema/historical checks.

Alternatives rejected: audit-only assertions do not prove patched behavior; source-string tests do not exercise real SVG/theme compatibility; an extra browser npm dependency is unnecessary for headless dump-dom qualification.

References: [KaTeX advisory](https://github.com/advisories/GHSA-238p-pmpm-9mq7), [Mermaid 12.2.0 release](https://github.com/mermaid-js/mermaid/releases/tag/mermaid%4012.2.0).

## Quality review

Notice research passes CHK001-004, CHK006, CHK008-010. Renderer behavior and portability research resolve CHK005 and CHK007; parent review confirms all ten written requirements specify measurable scope, preserved boundaries and meaningful validation. These are requirements-writing assessments only.

## Second-review cache integrity correction (2026-10-13)

The P1 finding demonstrates that recorded Go sums and directory identity do not authenticate currently cached bytes. Decision: hash only each selected effective source directory using the Go h1 format and compare directly with its binary Sum, then write the notice bytes captured during that verified hash. This avoids dependence on a potentially altered cached ziphash, unrelated tooling graph traversal/downloads and a later source reread. A real isolated file-proxy/Go-build fixture proves the recorded metadata stays unchanged after license tampering; the original collector accepted it and the corrected collector rejects it. A post-verification mutation fixture proves copied notice bytes stay at the verified snapshot.

Alternatives considered: go mod verify is trusted but verifies the whole selected graph rather than only linked sources; copied-file equality alone verifies no upstream identity. The format preserves logical module/version paths inside the hash, even though cache directory names are escaped. [Go directory hash implementation](https://raw.githubusercontent.com/golang/mod/v0.40.0/sumdb/dirhash/hash.go), [Go cache verification](https://raw.githubusercontent.com/golang/go/go1.27.1/src/cmd/go/internal/modcmd/verify.go).
