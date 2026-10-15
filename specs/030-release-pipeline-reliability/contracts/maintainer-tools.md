# Maintainer tools

## Acquisition

`python scripts/bootstrap-actions.py setup-go --output build/actions/setup-go`

Allowlisted action, adjacent immutable lock, no authentication, <=3 attempts and
60 seconds total. Verify digest before confined complete-tree extraction. Refuse
existing destinations. Emit action/revision/digest/attempt count; errors exit 1.

## Public verification

`python scripts/release.py verify-public --candidate DIR --source-root DIR --site https://insonic.io/ --pages-commit SHA`

Reuse local candidate verification and GET-only release/tag/asset identity.
Original docs must be clean release-target bytes. HTTP phase: <=120 seconds,
<=3 attempts/request, <=5 redirects/request and 16 MiB/response. HTTPS same-origin
redirects only; no authorization/cookies. Optional output stores a JSON receipt.
Omit Pages commit for explicit generic-site content proof without Pages proof.
All mismatches exit nonzero. No external writes or publication.
