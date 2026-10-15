# S031 verification

## 2026-10-15: specification and local verification

Spec Kit specify, clarify, checklist, plan, tasks, analyze and implementation prerequisites completed for `specs/031-landing-download-clarity` on `codex/031-landing-download-clarity`, scope #101, milestone 14 and Project slice S031. No extension hooks are installed. The analysis gate has no blocking finding.

Rendered regression checks first failed for the original ambiguous package labels and five competing primary actions. The implementation passes all four new rendered-state cases. Geometry/activation regressions cover undersized/overlapping/touching targets, clipped/obscured labels and exact native-link interception.

Local parity through `scripts/run-hidden.ps1` (CREATE_NO_WINDOW, redirected noninteractive I/O):

- Repository/brand integrity: 1251 text files checked, zero problems.
- Schema check: 25 schemas, 22 contracts and 30 examples, version 1.0.1.
- Root Node tests: 106 passed, zero skipped/failed. Restricted-run symlink EPERM was resolved by the authorized hidden launcher outside that sandbox; isolation tests were retained.
- Site Node tests: 14 passed, zero skipped/failed.
- Full site build: public and offline exports each passed the 111-page relative-link/anchor inventory. No new dependency or kit-byte update.

The five package destinations match the fresh official v1.0.1 API and the original release manifest whose SHA-256 matches that API. See [link inventory](link-inventory.md). No binary download/install or package publication was performed.

## Hosted-browser and delivery acceptance

Initial PR head `f0040d8ac215bf4e94c6f0d2926fbdb2b8c99d49`, hosted run `38740731965`: documentation UI passed in 131.4 seconds (131.2-second qualification phase). It inspected 114 public and 111 offline HTML files, passed 19 case groups, and produced 18 full-page landing captures: eight width/theme cases per state plus two actual 200% text cases. Reported 25 actual Tab focus targets and all 22 native links activated independently through trusted keyboard, pointer and touch inputs. Candidate inventory: 17 links/zero downloads; released fixture: 22 links/five downloads. No browser error, external HTTP request or WebSocket escape; all contexts/server/owned files cleaned up.

Inspected all 16 complete-page viewport/theme captures through full-page comparison sheets and all eight released download-region crops at their original sizes, plus both 200% text crops. Labels wrap without touching or clipping; hierarchy, header/footer and preview remain readable in both themes. Measurements verify minimum 44px controls, at least 12px peer action gaps and non-overlap. 320 CSS pixels is explicitly the reflow equivalent from 1280px at 400%, not a browser zoom UI claim.

Both round-one Codex code and security reviews completed on `f0040d8` without findings; bot thumbs-up and explicit security result observed. Self-review then aligned fixture links with the real postbuild portable URL rewrite and added literal-replacement/link regressions. Final-head hosted CI and the second/final code/security round are required after that change. The 180-second browser and 660-second CI bounds remain unchanged.

No local Windows browser, personal profile or installer was launched. Browser screenshots/report will distinguish same-source released fixtures from real exported state. Neither the automated checks nor image inspection certifies assistive technology or other browser engines. Live deployed inspection follows an authorized deployment after merge; this PR handoff cannot establish deployed behavior.

Review cap: two rounds maximum for this PR. Final exact-head CI/review evidence will be recorded on the PR after the last code commit.

## Convergence

Post-implementation Spec Kit convergence checked all ten functional requirements, five success criteria, three user journeys, plan decisions and constitution constraints against current source and the observed checks. No unbuilt code, contradictory behavior or unrequested addition remains. Existing T013 retains final exact-head CI/review delivery; no duplicate convergence phase/task is appended.
