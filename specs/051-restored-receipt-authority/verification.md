# S051 verification

## 2026-10-27 planning and test-first repair

- Spec Kit specify/clarify/plan/tasks/analyze completed: eight FRs, three SCs, six acceptance scenarios, nine traceable tasks, 16/16 built-in quality checklist, five constitution principles and no blocking analysis findings. No extension hooks or custom reviewer-owned checklist.
- Test-first authority mutations reproduced 113 accepted-malformation failures and 16 wrong-exception errors before repair. Corrected full restored suite passes 10 tests including 157 new negative authority cases and valid newer-prior/legacy controls. Focused journey suites pass 41 tests.
- The corrected validator requires canonical unique identities and exact positive integer revisions across all source cases, including enrollment. Destination equality and explicit source <= prior < new ordering remain. Legacy source-only v1 uses its existing validation path.

## 2026-10-27 retained actual evidence revalidation

Recovered [original successful CPU receipt](https://github.com/shruggietech/insonic/pull/179#issuecomment-6434152258), verified 97274 exact bytes with SHA-256 e45118f36954a8486cb65289fdd3a68bfb9cb3f7906dc9127c93d667889c3e29 and validated unchanged against the frozen corpus. Current validator SHA-256 0790d47ac899d7f9a1a7e2e120a52fb49ab7e30870af4fd548a2c90040155ce5; validation took 0.016s. Six source/four destination cases, eight fresh terminal works, twelve preserved history entries and original cleanup pass.

Original package/harness revision remains 9183e083327a5298e7d24c2e1e10ebff8a0e6a98; its source tree matched merged S050 ae8c608f15594fec5ded2f49919160a6258d8cec. Original preparation 214.610s + qualification 938.421s = combined 1153.031s. Revalidation is not a new inference run. S050 records now include a dated reconciliation of actual acceptance, CI and owner merge before final convergence/handoff.

## Remaining delivery

Full local checks, final convergence, final-head CI/reviews, compact durable evidence retention and unmerged owner handoff remain required. Broad #147 remains open.

## 2026-10-27 local verification environment

The first site test run passed 14/17; three owned Git-history fixtures were denied writes to their temporary .git/config inside the restricted runner. No site implementation failure was established. Rerun the same suite through the approved hidden launcher outside that filesystem restriction.

## 2026-10-27 local acceptance and convergence

- Full Python 618 tests pass in 68.041s with two declared skips; root 205/205 and site 17/17 pass. Site fixtures passed unchanged outside the restricted temporary Git-directory boundary. Public/offline build validates 111 exported pages. Root encoding/BOM/LF/link/version/upstream checks plus 25 schemas, 22 contracts and 30 examples pass.
- S051 implementation convergence checked eight FRs, three SCs, six acceptance scenarios, seven plan decisions and five constitution principles: no additional implementation gaps; tasks were unchanged during convergence. T008/T009 remain delivery gates. S050 current implementation requirements now include strict authority refusal; its historical owner-merge/handoff limitation remains explicitly reconciled rather than rewritten.

## 2026-10-27 initial code-head CI and delivery finalization

Initial head `f103bf9469535bdc9341f02ca16064529ad2b409` and test-merge `6f00e4f6fb7c371adaea08b96f0a2b137ac79c15` passed [CI 40455412727](https://github.com/shruggietech/insonic/actions/runs/40455412727): 30 successful jobs, 18 expected skips and every elected result. Independent terminal audit: preparation 178s + qualification 646s = combined 824s; no target overrun. Readiness 05:30:22Z, worker terminal 05:41:07Z, metadata 05:41:08Z on 2026-10-27. [Compact audit](https://github.com/shruggietech/insonic/pull/181#issuecomment-6434734054) is retained before another full qualification. Codex round one completed on this head without findings or review threads.

The finalization commit changes only internal delivery records; validator SHA-256 `0790d47ac899d7f9a1a7e2e120a52fb49ab7e30870af4fd548a2c90040155ce5` and tests remain unchanged. Final-head CI and the second/final Codex review are live handoff gates; terminal audits are retained on PR #181 before readiness is announced. Issue #180 and milestone 35 remain open, with project status In review and auto-merge disabled. Owned temporary inputs are removed after evidence readback. Owner libraries, the installed release and Downloads remain unchanged.
