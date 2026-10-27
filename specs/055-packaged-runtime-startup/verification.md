# S055 verification

## 2026-10-27: Local implementation

- Blocking Spec Kit analysis: eight functional requirements and four success criteria covered; no constitution/scope findings. Requirements checklist 8/8.
- Before repair, real IPC/held-owner regressions failed with one unnecessary startup launch and startup_launch. After repair, focused startup/cold-race/input/deadline tests passed. Owner release permits replacement and preserves actual bootstrap failure.
- Full go test ./... and go vet ./... passed with Go 1.27.1. Repository check passed; root tests 206/206. Python qualification tests: 663, three declared skips. Focused headless terminal/evidence tests: 21/21.
- Documentation checks, 18 site tests and 140-page public/offline export passed.
- Initial local Python checks exposed unselected compiler and inherited C.UTF-8 locale prerequisites; selecting pinned Go PATH and C locale resolved both without product changes. One root-test inventory omission was corrected by classifying the new evidence helper.
- Existing main failure remains [durably retained](https://github.com/shruggietech/insonic/pull/190#issuecomment-6452908340). The reproduced arbitration defect is established; its occurrence and native errno in that historical failure remain unproven.

## Delivery acceptance

- Exact-head CI and independent terminal timing/artifact validation pending.
- Codex review rounds used: 0 of 2. All actionable external findings must be resolved.
- Owner merge reserved. No tag, release, deployment or merged-main acceptance is claimed.

## 2026-10-27: First CI attempt and correction

- CI 40586738671 at PR head e31ddda (tested merge 6fd62da) failed the Windows archive timeout case: actual exit 1 / generic cancellation, expected exit 4 / wait_timeout; work and its owner remained running. This is distinct from the original startup refusal.
- Independent rejected audit: 220 s preparation + 604 s qualification = 824 s combined. Eight original ZIP sizes/digests and five completed terminal receipts / 40 cases verified. [Complete durable evidence](https://github.com/shruggietech/insonic/pull/192#issuecomment-6454336247) was recovered byte-for-byte before another run.
- Controlled virtual-time regressions failed before the deadline-notification repair and passed afterward. Affected command/runtime tests, vet, repository checks and all 206 root tests passed. Existing clocks and eight-case acceptance are unchanged.
- First Codex review completed without findings on e31ddda. The deadline correction requires the remaining authorized round; no third round may be requested.

## 2026-10-27: Corrected product proof and setup enumeration

- CI 40589369443 at PR head c9f3948 (tested merge 8813ae3, matching tracked tree) passed the Windows desktop archive campaign. All six source/package terminal receipts and 48 cases independently validated.
- Overall acceptance still failed: setup stopped before product work in wait-inputs with an incomplete job inventory; join/candidate/aggregate correctly failed downstream. Its exact page counts are unknown.
- [Second complete retained audit](https://github.com/shruggietech/insonic/pull/192#issuecomment-6454665442): 258 s preparation + 556 s qualification = 814 s combined; eight original ZIP sizes/digests verified and durable payload recovered byte-for-byte before another run.
- Four new inventory regressions failed before the bounded resnapshot repair. All 24 acceptance-helper tests and all 210 root tests passed afterward, along with repository/schema checks. Strict final inventories, attempt/readiness identity, fixed deadline, duplicate refusal and finite retry bounds remain enforced.
