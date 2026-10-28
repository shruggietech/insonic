# Validation: Import interruption recovery

1. Prepare the repository's pinned media tools and normal native prerequisites using the documented development workflow. Windows tooling must use the existing hidden ProcessTree launcher.
2. Run native library recovery regressions with `INSONIC_LIBRARY_TOOLS_FILE` selecting the verified `build/native/media-tools.json`.
3. Run `python -m unittest discover -s tests -p '*recovery*test.py'` and `python -m unittest discover -s tests -p 'file_open_barrier_test.py'`, followed by the full maintained Python suite.
4. Run `npm run check`, `npm test` and affected site tests/build.
5. Run normal source CLI qualification (`python scripts/qualify.py cli --native`) and extracted CLI/desktop package qualification. The shared contract now includes import recovery, retaining each context's actual image identity.
6. Require all three actual interruption scenarios and strict receipt validation. Inspect exact-source terminal CI receipts independently, retain compact audited evidence outside workflow storage, and remove owned temporary staging after readback verification.
7. Resolve all review findings through at most two Codex rounds. Present the unmerged PR with exact final head, check/review state, evidence and any remaining limits.

Expected same-source result: one recording, original candidate bytes/captured provenance, distinguishable attempts and duplicate-free replay. Changed-source/corrupt-report cases refuse stale recovery. No owner libraries or installed packages are modified.
