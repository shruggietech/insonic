# Verification: Release pipeline reliability

## 2026-10-15: local implementation and convergence

Baseline is repaired main `8a466652aa2ed36c6b8aa694fb13cf0dd95fa860`.
Spec Kit specify, clarify, checklist-template resolution, plan, delegated
research, tasks, read-only analyze and implement completed in sequence. No
extension hooks are configured. Analysis has full FR coverage and no critical
or high findings. Convergence found one resolved-URL coverage gap (T014), which
was appended, implemented and verified. Final buildable requirement coverage is
complete; CI/review delivery remains the explicit pending acceptance gate.

| Check | Result |
| --- | --- |
| Root `npm test` | 104 passed |
| Python discovery `*_test.py` | 278 passed |
| Site tests | 10 passed |
| Public and offline site build | Passed; each export's 111 documentation pages validated |
| `npm run check` | Passed; repository, links, versions, YAML/JSON, schemas and brand bytes |
| New Python text integrity | UTF-8 without BOM, LF and no detected mojibake |
| `git diff --check` | Passed |

Initial root failures selected remote action paths; selectors now inspect the
same cache inputs on local actions. One initial existing media-transfer test
hit Windows filesystem access denial; the unchanged test passed in the complete
rerun. Initial absent-tool failure fixtures preceded bootstrap implementation.
Focused tests cover transient recovery/exhaustion, deadlines, TLS/auth failures,
redirect confinement, digest/size/path/archive rejection, deployment races and
stale public responses. No security or tenancy check was weakened.

## 2026-10-15: read-only live validation

Both approved official action archives were independently retrieved again and
matched the committed locks. Complete trees were staged without execution;
normal action main/post behavior is reserved for disposable hosted CI proof.

The actual `release.py verify-public` command validated all nine original local
v1.0.1 assets against their package/installer/documentation contracts, exact
public release/tag identity and full API asset digests/sizes. It compared 290
live URLs representing 212 unique original archive files in a 42.204-second
content phase. The original archive contains 813 files. Checked URLs include
archived HTML, schemas, stable aliases, navigation destinations and directly
referenced resources. This is byte verification, not a visual or assistive
technology assessment.

- Product source/tag: `136f5c0307a6fce00ba8240ef69c907a944cd8c0`.
- Exact Pages branch and completed build, before and after content requests:
  `85b1476d86a9bfe8fbaac159b4df98226bc2393a`.
- Public release: <https://github.com/shruggietech/insonic/releases/tag/v1.0.1>.
- Stable documentation: <https://insonic.io/docs/latest/index.html>.

The first anonymous Pages metadata read failed closed because that API needs
authenticated Pages read access. With the existing credential held only in
memory, the GET-only command passed. No credential was logged or persisted.
The release, tag and deployment were not changed. No application, MSI, personal
browser or desktop interaction was used.

## Delivery boundary

The official PR records final head, measured CI turnaround and every review
thread. Opening review is round one; at most one further review request is
authorized. Stop for owner final review and merge. Green PR checks do not
establish the later squash commit's main-branch checks.
