# Data model: Qualification timing

## Stage policy

Each key ends in `_seconds`, has a positive finite integer value and names a successful-execution allowance. Consumers convert units exactly once. Qualification-only callers can use explicit smaller injected clocks for failure tests; ordinary product settings are separate.

## Aggregate policy

The existing three fields remain `qualification_target_seconds=1800`, `qualification_limit_seconds=3600`, `finalization_reserve_seconds=60`. Target is below hard limit, reserve is positive and below it. Terminal accounting uses zero provisional reserve and authoritative conservative completion timestamps.

## Deadline inventory

Entries identify source owner, prior/effective policy, caller, origin, parent/shared cap, platform and concrete retained-bound exceptions. Poll cadence, negative refusal probes, injected expiry fixtures and ordinary production contracts are distinct dispositions.

## Help observation

A fixed allowlisted stage, nonnegative monotonic elapsed duration, effective test bounds and terminal outcome. Native receive time and frontend elapsed time are independent observations; no private values/freeform input are retained. Forced cleanup never transitions failure to success.
