# S065 research decisions

## D1: Maintained minor-version preparation

Decision: Run scripts/release.py prepare for 1.1.0, carrying all accepted S061-S064 behavior into a candidate. Rationale: the previous release is 1.0.3 and long-recording operations are additive public capabilities. Alternatives: a patch understates scope; a major release implies unnecessary incompatibility. Existing preparation regressions cover ownership, history, version alignment, prior-document support and upstream independence.

## D2: Truthful capability metadata

Decision: Extend current candidate advanced-CLI declarations and matching documentation for long-recording policy/resume, coverage and repair. Rationale: collection currently declares only backup despite documented absent dedicated GUI controls. Alternative: add GUI controls, which would expand the approved release scope. Historical fixtures retain original capabilities.

## D3: Historical authority and status

Decision: Copy forward first, restore historical docs/schemas from v1.0.3, then elect historical released status in docs/versions.json. Rationale: 11 documentation and seven schema modifications plus four new schemas accumulated under the prior current route; they belong in 1.1.0. The original index candidate sentence is projected as released by site/lib/release-status.mjs, so its original source bytes should remain intact. Alternatives: rewrite history or relabel the current snapshots, both of which would misstate released behavior.

## D4: Genuine capture bootstrap

Decision: Omit only new-version screenshot walkthrough sections in a clean bootstrap, dispatch full CI, then import original validated capture bytes and restore version-correct embeds. Rationale: preparation intentionally does not copy/relabel image authority. Four receipts cover 25 application, density, restricted-installer and full-transaction PNGs. Alternatives: old screenshots with new labels or local focus-stealing captures, both rejected by evidence/desktop requirements.

## D5: Qualification and retention

Decision: Keep local checks headless in foreground and use isolated CI for package/native/GUI/installer acceptance. Retain original acceptance/capture archives and metadata, verify their digests, independently audit terminal timing and record preparation/qualification/combined durations. Alternative: rely on expiring workflow artifacts or final job success alone, which omits retained provenance and terminal acceptance.

## D6: Publication boundary

Decision: Push and official PR are already authorized; exactly two external review rounds maximum. Owner merge and later publication remain separate. Real-model CPU/NVIDIA qualification is not newly elected by this preparation request. Retain an executable continuation for an exact accepted-main candidate and publication of those unchanged bytes. Alternatives: publish this pre-merge candidate or claim prior/model fixtures qualify current source, neither supported by authorization or evidence.
