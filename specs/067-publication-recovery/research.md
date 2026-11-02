# S067 Research

## Durable identity and exact recovery

Decision: extend the existing Pages promotion path with phase observations and atomic receipts; provide separate read-only reconciliation.

Rationale: pages.py currently loses the staged identity when its immediate ref read disagrees. website.py already records on-staged identity, but its truncating writer is unsuitable for preserving the last complete failure record. The release workflow currently retains no promotion receipt.

Alternatives: repeat publication or rebuild (rejected because original bytes/identity are authority); retry arbitrary heads (rejected because it conceals concurrent writers); new deployment subsystem (unnecessary).

Preserve original old-head and failed-request categories without claiming their cause. Validate commit ancestry except the explicit no-op case where staged equals previous. Always upload receipt evidence, including failure, with run attempt in artifact name.

## Timing authority

Decision: document actual executed ceilings from policy and callers, not similarly named unused constants.

Rationale: website.py Git uses subprocess_seconds=600; public verification network phase is 600 with request_seconds=120 clipped to remaining time; live browser is 900; Pages aggregate is 3600; website worker ceiling is 5430 with 30-second cleanup. bootstrap-actions uses 600 seconds for download and extraction.

Alternative: change runtime policy to match stale prose (rejected; #240 requests truthful guidance, not narrower qualification).

Research used the two read-only agents required by speckit-plan. No new dependency, external API contract or unresolved architectural choice.
