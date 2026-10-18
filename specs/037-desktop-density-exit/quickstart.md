# S037 validation guide

1. Run required `npm run check` and `npm test`, frontend `npm --prefix desktop/frontend run check` / `test`, Python maintainer tests and affected Go desktop/lifecycle tests through hidden owned launchers.
2. Use the existing native dependency/toolchain preparation. Build current CLI and capture bridge from clean source; run `python scripts/qualify-ui-capture.py --output build/ui-capture`. Expect all declared density scenarios and source-bound images, with no external requests.
3. Run native desktop qualification on disposable Windows/Linux CI. Expect all previous journey flags plus compact geometry and native Exit/durable-survival receipts. Do not launch native UI on the owner's desktop for this proof.
4. Copy only validated clean-source capture images and original receipt into the versioned screenshot directory, update their public guide, and run affected site checks/build.
5. Push and inspect exact-head CI, resolve at most two review rounds, verify all threads closed, and leave the PR ready and unmerged for owner review.
