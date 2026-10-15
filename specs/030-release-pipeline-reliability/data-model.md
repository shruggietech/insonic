# Data model

## Approved action

Key is setup-go or setup-node, with immutable 40-character lowercase revision
and archive SHA256. Official repository is fixed to actions/<key>. Code-owned
limits bound acquisition and extraction independently of lock contents.

State: absent -> acquisition -> digest verification -> confined staging ->
atomic destination. Failures remove staging and expose no executable action.

## Publication verification

Candidate identifies repository/tag/version/source and complete asset inventory.
Original public archive provides expected bytes. An elected Pages commit must
match current branch and completed build both before and after HTTP checks.
Receipt records identity, exact checked paths/digests, explicit proof levels and
time. It contains no tokens, headers or credential-bearing URLs.
