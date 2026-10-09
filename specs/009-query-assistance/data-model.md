# Data model: S009

Assistance settings: one current named catalog setting with expected per-setting revision and receipt-reconciled writes. Configuration fields: enabled, adapter/contract, local/hosted mode, endpoint/model/credential_id, suggest/auto-run mode and finite limits. Old configuration versions carry no content/results. Absent setting has revision zero and disabled defaults.

Assistance request: prompt, optional explicit configuration, optional saved revision fence, optional mode election and optional normalized current context query. Provider response: contract_version, query (complete QueryInput), explanation. Unknown/duplicate fields and trailing JSON fail. Provider cannot provide configuration or execution authority.

Assistance result: proposal, structural/backend validation, settings revision, selected backend/model/adapter/context digest and optional current execution. Proposal/results remain ephemeral; existing query.save persists only definitions. States: disabled -> configured -> generating -> rejected/pending/validated -> optionally executing -> current result/error. Configuration/profile changes cause conflict before accepting provider output.
