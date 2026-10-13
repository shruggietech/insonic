# Shared S022 contracts

Pre-owner workspace plan/preflight/provision operations accept the same complete backend-plan in CLI and native desktop bridge. Protected secret input is separate, excluded from persisted plans and diagnostics. Provision is explicit; preflight never provisions. Existing workspace config inspection exposes exact generated profile identities/revisions.

Runtime operations query.services and query.assistance-test use versioned data envelopes, existing workspace/request identity, aggregate contexts and bounded payloads. Service discovery returns observations only. Assistance-set elects adapter/endpoint/model/route at expected_revision. Native assistance and optional synthetic tests retain strict proposal validation; normal assist additionally performs the existing backend validation, query budget and revision fencing.
