# Validation

1. Use pinned Go 1.27.1 and hidden noninteractive tooling. Install root/site locks with bounded audited npm requests and ignored install scripts.
2. After selected Go upgrades and tidy, explicitly download the two pinned pristine replacement upstreams, then run source/notice verification; missing and altered checksum regressions must fail.
3. Run root npm check/test, Python unittest discovery, site check/test/build/export and affected native Go tests/vet. Focused capture-pin tests require no browser or owner GUI.
4. Push the candidate and inspect complete exact-head Windows/Linux CI, all package scopes and CLI-corpus receipts. Verify terminal preparation/qualification/combined timing and resolve bounded reviews.
5. Reconcile #168 and superseded #157-#159 only after qualification; stop with the replacement PR unmerged.

## Local verification

Root foundation/schema checks, all 181 Node tests, all 17 site tests and a fresh 111-page public/offline export pass. All 518 Python tests pass in 203.063 seconds with an owned process tree and a complete isolated profile. A prior parallel run had timed out and the original environment lacked APPDATA; neither is reported as passing.

Go vet passes. With the complete profile, all Go packages pass except the unchanged Windows process timing suite on this host. The baseline and refreshed process test binaries are byte-identical (SHA-256 7df2ed8a35546951c295515a1bff34ecc4a0b691d5c55b9ce1665600bf1f22b8), and the full baseline suite reproduces the same deadline failures. Their only external module is unchanged golang.org/x/sys v0.49.0. Required full hosted Go/native/package checks remain authoritative and cannot be waived.

## Hosted acceptance and reviews

[Full CI 39827567369](https://github.com/shruggietech/insonic/actions/runs/39827567369) passed on head 064070d3f8808247e84992969097045cd0d86bf6 (merge source 9b31bde053459dec42c7d98e2626c216422d3725): 30 successful jobs, 18 legitimate skips. Terminal policy accepted 530s preparation, 703s qualification and 1233s total, with a 43s advisory overrun and no hard-limit breach. Preparation includes the superseded-run scheduling wait. The [acceptance artifact](https://github.com/shruggietech/insonic/actions/runs/39827567369/artifacts/12447819227) and [complete candidate](https://github.com/shruggietech/insonic/actions/runs/39827567369/artifacts/12448986209) retain canonical evidence. Hosted Windows core passes the complete Go suite, including internal/process, and all 518 Python tests. No local timing failure or cancelled obsolete run is represented as a pass.

Both authorized Codex rounds completed. The first P1 about module-file checksum enforcement was reproduced, fixed, replied to and resolved; the second round reviewed 064070d and found no major issues. No third round was requested. #157-#159 are closed as superseded by the unmerged replacement [PR #169](https://github.com/shruggietech/insonic/pull/169). That PR records final planning-closeout head checks and terminal timing. #147 and #154 retain their outstanding criteria. No owner-desktop GUI or release publication was performed.
